package templkit

import "strconv"

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
