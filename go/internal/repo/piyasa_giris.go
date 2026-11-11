package repo

// Ana sayfa "giriş hero"su için GERÇEK piyasa verisi.
//
// NEDEN AYRI DOSYA: `fiyat.go` fiyat bandını (25 coin) besliyor; buradaki
// veri farklı uçlardan geliyor ve farklı TTL istiyor. Aynı fonksiyona
// sıkıştırmak, fiyat bandını 10 dakikalık önbelleğe mahkûm ederdi.
//
// KAYNAKLAR (hepsi ücretsiz, anahtarsız — 11 Kasım 2026'da canlı sınandı):
//   - CoinGecko /global          → 24s hacim, BTC hakimiyeti
//   - CoinGecko market_chart     → BTC 30 günlük kapanış (sparkline + RSI)
//   - alternative.me /fng        → Korku & Açgözlülük endeksi
//   - CoinGecko coins/markets    → BTC/ETH/SOL fiyat + 24s değişim (repo.Fiyatlar)
//
// KURAL: Veri gelmezse alan BOŞ kalır ve şablon o parçayı basmaz. Uydurma
// sayı yazmak yasak (kullanıcı isteği, tüm proje boyunca geçerli).
//
// ÖNBELLEK: 10 dakika. CoinGecko ücretsiz katmanı ~30 istek/dk; giriş
// hero'su her ana sayfa açılışında 3 istek atsaydı kolayca aşılırdı.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const (
	girisCacheKind = "giris"
	girisCacheKey  = "piyasa"
	girisTTL       = 10 * time.Minute

	girisGlobalURL = "https://api.coingecko.com/api/v3/global"
	// days=30 + interval=daily: sparkline son 7 günü kullanır, RSI(14)
	// için 15 kapanış gerekir. 30 gün ikisini de tek istekte karşılıyor.
	girisGrafikURL = "https://api.coingecko.com/api/v3/coins/bitcoin/market_chart" +
		"?vs_currency=usd&days=30&interval=daily"
	girisFNGURL = "https://api.alternative.me/fng/?limit=8"
)

// PiyasaGiris ana sayfa hero'sunun ham verisidir.
//
// Sıfır değer "veri yok" demektir; şablon bunu görüp modülü basmaz.
type PiyasaGiris struct {
	BTCFiyat   float64 `json:"btc_fiyat"`
	BTC24      float64 `json:"btc_24"`
	BTCYol     string  `json:"btc_yol"` // sparkline çizgi (viewBox 320×120)
	BTCAlan    string  `json:"btc_alan"`
	Hacim24    float64 `json:"hacim_24"`    // USD
	BTCDominan float64 `json:"btc_dominan"` // yüzde
	ETHFiyat   float64 `json:"eth_fiyat"`
	ETH24      float64 `json:"eth_24"`
	SOLFiyat   float64 `json:"sol_fiyat"`
	SOL24      float64 `json:"sol_24"`
	RSI        float64 `json:"rsi"` // BTC 14 günlük (Cutler) RSI
	FNG        int     `json:"fng"`
	FNGDun     int     `json:"fng_dun"`
	FNGHafta   int     `json:"fng_hafta"`
	FNGSinif   string  `json:"fng_sinif"` // kaynağın kendi sınıflandırması
	Cekildi    int64   `json:"cekildi"`
	Hazir      bool    `json:"hazir"`
}

// girisCache zarfı: TTL kararı `cekildi`ye bakar (saat dilimi sorunu yok).
type girisCache struct {
	Cekildi int64       `json:"cekildi"`
	Veri    PiyasaGiris `json:"veri"`
}

