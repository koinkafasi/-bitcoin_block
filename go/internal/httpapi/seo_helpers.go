package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"
)

// crumb kırıntı yolu öğesidir (JSON-LD BreadcrumbList için).
type crumb struct {
	Name string
	URL  string
}

// breadcrumbJSONLD, BreadcrumbList yapısal verisini üretir.
func breadcrumbJSONLD(items []crumb) template.JS {
	list := make([]map[string]any, 0, len(items))
	for i, it := range items {
		if it.Name == "" || it.URL == "" {
			continue
		}
		list = append(list, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     it.Name,
			"item":     it.URL,
		})
	}
	if len(list) == 0 {
		return ""
	}
	b, _ := json.Marshal(map[string]any{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": list,
	})
	return template.JS(b)
}

// truncateRunes metni rune bazında kısaltır (Türkçe karakter güvenli).
func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	cut := string(r[:max])
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

// metaDesc, SEO açıklaması için sırayla verilen adaylardan ilk dolu olanı
// 158 karaktere kısaltarak döndürür.
func metaDesc(candidates ...string) string {
	for _, c := range candidates {
		if strings.TrimSpace(c) != "" {
			return truncateRunes(c, 158)
		}
	}
	return ""
}
