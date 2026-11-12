package scoring

import (
	"math"
	"time"
)

// OncelikGirdi bir içeriğin dört eksenini hesaplamak için gereken her şeydir.
// Paket veritabanı bilmez; bu alanları çağıran taraf doldurur.
type OncelikGirdi struct {
	Tamlik Input // başlık/spot/etiket/kapak/kaynak sinyalleri

	TrendEslesme    int
	TrendSonGorulme time.Time
	TrendTrafik     int // trend_terms.approx_traffic; 0 = bilinmiyor

	// Kaynak güveni. feeds.trust (0–100). Agregatör akışları (Google News
	// gibi) özgün kaynak değildir, bu yüzden ayrıca cezalandırılır.
	KaynakGuven int
	Agregator   bool

	// İçeriğin yaşı için referans noktaları.
	OlusturmaZamani time.Time
	Simdi           time.Time // sıfırsa time.Now() kullanılır
}

// Eksenler dört ekseni ve birleşik önceliği taşır. Hepsi 0–100.
type Eksenler struct {
	Tamlik  int `json:"tamlik"`
	Trend   int `json:"trend"`
	Guven   int `json:"guven"`
	Tazelik int `json:"tazelik"`
	Oncelik int `json:"oncelik"`
}

// Ağırlıklar tek yerde; değiştirilirse kayıtlı öncelikler yeniden
// hesaplanmalıdır (cmd/backfill -is=oncelik).
const (
	AgirlikTrend   = 45
	AgirlikTazelik = 35
	AgirlikGuven   = 10
	AgirlikTamlik  = 10
)

const (
	tazelikIlkGun   = 24 * time.Hour
	tazelikSinir    = 7 * 24 * time.Hour
	tazelikIlkGunDe = 70.0 // ilk günün sonundaki değer
)

// Tazelik yaşa göre 100'den 0'a iner.
func Tazelik(yas time.Duration) int {
	if yas <= 0 {
		return 100
	}
	if yas >= tazelikSinir {
		return 0
	}
	var v float64
	if yas <= tazelikIlkGun {
		v = 100 - (100-tazelikIlkGunDe)*(yas.Hours()/tazelikIlkGun.Hours())
	} else {
		kalan := (yas - tazelikIlkGun).Hours() / (tazelikSinir - tazelikIlkGun).Hours()
		v = tazelikIlkGunDe * (1 - kalan)
	}
	return clamp(int(math.Round(v)))
}

// trendTazelikSiniri: eşleşilen terim bu süreden eskiyse trend sinyali
// sayılmaz. Google Trends "şu an" demektir; dünkü trend bugünün trendi değil.
const trendTazelikSiniri = 48 * time.Hour

func TrendUyumu(eslesme int, sonGorulme, simdi time.Time, trafik int) int {
	if eslesme <= 0 {
		return 0
	}
	taze := 1.0
	if !sonGorulme.IsZero() {
		yas := simdi.Sub(sonGorulme)
		switch {
		case yas <= 0:
			taze = 1.0
		case yas >= trendTazelikSiniri:
			return 0
		default:
			taze = 1 - yas.Hours()/trendTazelikSiniri.Hours()
		}
	}
	bonus := 0.0
	if trafik >= 50000 {
		bonus = 10
	} else if trafik >= 10000 {
		bonus = 5
	}
	return clamp(int(math.Round(float64(clamp(eslesme))*taze + bonus)))
}

func KaynakGuveni(trust int, agregator bool) int {
	v := trust
	if v <= 0 {
		v = 50
	}
	if agregator {
		v -= 15
	}
	return clamp(v)
}

// Hesapla dört ekseni ve birleşik önceliği üretir.
func Hesapla(g OncelikGirdi) Eksenler {
	simdi := g.Simdi
	if simdi.IsZero() {
		simdi = time.Now()
	}
	var yas time.Duration
	if !g.OlusturmaZamani.IsZero() {
		yas = simdi.Sub(g.OlusturmaZamani)
	}

	tamlik, _ := Explain(g.Tamlik)
	e := Eksenler{
		Tamlik:  tamlik,
		Trend:   TrendUyumu(g.TrendEslesme, g.TrendSonGorulme, simdi, g.TrendTrafik),
		Guven:   KaynakGuveni(g.KaynakGuven, g.Agregator),
		Tazelik: Tazelik(yas),
	}
	toplam := e.Trend*AgirlikTrend + e.Tazelik*AgirlikTazelik +
		e.Guven*AgirlikGuven + e.Tamlik*AgirlikTamlik
	e.Oncelik = clamp(int(math.Round(float64(toplam) /
		float64(AgirlikTrend+AgirlikTazelik+AgirlikGuven+AgirlikTamlik))))
	return e
}

func TazelikSinirSaat() int {
	return int(tazelikSinir.Hours())
}
