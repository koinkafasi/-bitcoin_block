package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

func TestCDYuzdeSayi(t *testing.T) {
	for _, c := range []struct {
		giris string
		bekle float64
		ok    bool
	}{
		{"+2,17", 2.17, true},
		{"-0.20%", -0.2, true},
		{"0", 0, true},
		{"", 0, false},
		{"abc", 0, false},
		{"1.2.3", 0, false},
	} {
		v, ok := cdYuzdeSayi(c.giris)
		assert.Equal(t, c.ok, ok, c.giris)
		if c.ok {
			assert.InDelta(t, c.bekle, v, 0.0001, c.giris)
		}
	}
}

func TestCDKoleksiyonJSONLD(t *testing.T) {
	h := cdCollectionJSONLD("BitCoŞaR", "https://x.test", "Politika", "https://x.test/policy",
		[]cdLDItem{
			{Headline: "A", URL: "https://x.test/a", Author: "Yazar", Published: "2026-11-06T10:00:00Z"},
			{Headline: "B", URL: "https://x.test/b"},
		})
	s := string(h)
	require.Contains(t, s, `"@type":"CollectionPage"`)
	require.Contains(t, s, `"@type":"ItemList"`)
	require.Contains(t, s, `"headline":"A"`)
	require.Contains(t, s, `"position":2`)

	// Yapı geçerli JSON olmalı: templ'in ham yazdığı blok ayrıştırılabilir.
	jsonStr := strings.TrimSuffix(strings.TrimPrefix(s, `<script type="application/ld+json">`), `</script>`)
	var v map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &v))
	assert.Equal(t, "CollectionPage", v["@type"])
}

func TestCDKisiJSONLDSarmalayicisiz(t *testing.T) {
	// templ `<script>` içine ham yazar; bu yüzden JSON gövdesi SARMALAYICISIZ
	// olmak zorunda (çift script etiketi 7 Kasım 2026'da sayfayı bozdu).
	js := string(cdKisiJSONLD("BitCoŞaR", "https://x.test/yazar/a", "A", "Muhabir", "bio", "", nil))
	assert.False(t, strings.Contains(js, "<script"), "gövde script etiketi içermemeli")
	var v map[string]any
	require.NoError(t, json.Unmarshal([]byte(js), &v))
	assert.Equal(t, "WebPage", v["@type"])
	me, _ := v["mainEntity"].(map[string]any)
	require.NotNil(t, me)
	assert.Equal(t, "Person", me["@type"])
	assert.Equal(t, "Muhabir", me["jobTitle"])
}

func TestCDCoinJSONLDFinansalUrun(t *testing.T) {
	s := string(cdCoinJSONLD("BitCoŞaR", "https://x.test/price/bitcoin", "Bitcoin", "BTC", "75.921,29 $", "+2,17"))
	require.Contains(t, s, `"@type":["WebPage","FinancialProduct"]`)
	require.Contains(t, s, `"Ticker Symbol"`)
	require.Contains(t, s, `"Price Change (24 hours)"`)
	// Oran hesaplanmadığı için BASILMAZ (uydurma değer yasağı).
	assert.NotContains(t, s, "Volume-to-Market-Cap Ratio")
}

func TestCDMakaleJSONLDFields(t *testing.T) {
	v := templkit.CDArticleView{
		Title: "T", Desc: "D", Canonical: "https://x.test/policy/2026/11/06/a",
		Headline: "Başlık", SectionName: "Politika", SectionHref: "/policy",
		Author: "Yazar", BaseURL: "https://x.test",
		Shell: templkit.CDShell{SiteName: "BitCoŞaR"},
	}
	s := string(makaleJSONLD(makaleLDGirdi{
		View: v, YayinISO: "2026-11-06T10:00:00Z", GuncelISO: "2026-11-06T11:00:00Z",
		ID: "42", YazarSlug: "yazar", Keywords: []string{"kripto"},
	}))
	for _, gerekli := range []string{
		`"@type":"NewsArticle"`, `"inLanguage":"tr"`, `"identifier":"42"`,
		`"abstract":"D"`, `"articleSection":"Politika"`,
		`"datePublished":"2026-11-06T10:00:00Z"`,
		`"dateModified":"2026-11-06T11:00:00Z"`,
		`"url":"https://x.test/yazar/yazar"`, `"keywords":["kripto"]`,
		`"@type":"NewsMediaOrganization"`,
	} {
		assert.Contains(t, s, gerekli)
	}
	// CoinDesk tek nesne basıyor: @graph YOK.
	assert.NotContains(t, s, "@graph")
}

func TestCDMakaleBreadcrumbAyriBlok(t *testing.T) {
	v := templkit.CDArticleView{
		Headline: "Başlık", SectionName: "Politika", SectionHref: "/policy",
		Canonical: "https://x.test/policy/2026/11/06/a", BaseURL: "https://x.test",
	}
	s := string(makaleBreadcrumbJSONLD(v))
	assert.True(t, strings.HasPrefix(s, `<script type="application/ld+json">`))
	assert.True(t, strings.HasSuffix(s, `</script>`))
	assert.Contains(t, s, `"@type":"BreadcrumbList"`)
	assert.Equal(t, 1, strings.Count(s, "<script"), "tek script etiketi")
}

