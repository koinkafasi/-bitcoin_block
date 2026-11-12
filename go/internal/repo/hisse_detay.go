package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const yahooChartAralik = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s"

// HisseNokta grafikteki tek nokta (unix saniye + OHLC).
type HisseNokta struct {
	T      int64   `json:"t"`
	C      float64 `json:"c"`
	Acilis float64 `json:"o,omitempty"`
	Yuksek float64 `json:"h,omitempty"`
	Dusuk  float64 `json:"l,omitempty"`
}

// HisseDetay tek hissenin kotasyonu + seçilen aralığın serisi.
type HisseDetay struct {
	Sembol string `json:"sembol"`
	Ad     string `json:"ad"`
	Borsa  string `json:"borsa"`
	Para   string `json:"para"`

	Fiyat      float64 `json:"fiyat"`
	Degisim    float64 `json:"degisim"` // günlük % (CoinDesk üst fiyatı)
	Onceki     float64 `json:"onceki"`  // önceki kapanış
	GunMin     float64 `json:"gun_min"`
	GunMaks    float64 `json:"gun_maks"`
	H52Min     float64 `json:"h52_min"`
	H52Maks    float64 `json:"h52_maks"`
	Hacim      float64 `json:"hacim"`  // günlük işlem hacmi (adet)
	Acilis     float64 `json:"acilis"` // son noktanın açılışı (OHLC okuması)
	Yuksek     float64 `json:"yuksek"`
	Dusuk      float64 `json:"dusuk"`
	Guncelleme int64   `json:"guncelleme"` // borsa saati (unix)
	Dilim      int     `json:"dilim"`      // borsa saat dilimi kayması (saniye, EST=-18000)

	Noktalar []HisseNokta `json:"noktalar"`
}

// HisseAralik tek zaman sekmesi (CoinDesk §10.6: 1D · 5D · 1M · YTD · 1Y · All).
type HisseAralik struct {
	Kod      string // adres parametresi (?aralik=1d)
	Ad       string // sekme etiketi
	Interval string // Yahoo interval
	Range    string // Yahoo range
}

// HisseAraliklari sekmelerin TANIM SIRASI (şablonda da bu sıra basılır).
var HisseAraliklari = []HisseAralik{
	{Kod: "1d", Ad: "1D", Interval: "5m", Range: "1d"},
	{Kod: "5d", Ad: "5D", Interval: "30m", Range: "5d"},
	{Kod: "1m", Ad: "1M", Interval: "1d", Range: "1mo"},
	{Kod: "ytd", Ad: "YTD", Interval: "1d", Range: "ytd"},
	{Kod: "1y", Ad: "1Y", Interval: "1d", Range: "1y"},
	{Kod: "all", Ad: "All", Interval: "1mo", Range: "max"},
}

// HisseAralikBul kodu eşler; bilinmeyen kodda "1d" döner (bozuk parametre
// sayfayı düşürmesin).
func HisseAralikBul(kod string) HisseAralik {
	k := strings.ToLower(strings.TrimSpace(kod))
	for _, a := range HisseAraliklari {
		if a.Kod == k {
			return a
		}
	}
	return HisseAraliklari[0]
}

var hisseSembolRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9.\-=^]{0,11}$`)

// HisseSembolGecerli sembolün güvenli olduğunu söyler.
func HisseSembolGecerli(s string) bool { return hisseSembolRe.MatchString(s) }

// yahooMeta grafik yanıtının kotasyon bölümü.
type yahooMeta struct {
	Currency           string  `json:"currency"`
	Symbol             string  `json:"symbol"`
	ExchangeName       string  `json:"exchangeName"`
	FullExchangeName   string  `json:"fullExchangeName"`
	LongName           string  `json:"longName"`
	ShortName          string  `json:"shortName"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	PreviousClose      float64 `json:"previousClose"`
	ChartPreviousClose float64 `json:"chartPreviousClose"`
	ChangePercent      float64 `json:"regularMarketChangePercent"`
	DayHigh            float64 `json:"regularMarketDayHigh"`
	DayLow             float64 `json:"regularMarketDayLow"`
	Volume             float64 `json:"regularMarketVolume"`
	Week52High         float64 `json:"fiftyTwoWeekHigh"`
	Week52Low          float64 `json:"fiftyTwoWeekLow"`
	MarketTime         int64   `json:"regularMarketTime"`
	Gmtoffset          int     `json:"gmtoffset"`
}

// yahooQuote tek serinin OHLC dizileri.
type yahooQuote struct {
	Open  []*float64 `json:"open"`
	High  []*float64 `json:"high"`
	Low   []*float64 `json:"low"`
	Close []*float64 `json:"close"`
}