// PiyasaGirisVeri hero verisini döndürür. Önbellek taze ise oradan,
// değilse kaynaklardan; kaynak hata verirse ELDEKİ veri döner (bayat
// veri, boş panelden iyidir — şerit de aynı ilkeyi izliyor).
func PiyasaGirisVeri(ctx context.Context, pool DB) PiyasaGiris {
	var mevcut girisCache
	if b, err := CacheGet(ctx, pool, girisCacheKind, girisCacheKey); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &mevcut)
		if time.Since(time.Unix(mevcut.Cekildi, 0)) < girisTTL && mevcut.Veri.Hazir {
			return mevcut.Veri
		}
	}

	// Üç uç BİRBİRİNDEN BAĞIMSIZ: sırayla çağırmak ana sayfa açılışına
	// 3x gecikme eklerdi. Sonuçlar TİPLİ kanaldan toplanır — paylaşılan
	// değişkene yazan yok, yarış koşulu da yok.
	type globalSonuc struct{ v girisGlobal }
	type kapanisSonuc struct{ v []float64 }
	type fngSonuc struct{ v []girisFNG }
	gKanal := make(chan globalSonuc, 1)
	kKanal := make(chan kapanisSonuc, 1)
	fKanal := make(chan fngSonuc, 1)
	go func() {
		g, err := girisGlobalCek(ctx)
		if err != nil {
			gKanal <- globalSonuc{}
			return
		}
		gKanal <- globalSonuc{g}
	}()
	go func() {
		k, err := girisBTCKapanis(ctx)
		if err != nil {
			kKanal <- kapanisSonuc{}
			return
		}
		kKanal <- kapanisSonuc{k}
	}()
	go func() {
		f, err := girisFNGCek(ctx)
		if err != nil {
			fKanal <- fngSonuc{}
			return
		}
		fKanal <- fngSonuc{f}
	}()
	global := (<-gKanal).v
	kapanis := (<-kKanal).v
	fng := (<-fKanal).v

	v := mevcut.Veri // bayat veriyi taban al (kısmi başarıda alanlar korunur)
	if global.HacimUsd > 0 {
		v.Hacim24 = global.HacimUsd
		v.BTCDominan = global.BTCDominan
	}
	if yol, alan := cdSparkline(kapanis, 320, 120, 8); yol != "" {
		v.BTCYol, v.BTCAlan = yol, alan
	}
	if r, ok := cdRSI(kapanis, 14); ok {
		v.RSI = r
	}
	if len(fng) > 0 {
		v.FNG = fng[0].Deger
		v.FNGSinif = fng[0].Sinif
		if len(fng) > 1 {
			v.FNGDun = fng[1].Deger
		}
		if len(fng) > 7 {
			v.FNGHafta = fng[7].Deger
		}
	}
	// BTC/ETH/SOL fiyatları fiyat bandının önbelleğinden (60 sn) geliyor.
	for _, s := range Fiyatlar(ctx, pool) {
		switch s.Sym {
		case "BTC":
			v.BTCFiyat, v.BTC24 = s.Price, s.Change
		case "ETH":
			v.ETHFiyat, v.ETH24 = s.Price, s.Change
		case "SOL":
			v.SOLFiyat, v.SOL24 = s.Price, s.Change
		}
	}

	v.Hazir = v.BTCFiyat > 0 && v.BTCYol != ""
	v.Cekildi = time.Now().Unix()
	// Hazır değilse YAZMA: boş iskelet 10 dakika önbellekte kalırsa
	// kaynak düzeldiğinde hero boş kalırdı.
	if v.Hazir {
		if b, err := json.Marshal(girisCache{Cekildi: v.Cekildi, Veri: v}); err == nil {
			_ = CacheSet(ctx, pool, girisCacheKind, girisCacheKey, b)
		}
	}
	return v
}

// girisGlobal /global ucunun ihtiyacımız olan alanları.
type girisGlobal struct {
	HacimUsd   float64
	BTCDominan float64
}

