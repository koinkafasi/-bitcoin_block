package templkit

import "strings"

// CoinDesk mega menüsü — veri modeli.
//
// Yapı `docs/COINDESK-SPEC.md` §3 ölçümünden çıkarıldı (başlık sırası,
// gruplama, sütun düzeni). Ancak **her satırın bir karşılığı olmalı**:
// ölçümdeki CoinDesk program adları (CoinDesk Daily, Public Keys…), kendi
// endeksleri (CD20, XBX) ve ABD hisse sembolleri bizde YOK.
//
// KURAL (bu dosyanın sözleşmesi): menüye yalnız **var olan** sayfa yazılır.
// Ölü bağlantı (404) kullanıcıya "site yarım" der; eksik menü satırı ise
// yalnız "daha az" der. Ölçüm (5 Kasım 2026): menüdeki 49 hedefin 29'u
// 404'tü — yapı tamam görünüyordu, site yarımdı.
//
// VERİYE BAĞLI SÜTUNLAR:
//   · Etiketler (Bitcoin/Ethereum…) — yalnız içeriği olan etiketler
//   · Fiyatlar — canlı ilk 5 coin (CoinGecko önbelleğinden)
// Böylece veri geldikçe menü kendiliğinden dolar, boş sayfa bağlanmaz.

// CDMenuItem bir açılır menü satırı.
//
// Kind alanı satırın rolünü taşır; şablon buna göre biçim verir:
//
//	""        normal bağlantı
//	"baslik"  grup başlığı (tıklanmaz)
//	"tumu"    "tümünü gör" satırı (ayraçtan sonra, vurgulu)
type CDMenuItem struct {
	Label string
	Href  string
	Kind  string
}

// CDMenuColumn açılır menüdeki tek sütun.
type CDMenuColumn struct {
	Items []CDMenuItem
}

// CDMenuGroup üst gezinmedeki tek başlık ve (varsa) açılır menüsü.
type CDMenuGroup struct {
	Label   string
	Href    string
	Columns []CDMenuColumn
}

// cdMenuSabit menünün veriye bağlı OLMAYAN kısmı.
//
// Sıra CoinDesk ölçümüyle aynı: News · Video · Prices · Research · Sponsored.
// "Events" ve "Data & Indices" ÇIKARILDI: kripto etkinliğimiz ve kendi
// endeks metodolojimiz yok (uydurma endeks göstermek yanlış veri olurdu).
// ABD hisse fiyatları da çıkarıldı: veri kaynağı bağlanmadı.
var cdMenuSabit = []CDMenuGroup{
	{
		Label: "News", Href: "/latest-crypto-news",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Latest Crypto News", Href: "/latest-crypto-news"},
				{Label: "Markets", Href: "/markets"},
				{Label: "Finance", Href: "/business"},
				{Label: "Tech", Href: "/tech"},
				{Label: "Policy", Href: "/policy"},
				{Label: "Focus", Href: "/focus"},
				{Label: "Opinion", Href: "/opinion"},
			}},
		},
	},
	{
		Label: "Video", Href: "/videos",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Videos", Href: "/videos"},
				{Label: "Podcasts", Href: "/podcasts"},
			}},
		},
	},
	{
		Label: "Prices", Href: "/price",
		// Sütunlar handler'da canlı veriyle doldurulur (cdMenuFiyatSutunlari).
	},
	{Label: "Research", Href: "/research"},
	{
		Label: "Sponsored", Href: "/sponsored-content",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Thought Leadership", Href: "/sponsored-content"},
				{Label: "Press Releases", Href: "/press-release"},
			}},
		},
	},
}

// cdMenuGroups menüyü, kabuktaki veriyle birleştirir.
//
// `s.MenuTags` ve `s.MenuCoins` boşsa ilgili sütun **hiç basılmaz**:
// içeriği olmayan etikete bağlantı vermek 404'e götürür.
func cdMenuGroups(s CDShell) []CDMenuGroup {
	out := make([]CDMenuGroup, 0, len(cdMenuSabit)+1)
	for _, g := range cdMenuSabit {
		// Prices grubu: canlı coinler.
		if g.Label == "Prices" {
			if len(s.MenuCoins) == 0 {
				// Fiyat servisi yanıt vermediyse menü yine çalışsın:
				// yalnız "tüm fiyatlar" satırı.
				g.Columns = []CDMenuColumn{{Items: []CDMenuItem{
					{Label: "Latest Crypto Prices", Kind: "baslik"},
					{Label: "See all crypto prices", Href: "/price", Kind: "tumu"},
				}}}
			} else {
				items := []CDMenuItem{{Label: "Latest Crypto Prices", Kind: "baslik"}}
				items = append(items, s.MenuCoins...)
				items = append(items, CDMenuItem{Label: "See all crypto prices", Href: "/price", Kind: "tumu"})
				g.Columns = []CDMenuColumn{{Items: items}}
			}
		}
		// News grubuna etiket sütunu eklenir (içeriği olan etiketler).
		if g.Label == "News" && len(s.MenuTags) > 0 {
			tagItems := []CDMenuItem{{Label: "Crypto News", Kind: "baslik"}}
			tagItems = append(tagItems, s.MenuTags...)
			tagItems = append(tagItems,
				CDMenuItem{Label: "News Analysis", Href: "/news-analysis"},
				CDMenuItem{Label: "Press Releases", Href: "/press-release"},
			)
			g.Columns = append(g.Columns, CDMenuColumn{Items: tagItems})
		}
		out = append(out, g)
	}
	return out
}

// cdMenuID açılır menünün aria-controls/id değerini üretir.
// cdSlug yalnız harf/rakam bıraktığı için "Data & Indices" → "dataindices".
func cdMenuID(label string) string { return "cd-mega-" + cdSlug(label) }

// cdHasMenu başlığın açılır menüsü var mı.
func cdHasMenu(g CDMenuGroup) bool { return len(g.Columns) > 0 }

// cdMenuAktif, üst başlığın aktif olup olmadığını s.Nav'dan türetir.
//
// CDShell'de "geçerli yol" alanı yok; aktiflik, handler'ın s.Nav içinde
// işaretlediği bağlantıdan okunur. Böylece mega menü, CDShell'e yeni alan
// eklemeden ve handler'lara dokunmadan mevcut aria-current davranışını
// korur. Karşılaştırma son ek üzerinden yapılır: s.Nav yolları dil ön eki
// taşıyabilir ("/tr/markets"), menü tanımı taşımaz ("/markets").
func cdMenuAktif(nav []CDNavLink, href string) bool {
	if href == "" {
		return false
	}
	for _, l := range nav {
		if !l.Active || l.Href == "" {
			continue
		}
		if l.Href == href || strings.HasSuffix(l.Href, href) {
			return true
		}
	}
	return false
}
