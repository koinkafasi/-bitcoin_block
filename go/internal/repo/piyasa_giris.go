package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

const (
	girisCacheKind = "giris"
	girisCacheKey  = "piyasa"
	// Ham seri TTL'i: saatlik kapanışlar için 10 dk fazlasıyla yeterli.
	girisTTL = 10 * time.Minute

	girisGlobalURL = "https://api.coingecko.com/api/v3/global"
	girisGrafikURL = "https://api.coingecko.com/api/v3/coins/bitcoin/market_chart" +
		"?vs_currency=usd&days=30"
	girisFNGURL = "https://api.alternative.me/fng/?limit=8"

	// Çizgide gösterilen pencere: son 7 gün (saatlik 168 nokta).
	girisCizgiNokta = 168
	// Saatlik seriden günlük kapanış üretirken gün başına nokta sayısı.
	girisGunlukNokta = 24
)

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
	RSI        float64 `json:"rsi"` // BTC günlük (Cutler) RSI
	FNG        int     `json:"fng"`
	FNGDun     int     `json:"fng_dun"`
	FNGHafta   int     `json:"fng_hafta"`
	FNGSinif   string  `json:"fng_sinif"` // kaynağın kendi sınıflandırması
	// Cekildi FİYATIN çekildiği an. "Az önce" yazısı buna bakar — geçmiş
	// serinin yaşı değil, ekranda gördüğün sayının yaşı önemli.
	Cekildi int64 `json:"cekildi"`
	Hazir   bool  `json:"hazir"`
}

// girisHam önbellekte duran HAM veridir (türetilmiş yollar saklanmaz:
// uç nokta her istekte güncel fiyatla yeniden hesaplanır).
type girisHam struct {
	Cekildi  int64     `json:"cekildi"`
	Grafik   []float64 `json:"grafik"` // saatlik kapanışlar (eskiden yeniye)
	Hacim24  float64   `json:"hacim_24"`
	BTCDomin float64   `json:"btc_dominan"`
	FNG      int       `json:"fng"`
	FNGDun   int       `json:"fng_dun"`
	FNGHafta int       `json:"fng_hafta"`
	FNGSinif string    `json:"fng_sinif"`
}

// girisYenileme tek uçuşu garanti eder: önbellek eskidiğinde 100 istek
// 100 arka plan görevi değil, 1 görev başlatır.
var (
	girisKilit       sync.Mutex
	girisYenileniyor bool
)

func PiyasaGirisVeri(ctx context.Context, pool DB) PiyasaGiris {
	ham := girisHamOku(ctx, pool)
	fiyatlar, fiyatCekildi := FiyatlarZamanli(ctx, pool)

	var v PiyasaGiris
	for _, s := range fiyatlar {
		switch s.Sym {
		case "BTC":
			v.BTCFiyat, v.BTC24 = s.Price, s.Change
		case "ETH":
			v.ETHFiyat, v.ETH24 = s.Price, s.Change
		case "SOL":
			v.SOLFiyat, v.SOL24 = s.Price, s.Change
		}
	}

	// ÇİZGİ: geçmiş serinin SON NOKTAASI güncel fiyatla değiştirilir.
	// Böylece grafiğin ucu canlı fiyatla hareket eder; gövde gerçek geçmiş.
	seri := make([]float64, 0, girisCizgiNokta)
	if n := len(ham.Grafik); n > 0 {
		bas := 0
		if n > girisCizgiNokta {
			bas = n - girisCizgiNokta
		}
		seri = append(seri, ham.Grafik[bas:]...)
	}
	if v.BTCFiyat > 0 && len(seri) > 0 {
		seri[len(seri)-1] = v.BTCFiyat
	}
	if yol, alan := cdSparkline(seri, 320, 120, 8); yol != "" {
		v.BTCYol, v.BTCAlan = yol, alan
	}

	// RSI: saatlik seriden GÜNLÜK kapanış üretilir (14 gün = 15 kapanış).
	if gunluk := girisGunlukKapanis(ham.Grafik, girisGunlukNokta); len(gunluk) > 0 {
		if v.BTCFiyat > 0 {
			gunluk[len(gunluk)-1] = v.BTCFiyat // bugünün kapanışı = güncel fiyat
		}
		if r, ok := cdRSI(gunluk, 14); ok {
			v.RSI = r
		}
	}

	v.Hacim24 = ham.Hacim24
	v.BTCDominan = ham.BTCDomin
	v.FNG = ham.FNG
	v.FNGDun = ham.FNGDun
	v.FNGHafta = ham.FNGHafta
	v.FNGSinif = ham.FNGSinif
	v.Cekildi = fiyatCekildi
	v.Hazir = v.BTCFiyat > 0 && v.BTCYol != ""
	return v
}

