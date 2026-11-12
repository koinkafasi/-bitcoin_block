package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"

	"theblock/site/internal/repo"
)

// orgJSONLD ana sayfa için Organization + WebSite + SearchAction.
func (s *Server) orgJSONLD() template.JS {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	obj := []map[string]any{
		{"@context": "https://schema.org", "@type": "Organization", "name": s.siteName, "url": base + "/"},
		{"@context": "https://schema.org", "@type": "WebSite", "name": s.siteName, "url": base + "/", "inLanguage": "tr",
			"potentialAction": map[string]any{
				"@type":       "SearchAction",
				"target":      map[string]any{"@type": "EntryPoint", "urlTemplate": base + "/arama?q={search_term_string}"},
				"query-input": "required name=search_term_string",
			}},
	}
	b, _ := json.Marshal(obj)
	return template.JS(b)
}

// sitemapLangRefs çeviri köklerini gruplar (hreflang üretimi için).
func sitemapLangRefs(refs []repo.LangRef) map[int64][]repo.LangRef {
	out := map[int64][]repo.LangRef{}
	for _, r := range refs {
		out[r.RootID] = append(out[r.RootID], r)
	}
	return out
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")
	return r.Replace(s)
}
