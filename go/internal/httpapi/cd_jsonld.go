package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// JSON-LD üretimi — Go tarafında.
//
// NEDEN templ içinde değil: templ `<script>` içeriğini HAM metin sayar;
// `{{ }}` ifadeleri çalıştırılmaz. Ölçüm (5 Kasım 2026): çıktıda literal
// `{{ templ.JSONString(...) }}` göründü, JSON geçersiz oldu, Google yapısal
// veriyi okuyamadı. theblock da aynı nedenle jsonld.go'da strings.Builder
// kullanıyor.

// jsonldScript tek bir yapısal veri bloğunu <script> etiketiyle sarar.
//
// `</script>` enjeksiyonuna karşı: json.Marshal `</` dizisini `<\/` yapar
// (Go 1.7+ SetEscapeHTML varsayılanı), bu yüzden ek kaçış gerekmez.
func jsonldScript(v any) template.HTML {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(b) + `</script>`)
}

// orgJSONLD ana sayfa/kurumsal: Organization + WebSite.
func orgJSONLD(siteName, base, canonical string) template.HTML {
	return jsonldScript(map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{"@type": "Organization", "name": siteName, "url": base + "/"},
			map[string]any{
				"@type":      "WebSite",
				"name":       siteName,
				"url":        base + "/",
				"inLanguage": "tr",
				"potentialAction": map[string]any{
					"@type":       "SearchAction",
					"target":      base + "/arama?q={search_term_string}",
					"query-input": "required name=search_term_string",
				},
			},
		},
		"url": canonical,
	})
}

// makaleLDGirdi NewsArticle üretimi için girdi.
type makaleLDGirdi struct {
	View      templkit.CDArticleView
	YayinISO  string // datePublished
	GuncelISO string // dateModified (yoksa YayinISO)
	ID        string // content_id / identifier
	YazarSlug string
	Keywords  []string
	GorselAlt string
	// Gorsel og:image / JSON-LD için MUTLAK görsel adresi (göreli
	// `/kg/...` Open Graph'ta geçersizdir).
	Gorsel string
}

// makaleJSONLD NewsArticle — CoinDesk makale sayfasıyla aynı alan seti.
//
// CoinDesk tek nesne basıyor (@graph YOK) ve şu alanları taşıyor:
// inLanguage, url, identifier, publisher, headline, abstract,
// mainEntityOfPage, thumbnailUrl, image, datePublished, dateModified,
// articleSection, author, keywords.
//
// FARK (bilinçli): `author` CoinDesk'te Person + jobTitle + worksFor +
// description + sameAs + image taşıyor. Bizde yazarın biyografisi/sosyal
// hesabı veritabanında doğrulanmış alan olarak yoksa yalnız name + url
// basılır — uydurma jobTitle yazmak E-E-A-T'yi zayıflatır.
func makaleJSONLD(g makaleLDGirdi) template.HTML {
	v := g.View
	yazar := map[string]any{"@type": "Person", "name": v.Author}
	if g.YazarSlug != "" {
		yazar["url"] = strings.TrimRight(v.BaseURL, "/") + "/yazar/" + g.YazarSlug
	}

	makale := map[string]any{
		"@type":            "NewsArticle",
		"inLanguage":       "tr",
		"url":              v.Canonical,
		"headline":         v.Headline,
		"abstract":         v.Desc,
		"mainEntityOfPage": v.Canonical,
		"articleSection":   v.SectionName,
		"author":           []any{yazar},
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  v.Shell.SiteName,
			"logo":  "",
			"url":   strings.TrimRight(v.BaseURL, "/"),
		},
	}
	if g.ID != "" {
		makale["identifier"] = g.ID
	}
	gorsel := g.Gorsel
	if gorsel == "" {
		gorsel = v.Image
	}
	if gorsel != "" {
		makale["thumbnailUrl"] = gorsel
		makale["image"] = map[string]any{"@type": "ImageObject", "url": gorsel}
	}
	if g.YayinISO != "" {
		makale["datePublished"] = g.YayinISO
		deg := g.GuncelISO
		if deg == "" {
			deg = g.YayinISO
		}
		makale["dateModified"] = deg
	}
	if len(g.Keywords) > 0 {
		makale["keywords"] = g.Keywords
	}

	makale["@context"] = "http://schema.org"
	return jsonldScript(makale)
}

// makaleBreadcrumbJSONLD BreadcrumbList — ayrı blok.
//
// CoinDesk makalede kırıntı basmıyor; ama Google SERP yolunu bu blok
// besliyor ve sayfada GÖRÜNÜR kırıntı yok (kullanıcı isteği) — yapısal
// veri arka planda kalıyor.
func makaleBreadcrumbJSONLD(v templkit.CDArticleView) template.HTML {
	return breadcrumbScript([]crumb{
		{Name: "Ana sayfa", URL: strings.TrimRight(v.BaseURL, "/") + "/"},
		{Name: v.SectionName, URL: strings.TrimRight(v.BaseURL, "/") + v.SectionHref},
		{Name: v.Headline, URL: v.Canonical},
	})
}

// breadcrumbScript kırıntı yolunu <script> etiketiyle sarar.
//
// NEDEN SARMALAYICI BURADA: templ bir `<script>` etiketinin İÇİNDEKİ
// ifadeleri çalıştırmıyor, ham metin sayıyor (7 Kasım 2026'da bu tuşla
// sayfaya literal `@templ.Raw(...)` yazıldı). Bu yüzden script etiketi
// Go'da kurulur ve templ'e tek parça `templ.Raw` olarak verilir.
func breadcrumbScript(items []crumb) template.HTML {
	js := breadcrumbJSONLD(items)
	if js == "" {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(js) + `</script>`)
}

// siteAdiJSONLD yardımcı: sadece adı olan kurum bloğu (liste sayfaları).
func siteAdiJSONLD(siteName, canonical string) template.HTML {
	return jsonldScript(map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     siteName,
		"url":      canonical,
	})
}

// makaleYayinISO içerik yayın zamanını ISO-8601'e çevirir (boş olabilir).
// `repo` bağımlılığı burada kullanılmıyor; imza tutarlılığı için alınıyor.
func makaleYayinISO(c repo.Content) string {
	if c.PublishedAt == nil {
		return ""
	}
	return c.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
}

// trKucult yardımcı (menü/etiket karşılaştırmaları).
func trKucult(s string) string { return strings.ToLower(s) }
