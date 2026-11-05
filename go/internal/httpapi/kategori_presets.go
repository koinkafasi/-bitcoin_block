package httpapi

// Kategori tasarım şablonları: yönetimden tek tıkla uygulanan hazır
// blok düzenleri + liste görünümü. Şablon "kategori:<slug>" page_layout
// anahtarına yazılır; böylece Sayfa Tasarımcısı'nda sonradan düzenlenebilir.

type categoryPreset struct {
	Key    string
	Name   string
	Desc   string
	View   string // liste görünümü; boş = genel ayar
	Blocks string // page_layouts.blocks JSON; boş = modülsüz
}

var categoryPresets = []categoryPreset{
	{
		Key:  "genel",
		Name: "Genel Düzen",
		Desc: "Kategoriye özel düzen ve görünüm kullanılmaz; genel kategori ayarları geçerli olur.",
	},
	{
		Key:  "piyasa",
		Name: "Piyasa / Ekonomi",
		Desc: "Üstte manşet şeridi, ortada haber listesi, sağda numaralı trend sütunu.",
		Blocks: `[
			{"id":"cp-ust","type":"ust_manset","visible":true,"opts":{"count":6,"zone":"ust"}},
			{"id":"cp-liste","type":"kategori_liste","visible":true,"opts":{"zone":"govde"}},
			{"id":"cp-trend","type":"trend_numarali","visible":true,"opts":{"count":5,"zone":"yan"}}
		]`,
	},
	{
		Key:  "manset",
		Name: "Manşet + Liste",
		Desc: "1 büyük manşet, altında haber listesi ve sağda gündem etiketleri.",
		Blocks: `[
			{"id":"cp-manset","type":"manset","visible":true,"opts":{"count":5,"zone":"ust"}},
			{"id":"cp-liste","type":"kategori_liste","visible":true,"opts":{"zone":"govde"}},
			{"id":"cp-trend","type":"trend","visible":true,"opts":{"count":10,"zone":"yan"}}
		]`,
	},
	{
		Key:  "dergi",
		Name: "Dergi",
		Desc: "Manşet, haber listesi ve sağda foto galeri.",
		Blocks: `[
			{"id":"cp-manset","type":"manset","visible":true,"opts":{"count":5,"zone":"ust"}},
			{"id":"cp-liste","type":"kategori_liste","visible":true,"opts":{"zone":"govde"}},
			{"id":"cp-galeri","type":"galeri_foto","visible":true,"opts":{"count":6,"zone":"yan"}}
		]`,
	},
	{
		Key:  "klasik",
		Name: "Klasik",
		Desc: "Modülsüz klasik liste: 1 büyük haber + kartlar.",
		View: "klasik",
	},
	{
		Key:  "sade-liste",
		Name: "Sade Liste",
		Desc: "Modülsüz, görsel solda sıkı haber listesi.",
		View: "liste",
	},
	{
		Key:  "izgara",
		Name: "Izgara",
		Desc: "Modülsüz kart ızgarası.",
		View: "izgara",
	},
	{
		Key:  "mozaik",
		Name: "Mozaik",
		Desc: "Modülsüz mozaik düzen.",
		View: "mozaik",
	},
}

func categoryPresetByKey(key string) (categoryPreset, bool) {
	for _, p := range categoryPresets {
		if p.Key == key {
			return p, true
		}
	}
	return categoryPreset{}, false
}

func validCategoryView(v string) bool {
	switch v {
	case "", "klasik", "liste", "izgara", "mozaik", "akis":
		return true
	}
	return false
}
