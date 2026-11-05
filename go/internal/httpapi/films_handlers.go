package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// ---- Sinema / film listesi (spec 02 §217) ----

type filmsData struct {
	Items []repo.Film
	Week  string
	Weeks []string
	Total int64
}

func handleFilmWeek(v string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	now := time.Now()
	offset := (int(now.Weekday()) + 6) % 7 // Pazartesi = 0
	return now.AddDate(0, 0, -offset).Format("2006-01-02")
}

// handleFilmsAdmin panel: film listesi yönetimi.
func (s *Server) handleFilmsAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildFilmsData(r)
	if IsHTMX(r) {
		s.renderSinemalarPanel(w, r, d)
		return
	}
	s.renderSinemalarPage(w, r, d)
}

func (s *Server) buildFilmsData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	week := strings.TrimSpace(r.URL.Query().Get("hafta"))
	items, last, _ := repo.ListFilms(ctx, s.pool, sid, week)
	weeks, _ := repo.FilmWeeks(ctx, s.pool, sid, 20)
	total, _ := repo.CountFilms(ctx, s.pool, sid)
	d := s.baseView("tr", "/admin", "Sinemalar")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.FilmlerPanel = &filmsData{Items: items, Week: last, Weeks: weeks, Total: total}
	return d
}

func (s *Server) renderFilmsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildFilmsData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderSinemalarPanel(w, r, d)
}

func (s *Server) filmResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderFilmsPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/sinemalar", kind, text)
}

// handleFilmSave film ekler/günceller.
func (s *Server) handleFilmSave(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.FormValue("film"))
	if title == "" {
		s.filmResult(w, r, "error", "Film adı gerekli.")
		return
	}
	f := &repo.Film{
		SiteID:      s.curSiteID(r),
		Cinema:      firstNonEmpty(strings.TrimSpace(r.FormValue("sinema")), "Genel"),
		Title:       title,
		Genre:       strings.TrimSpace(r.FormValue("tur")),
		DurationMin: parseIntOr(r.FormValue("sure")),
		PosterURL:   strings.TrimSpace(r.FormValue("afis")),
		TrailerURL:  strings.TrimSpace(r.FormValue("fragman")),
		Synopsis:    strings.TrimSpace(r.FormValue("ozet")),
		WeekStart:   handleFilmWeek(r.FormValue("hafta")),
		Sort:        parseIntOr(r.FormValue("sira")),
	}
	if err := repo.UpsertFilm(r.Context(), s.pool, f); err != nil {
		s.logger.Error("film kaydı", "error", err)
		s.filmResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	s.audit(r, "film_kaydet", title)
	if IsHTMX(r) {
		r2 := withHTMXCurrentQuery(r)
		q := r2.URL.Query()
		q.Set("hafta", f.WeekStart)
		r2.URL.RawQuery = q.Encode()
		d := s.buildFilmsData(r2)
		d.Flash = &flashMsg{Kind: "info", Text: "Film kaydedildi: " + title}
		s.renderSinemalarPanel(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/sinemalar?hafta="+f.WeekStart, "info", "Film kaydedildi: "+title)
}

// handleFilmDelete filmi siler.
func (s *Server) handleFilmDelete(w http.ResponseWriter, r *http.Request) {
	if err := repo.DeleteFilm(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.filmResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "film_sil", r.PathValue("id"))
	s.filmResult(w, r, "info", "Film silindi.")
}

// handleFilmsPublic genel sinemalar sayfası.
func (s *Server) handleFilmsPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	week := strings.TrimSpace(r.URL.Query().Get("hafta"))
	items, last, _ := repo.ListFilms(ctx, s.pool, site.ID, week)
	weeks, _ := repo.FilmWeeks(ctx, s.pool, site.ID, 20)
	d := s.baseView(locale, s.langBase(locale), "Sinemalar")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Filmler = items
	d.FilmHafta = last
	d.FilmHaftalar = weeks
	s.renderPublic(w, r, "sinemalar", d, cacheControlPublic)
}
