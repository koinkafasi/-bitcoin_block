package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Kripto fiyat verisi — CoinGecko (ücretsiz uç).
//
// NEDEN CoinGecko: anahtarsız çalışıyor, 250+ coin tek istekte, sıralama ve
// piyasa değeri hazır. Binance yalnız saatlik hacim verir ve coin adı/logosu
// yoktur; CoinDesk'in fiyat bandı için sembol + ad + yüzde gerekir.
//
// ÖNBELLEK: 60 saniye (veritabanında `services_cache`, kind `fiyat`).
// CoinGecko ücretsiz katmanı ~30 istek/dk; sayfa başına istek atılırsa
// kolayca aşılır. Tek satır önbellek tüm sayfaları besler.
const (
	fiyatCacheKind = "fiyat"
	fiyatCacheKey  = "coingecko"
	fiyatTTL       = 60 * time.Second
	fiyatURL       = "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd&order=market_cap_desc&per_page=25&page=1&sparkline=false&price_change_percentage=24h"
)

// FiyatSatiri tek bir kripto varlığın fiyat satırıdır.
type FiyatSatiri struct {
	Rank    int     `json:"rank"`
	Sym     string  `json:"sym"`
	Name    string  `json:"ad"`
	Price   float64 `json:"fiyat"`
	Change  float64 `json:"degisim"` // 24s yüzde
	HasData bool    `json:"veri"`
}

// fiyatCache zarfı: zaman + veri. TTL kararı `cekildi` alanına bakar.
type fiyatCache struct {
	// Cekildi UNIX saniye. DB'de updated_at de var ama okuma anında
	// saat dilimi/kayma sorunlarına girmemek için gövdede de tutulur.
	Cekildi  int64         `json:"cekildi"`
	Satirlar []FiyatSatiri `json:"satirlar"`
}

// Fiyatlar kripto fiyat satırlarını döndürür.
//
// Önbellek taze değilse CoinGecko'ya gidilir; oradan da veri gelmezse
// ELDEKİ (eski) veri döndürülür. Boş liste döndürmek fiyat bandını
// kaybettirir; eski fiyat yeni fiyattan iyidir (ve şerit saatini gösterir).
func Fiyatlar(ctx context.Context, pool DB) []FiyatSatiri {
	var mevcut fiyatCache
	if b, err := CacheGet(ctx, pool, fiyatCacheKind, fiyatCacheKey); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &mevcut)
		if time.Since(time.Unix(mevcut.Cekildi, 0)) < fiyatTTL && len(mevcut.Satirlar) > 0 {
			return mevcut.Satirlar
		}
	}

	satirlar, err := fiyatCek(ctx)
	if err != nil || len(satirlar) == 0 {
		return mevcut.Satirlar
	}
	if b, err := json.Marshal(fiyatCache{Cekildi: time.Now().Unix(), Satirlar: satirlar}); err == nil {
		_ = CacheSet(ctx, pool, fiyatCacheKind, fiyatCacheKey, b)
	}
	return satirlar
}

// fiyatCek CoinGecko'dan fiyatları çeker.
func fiyatCek(ctx context.Context) ([]FiyatSatiri, error) {
	istek, err := http.NewRequestWithContext(ctx, http.MethodGet, fiyatURL, nil)
	if err != nil {
		return nil, err
	}
	// CoinGecko UA'sız istekleri sınırlıyor; tanıtıcı UA zorunlu.
	istek.Header.Set("User-Agent", "bitcosar/1.0 (+https://bitcosar.com)")
	istek.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(istek)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko: HTTP %d", resp.StatusCode)
	}
	govde, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var ham []struct {
		Symbol         string  `json:"symbol"`
		Name           string  `json:"name"`
		CurrentPrice   float64 `json:"current_price"`
		MarketCapRank  int     `json:"market_cap_rank"`
		PriceChange24h float64 `json:"price_change_percentage_24h"`
	}
	if err := json.Unmarshal(govde, &ham); err != nil {
		return nil, fmt.Errorf("coingecko: JSON: %w", err)
	}

	out := make([]FiyatSatiri, 0, len(ham))
	for i, h := range ham {
		sym := strings.ToUpper(strings.TrimSpace(h.Symbol))
		if sym == "" {
			continue
		}
		sira := h.MarketCapRank
		if sira <= 0 {
			sira = i + 1
		}
		out = append(out, FiyatSatiri{
			Rank: sira, Sym: sym, Name: h.Name,
			Price: h.CurrentPrice, Change: h.PriceChange24h, HasData: true,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out, nil
}

// FiyatBicim fiyatı okunur metne çevirir.
//
// Kural (CoinDesk ölçümü): 1 USD üstü 2 ondalık; 1 USD altı anlamlı
// basamak korunur (0.00001234 gibi fiyatlar kaybolmasın).
func FiyatBicim(f float64) string {
	switch {
	case f <= 0:
		return "—"
	case f >= 1000:
		return "$" + binlikAyrac(f, 2)
	case f >= 1:
		return "$" + binlikAyrac(f, 2)
	case f >= 0.01:
		return "$" + binlikAyrac(f, 4)
	default:
		// Çok küçük: 6 anlamlı basamak.
		return "$" + strconv.FormatFloat(f, 'f', 6, 64)
	}
}

// binlikAyrac 1234567.89 → "1,234,567.89".
func binlikAyrac(f float64, ondalik int) string {
	s := strconv.FormatFloat(f, 'f', ondalik, 64)
	parca := strings.SplitN(s, ".", 2)
	tam := parca[0]
	var b strings.Builder
	for i, r := range tam {
		if i > 0 && (len(tam)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if len(parca) == 2 {
		return b.String() + "." + parca[1]
	}
	return b.String()
}

// DegisimBicim +3.48% / -1.42%.
func DegisimBicim(f float64) string {
	isaret := "+"
	if f < 0 {
		isaret = ""
	}
	return isaret + strconv.FormatFloat(f, 'f', 2, 64) + "%"
}
