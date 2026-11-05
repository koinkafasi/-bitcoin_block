package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// News sitemap adaptörü: RSS yayınlamayan ama Google News sitemap'i sunan
// sitelerden başlık, URL ve yayın tarihi alır. İlk birkaç yeni öğe için
// makale özeti ayrıca zenginleştirilir.

const (
	sitemapMaxBodyBytes   = 8 << 20
	sitemapDetailBudget   = 6
	sitemapMaxItemsPerRun = 40
)

type newsSitemapDoc struct {
	URLs []newsSitemapURL `xml:"url"`
}

type newsSitemapURL struct {
	Loc  string `xml:"loc"`
	News struct {
		Title   string `xml:"title"`
		PubDate string `xml:"publication_date"`
	} `xml:"news"`
}

func ScanSitemapFeed(ctx context.Context, pool *pgxpool.Pool, siteID int64, f Feed) ([]RSSItem, error) {
	listing, err := FetchNewsSitemap(ctx, f.URL)
	if err != nil {
		return nil, err
	}
	if len(listing) == 0 {
		return nil, errors.New("haber sitemap'inde öğe bulunamadı")
	}
	out := make([]RSSItem, 0, sitemapMaxItemsPerRun)
	detailBudget := sitemapDetailBudget
	for _, it := range listing {
		if len(out) >= sitemapMaxItemsPerRun {
			break
		}
		h := sha1.Sum([]byte(it.Link))
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_source_items WHERE url_hash=$1)`,
			hex.EncodeToString(h[:])).Scan(&exists)
		if exists {
			continue
		}
		if detailBudget > 0 {
			detailBudget--
			if meta, err := FetchHTMLArticle(ctx, it.Link); err == nil {
				it.Enriched = true
				if meta.Title != "" {
					it.Title = meta.Title
				}
				if meta.Description != "" {
					it.Description = meta.Description
				}
				if !meta.Published.IsZero() {
					it.PubDate = meta.Published
				}
				if meta.Section != "" {
					it.Section = meta.Section
				}
			}
		}
		out = append(out, it)
	}
	return out, nil
}

func FetchNewsSitemap(ctx context.Context, rawURL string) ([]RSSItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "application/xml, text/xml, application/xhtml+xml, */*;q=0.8")
	req.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.7")
	resp, err := htmlHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("sitemap HTTP " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, sitemapMaxBodyBytes))
	if err != nil {
		return nil, err
	}
	return parseNewsSitemap(body), nil
}

func parseNewsSitemap(body []byte) []RSSItem {
	var doc newsSitemapDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]RSSItem, 0, len(doc.URLs))
	for _, u := range doc.URLs {
		link := strings.TrimSpace(u.Loc)
		title := cleanHTMLText(u.News.Title)
		if link == "" || title == "" || seen[link] {
			continue
		}
		seen[link] = true
		out = append(out, RSSItem{
			Title:   truncateRunes(title, 300),
			Link:    link,
			PubDate: parseRSSDate(strings.TrimSpace(u.News.PubDate)),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PubDate.After(out[j].PubDate)
	})
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
