package httpapi

import (
	"strings"
	"time"

	"theblock/site/internal/repo"
)

const bikHaberMerkezi = "HABER MERKEZI"

type bikMeta struct {
	Title     string // <meta name="title">
	URL       string // <meta name="url"> — tam (mutlak) adres
	Section   string // <meta name="articleSection"> — kategori
	Published string // <meta name="datePublished"> — RFC3339
	Modified  string // <meta name="dateModified"> — RFC3339
	Author    string // <meta name="articleAuthor"> — kaynak
}

func buildBikMeta(c repo.Content, author *repo.PersonRef, canonical string) bikMeta {
	m := bikMeta{
		Title:   strings.TrimSpace(c.Title),
		URL:     strings.TrimSpace(canonical),
		Section: strTrimPtr(c.CategoryName),
		Author:  bikArticleAuthor(c, author),
	}
	if c.PublishedAt != nil && !c.PublishedAt.IsZero() {
		m.Published = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		m.Modified = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	// Güncelleme yayından önce görünüyorsa (saat kayması, toplu içe aktarma)
	// yayın zamanını kullan: dateModified < datePublished mantıksızdır.
	if m.Published != "" && c.PublishedAt != nil &&
		!c.UpdatedAt.IsZero() && c.UpdatedAt.Before(*c.PublishedAt) {
		m.Modified = m.Published
	}
	return m
}

func bikArticleAuthor(c repo.Content, author *repo.PersonRef) string {
	if author != nil {
		if ad := strings.TrimSpace(author.Name); ad != "" {
			return ad
		}
	}
	if ad := strTrimPtr(c.SourceName); ad != "" {
		return ad
	}
	return bikHaberMerkezi
}

func strTrimPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
