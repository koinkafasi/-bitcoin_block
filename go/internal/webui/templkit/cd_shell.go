package templkit

import (
	"strconv"
	"strings"
)

// CDTickerItem fiyat şeridindeki tek kalem.
type CDTickerItem struct {
	Sym    string
	Slug   string // kanonik yol parçası (tam ad): "bitcoin"
	Href   string // tıklama hedefi; boşsa Slug'tan /price/ yolu kurulur
	Price  string
	Change string // "+%3.48" gibi, işaret dahil
	Up     bool
}

// CDNavLink üst gezinmedeki tek bağlantı.
type CDNavLink struct {
	Label  string
	Href   string
	Active bool
}

// CDDil dil seçicideki tek dil.
type CDDil struct {
	Kod   string
	Ad    string
	Aktif bool
	Href  string
}

// CDShell, CoinDesk kabuğunun sayfa verisidir.
type CDShell struct {
	SiteName    string
	LangBase    string
	Nav         []CDNavLink
	Ticker      []CDTickerItem
	TickerTab   string // "markets" | "stocks"
	TickerLive  string // "Markets Live" / "Crypto Stocks"
	SearchHref  string
	LoginHref   string
	SignupHref  string
	LoggedIn    bool
	UserName    string
	UserInitial string
	UserHref    string // hesap merkezi ("/hesabim")
	UserUnread  int    // okunmamış bildirim (0 = rozet basılmaz)
	Locale      string
	// Diller dil seçicisi (yalnız gerçek diller).
	Diller []CDDil
	// MenuTags mega menüdeki etiket sütunu (yalnız içeriği olan etiketler).
	MenuTags []CDMenuItem
	// MenuCoins mega menüdeki canlı fiyat satırları (ilk 5 coin).
	MenuCoins    []CDMenuItem
	MenuBolumler map[string]bool
	BaseURL      string

	LogoURL string
	// LogoSimgeURL soldaki SİMGE. Doluysa kilit "simge | yazı+slogan"
	// olarak kurulur ve slogan yazının altına YAPISAL olarak hizalanır.
	LogoSimgeURL string
	// LogoSimgeYukseklik simgenin çizim yüksekliği (px). 0 → LogoHedefYukseklik.
	LogoSimgeYukseklik int

	// LogoAcikURL açık tema için ayrı logo (markanın koyu renkli sürümü).
	// Boşsa LogoURL iki temada da kullanılır.
	LogoAcikURL string

	LogoGenislik  int
	LogoYukseklik int

	LogoHedefYukseklik int
	// LogoAzamiGenislik panelden ayarlanan en fazla genişlik (px).
	// 0 = 260.
	LogoAzamiGenislik int
	SloganGirinti     int

	// Tagline logonun altındaki slogan (marka ayarından).
	Tagline string

	// FaviconURL yüklenen favicon; boşsa statik simge kullanılır.
	FaviconURL string

	// KonumKapi true ise konum izin penceresi KAPATILAMAZ.
	// Üyelik paneli gibi "konum olmadan devam edilemez" sayfalarda açılır.
	KonumKapi bool
}

var cdFooterColumns = []cdFooterBolum{
	{Title: "Hakkımızda", Links: [][2]string{
		{"Hakkımızda", "/hakkimizda"},
		{"Kariyer", "/kariyer"},
	}},
	{Title: "İletişim", Links: [][2]string{
		{"Bize Ulaşın", "/iletisim"},
		{"Gizlilik", "/gizlilik-politikasi"},
		{"Kullanım Koşulları", "/kullanim-kosullari"},
	}},
	{Title: "Newsletters", Links: [][2]string{
		{"Bültenler", "/newsletters"},
	}},
	{Title: "Haberler", Links: [][2]string{
		{"Son Haberler", "/latest-crypto-news"},
		{"Piyasalar", "/markets"},
		{"Finans", "/business"},
		{"Teknoloji", "/tech"},
		{"Politika", "/policy"},
		{"Haber Analizi", "/news-analysis"},
	}},
	{Title: "Videolar", Links: [][2]string{
		{"Video", "/videos"},
	}},
	{Title: "Podcasts", Links: [][2]string{
		{"Podcastlar", "/podcasts"},
	}},
	{Title: "Kripto paralar", Links: [][2]string{
		{"Tüm fiyatlar", "/price"},
		{"Bitcoin (BTC)", "/price/bitcoin"},
		{"Ethereum (ETH)", "/price/ethereum"},
		{"XRP", "/price/xrp"},
		{"Solana (SOL)", "/price/solana"},
	}},
	{Title: "Veri ve Endeksler", Links: [][2]string{
		{"Piyasa Verileri", "/price"},
	}},
	{Title: "Fikir Birliği", Links: [][2]string{
		{"Etkinlikler", "/etkinlikler"},
	}},
	{Title: "Sponsorlu", Links: [][2]string{
		{"Düşünce Liderliği", "/sponsored-content"},
		{"Basın Bültenleri", "/press-release"},
	}},
	{Title: "Araştırma", Links: [][2]string{
		{"Araştırma Merkezi", "/research"},
		{"Görüş", "/opinion"},
		{"Odak", "/focus"},
	}},
}

