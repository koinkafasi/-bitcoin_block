package templkit

import (
	"math"
	"strconv"
)

// CDGiris — ana sayfanın GİRİŞ HERO'su (11 Kasım 2026).
//
// KAYNAK TASARIM: ghost.bitcosar.com ana sayfasındaki `.ch-hero`
// (ölçüldü 11 Kasım 2026, 1440×900: yükseklik 532 px, ızgara
// 1.05fr / 1fr / 300px, gap 30, dikey dolgu 56/60, görsel yuvası
// min-height 400). Yerleşim birebir korunuyor; renkler ve yazı tipleri
// cd tasarım jetonlarına bağlandı (site krem/koyu temalı).
//
// VERİ KURALI: sayı UYDURULMAZ. Karşılığı olmayan parça basılmaz —
// FNG gelmezse o gösterge kartı, RSI gelmezse RSI kartı, sparkline
// gelmezse grafik hiç çizilmez. `Hazir` yanlışsa modülün tamamı düşer.
type CDGiris struct {
	Hazir bool

	// Metin bloğu (marka söylemi — sayısal iddia yok).
	Eyebrow     string
	BaslikOnce  string
	BaslikVurgu string
	BaslikSonra string
	Aciklama    string
	Cta1Ad      string
	Cta1Href    string
	Cta2Ad      string
	Cta2Href    string
	Maddeler    []CDGirisMadde
	// Guncelleme: verinin yaşı ("3 dk önce"). "CANLI" yazmıyoruz:
	// veri 10 dakikalık önbellekten geliyor, canlı demek yanlış olurdu.
	Guncelleme string

	// BTC paneli — gerçek CoinGecko verisi.
	BTCFiyat   string
	BTCDegisim string
	BTCUp      bool
	BTCYol     string // sparkline çizgi yolu (viewBox 320×120)
	BTCAlan    string // aynı serinin dolgusu
	Hacim      string // 24s toplam piyasa hacmi
	Dominans   string // BTC hakimiyeti
	Endeks     string // Korku & Açgözlülük değeri
	EndeksUp   bool

	// Yüzen kartlar (aynı fiyat önbelleğinden).
	ETHFiyat   string
	ETHDegisim string
	ETHUp      bool
	SOLFiyat   string
	SOLDegisim string
	SOLUp      bool

	// Göstergeler. *Var alanları ZORUNLU: 0 geçerli bir RSI/FNG değeri,
	// "veri yok" ile karıştırılmamalı.
	FNGVar    bool
	FNGDeger  int
	FNGEtiket string
	FNGUp     bool
	FNGDun    int
	FNGHafta  int

	RSIVar    bool
	RSI       int
	RSIEtiket string
	RSIUp     bool
}

// CDGirisMadde metin bloğunun altındaki numaralı üç kısa ifade.
type CDGirisMadde struct {
	No    string
	Metin string
}

// cdGirisYay yarım daire yayın dasharray değerini üretir.
//
// Yarıçap 84 (ghost ölçümü) → yay uzunluğu π·84 ≈ 264. Ghost bu değeri
// JS ile yazıyordu; biz sunucuda hesaplıyoruz (CSP: script-src 'self',
// harici JS yok).
func cdGirisYay(pct int) string {
	uzunluk := math.Pi * 84 * float64(cdGirisKirp(pct)) / 100
	return strconv.FormatFloat(uzunluk, 'f', 1, 64) + " 264"
}

// cdGirisNoktaX / cdGirisNoktaY yayın ucundaki noktanın konumu.
//
// Yay 180°'den 0°'ye: p=0 → (16,100), p=100 → (184,100). Ghost'un
// başlangıç dairesi cx=16 cy=100 ile birebir aynı.
func cdGirisNoktaX(pct int) string {
	x, _ := cdGirisNokta(pct)
	return strconv.FormatFloat(x, 'f', 1, 64)
}

func cdGirisNoktaY(pct int) string {
	_, y := cdGirisNokta(pct)
	return strconv.FormatFloat(y, 'f', 1, 64)
}

func cdGirisNokta(pct int) (float64, float64) {
	teta := math.Pi * (1 - float64(cdGirisKirp(pct))/100)
	return 100 + 84*math.Cos(teta), 100 - 84*math.Sin(teta)
}

func cdGirisKirp(pct int) int {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}
