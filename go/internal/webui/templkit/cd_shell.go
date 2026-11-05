package templkit

import "strconv"

// CoinDesk tasarım kabuğu (bitcosar).
// Kaynak: docs/COINDESK-SPEC.md §3 (üst gezinme + fiyat şeridi) ve §6
// (ana sayfa modül sırası). Değerler ölçüldü: nav sırası, şerit içeriği,
// alt bilgi sütunları CoinDesk'in kendi HTML'inden alındı.
//
// TASARIM İLKESİ: gölge yok, ayırma çizgiyle; köşe 2 px; JS yalnız
// etkileşim için (şerit sekmesi). Makale sayfası JS'siz okunur.

// CDTickerItem fiyat şeridindeki tek kalem.
type CDTickerItem struct {
	Sym    string
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

// CDShell, CoinDesk kabuğunun sayfa verisidir.
type CDShell struct {
	SiteName   string
	LangBase   string
	Nav        []CDNavLink
	Ticker     []CDTickerItem
	TickerTab  string // "markets" | "stocks"
	TickerLive string // "Markets Live" / "Crypto Stocks"
	SearchHref string
	LoginHref  string
	SignupHref string
	Locale     string
	// Footer sütunları ölçülen yapıya göre sabittir (aşağıda).
}

// cdFooterColumns CoinDesk alt bilgi sütunları (ölçüm §3).
var cdFooterColumns = []struct {
	Title string
	Links [][2]string // {etiket, yol}
}{
	{"Haberler", [][2]string{
		{"Son Haberler", "/latest-crypto-news"},
		{"Markets", "/markets"},
		{"Finance", "/business"},
		{"Tech", "/tech"},
		{"Policy", "/policy"},
		{"Focus", "/focus"},
	}},
	{"Piyasalar", [][2]string{
		{"Tüm fiyatlar", "/price"},
		{"Bitcoin (BTC)", "/price/bitcoin"},
		{"Ethereum (ETH)", "/price/ethereum"},
		{"XRP", "/price/xrp"},
		{"Solana (SOL)", "/price/solana"},
	}},
	{"Görüş & Analiz", [][2]string{
		{"Görüş", "/opinion"},
		{"Analiz", "/news-analysis"},
		{"Araştırma", "/research"},
		{"Endeksler", "/index"},
	}},
	{"Medya", [][2]string{
		{"Video", "/videos"},
		{"Podcast", "/podcasts"},
		{"Bültenler", "/newsletters"},
		{"RSS", "/rss.xml"},
	}},
	{"Kurumsal", [][2]string{
		{"Hakkımızda", "/hakkimizda"},
		{"İletişim", "/iletisim"},
		{"Gizlilik", "/gizlilik-politikasi"},
		{"Kullanım Koşulları", "/kullanim-kosullari"},
		{"Çerez Politikası", "/cerez-politikasi"},
	}},
}

// CDSekme etiketi: "markets" → "Markets Live".
func cdSekmeEtiketi(tab string) string {
	if tab == "stocks" {
		return "Crypto Stocks"
	}
	return "Markets Live"
}

// cdSiraya numarayı 1-tabanlı metne çevirir (0 → "1").
func cdSiraya(i int) string {
	return strconv.Itoa(i + 1)
}

// cdOkumaMetni okuma süresi metni.
func cdOkumaMetni(dk int) string {
	if dk <= 0 {
		dk = 1
	}
	return strconv.Itoa(dk) + " dk okuma"
}

// cdSlug sembolü fiyat yolu parçasına çevirir (BTC → btc).
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
