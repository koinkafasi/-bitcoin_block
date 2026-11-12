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
	"sync"
	"time"
)

const (
	fiyatMetaKind = "fiyat"
	fiyatMetaKey  = "coingecko"
	// Meta TTL'i: ad + sıra. Uzun tutuluyor ki 429 bizi etkilemesin.
	fiyatMetaTTL = 30 * time.Minute

	fiyatURL = "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd&order=market_cap_desc&per_page=25&page=1" +
		"&sparkline=false&price_change_percentage=24h"
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

func Fiyatlar(ctx context.Context, pool DB) []FiyatSatiri {
	satirlar, _ := FiyatlarZamanli(ctx, pool)
	return satirlar
}

func FiyatlarZamanli(ctx context.Context, pool DB) ([]FiyatSatiri, int64) {
	meta, metaCekildi := fiyatMetaOku(ctx, pool)

	// Canlı fiyat: Binance.
	syms := make([]string, 0, len(meta))
	for _, m := range meta {
		syms = append(syms, m.Sym)
	}
	canli, canliCekildi := fiyatCanliOku(ctx, pool, syms)

	if len(canli) == 0 {
		// Binance yok: meta zaten CoinGecko fiyatlarını taşıyor.
		return meta, metaCekildi
	}
	return fiyatBirlestir(meta, canli), canliCekildi
}

func fiyatBirlestir(meta []FiyatSatiri, canli map[string]binanceFiyat) []FiyatSatiri {
	out := make([]FiyatSatiri, 0, len(meta))
	for _, m := range meta {
		if c, ok := canli[m.Sym]; ok && c.Fiyat > 0 {
			m.Price = c.Fiyat
			m.Change = c.Degisim
			m.HasData = true
		}
		out = append(out, m)
	}
	return out
}

func fiyatCanliOku(ctx context.Context, pool DB, syms []string) (map[string]binanceFiyat, int64) {
	type canliCache struct {
		Cekildi int64                   `json:"cekildi"`
		Fiyat   map[string]binanceFiyat `json:"fiyat"`
	}
	var mevcut canliCache
	if b, err := CacheGet(ctx, pool, "fiyat", "binance"); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &mevcut)
	}
	taze := time.Since(time.Unix(mevcut.Cekildi, 0)) < binanceTTL
	if taze && len(mevcut.Fiyat) > 0 {
		return mevcut.Fiyat, mevcut.Cekildi
	}

	// Tazeleme hakkı: yalnız bir istek çeker; diğerleri eldekiyle döner.
	if !fiyatKilit.Take() {
		return mevcut.Fiyat, mevcut.Cekildi
	}
	defer fiyatKilit.Release()

	if len(syms) == 0 {
		return mevcut.Fiyat, mevcut.Cekildi
	}
	tazeFiyat := binanceFiyatlar(ctx, pool, syms)
	if len(tazeFiyat) == 0 {
		return mevcut.Fiyat, mevcut.Cekildi
	}
	cekildi := time.Now().Unix()
	if b, err := json.Marshal(canliCache{Cekildi: cekildi, Fiyat: tazeFiyat}); err == nil {
		_ = CacheSet(ctx, pool, "fiyat", "binance", b)
	}
	return tazeFiyat, cekildi
}

func fiyatMetaOku(ctx context.Context, pool DB) ([]FiyatSatiri, int64) {
	var mevcut fiyatCache
	var varMi bool
	if b, err := CacheGet(ctx, pool, fiyatMetaKind, fiyatMetaKey); err == nil && len(b) > 0 {
		if json.Unmarshal(b, &mevcut) == nil && len(mevcut.Satirlar) > 0 {
			varMi = true
		}
	}
	if varMi && time.Since(time.Unix(mevcut.Cekildi, 0)) < fiyatMetaTTL {
		return mevcut.Satirlar, mevcut.Cekildi
	}

	if !varMi {
		// Hiç liste yok: ilk açılış, beklemek zorundayız.
		satirlar, err := fiyatCek(ctx)
		if err != nil || len(satirlar) == 0 {
			return nil, 0
		}
		if b, err := json.Marshal(fiyatCache{Cekildi: time.Now().Unix(), Satirlar: satirlar}); err == nil {
			_ = CacheSet(ctx, pool, fiyatMetaKind, fiyatMetaKey, b)
		}
		return satirlar, time.Now().Unix()
	}

	fiyatMetaArkaPlandaYenile(pool)
	return mevcut.Satirlar, mevcut.Cekildi
}

var (
	fiyatMetaKilit       sync.Mutex
	fiyatMetaYenileniyor bool
)

func fiyatMetaArkaPlandaYenile(pool DB) {
	fiyatMetaKilit.Lock()
	if fiyatMetaYenileniyor {
		fiyatMetaKilit.Unlock()
		return
	}
	fiyatMetaYenileniyor = true
	fiyatMetaKilit.Unlock()

	go func() {
		defer func() {
			fiyatMetaKilit.Lock()
			fiyatMetaYenileniyor = false
			fiyatMetaKilit.Unlock()
		}()
		ctx, iptal := context.WithTimeout(context.Background(), 20*time.Second)
		defer iptal()
		satirlar, err := fiyatCek(ctx)
		if err != nil || len(satirlar) == 0 {
			return
		}
		if b, err := json.Marshal(fiyatCache{Cekildi: time.Now().Unix(), Satirlar: satirlar}); err == nil {
			_ = CacheSet(ctx, pool, fiyatMetaKind, fiyatMetaKey, b)
		}
	}()
}

type tekUcus struct {
	kanal chan struct{}
}

func (t *tekUcus) Take() bool {
	select {
	case t.kanal <- struct{}{}:
		return true
	default:
		return false
	}
}

func (t *tekUcus) Release() { <-t.kanal }

var fiyatKilit = &tekUcus{kanal: make(chan struct{}, 1)}

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

func FiyatBicimP(f float64, para string) string {
	if f <= 0 {
		return "—"
	}
	if para == "TRY" {
		return "₺" + binlikAyrac(f, 2)
	}
	return FiyatBicim(f)
}

func SayiBicim(f float64, ondalik int) string {
	return binlikAyrac(f, ondalik)
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
