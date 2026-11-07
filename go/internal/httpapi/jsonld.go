package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// articleJSONLD makale için NewsArticle yapısal verisi üretir.
func (s *Server) articleJSONLD(c repo.Content, url string, author *repo.PersonRef, tags []string) template.JS {
	img := ""
	if c.CoverMediaID != nil {
		img = strings.TrimRight(s.cfg.BaseURL, "/") + s.mediaURL(c.CoverMediaID)
	}
	pub := ""
	mod := ""
	if c.PublishedAt != nil {
		pub = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	mod = c.UpdatedAt.UTC().Format(time.RFC3339)
	if pub == "" {
		pub = mod
	}
	authorObj := map[string]any{"@type": "Person", "name": c.AuthorName}
	if author != nil {
		authorObj["name"] = author.Name
		authorObj["url"] = strings.TrimRight(s.cfg.BaseURL, "/") + "/yazar/" + author.Slug
	}
	obj := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "NewsArticle",
		"headline":         c.Title,
		"url":              url,
		"description":      orStrAny(c.SeoDesc, c.Spot),
		"inLanguage":       c.Locale,
		"datePublished":    pub,
		"dateModified":     mod,
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": url},
		"author":           authorObj,
		"publisher": map[string]any{
			"@type": "Organization",
			"name":  s.siteName,
			"logo":  map[string]any{"@type": "ImageObject", "url": strings.TrimRight(s.cfg.BaseURL, "/") + "/icon-512.png"},
		},
	}
	if c.CategoryName != nil && *c.CategoryName != "" {
		obj["articleSection"] = *c.CategoryName
	}
	if len(tags) > 0 {
		obj["keywords"] = strings.Join(tags, ", ")
	}
	if img != "" {
		obj["image"] = []string{img}
	}
	b, _ := json.Marshal(obj)
	return template.JS(b)
}

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

func orStrAny(a *string, b string) string {
	if a != nil && *a != "" {
		return *a
	}
	return b
}

// newsSitemap taşındı: cd_sitemaps.go (CoinDesk biçimi: lastmod +
// image:image + CDATA başlık). Burada bırakılan kopya derleme hatası
// veriyordu.

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