func TestCDMakaleMetasYalnizDoluAlan(t *testing.T) {
	m := cdMakaleMetas("BitCoŞaR", makaleMetaGirdi{
		ID: "7", Baslik: "B", Bolum: "Politika", BolumSlug: "policy",
		Yazar: "Y", Kanonik: "https://x.test/a", Etiketler: []string{"kripto", "btc"},
	})
	adlar := map[string]string{}
	for _, x := range m {
		adlar[x.Name] = x.Content
	}
	assert.Equal(t, "7", adlar["content_id"])
	assert.Equal(t, "policy", adlar["content_vertical"])
	assert.Equal(t, "kripto,btc", adlar["tags"])
	assert.Equal(t, "btc", adlar["tag_2"])
	assert.Equal(t, "false", adlar["sponsored"])
	_, varMi := adlar["thumbnail"] // görsel yok → basılmamalı
	assert.False(t, varMi)
	_, varMi = adlar["parsely-image-url"]
	assert.False(t, varMi)
}

func TestCDNewsSitemapCoinDeskBicimi(t *testing.T) {
	s := &Server{siteName: "BitCoŞaR"}
	yayin := time.Now().Add(-2 * time.Hour)
	smap := s.newsSitemap("https://x.test", []repo.Content{{
		ID: 5, Title: `Başlık & <test>`, Locale: "tr",
		PublishedAt: &yayin, UpdatedAt: yayin, ImageURL: "/kg/abc",
	}})
	assert.Contains(t, smap, `<lastmod>`)
	assert.Contains(t, smap, `xmlns:image=`)
	assert.Contains(t, smap, `<image:loc>https://x.test/kg/abc</image:loc>`)
	assert.Contains(t, smap, `<![CDATA[ Başlık & <test> ]]>`)
	assert.Contains(t, smap, `<news:language>tr</news:language>`)
	// 48 saatten eski içerik girmemeli.
	eski := time.Now().Add(-72 * time.Hour)
	smap = s.newsSitemap("https://x.test", []repo.Content{{ID: 6, Title: "Eski", Locale: "tr", PublishedAt: &eski, UpdatedAt: eski}})
	assert.NotContains(t, smap, "<loc>")
}

func TestCDRSSCoinDeskBicimi(t *testing.T) {
	s := &Server{siteName: "BitCoŞaR"}
	yayin := time.Now().Add(-time.Hour)
	rss := s.cdRSSXML("BitCoŞaR", "https://x.test/", "açıklama", "https://x.test", "https://x.test/rss.xml",
		[]repo.Content{{
			ID: 9, Title: "Haber", Spot: "Spot", Locale: "tr", AuthorName: "Yazar",
			PublishedAt: &yayin, UpdatedAt: yayin, ImageURL: "/kg/x",
			CategoryName: strPtr("Politika"), CategorySlug: strPtr("policy"),
		}},
		map[int64][]repo.Etiket{9: {{Ad: "Kripto", Slug: "kripto"}}})
	for _, gerekli := range []string{
		`xmlns:media=`, `<ttl>5</ttl>`, `<sy:updatePeriod>hourly</sy:updatePeriod>`,
		`<atom:link rel="self"`, `<dc:creator>Yazar</dc:creator>`,
		`<media:content url="https://x.test/kg/x"`, `<guid isPermaLink="false">9</guid>`,
		`<category domain="tag"><![CDATA[ Kripto ]]>`,
		`<category domain="https://x.test/policy">`,
	} {
		assert.Contains(t, rss, gerekli)
	}
}

func TestCDMakaleDilimi(t *testing.T) {
	if n, ok := cdMakaleDilimi("articles-7.xml"); !ok || n != 7 {
		t.Fatalf("articles-7.xml → %d/%v", n, ok)
	}
	for _, kötü := range []string{"static-1.xml", "articles-.xml", "articles-x.xml", "crypto-1.xml"} {
		if _, ok := cdMakaleDilimi(kötü); ok {
			t.Errorf("%q dilim sayılmamalı", kötü)
		}
	}
}

func TestCDHeadTemplEtiketleri(t *testing.T) {
	// cdHead'in ürettiği HTML'i templ üzerinden al.
	var sb strings.Builder
	err := templkit.CDHeadParca(templkit.CDHead{
		Title: "T", Desc: "D", Canonical: "https://x.test/a",
		SiteName: "BitCoŞaR", OGType: "article", Image: "https://x.test/i.jpg",
		ImageAlt: "alt", Locale: "tr", PageCategory: "article_page",
		ContentCategory: "policy",
	}).Render(t.Context(), &sb)
	require.NoError(t, err)
	s := sb.String()
	for _, gerekli := range []string{
		`<meta name="googlebot" content="index, follow, max-video-preview:-1, max-image-preview:large, max-snippet:-1"`,
		`<meta name="page_url" content="https://x.test/a"`,
		`<meta name="page_category" content="article_page"`,
		`<meta name="content_category" content="policy"`,
		`<meta name="content_language" content="tr"`,
		`<meta property="og:locale" content="tr_TR"`,
		`<meta property="og:image:width" content="1200"`,
		`<meta property="og:image:alt" content="alt"`,
		`<meta name="twitter:image:height" content="675"`,
		`<link rel="hub" href="https://pubsubhubbub.appspot.com/"`,
	} {
		assert.Contains(t, s, gerekli)
	}
}

func strPtr(s string) *string { return &s }
