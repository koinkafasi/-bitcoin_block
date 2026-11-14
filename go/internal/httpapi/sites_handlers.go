package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// sitesData Sites yönetim sayfası verisidir.
type sitesData struct {
	Items []repo.Site
	Stats []repo.SiteStat
	RPM   float64
	Tot   repo.SiteStat
}

// curSiteID çerezden seçili siteyi döndürür (yoksa 1).
func (s *Server) curSiteID(r *http.Request) int64 {
	if c, err := r.Cookie("tb_site"); err == nil {
		if n, e := strconv.ParseInt(c.Value, 10, 64); e == nil && n > 0 {
			return n
		}
	}
	return 1
}

// siteByDomain host'tan aktif siteyi çözer; bulunamazsa varsayılan site 1.
func (s *Server) siteByDomain(ctx context.Context, host string) repo.Site {
	host = strings.TrimSpace(host)
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	domain := strings.ToLower(host)
	if domain == "" {
		domain = varsayilanDomain
	}
	// bellek önbelleği (60 sn) — her istekte DB sorgusunu önler
	s.siteMu.Lock()
	if e, ok := s.siteCache[domain]; ok && time.Since(e.at) < 60*time.Second {
		s.siteMu.Unlock()
		return e.site
	}
	s.siteMu.Unlock()

	var st repo.Site
	if got, err := repo.GetSiteByDomain(ctx, s.pool, domain); err == nil && got.ID > 0 {
		st = got
	} else if d, err2 := repo.GetSite(ctx, s.pool, 1); err2 == nil {
		st = d
	}
	s.siteMu.Lock()
	s.siteCache[domain] = siteEntry{at: time.Now(), site: st}
	s.siteMu.Unlock()
	return st
}

type siteEntry struct {
	at   time.Time
	site repo.Site
}

// handleSitesGet siteleri listeler.
func (s *Server) handleSitesGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildSitesData(r)
	if IsHTMX(r) {
		s.renderSitelerPanelV2(w, r, d)
		return
	}
	s.renderSitelerPageV2(w, r, d)
}

func (s *Server) buildSitesData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Siteler (Yayın Ağı)")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ctx := r.Context()
	items, _ := repo.ListSites(ctx, s.pool)
	// P5-G: pilot dashboard (site × içerik/trafik/gelir/maliyet)
	rpm := repo.GetDefaultRPM(ctx, s.pool)
	stats, _ := repo.SiteStats(ctx, s.pool, rpm)
	sd := &sitesData{Items: items, Stats: stats, RPM: rpm}
	for _, st := range stats {
		sd.Tot.Content += st.Content
		sd.Tot.Published += st.Published
		sd.Tot.Hits += st.Hits
		sd.Tot.Revenue += st.Revenue
		sd.Tot.AICost += st.AICost
		sd.Tot.Profit += st.Profit
	}
	d.Siteler = sd
	return d
}

func (s *Server) renderSitelerPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildSitesData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderSitelerPanelV2(w, r, d)
}

func (s *Server) sitesResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderSitelerPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/siteler", kind, text)
}

// handleSitesEkle yeni site ekler.
func (s *Server) handleSitesEkle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	if name == "" || domain == "" {
		s.sitesResult(w, r, "error", "Ad ve alan adı gerekli.")
		return
	}
	if _, err := repo.CreateSite(r.Context(), s.pool, name, domain, r.FormValue("brand"), r.FormValue("locale")); err != nil {
		s.sitesResult(w, r, "error", "Eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "site_ekle", name)
	s.sitesResult(w, r, "info", name+" eklendi.")
}

// handleSitesGuncelle site günceller.
func (s *Server) handleSitesGuncelle(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		s.sitesResult(w, r, "error", "Geçersiz site.")
		return
	}
	st := &repo.Site{
		ID:            id,
		Name:          strings.TrimSpace(r.FormValue("name")),
		Domain:        strings.ToLower(strings.TrimSpace(r.FormValue("domain"))),
		Status:        r.FormValue("status"),
		DefaultLocale: r.FormValue("locale"),
	}
	st.Brand = optStr(r.FormValue("brand"))
	if st.Status != "aktif" && st.Status != "pasif" {
		st.Status = "aktif"
	}
	if err := repo.UpdateSite(r.Context(), s.pool, st); err != nil {
		s.sitesResult(w, r, "error", "Güncellenemedi: "+err.Error())
		return
	}
	s.sitesResult(w, r, "info", "Güncellendi.")
}

// handleSitesSil site siler.
func (s *Server) handleSitesSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 1 {
		s.sitesResult(w, r, "error", "Varsayılan site silinemez.")
		return
	}
	if err := repo.DeleteSite(r.Context(), s.pool, id); err != nil {
		s.sitesResult(w, r, "error", "Silinemedi (içerik bağlı olabilir).")
		return
	}
	s.sitesResult(w, r, "info", "Silindi.")
}

// handleSiteSec seçili siteyi çereze yazar.
func (s *Server) handleSiteSec(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Query().Get("site"))
	if id == 0 {
		id = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "tb_site",
		Value:    itoa(int(id)),
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	target := s.ayniSiteYolu(r, r.Referer())
	if target == "" {
		target = "/admin"
	}
	if IsHTMX(r) {
		// HX-Redirect: htmx isteklerinde istemci tarafı navigasyon (reload yok).
		w.Header().Set("HX-Redirect", target)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
