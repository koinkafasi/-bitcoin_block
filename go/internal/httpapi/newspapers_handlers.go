package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Gazete manşetleri (spec 02 §217) ----

type newspapersData struct {
	Items []repo.NewspaperPage
	Date  string
	Dates []string
	Total int64
}

// handleNewspapersAdmin panel: gazete manşetleri yönetimi.
func (s *Server) handleNewspapersAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildNewspapersData(r)
	if IsHTMX(r) {
		s.renderGazetelerPanelV2(w, r, d)
		return
	}
	s.renderGazetelerPageV2(w, r, d)
}

func (s *Server) buildNewspapersData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	date := strings.TrimSpace(r.URL.Query().Get("tarih"))
	items, last, _ := repo.ListNewspaperPages(ctx, s.pool, sid, date)
	dates, _ := repo.NewspaperDates(ctx, s.pool, sid, 30)
	total, _ := repo.CountNewspaperPages(ctx, s.pool, sid)
	d := s.baseView("tr", "/admin", "Gazete Manşetleri")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.GazetelerPanel = &newspapersData{Items: items, Date: last, Dates: dates, Total: total}
	return d
}

func (s *Server) renderNewspapersPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildNewspapersData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderGazetelerPanelV2(w, r, d)
}

func (s *Server) newspaperResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderNewspapersPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/gazeteler", kind, text)
}

// handleNewspaperSave gazete manşetini ekler/günceller.
func (s *Server) handleNewspaperSave(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("gazete"))
	img := strings.TrimSpace(r.FormValue("gorsel"))
	if name == "" || img == "" {
		s.newspaperResult(w, r, "error", "Gazete adı ve görsel URL gerekli.")
		return
	}
	date := strings.TrimSpace(r.FormValue("tarih"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	slug := platform.Slugify(name)
	if slug == "" {
		slug = "gazete-" + rndSuffix()
	}
	p := &repo.NewspaperPage{
		SiteID:   s.curSiteID(r),
		Name:     name,
		Slug:     slug,
		PageDate: date,
		ImageURL: img,
		ThumbURL: strings.TrimSpace(r.FormValue("kucuk")),
		Sort:     parseIntOr(r.FormValue("sira")),
	}
	if err := repo.UpsertNewspaperPage(r.Context(), s.pool, p); err != nil {
		s.logger.Error("gazete manşet kaydı", "error", err)
		s.newspaperResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	s.audit(r, "gazete_kaydet", name)
	if IsHTMX(r) {
		r2 := withHTMXCurrentQuery(r)
		q := r2.URL.Query()
		q.Set("tarih", date)
		r2.URL.RawQuery = q.Encode()
		d := s.buildNewspapersData(r2)
		d.Flash = &flashMsg{Kind: "info", Text: "Manşet kaydedildi: " + name}
		s.renderGazetelerPanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/gazeteler?tarih="+date, "info", "Manşet kaydedildi: "+name)
}

// handleNewspaperDelete manşeti siler.
func (s *Server) handleNewspaperDelete(w http.ResponseWriter, r *http.Request) {
	if err := repo.DeleteNewspaperPage(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.newspaperResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "gazete_sil", r.PathValue("id"))
	s.newspaperResult(w, r, "info", "Manşet silindi.")
}

// handleNewspaperFetch gazete manşetlerini kaynaktan (gazeteoku.com) çeker.
//
// Paneldeki "Kaynaktan çek" düğmesi bunu çağırır. Zamanlayıcı aynı işi
// günde bir kendiliğinden yapar; bu düğme elle/ZORUNLU çekim içindir.
func (s *Server) handleNewspaperFetch(w http.ResponseWriter, r *http.Request) {
	n, err := repo.FetchGazeteMansetleri(r.Context(), s.pool)
	if err != nil {
		s.logger.Warn("gazete manşet çekimi", "error", err)
		s.newspaperResult(w, r, "error", "Kaynaktan çekilemedi: "+err.Error())
		return
	}
	s.audit(r, "gazete_cek", strconv.Itoa(n))
	d := s.buildNewspapersData(withHTMXCurrentQuery(r))
	d.Flash = &flashMsg{Kind: "info", Text: "Kaynaktan " + strconv.Itoa(n) + " gazete manşeti alındı."}
	if IsHTMX(r) {
		s.renderGazetelerPanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/gazeteler", "info", d.Flash.Text)
}

// handleNewspapersPublic genel gazete manşetleri galerisi.
func (s *Server) handleNewspapersPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	date := strings.TrimSpace(r.URL.Query().Get("tarih"))
	items, last, _ := repo.ListNewspaperPages(ctx, s.pool, site.ID, date)
	dates, _ := repo.NewspaperDates(ctx, s.pool, site.ID, 30)
	d := s.baseView(locale, s.langBase(locale), "Gazete Manşetleri")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Gazeteler = items
	d.GazeteTarih = last
	d.GazeteTarihler = dates
	s.renderPublic(w, r, "gazete-mansetleri", d, cacheControlPublic)
}