// cdFooterSutunlari bölümleri DÖRT sütuna sırayla dağıtır (CoinDesk
// `grid-cols-4`). Sıra korunur; sütun yükseklikleri dengeli kalır.
func cdFooterSutunlari(s CDShell) [][]cdFooterBolum {
	const sutunSayisi = 4
	out := make([][]cdFooterBolum, sutunSayisi)
	for i, b := range cdFooterColumns {
		// ÇEVİRİ: sütun başlığı ve bağlantı metinleri arayüz kabuğudur,
		// çevrilir (bkz. cd_sozluk.go). Bağlantı ADRESLERİ dokunulmaz.
		for j := range b.Links {
			b.Links[j][0] = cdT(s.Locale, b.Links[j][0])
		}
		b.Title = cdT(s.Locale, b.Title)
		out[i%sutunSayisi] = append(out[i%sutunSayisi], b)
	}
	return out
}

// cdFooterBolum alt bilgide tek başlık + bağlantıları.
type cdFooterBolum struct {
	Title string
	Links [][2]string
}

// cdSiraya numarayı 1-tabanlı metne çevirir (0 → "1").
func cdSiraya(i int) string {
	return strconv.Itoa(i + 1)
}

// cdSayiMetni sayacı metne çevirir (hesap menüsü/kutuları).
func cdSayiMetni(n int) string {
	return strconv.Itoa(n)
}

// cdBildirimSayi bildirim ROZETİ metni — 9 üstü "9+" (eski header ile
// aynı sınır; rozet kutusu 3 haneli sayıya şişmesin).
func cdBildirimSayi(n int) string {
	if n > 9 {
		return "9+"
	}
	return strconv.Itoa(n)
}

// cdOkumaMetni okuma süresi metni.
func cdOkumaMetni(dk int) string {
	if dk <= 0 {
		dk = 1
	}
	return strconv.Itoa(dk) + " dk okuma"
}

func cdSlug(sym string) string {
	out := make([]rune, 0, len(sym))
	for _, r := range sym {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		}
	}
	return string(out)
}

func cdLogoOlcu(deger, varsayilan int) string {
	if deger <= 0 || deger > 4000 {
		deger = varsayilan
	}
	return strconv.Itoa(deger)
}

func cdDolu(s string) bool { return strings.TrimSpace(s) != "" }

// cdKirp baştaki/sondaki boşluğu atar (templ'den çağrılıyor).
func cdKirp(s string) string { return strings.TrimSpace(s) }

func cdLogoStil(h, w, girinti int) string {
	if h <= 0 || h > 120 {
		h = 32
	}
	if w <= 0 || w > 600 {
		w = 260
	}
	if girinti < 0 || girinti > 600 {
		girinti = 0
	}
	return "--cd-logo-h:" + strconv.Itoa(h) + "px;--cd-logo-max:" + strconv.Itoa(w) +
		"px;--cd-logo-indent:" + strconv.Itoa(girinti) + "px"
}

func itoaOlcu(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func cdSimgeStil(simgeH, logoH int) string {
	h := simgeH
	if h <= 0 {
		h = logoH
	}
	if h <= 0 || h > 120 {
		h = 32
	}
	return "--cd-simge-h:" + strconv.Itoa(h) + "px"
}
