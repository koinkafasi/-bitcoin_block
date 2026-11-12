package templkit

import (
	"math"
	"strconv"
)

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
	// Cekildi fiyatın çekildiği an (UNIX). Tarayıcı "az önce" yazısını
	// buradan hesaplar; sunucu saatiyle birlikte gönderilir (/canli/giris).
	Cekildi int64

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

func cdGirisYay(pct int) string {
	uzunluk := math.Pi * 84 * float64(cdGirisKirp(pct)) / 100
	return strconv.FormatFloat(uzunluk, 'f', 1, 64) + " 264"
}

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