// girisHamOku önbellekten ham veriyi döndürür; gerekirse tazeler.
func girisHamOku(ctx context.Context, pool DB) girisHam {
	var mevcut girisHam
	var varMi bool
	if b, err := CacheGet(ctx, pool, girisCacheKind, girisCacheKey); err == nil && len(b) > 0 {
		if json.Unmarshal(b, &mevcut) == nil && len(mevcut.Grafik) > 0 {
			varMi = true
		}
	}

	if varMi && time.Since(time.Unix(mevcut.Cekildi, 0)) < girisTTL {
		return mevcut
	}

	if !varMi {
		// İlk açılış: veri hiç yok. Beklemek zorundayız (hero boş
		// görünmesin); bu yol ömür boyu bir kez yaşanır.
		taze, _ := girisHamCek(ctx, pool, mevcut)
		return taze
	}

	// Bayat: ESKİ veriyi hemen ver, tazelemeyi arkaya at.
	girisArkaPlandaYenile(pool, mevcut)
	return mevcut
}

// girisArkaPlandaYenile tek uçuşlu arka plan tazelemesi.
func girisArkaPlandaYenile(pool DB, mevcut girisHam) {
	girisKilit.Lock()
	if girisYenileniyor {
		girisKilit.Unlock()
		return
	}
	girisYenileniyor = true
	girisKilit.Unlock()

	go func() {
		defer func() {
			girisKilit.Lock()
			girisYenileniyor = false
			girisKilit.Unlock()
		}()
		// İsteğin context'i bitmiş olacak; kendi bütçemizi kuruyoruz.
		ctx, iptal := context.WithTimeout(context.Background(), 30*time.Second)
		defer iptal()
		_, _ = girisHamCek(ctx, pool, mevcut)
	}()
}

func girisHamCek(ctx context.Context, pool DB, taban girisHam) (girisHam, bool) {
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
		k, err := girisBTCSeri(ctx)
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
	seri := (<-kKanal).v
	fng := (<-fKanal).v

	out := taban // eski değerler taban
	if global.HacimUsd > 0 {
		out.Hacim24 = global.HacimUsd
		out.BTCDomin = global.BTCDominan
	}
	if len(seri) > 0 {
		out.Grafik = seri
	}
	if len(fng) > 0 {
		out.FNG = fng[0].Deger
		out.FNGSinif = fng[0].Sinif
		if len(fng) > 1 {
			out.FNGDun = fng[1].Deger
		}
		if len(fng) > 7 {
			out.FNGHafta = fng[7].Deger
		}
	}

	// Grafik olmadan yazmanın anlamı yok: hero çizgisiz basılmaz.
	if len(out.Grafik) == 0 {
		return out, false
	}
	out.Cekildi = time.Now().Unix()
	if b, err := json.Marshal(out); err == nil {
		_ = CacheSet(ctx, pool, girisCacheKind, girisCacheKey, b)
	}
	return out, true
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

func girisBTCSeri(ctx context.Context) ([]float64, error) {
	seri, err := girisBTCSeriCoinGecko(ctx)
	if err == nil && len(seri) > 0 {
		return seri, nil
	}
	// Yedek: Binance 30 gün × saatlik = 720 mum.
	seri, bErr := binanceKlines(ctx, "BTC", 720)
	if bErr != nil || len(seri) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, bErr
	}
	return seri, nil
}

func girisBTCSeriCoinGecko(ctx context.Context) ([]float64, error) {
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
	if len(out) < girisCizgiNokta/2 {
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
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(istek)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func girisGunlukKapanis(seri []float64, nokta int) []float64 {
	if nokta <= 0 || len(seri) < nokta*2 {
		return nil
	}
	gun := len(seri) / nokta
	out := make([]float64, 0, gun)
	for i := 1; i <= gun; i++ {
		// i. günün SON noktası = kapanış. (Gün başını almak RSI'ı bir
		// gün kaydırıyordu; test yakaladı — 11 Kasım 2026.)
		out = append(out, seri[len(seri)-(i-1)*nokta-1])
	}
	// Geriye doğru topladık; eskiden yeniye çevir.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

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
			kayip += -fark
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
