package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func TestParseHTMLListingExtractsArticles(t *testing.T) {
	doc := `
<html><body>
<nav>
  <a href="/kategori/gundem">Gündem kategorisindeki tüm haberleri görüntüle</a>
  <a href="/etiket/ekonomi">Ekonomi etiketiyle ilgili haberleri görüntüle</a>
</nav>
<main>
  <a href="/gundem/buyuk-haber-bugun-boyle-oldu-123456">Büyük haber bugün böyle oldu ve gelişmeler</a>
  <a href="/gundem/buyuk-haber-bugun-boyle-oldu-123456?utm_source=home">Büyük haber bugün böyle oldu ve gelişmeler</a>
  <a href="https://example.com/ekonomi/piyasalarda-son-durum-ve-beklentiler-654321"><span>Piyasalarda son durum ve beklentiler</span></a>
  <a href="https://baska.example/haber/dis-haber-987654">Dış kaynaktaki haber başlığı burada</a>
  <a href="/foto-galeri/resimli-haber-555555">Foto galeri başlığı burada</a>
  <a href="/forum/mchose-kulaklik-alinir-mi-777777">MCHOSE V9 Pro kulaklık alınır mı?</a>
  <a href="/spor/canli-58-lille-galatasaray-888888">CANLI 58' Lille 2 Galatasaray 3</a>
  <a href="/spor/bugun-2200-aston-villa-viking-999999">Bugün 22:00 Aston Villa Viking</a>
  <a href="/haber/">Kısa</a>
</main>
</body></html>`
	items := parseHTMLListing("https://example.com", doc)
	require.Len(t, items, 2)
	assert.Equal(t, "Büyük haber bugün böyle oldu ve gelişmeler", items[0].Title)
	assert.Equal(t, "https://example.com/gundem/buyuk-haber-bugun-boyle-oldu-123456", items[0].Link)
	assert.NotContains(t, items[0].Link, "utm_")
	assert.Equal(t, "Piyasalarda son durum ve beklentiler", items[1].Title)
	assert.Equal(t, "https://example.com/ekonomi/piyasalarda-son-durum-ve-beklentiler-654321", items[1].Link)
}

func TestParseHTMLListingDeduplicatesTitles(t *testing.T) {
	doc := `
<a href="/gundem/ayni-haber-basligi-burada-123456">Aynı haber başlığı burada ve ayrıntılar</a>
<a href="/gundem/ayni-haber-basligi-burada-123457">Aynı haber başlığı burada ve ayrıntılar</a>`
	items := parseHTMLListing("https://example.com", doc)
	assert.Len(t, items, 1)
}

func TestParseHTMLArticleMetaJSONLD(t *testing.T) {
	doc := `<html><head>
<script type="application/ld+json">
{"@context":"https://schema.org","@graph":[
  {"@type":"WebSite","name":"Örnek"},
  {"@type":"NewsArticle","headline":"Ekonomide kritik karar","description":"Merkez Bankası yeni kararını açıkladı.","articleSection":"Ekonomi","datePublished":"2026-10-21T19:06:00+03:00"}
]}
</script></head></html>`
	meta := parseHTMLArticleMeta(doc)
	assert.Equal(t, "Ekonomide kritik karar", meta.Title)
	assert.Equal(t, "Merkez Bankası yeni kararını açıkladı.", meta.Description)
	assert.Equal(t, "Ekonomi", meta.Section)
	assert.Equal(t, 2026, meta.Published.Year())
	assert.Equal(t, time.October, meta.Published.Month())
}

func TestParseHTMLArticleMetaFallback(t *testing.T) {
	doc := `<html><head>
<title>Son dakika haberi - Örnek Site</title>
<meta property="og:title" content="Son dakika haberi">
<meta name="description" content="Haberin kısa açıklaması burada yer alıyor.">
<meta property="article:published_time" content="2026-10-21T20:05:00+03:00">
</head></html>`
	meta := parseHTMLArticleMeta(doc)
	assert.Equal(t, "Son dakika haberi", meta.Title)
	assert.Equal(t, "Haberin kısa açıklaması burada yer alıyor.", meta.Description)
	assert.False(t, meta.Published.IsZero())
}

func TestSectionCategorySlug(t *testing.T) {
	cases := map[string]string{
		"GÜNDEM":          "gundem",
		"Güncel":          "gundem",
		"ASAYİŞ":          "gundem",
		"Politika":        "gundem",
		"EKONOMİ":         "ekonomi",
		"Borsa Haberleri": "ekonomi",
		"KAP Haberleri":   "ekonomi",
		"Spor":            "spor",
		"DÜNYA":           "dunya",
		"Teknoloji":       "teknoloji",
		"Kültür-Sanat":    "egitim-kultur",
		"Sağlık":          "yasam",
		"Bilinmeyen":      "",
	}
	for section, want := range cases {
		assert.Equal(t, want, sectionCategorySlug(section), section)
	}
}

func TestParseHTMLArticleBreadcrumbSection(t *testing.T) {
	doc := `<html><head>
<meta name=articleSection content=news>
<script type=application/ld+json>{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"item":{"@id":"https://example.com","name":"Anasayfa"}},{"@type":"ListItem","position":2,"item":{"@id":"https://example.com/kategori/spor-12","name":"Spor"}}]}</script>
</head></html>`
	meta := parseHTMLArticleMeta(doc)
	assert.Equal(t, "Spor", meta.Section)
}

func TestDecodeHTMLCharsetWindows1254(t *testing.T) {
	raw := []byte(`<meta charset="windows-1254"><a href="/spor/sergen-yalcin-imza-toreni-123456">Sergen Yalçın için imza töreni düzenlenecek</a>`)
	encoded, err := charmap.Windows1254.NewEncoder().Bytes(raw)
	require.NoError(t, err)
	decoded := string(decodeHTMLCharset(encoded))
	assert.Contains(t, decoded, "Sergen Yalçın için imza töreni")
}

func TestCleanHTMLTitleRelativeTime(t *testing.T) {
	assert.Equal(t, "Yapay Zekâya Komut Yaz", cleanHTMLTitle("Yapay Zekâya Komut Yaz 1 gün önce"))
	assert.Equal(t, "Son dakika gelişmesi", cleanHTMLTitle("Son dakika gelişmesi dün"))
}
