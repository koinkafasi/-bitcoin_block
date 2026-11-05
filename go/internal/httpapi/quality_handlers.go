package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// handleContentQuality içeriğin güncel metnini ölçüp kalite panelini döndürür.
// Sonuç hiçbir şeyi otomatik yayınlamaz; editör "Yeniden ölç" ile tazeler.
func (s *Server) handleContentQuality(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, ok := s.editableContent(w, r, id)
	if !ok {
		return
	}
	report := s.contentQuality(r, c)
	view := qualityView(id, report)
	s.attachSimilarity(r, c, &view)
	s.renderTemplPanel(w, r, templkit.KalitePanel(view))
}

// contentQuality içerik başlık + spot + gövdesini ölçer.
func (s *Server) contentQuality(r *http.Request, c repo.Content) quality.Report {
	text := qualityText(c)
	rep := quality.Analyze(text, c.Locale)
	// Sözlük tabanlı denetim yalnız Türkçe ve hunspell kuruluysa çalışır;
	// yoksa kural tabanlı bulgularla devam edilir (sessiz düşüş).
	if s.spell != nil && strings.EqualFold(c.Locale, defaultLocale) &&
		s.spell.Kullanilabilir(r.Context()) {
		words, err := s.spell.Denetle(r.Context(), text)
		if err == nil {
			rep.Imla = append(rep.Imla, quality.HecelemeBulguHazirla(words)...)
		}
	}
	return rep
}

// attachSimilarity yayındaki son içeriklerle yakın-kopya ölçümünü görünüme
// ekler. Hata olursa panel sessizce benzerlik bölümü olmadan gösterilir.
func (s *Server) attachSimilarity(r *http.Request, c repo.Content, v *templkit.IcerikKalite) {
	recent, err := repo.RecentPublishedForCompare(r.Context(), s.pool, c.SiteID, c.ID, 50)
	if err != nil {
		s.logger.Warn("benzerlik adayları", "content", c.ID, "error", err)
		return
	}
	cands := make([]quality.Candidate, 0, len(recent))
	for _, rc := range recent {
		cands = append(cands, quality.Candidate{
			Ref:   strconv.FormatInt(rc.ID, 10),
			Title: rc.Title,
			Text:  qualityText(rc),
		})
	}
	rep := quality.CheckDuplicates(qualityText(c), cands)
	if rep.OK() {
		return
	}
	v.BenzerlikVar = true
	v.BenzerlikOran = fmt.Sprintf("%.0f%%", rep.MaxScore*100)
	for _, m := range rep.Matches {
		v.BenzerlikHits = append(v.BenzerlikHits, templkit.IcerikKaliteBulgu{
			Title: m.Title, Detail: m.Detail,
		})
	}
}

// qualityView raporu templ görünümüne çevirir.
func qualityView(contentID int64, rep quality.Report) templkit.IcerikKalite {
	v := templkit.IcerikKalite{
		Checked:     true,
		Score:       rep.Score,
		Grade:       rep.Grade,
		Words:       strconv.Itoa(rep.Metrics.Words),
		Sentences:   strconv.Itoa(rep.Metrics.Sentences),
		AvgSentence: formatFloat1(rep.Metrics.AvgSentenceWords),
		StdDev:      formatFloat1(rep.Metrics.SentenceStdDev),
		MATTR:       formatFloat3(rep.Metrics.MATTR),
		Cliches:     strconv.Itoa(rep.Metrics.Cliques),
		Repeats:     strconv.Itoa(rep.Metrics.RepeatedPhrases),
		RefreshURL:  "/admin/icerikler/" + itoa64(contentID) + "/kalite",
	}
	for _, b := range rep.Blockers {
		v.Blockers = append(v.Blockers, templkit.IcerikKaliteBulgu{Title: b.Title, Detail: b.Detail})
	}
	for _, wn := range rep.Warnings {
		v.Warnings = append(v.Warnings, templkit.IcerikKaliteBulgu{Title: wn.Title, Detail: wn.Detail})
	}
	for _, im := range rep.Imla {
		v.Imla = append(v.Imla, templkit.IcerikKaliteBulgu{Title: im.Code, Detail: im.Message})
	}
	return v
}

func formatFloat1(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}

func formatFloat3(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