func girisGlobalCek(ctx context.Context) (girisGlobal, error) {
	var out girisGlobal
	govde, err := cdJSONCek(ctx, girisGlobalURL)
	if err != nil {
		return out, err
	}
	var ham struct {
		Data struct {
			TotalVolume struct {
				Usd float64 `json:"usd"`
			} `json:"total_volume"`
			MarketCapPct struct {
				BTC float64 `json:"btc"`
			} `json:"market_cap_percentage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(govde, &ham); err != nil {
		return out, fmt.Errorf("coingecko global: JSON: %w", err)
	}
	out.HacimUsd = ham.Data.TotalVolume.Usd
	out.BTCDominan = ham.Data.MarketCapPct.BTC
	return out, nil
}

// girisBTCKapanis BTC'nin günlük kapanışlarını (USD) döndürür.
func girisBTCKapanis(ctx context.Context) ([]float64, error) {
	govde, err := cdJSONCek(ctx, girisGrafikURL)
	if err != nil {
		return nil, err
	}
	var ham struct {
		Prices [][2]float64 `json:"prices"`
	}
	if err := json.Unmarshal(govde, &ham); err != nil {
		return nil, fmt.Errorf("coingecko chart: JSON: %w", err)
	}
	out := make([]float64, 0, len(ham.Prices))
	for _, p := range ham.Prices {
		if p[1] > 0 {
			out = append(out, p[1])
		}
	}
	if len(out) < 8 {
		return nil, fmt.Errorf("coingecko chart: yetersiz nokta (%d)", len(out))
	}
	return out, nil
}

// girisFNG Korku & Açgözlülük endeksi (0-100).
type girisFNG struct {
	Deger int
	Sinif string
}

func girisFNGCek(ctx context.Context) ([]girisFNG, error) {
	govde, err := cdJSONCek(ctx, girisFNGURL)
	if err != nil {
		return nil, err
	}
	var ham struct {
		Data []struct {
			Value string `json:"value"`
			Sinif string `json:"value_classification"`
		} `json:"data"`
	}
	if err := json.Unmarshal(govde, &ham); err != nil {
		return nil, fmt.Errorf("alternative.me: JSON: %w", err)
	}
	out := make([]girisFNG, 0, len(ham.Data))
	for _, d := range ham.Data {
		n := 0
		if _, err := fmt.Sscanf(d.Value, "%d", &n); err != nil || n < 0 || n > 100 {
			continue
		}
		out = append(out, girisFNG{Deger: n, Sinif: d.Sinif})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("alternative.me: boş veri")
	}
	return out, nil
}

// cdJSONCek küçük JSON uçlarını ortak başlıklarla çeker.
func cdJSONCek(ctx context.Context, url string) ([]byte, error) {
	istek, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// CoinGecko UA'sız istekleri sınırlıyor (fiyat.go ile aynı gerekçe).
	istek.Header.Set("User-Agent", "bitcosar/1.0 (+https://bitcosar.com)")
	istek.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(istek)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// cdSparkline değerleri SVG çizgi+dolgu yoluna çevirir.
//
// viewBox w×h, kenarlarda pad kadar boşluk. Tek değer/eşit seri düz
// çizgi verir (sıfıra bölme yok). Değer yoksa "".
func cdSparkline(deger []float64, w, h, pad float64) (yol, alan string) {
	if len(deger) < 2 {
		return "", ""
	}
	min, maks := deger[0], deger[0]
	for _, d := range deger {
		if d < min {
			min = d
		}
		if d > maks {
			maks = d
		}
	}
	ic := h - 2*pad
	adim := (w - 2*pad) / float64(len(deger)-1)
	nokta := func(i int, d float64) (float64, float64) {
		oran := 0.5
		if maks > min {
			oran = (d - min) / (maks - min)
		}
		// SVG'de y aşağı büyür: yüksek değer küçük y.
		return pad + float64(i)*adim, pad + (1-oran)*ic
	}
	var b []byte
	b = append(b, 'M')
	for i, d := range deger {
		x, y := nokta(i, d)
		if i > 0 {
			b = append(b, 'L')
		}
		b = append(b, []byte(fmt.Sprintf("%.2f %.2f", x, y))...)
	}
	yol = string(b)
	sx, _ := nokta(0, deger[0])
	ex, _ := nokta(len(deger)-1, deger[len(deger)-1])
	alan = fmt.Sprintf("%s L%.2f %.2f L%.2f %.2f Z", yol, ex, h-pad, sx, h-pad)
	return yol, alan
}

// cdRSI Wilder/Cutler basitleştirmesi: son `periyot` kapanışın ortalama
// kazanç/kaybının oranı. Yetersiz veri varsa ok=false (uydurma yok).
func cdRSI(kapanis []float64, periyot int) (float64, bool) {
	if periyot <= 0 || len(kapanis) < periyot+1 {
		return 0, false
	}
	dilim := kapanis[len(kapanis)-(periyot+1):]
	var kazanc, kayip float64
	for i := 1; i < len(dilim); i++ {
		fark := dilim[i] - dilim[i-1]
		if fark > 0 {
			kazanc += fark
		} else {
			kayip -= fark
		}
	}
	if kazanc == 0 && kayip == 0 {
		return 50, true
	}
	if kayip == 0 {
		return 100, true
	}
	rs := (kazanc / float64(periyot)) / (kayip / float64(periyot))
	return math.Round(100 - 100/(1+rs)), true
}