// yahooChartYanit grafik yanıtı (yalnız kullandığımız alanlar).
type yahooChartYanit struct {
	Chart struct {
		Result []struct {
			Meta       yahooMeta `json:"meta"`
			Timestamp  []int64   `json:"timestamp"`
			Indicators struct {
				Quote []yahooQuote `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func HisseCek(ctx context.Context, sembol, aralik string) (HisseDetay, error) {
	sym := strings.ToUpper(strings.TrimSpace(sembol))
	if !HisseSembolGecerli(sym) {
		return HisseDetay{}, fmt.Errorf("geçersiz hisse sembolü")
	}
	ar := HisseAralikBul(aralik)
	u := fmt.Sprintf(yahooChartAralik, url.PathEscape(sym), ar.Interval, ar.Range)

	raw, err := httpGetJSONUA(u, 10*time.Second)
	if err != nil {
		return HisseDetay{}, err
	}
	var y yahooChartYanit
	if err := json.Unmarshal(raw, &y); err != nil {
		return HisseDetay{}, err
	}
	if len(y.Chart.Result) == 0 {
		return HisseDetay{}, fmt.Errorf("sonuç yok")
	}
	r := y.Chart.Result[0]
	if r.Meta.RegularMarketPrice <= 0 {
		return HisseDetay{}, fmt.Errorf("fiyat yok")
	}

	d := HisseDetay{
		Sembol:     strings.ToUpper(firstDolu(r.Meta.Symbol, sym)),
		Ad:         firstDolu(r.Meta.LongName, r.Meta.ShortName, sym),
		Borsa:      firstDolu(r.Meta.FullExchangeName, r.Meta.ExchangeName),
		Para:       firstDolu(r.Meta.Currency, "USD"),
		Fiyat:      r.Meta.RegularMarketPrice,
		Onceki:     firstPozitif(r.Meta.PreviousClose, r.Meta.ChartPreviousClose),
		Degisim:    r.Meta.ChangePercent,
		GunMin:     r.Meta.DayLow,
		GunMaks:    r.Meta.DayHigh,
		H52Min:     r.Meta.Week52Low,
		H52Maks:    r.Meta.Week52High,
		Hacim:      r.Meta.Volume,
		Guncelleme: r.Meta.MarketTime,
		Dilim:      r.Meta.Gmtoffset,
	}
	// Yüzde alanı boşsa önceki kapanıştan hesapla (uydurma değil, aynı veriden).
	if d.Degisim == 0 && d.Onceki > 0 {
		d.Degisim = (d.Fiyat - d.Onceki) / d.Onceki * 100
	}

	if len(r.Indicators.Quote) > 0 {
		d.Noktalar = hisseSeri(r.Timestamp, r.Indicators.Quote[0])
	}
	if n := len(d.Noktalar); n > 0 {
		son := d.Noktalar[n-1]
		d.Acilis, d.Yuksek, d.Dusuk = son.Acilis, son.Yuksek, son.Dusuk
	}
	return d, nil
}

func hisseSeri(ts []int64, q yahooQuote) []HisseNokta {
	tam := make([]HisseNokta, 0, len(ts))
	for i, t := range ts {
		if i >= len(q.Close) || q.Close[i] == nil {
			continue
		}
		n := HisseNokta{T: t, C: *q.Close[i]}
		if i < len(q.Open) && q.Open[i] != nil {
			n.Acilis = *q.Open[i]
		}
		if i < len(q.High) && q.High[i] != nil {
			n.Yuksek = *q.High[i]
		}
		if i < len(q.Low) && q.Low[i] != nil {
			n.Dusuk = *q.Low[i]
		}
		tam = append(tam, n)
	}
	const ust = 240
	if len(tam) <= ust {
		return tam
	}
	out := make([]HisseNokta, 0, ust)
	adim := float64(len(tam)-1) / float64(ust-1)
	for i := 0; i < ust-1; i++ {
		out = append(out, tam[int(float64(i)*adim)])
	}
	out = append(out, tam[len(tam)-1])
	return out
}

func firstDolu(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstPozitif(v ...float64) float64 {
	for _, f := range v {
		if f > 0 {
			return f
		}
	}
	return 0
}

func HisseGunOHLC(seri []HisseNokta, dilimSaniye int) (acilis, yuksek, dusuk float64, ok bool) {
	if len(seri) == 0 {
		return 0, 0, 0, false
	}
	yer := time.FixedZone("borsa", dilimSaniye)
	sonGun := time.Unix(seri[len(seri)-1].T, 0).In(yer)
	y, m, d := sonGun.Date()
	for _, n := range seri {
		t := time.Unix(n.T, 0).In(yer)
		ny, nm, nd := t.Date()
		if ny != y || nm != m || nd != d {
			continue
		}
		if !ok {
			acilis = n.Acilis
			yuksek = n.Yuksek
			dusuk = n.Dusuk
			ok = true
			continue
		}
		if n.Yuksek > yuksek {
			yuksek = n.Yuksek
		}
		if n.Dusuk > 0 && (dusuk == 0 || n.Dusuk < dusuk) {
			dusuk = n.Dusuk
		}
	}
	return acilis, yuksek, dusuk, ok
}
