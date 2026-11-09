package templkit

import "strconv"

// Hisse detay sayfası (/price/stock/{sembol}) — CoinDesk §10.6.
//
// CoinDesk hisse şablonu coin şablonundan FARKLI: iki kolon (392 + 976),
// h1 yok (ad küçük), büyük fiyat 28/42, grafik 976×500, zaman sekmeleri
// 1D·5D·1M·YTD·1Y·All, "Basic/Advanced" kip düğmeleri ve altında
// performans satırı. Bu dosya yalnız VERİ taşır; biçimleme httpapi'de
// (repo.SayiBicim / FiyatBicimP) yapılır — templkit repo'ya bağlanmaz.

// CDHisseSekme zaman aralığı sekmesi.
type CDHisseSekme struct {
	Kod   string // ?aralik= değeri
	Ad    string // 1D, 5D …
	Href  string
	Aktif bool
}

// CDHisseStat "Key Stats" satırı (etiket subtle, değer kalın).
type CDHisseStat struct {
	Etiket string
	Deger  string
}

// CDHisseDonem performans satırındaki tek hücre.
type CDHisseDonem struct {
	Ad    string
	Yuzde string
	Up    bool
}

// CDHisseSayfa hisse detay sayfasının görünüm verisi.
type CDHisseSayfa struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string
	JSONLD    string
	BaseURL   string

	PageCategory    string
	ContentCategory string
	Locale          string

	Ad     string
	Sembol string
	Borsa  string
	Para   string

	Fiyat   string
	Degisim string
	Up      bool
	AsOf    string // "9 Kas 2026 15:59 EST" (borsa saati)

	// Profil: şirket künyesi ("About …"). Boşsa bölüm HİÇ basılmaz.
	// Sektör/çalışan Key Stats satırına handler'da eklenir.
	Ozet string

	Istatistik []CDHisseStat

	// Grafik: seri yoksa Yol boş kalır ve grafik bloğu hiç basılmaz.
	Yol        string
	Alan       string
	GrafikUp   bool
	GrafikVar  bool
	GrafikMin  string
	GrafikMaks string
	EtiketSol  string
	EtiketSag  string

	// OHLC okuması (grafiğin üstünde): "O:$632.30  H:$…".
	Acilis string
	Yuksek string
	Dusuk  string

	Sekmeler        []CDHisseSekme
	AktifKod        string
	Kip             string // "basic" | "advanced"
	KipHrefBasic    string
	KipHrefAdvanced string
	// Mumlar Advanced kipinde basılır (boşsa alan+çizgiye düşer).
	Mumlar []CDHisseMum

	Performans []CDHisseDonem
	Kaynak     string
}

// cdF SVG koordinatını kısa metne çevirir (mum geometrisi yazdırılırken).
func cdF(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}
