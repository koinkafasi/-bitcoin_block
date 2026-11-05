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

// makaleJSONLD NewsArticle + BreadcrumbList.
//
// BreadcrumbList Google Haber'de SERP yolunu gösterir. `image` zorunlu
// değil ama Google görsel zengin sonuç için istiyor; varsa eklenir.
func makaleJSONLD(v templkit.CDArticleView, yayinTarihi string) template.HTML {
	makale := map[string]any{
		"@type":            "NewsArticle",
		"headline":         v.Headline,
		"description":      v.Desc,
		"url":              v.Canonical,
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": v.Canonical},
		"articleSection":   v.SectionName,
		"inLanguage":       "tr",
		"author":           map[string]any{"@type": "Organization", "name": v.Author},
		"publisher":        map[string]any{"@type": "Organization", "name": v.Shell.SiteName},
	}
	if v.Image != "" {
		makale["image"] = []string{v.Image}
	}
	if yayinTarihi != "" {
		makale["datePublished"] = yayinTarihi
		makale["dateModified"] = yayinTarihi
	}

	kırıntı := map[string]any{
		"@type": "BreadcrumbList",
		"itemListElement": []any{
			map[string]any{"@type": "ListItem", "position": 1, "name": "Ana sayfa", "item": v.BaseURL + "/"},
			map[string]any{"@type": "ListItem", "position": 2, "name": v.SectionName, "item": v.BaseURL + v.SectionHref},
			map[string]any{"@type": "ListItem", "position": 3, "name": v.Headline},
		},
	}

	return jsonldScript(map[string]any{
		"@context": "https://schema.org",
		"@graph":   []any{makale, kırıntı},
	})
}

// siteAdiJSONLD yardımcı: sadece adı olan kurum bloğu (bölüm/etiket sayfaları).
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
