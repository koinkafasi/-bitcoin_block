package httpapi

import (
	"html"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// ---- P5-14: İç link önerisi + evergreen güncelleme (spec 03 P5-B) ----

type seoData struct {
	Evergreen []repo.EvergreenRow
	StaleDays int
	Contents  []repo.Content
	Target    *repo.Content
	Suggs     []linkSugView
}

type linkSugView struct {
	Title   string
	URL     string
	HTML    string
	Reasons string
	Score   int
}

// handleSEOPage iç link önerileri + evergreen güncelleme önerileri sayfası.
func (s *Server) handleSEOPage(w http.ResponseWriter, r *http.Request) {
	d := s.buildSEOData(r)
	if IsHTMX(r) {
		s.renderSeoMerkezPanelV2(w, r, d)
		return
	}
	s.renderSeoMerkezPageV2(w, r, d)
}

func (s *Server) buildSEOData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	locale := s.defaultLang(ctx)

	staleDays := 180
	if v := parseIntOr(r.URL.Query().Get("gun")); v >= 30 && v <= 3650 {
		staleDays = v
	}
	d := s.baseView("tr", "/admin", "SEO: İç Link & Evergreen")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	sd := &seoData{StaleDays: staleDays}
	sd.Evergreen, _ = repo.EvergreenCandidates(ctx, s.pool, sid, staleDays, 50)
	items, _, _ := repo.ListContents(ctx, s.pool, repo.ListFilter{Status: "yayinda", SiteID: sid, PerPage: 300})
	sd.Contents = items

	tid := parseID(r.URL.Query().Get("icerik"))
	if tid == 0 && len(items) > 0 {
		tid = items[0].ID
	}
	if tid > 0 {
		if tc, err := repo.GetContent(ctx, s.pool, tid); err == nil {
			sd.Target = &tc
			base := strings.TrimRight(s.cfg.BaseURL, "/")
			suggs, _ := repo.LinkSuggestions(ctx, s.pool, sid, tid, 10)
			for _, sg := range suggs {
				loc := sg.Locale
				if loc == "" {
					loc = locale
				}
				rel := publicArticleURL(loc, sg.Slug, sg.ID)
				sd.Suggs = append(sd.Suggs, linkSugView{
					Title:   sg.Title,
					URL:     rel,
					Reasons: sg.Reasons,
					Score:   sg.Score,
					HTML:    `<a href="` + html.EscapeString(base+rel) + `">` + htmlEscape(sg.Title) + `</a>`,
				})
			}
		}
	}

	d.Seo = sd
	return d
}

func (s *Server) renderSEOPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildSEOData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderSeoMerkezPanelV2(w, r, d)
}

// handleSEOTouch içeriği "güncellendi" olarak işaretler (updated_at + sürüm).
func (s *Server) handleSEOTouch(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.FormValue("id"))
	if id > 0 {
		if err := repo.TouchContentUpdate(r.Context(), s.pool, id); err != nil {
			if IsHTMX(r) {
				s.renderSEOPanel(w, r, "error", "İşaretlenemedi.")
				return
			}
			s.flashRedirect(w, r, "/admin/seo", "error", "İşaretlenemedi.")
			return
		}
		s.audit(r, "icerik_guncelle_isaretle", "id="+itoa(int(id)))
	}
	if IsHTMX(r) {
		s.renderSEOPanel(w, r, "info", "İçerik 'güncellendi' olarak işaretlendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/seo", "info", "İçerik 'güncellendi' olarak işaretlendi.")
}
