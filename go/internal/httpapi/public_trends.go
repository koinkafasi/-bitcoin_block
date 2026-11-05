package httpapi

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

type trendRSS struct {
	XMLName xml.Name        `xml:"rss"`
	Version string          `xml:"version,attr"`
	Channel trendRSSChannel `xml:"channel"`
}

type trendRSSChannel struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	Language    string         `xml:"language"`
	LastBuild   string         `xml:"lastBuildDate"`
	Items       []trendRSSItem `xml:"item"`
}

type trendRSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// handleTrendRSS /gundem.xml — dış trendlerden RSS 2.0 üretir.
func (s *Server) handleTrendRSS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	items, _ := repo.ListTrendTerms(ctx, s.pool, repo.TrendFilter{
		SiteID: site.ID, MinScore: 20, Limit: 50,
	})
	base := s.absBase(r)
	doc := trendRSS{
		Version: "2.0",
		Channel: trendRSSChannel{
			Title:       s.siteName + " — Gündem",
			Link:        base + "/gundem.xml",
			Description: "Google Trends Türkiye ve site içi gündem terimleri",
			Language:    locale,
			LastBuild:   time.Now().Format(time.RFC1123Z),
		},
	}
	for _, t := range items {
		link := base + "/arama?q=" + url.QueryEscape(t.Term)
		desc := t.Term + " gündemde"
		if t.ApproxTraffic > 0 {
			desc += "; yaklaşık " + itoa(t.ApproxTraffic) + "+ arama"
		}
		if t.CoverageCount == 0 {
			desc += "; sitede henüz kapsama yok"
		} else {
			desc += "; sitede " + itoa(t.CoverageCount) + " içerik"
		}
		doc.Channel.Items = append(doc.Channel.Items, trendRSSItem{
			Title:       t.Term,
			Link:        link,
			GUID:        base + "/gundem.xml#trend-" + itoa(int(t.ID)),
			Description: desc,
			PubDate:     t.LastSeenAt.Format("Mon, 02 Jan 2006 15:04:05 -0700"),
		})
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(doc); err != nil {
		s.logger.Warn("gundem.xml yazılamadı", "error", err)
	}
}

// handleTrendSuggest /api/trend-suggest — editör ve arama kutusu için
// güncel trend terimlerinden otomatik tamamlama.
func (s *Server) handleTrendSuggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, _ := repo.ListTrendTerms(ctx, s.pool, repo.TrendFilter{
		SiteID: site.ID, Q: q, MinScore: 10, Limit: 10,
	})
	type row struct {
		Term          string `json:"term"`
		Score         int    `json:"score"`
		ApproxTraffic int    `json:"approx_traffic,omitempty"`
		CoverageCount int    `json:"coverage_count"`
	}
	data := make([]row, 0, len(items))
	for _, t := range items {
		data = append(data, row{Term: t.Term, Score: t.Score, ApproxTraffic: t.ApproxTraffic, CoverageCount: t.CoverageCount})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}
