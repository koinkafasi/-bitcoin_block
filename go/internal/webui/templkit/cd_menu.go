package templkit

import "strings"

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

func cdSutunKartMi(sut CDMenuColumn) bool {
	for _, it := range sut.Items {
		if strings.TrimSpace(it.Desc) != "" {
			return true
		}
	}
	return false
}

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

func cdMenuGroups(s CDShell) []CDMenuGroup {
	out := make([]CDMenuGroup, 0, len(cdMenuSabit)+1)
	for _, g := range cdMenuSabit {
		// Prices grubu: canlı coinler + hisse girişi.
		if g.Href == "/price" {
			hisse := []CDMenuItem{
				{Label: "Hisse Senetleri", Kind: "baslik"},
				{Label: "BIST hisse fiyatları", Href: "/borsa/hisseler"},
				{Label: "Tüm hisseleri gör", Href: "/borsa/hisseler", Kind: "tumu"},
			}
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
			if len(hisse) > 0 {
				g.Columns = append(g.Columns, CDMenuColumn{Items: hisse})
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
	// ÇEVİRİ — tek noktadan: mega menü ve mobil menü aynı veriyi kullanır,
	// ikisi de buradan geçer (bkz. cd_sozluk.go). Adresler dokunulmaz.
	// Etiketler içerik/kategori adı DEĞİL arayüz metnidir; dinamik etiket
	// sütunu (popüler etiketler) sözlükte yoksa Türkçe kalır — doğru.
	for i := range dolu {
		dolu[i].Label = cdT(s.Locale, dolu[i].Label)
		for j := range dolu[i].Columns {
			for k := range dolu[i].Columns[j].Items {
				it := &dolu[i].Columns[j].Items[k]
				it.Label = cdT(s.Locale, it.Label)
				it.Desc = cdT(s.Locale, it.Desc)
			}
		}
	}
	return dolu
}

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
