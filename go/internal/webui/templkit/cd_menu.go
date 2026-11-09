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
//
// Desc doluysa sütun "açıklamalı kart" tipine geçer (spec §7.2A Tip B:
// 257×96 kart, 14/20 w500 başlık + 12/16 w400 açıklama). Boşsa düz
// bağlantı listesi (Tip A) basılır. Tek alanla iki yerleşim: menüye yeni
// bir "tip" sabiti eklemek yerine VERİ neyi gerektiriyorsa şablon ona uyar.
type CDMenuItem struct {
	Label string
	Href  string
	Kind  string
	Desc  string
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
		Label: "Haberler", Href: "/latest-crypto-news",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Son Kripto Haberleri", Href: "/latest-crypto-news"},
				{Label: "Piyasalar", Href: "/markets"},
				{Label: "Finans", Href: "/business"},
				{Label: "Teknoloji", Href: "/tech"},
				{Label: "Politika", Href: "/policy"},
				{Label: "Odak", Href: "/focus"},
				{Label: "Görüş", Href: "/opinion"},
			}},
		},
	},
	{
		Label: "Video", Href: "/videos",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Videolar", Href: "/videos"},
				{Label: "Podcastlar", Href: "/podcasts"},
			}},
		},
	},
	{
		Label: "Fiyatlar", Href: "/price",
		// Sütunlar handler'da canlı veriyle doldurulur (cdMenuFiyatSutunlari).
	},
	{
		Label: "Araştırma", Href: "/research",
		// AÇIKLAMALI KART TİPİ (spec §7.2A Tip B). CoinDesk Research
		// panelinde kart başlığı 14/20 w500 + tek satır açıklama 12/16
		// w400. Bizde karşılığı olan gerçek sayfalar bunlar; açıklamalar
		// sayfaların kendi meta açıklamalarından kısaltıldı.
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Araştırma Merkezi", Href: "/research",
					Desc: "Araştırma raporları ve veri analizleri"},
				{Label: "Haber Analizi", Href: "/news-analysis",
					Desc: "Derinlemesine haber analizleri"},
				{Label: "Focus", Href: "/focus",
					Desc: "Odak dosyaları ve uzun okumalar"},
			}},
		},
	},
	{
		Label: "Sponsorlu", Href: "/sponsored-content",
		Columns: []CDMenuColumn{
			{Items: []CDMenuItem{
				{Label: "Düşünce Liderliği", Href: "/sponsored-content"},
				{Label: "Basın Bültenleri", Href: "/press-release"},
			}},
		},
	},
}

// cdSutunKartMi sütunun açıklamalı kart tipiyle basılacağını söyler.
//
// Tek ölçüt: satırlardan birinde Desc dolu mu. Açıklamayı yutup düz liste
// basmak, ölçülen bilgiyi çöpe atardı; tam tersi, açıklaması olmayan
// satırı karta çevirmek ise 96 px'lik boş kartlar üretirdi.
func cdSutunKartMi(sut CDMenuColumn) bool {
	for _, it := range sut.Items {
		if strings.TrimSpace(it.Desc) != "" {
			return true
		}
	}
	return false
}

// cdMenuPanelSinifi panelin genişlik/hizalama sınıfını üretir.
//
// CoinDesk panel genişliğini MENÜ BAŞINA sabitliyor (spec §7.2A):
// 589 · 728 · 750 · 836 · 960. Genişlik içeriğe göre seçilmiş; bizde
// karşılığı olan üç kademe var:
//
//	589  tek sütunlu liste (Video · Prices · Sponsored)
//	750  iki sütunlu liste (News — bölümler + canlı etiketler)
//	836  üç sütunlu açıklamalı kart ızgarası (Research)
//
// Sponsored ayrıca SAĞA hizalı: son başlık olduğu için `left: 0` paneli
// ekrandan taşırıyordu (spec §7.2A istisnası).
func cdMenuPanelSinifi(g CDMenuGroup) string {
	sinif := "cd-nav__grup--589"
	switch {
	case cdMenuKartVar(g):
		sinif = "cd-nav__grup--836"
	case len(g.Columns) > 1:
		sinif = "cd-nav__grup--750"
	}
	if strings.EqualFold(g.Label, "Sponsored") {
		sinif += " cd-nav__grup--sag"
	}
	return sinif
}

// cdMenuKartVar grubun herhangi bir sütunu kart tipinde mi.
func cdMenuKartVar(g CDMenuGroup) bool {
	for _, s := range g.Columns {
		if cdSutunKartMi(s) {
			return true
		}
	}
	return false
}

// cdMobilAnaGruplar ve cdMobilMedyaGruplar mobil menüyü CoinDesk'in İKİ
// bloğuna ayırır: ana bölümler, sonra medya (Video/Podcast).
//
// NEDEN: CoinDesk mobil menüsü (JS kapalı, ölçüldü 9 Kasım 2026) önce
// News/Prices/Data/Research/Events/Sponsored bloğunu, bir ayraçtan sonra
// Videos/Podcasts/Newsletters bloğunu basıyor. Masaüstü sırası DEĞİŞMEZ;
// yalnız mobil görünüm bu iki bloğa ayrılır.
func cdMobilAnaGruplar(s CDShell) []CDMenuGroup {
	out := make([]CDMenuGroup, 0, len(cdMenuSabit))
	for _, g := range cdMenuGroups(s) {
		if !cdMedyaGrubu(g) {
			out = append(out, g)
		}
	}
	return out
}

func cdMobilMedyaGruplar(s CDShell) []CDMenuGroup {
	out := make([]CDMenuGroup, 0, 2)
	for _, g := range cdMenuGroups(s) {
		if cdMedyaGrubu(g) {
			out = append(out, g)
		}
	}
	return out
}

