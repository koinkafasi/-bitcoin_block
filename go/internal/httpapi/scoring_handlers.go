package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"theblock/site/internal/repo"
	"theblock/site/internal/scoring"
)

// ---- Kategori AI alaka puanı (spec 06 K-A) ----

// handleAIScore içeriğin AI alaka puanını hesaplar ve kaydeder.
func (s *Server) handleAIScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	fail := func(msg string) {
		if IsHTMX(r) {
			s.writeHTMXTrigger(w, "aiScore", map[string]any{"ok": false, "hata": msg})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "hata": msg})
	}
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		if IsHTMX(r) {
			s.writeHTMXTrigger(w, "aiScore", map[string]any{"ok": false, "hata": "içerik yok"})
			return
		}
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "hata": "içerik yok"})
		return
	}
	if !canEditContent(userFromContext(ctx), c) {
		fail("bu içerikte işlem yapma yetkiniz yok")
		return
	}
	score, note, sinyaller := s.aiRelevanceScore(ctx, c)
	if err := repo.SetContentAIScore(ctx, s.pool, c.ID, score); err != nil {
		fail(err.Error())
		return
	}
	gate := repo.CheckScoreGate(ctx, s.pool, c)
	gate.HasScore = score
	s.audit(r, "icerik_ai_skor", "id="+itoa(int(c.ID))+" skor="+itoa(score))
	if IsHTMX(r) {
		s.writeHTMXTrigger(w, "aiScore", map[string]any{
			"ok": true, "skor": score, "not": note, "sinyaller": sinyaller,
			"kapi_gecer": gate.Allowed, "kapi_nedeni": gate.Reason,
			"min_skor": gate.MinScore, "kategori": gate.Category,
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "skor": score, "not": note, "sinyaller": sinyaller,
		"kapi_gecer": gate.Allowed, "kapi_nedeni": gate.Reason,
		"min_skor": gate.MinScore, "kategori": gate.Category,
	})
}

func (s *Server) aiRelevanceScore(ctx context.Context, c repo.Content) (int, string, []scoring.Signal) {
	tagCount := 0
	if tags, err := repo.TagsOf(ctx, s.pool, c.ID); err == nil {
		tagCount = len(tags)
	}
	score, sig := scoring.Explain(scoring.Input{
		Title:     c.Title,
		Spot:      c.Spot,
		TagCount:  tagCount,
		HasCover:  c.CoverMediaID != nil,
		HasSource: c.SourceURL != nil && *c.SourceURL != "",
	})
	return score, scoring.Note, sig
}

// ---- Trends (spec 06 K-B) ----

func (s *Server) handleAPITrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	days := 7
	if v := r.URL.Query().Get("gun"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	site := s.siteByDomain(ctx, r.Host)
	items, err := repo.ListCatTrends(ctx, s.pool, site.ID, locale, days, 20)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false})
		return
	}
	type row struct {
		Category      string `json:"category"`
		Tag           string `json:"trending_tag"`
		PostCount     int    `json:"post_count"`
		UpdatedAt     string `json:"updated_at"`
		Lang          string `json:"lang"`
		Source        string `json:"source,omitempty"`
		Kind          string `json:"kind,omitempty"`
		Geo           string `json:"geo,omitempty"`
		Score         int    `json:"score,omitempty"`
		ApproxTraffic int    `json:"approx_traffic,omitempty"`
		CoverageCount int    `json:"coverage_count,omitempty"`
		HasCoverage   bool   `json:"has_coverage"`
		External      bool   `json:"external,omitempty"`
	}
	data := make([]row, 0, len(items))
	for _, t := range items {
		data = append(data, row{Category: t.Category, Tag: t.Tag, PostCount: t.PostCount,
			UpdatedAt: t.TrendDate.Format("2006-01-02T15:04:05Z"), Lang: t.Lang})
	}
	// v2: dış trend terimleri (Google Trends) aynı zarfta, ek alanlarla.
	// ?kaynak=google ile yalnız dış terimler istenebilir.
	onlyExternal := r.URL.Query().Get("kaynak") == "google"
	if onlyExternal {
		data = data[:0]
	}
	ext, _ := repo.ListTrendTerms(ctx, s.pool, repo.TrendFilter{
		SiteID: site.ID, MinScore: 0, Limit: 25,
	})
	for _, t := range ext {
		if t.Score < 20 && !onlyExternal {
			continue
		}
		source := t.SourceName
		if source == "" {
			source = "trends"
		}
		data = append(data, row{
			Category: "", Tag: t.Term, PostCount: 0,
			UpdatedAt: t.LastSeenAt.Format("2006-01-02T15:04:05Z"), Lang: locale,
			Source: source, Kind: t.Kind, Geo: t.Geo, Score: t.Score,
			ApproxTraffic: t.ApproxTraffic, CoverageCount: t.CoverageCount,
			HasCoverage: t.CoverageCount > 0, External: true,
		})
		if len(data) >= 40 {
			break
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data, "version": 2})
}
