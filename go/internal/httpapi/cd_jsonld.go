package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

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
	// Kilitli içerik abonelik gerektiriyor mu. Google, paywall'lu
	// içeriğin BİLDİRİLMESİNİ istiyor: aksi hâlde Googlebot'un gördüğü
	// tam metinle kullanıcının gördüğü kesik metin arasındaki fark
	// "cloaking" sayılabiliyor.
	Kilitli bool
}

func makaleJSONLD(g makaleLDGirdi) template.HTML {
	v := g.View
	yazar := map[string]any{"@type": "Person", "name": v.Author}
	if g.YazarSlug != "" {
		yazar["url"] = strings.TrimRight(v.BaseURL, "/") + "/yazar/" + g.YazarSlug
	}

	makale := map[string]any{
		"@type":      "NewsArticle",
		"inLanguage": "tr",
		"url":        v.Canonical,
		"headline":   v.Headline,
		"abstract":   v.Desc,
		// Ücretsiz erişim durumu HER ZAMAN yazılıyor: `true` olduğunda da
		// açık bir sinyal, Google için belirsizlik bırakmıyor.
		"isAccessibleForFree": !g.Kilitli,
		"mainEntityOfPage":    v.Canonical,
		"articleSection":      v.SectionName,
		"author":              []any{yazar},
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

func makaleBreadcrumbJSONLD(v templkit.CDArticleView) template.HTML {
	return breadcrumbScript([]crumb{
		{Name: "Ana sayfa", URL: strings.TrimRight(v.BaseURL, "/") + "/"},
		{Name: v.SectionName, URL: strings.TrimRight(v.BaseURL, "/") + v.SectionHref},
		{Name: v.Headline, URL: v.Canonical},
	})
}

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