// cdMedyaGrubu video/podcast bölümlerini işaretler.
func cdMedyaGrubu(g CDMenuGroup) bool {
	return g.Href == "/videos" || g.Href == "/podcasts"
}

// cdMenuGroups menüyü, kabuktaki veriyle birleştirir.
//
// `s.MenuTags` ve `s.MenuCoins` boşsa ilgili sütun **hiç basılmaz**:
// içeriği olmayan etikete bağlantı vermek 404'e götürür.
//
// Aynı kural bölüm satırları için de geçerli: `s.MenuBolumler` doluysa
// yalnız YAYINDA İÇERİĞİ OLAN bölümlerin satırı basılır. Ölçüm (6 Kasım
// 2026): Focus · Videos · Podcasts · News Analysis · Press Releases
// bölümlerinde 0 içerik vardı; menü beş ölü bağlantı veriyordu.
func cdMenuGroups(s CDShell) []CDMenuGroup {
	out := make([]CDMenuGroup, 0, len(cdMenuSabit)+1)
	for _, g := range cdMenuSabit {
		// Prices grubu: canlı coinler.
		if g.Href == "/price" {
			if len(s.MenuCoins) == 0 {
				// Fiyat servisi yanıt vermediyse menü yine çalışsın:
				// yalnız "tüm fiyatlar" satırı.
				g.Columns = []CDMenuColumn{{Items: []CDMenuItem{
					{Label: "Son Kripto Fiyatları", Kind: "baslik"},
					{Label: "Tüm kripto fiyatlarını gör", Href: "/price", Kind: "tumu"},
				}}}
			} else {
				items := []CDMenuItem{{Label: "Son Kripto Fiyatları", Kind: "baslik"}}
				items = append(items, s.MenuCoins...)
				items = append(items, CDMenuItem{Label: "Tüm kripto fiyatlarını gör", Href: "/price", Kind: "tumu"})
				g.Columns = []CDMenuColumn{{Items: items}}
			}
		}
		// News grubuna etiket sütunu eklenir (içeriği olan etiketler).
		if g.Href == "/latest-crypto-news" && len(s.MenuTags) > 0 {
			tagItems := []CDMenuItem{{Label: "Kripto Haberleri", Kind: "baslik"}}
			tagItems = append(tagItems, s.MenuTags...)
			tagItems = append(tagItems,
				CDMenuItem{Label: "Haber Analizi", Href: "/news-analysis"},
				CDMenuItem{Label: "Basın Bültenleri", Href: "/press-release"},
			)
			g.Columns = append(g.Columns, CDMenuColumn{Items: tagItems})
		}
		// İçeriği olmayan bölüm satırları düşürülür.
		g.Columns = cdBosBolumleriEle(s, g.Columns)
		out = append(out, g)
	}
	// Tüm sütunları boşalan grup (ör. yayında videosu ve podcast'i yokken
	// "Video") hiç basılmaz: açılır menüsü boş bir başlık tıklanınca boş
	// sayfaya götürür.
	dolu := out[:0]
	for _, g := range out {
		if len(g.Columns) > 0 || strings.TrimSpace(g.Href) == "" {
			dolu = append(dolu, g)
			continue
		}
		// Sütunsuz grup yalnız hedef sayfası doluysa kalır (ör. Research
		// kart sütunu boşaldıysa ama /research doluysa başlık kalsın).
		if cdBolumVar(s, g.Href) {
			dolu = append(dolu, g)
		}
	}
	return dolu
}

// cdBosBolumleriEle sütunlardan içeriği olmayan bölüm satırlarını eler.
//
// Ölçüt `cdBolumVar`: yalnız TEK PARÇALI yollar (/videos, /focus …) bölüm
// sayılır. /latest-crypto-news ve /price gibi liste sayfaları ve /tag/x
// etiketleri bu süzgecin dışında — onların doluluğu MenuTags/MenuCoins ile
// ayrıca denetleniyor.
func cdBosBolumleriEle(s CDShell, sutunlar []CDMenuColumn) []CDMenuColumn {
	if len(s.MenuBolumler) == 0 {
		return sutunlar
	}
	out := make([]CDMenuColumn, 0, len(sutunlar))
	for _, sut := range sutunlar {
		items := make([]CDMenuItem, 0, len(sut.Items))
		for _, it := range sut.Items {
			if !cdBolumVar(s, it.Href) {
				// Bölüm sayfası ve yayında içeriği yok → satır basılmaz.
				if cdBolumYoluMu(it.Href) {
					continue
				}
			}
			items = append(items, it)
		}
		if len(items) == 0 {
			continue
		}
		out = append(out, CDMenuColumn{Items: items})
	}
	return out
}

// cdBolumVar hedef bölümün yayında içeriği var mı.
//
// MenuBolumler boşsa (veri yüklenmemiş) her yol "var" sayılır: filtre
// yalnız gerçek veriyle çalışır, yoksa menüyü kazara boşaltırdı.
func cdBolumVar(s CDShell, href string) bool {
	slug := cdBolumSlug(href)
	if slug == "" {
		return true
	}
	if len(s.MenuBolumler) == 0 {
		return true
	}
	return s.MenuBolumler[slug]
}

// cdBolumYoluMu yolun tek parçalı bir bölüm adresi olup olmadığı.
func cdBolumYoluMu(href string) bool { return cdBolumSlug(href) != "" }

// cdBolumSlug "/videos" → "videos"; "/tag/btc" ve "/" → "".
func cdBolumSlug(href string) string {
	h := strings.Trim(strings.TrimSpace(href), "/")
	if h == "" || strings.Contains(h, "/") {
		return ""
	}
	return h
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
