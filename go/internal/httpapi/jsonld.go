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

// newsSitemap son 48 saatteki yayınlarla Google News sitemap üretir.
func (s *Server) newsSitemap(abs string, items []repo.Content) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">`)
	cut := time.Now().Add(-48 * time.Hour)
	for _, it := range items {
		if it.PublishedAt == nil || it.PublishedAt.Before(cut) {
			continue
		}
		loc := abs + makaleURL(it)
		sb.WriteString("<url><loc>" + htmlEscape(loc) + "</loc><news:news>")
		sb.WriteString("<news:publication><news:name>" + htmlEscape(s.siteName) + "</news:name><news:language>" + htmlEscape(it.Locale) + "</news:language></news:publication>")
		sb.WriteString("<news:publication_date>" + it.PublishedAt.UTC().Format(time.RFC3339) + "</news:publication_date>")
		sb.WriteString("<news:title>" + htmlEscape(it.Title) + "</news:title>")
		sb.WriteString("</news:news></url>")
	}
	sb.WriteString("</urlset>")
	return sb.String()
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
