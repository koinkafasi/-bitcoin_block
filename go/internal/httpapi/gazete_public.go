package httpapi

import (
	"context"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// ---- Gazete arşiv & flip görüntüleyici (Faz 5) + abonelik (Faz 6) ----

// gazetePublicView herkese açık gazete görünümü.
type gazetePublicView struct {
	Issues   []repo.GazeteIssue
	Issue    *repo.GazeteIssue
	Pages    []template.HTML
	Period   string
	From     string
	To       string
	Message  string
	HasPDF   bool
	PageNums []int
}

// gazetePublicReady sitenin gazete modülü açık mı? (public 404 kapısı)
func (s *Server) gazetePublicReady(ctx context.Context, siteID int64) bool {
	on, err := repo.SiteFeatureEnabled(ctx, s.pool, siteID, "gazete")
	return err == nil && on
}

// handleGazeteArsiv GET /gazete — baskı arşivi (günlük/haftalık filtre + tarih aralığı).
func (s *Server) handleGazeteArsiv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	if !s.gazetePublicReady(ctx, site.ID) {
		http.NotFound(w, r)
		return
	}
	period := strings.TrimSpace(r.URL.Query().Get("periyot"))
	from := strings.TrimSpace(r.URL.Query().Get("bas"))
	to := strings.TrimSpace(r.URL.Query().Get("bit"))
	issues, _ := repo.ListGazeteIssues(ctx, s.pool, site.ID, period, from, to, 120)
	// Yalnızca yayındaki sayılar public listelenir (yönetici hariç).
	if userFromContext(ctx).ID == 0 {
		var pub []repo.GazeteIssue
		for _, i := range issues {
			if i.Status == "yayinda" {
				pub = append(pub, i)
			}
		}
		issues = pub
	}
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Gazete Arşivi")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.GazetePublic = &gazetePublicView{
		Issues: issues, Period: period, From: from, To: to,
		Message: strings.TrimSpace(r.URL.Query().Get("mesaj")),
	}
	s.renderPublic(w, r, "gazete-arsiv", d, cacheControlPublic)
}

// handleGazeteOkuyucu GET /gazete/{tarih} — flip görüntüleyici.
func (s *Server) handleGazeteOkuyucu(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	if !s.gazetePublicReady(ctx, site.ID) {
		http.NotFound(w, r)
		return
	}
	date := strings.TrimSpace(r.PathValue("tarih"))
	period := strings.TrimSpace(r.URL.Query().Get("periyot"))
	if period == "" {
		period = "gunluk"
	}
	iss, err := repo.GetGazeteIssueByDate(ctx, s.pool, site.ID, period, date)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if iss.Status != "yayinda" && userFromContext(ctx).ID == 0 {
		http.NotFound(w, r)
		return
	}
	pages, _ := repo.ListGazetePages(ctx, s.pool, iss.ID)
	var rendered []template.HTML
	for _, p := range pages {
		rendered = append(rendered, template.HTML(gazetePageHTML(p, "Sayfa "+strconv.Itoa(p.PageNo))))
	}
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Gazete — "+iss.IssueDate)
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	v := &gazetePublicView{Issue: &iss, Pages: rendered, Period: period, HasPDF: iss.PDFPath != ""}
	for i := range rendered {
		v.PageNums = append(v.PageNums, i+1)
	}
	d.GazetePublic = v
	s.renderPublic(w, r, "gazete-okuyucu", d, "public, max-age=300")
}

// handleGazetePDFIndir GET /gazete/{tarih}/pdf — PDF indirme (yayında veya yetkili).
func (s *Server) handleGazetePDFIndir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	if !s.gazetePublicReady(ctx, site.ID) {
		http.NotFound(w, r)
		return
	}
	date := strings.TrimSpace(r.PathValue("tarih"))
	period := strings.TrimSpace(r.URL.Query().Get("periyot"))
	if period == "" {
		period = "gunluk"
	}
	iss, err := repo.GetGazeteIssueByDate(ctx, s.pool, site.ID, period, date)
	if err != nil || iss.PDFPath == "" {
		http.NotFound(w, r)
		return
	}
	if iss.Status != "yayinda" && userFromContext(ctx).ID == 0 {
		http.NotFound(w, r)
		return
	}
	store := s.storeForSite(ctx, site.ID)
	data, err := store.Get(iss.PDFPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="gazete-`+iss.IssueDate+`.pdf"`)
	w.Header().Set("Cache-Control", cacheControlPublic)
	_, _ = w.Write(data)
}

// handleGazeteAboneOl POST /gazete/abone — herkese açık abonelik (KVKK opt-in).
func (s *Server) handleGazeteAboneOl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	if !s.gazetePublicReady(ctx, site.ID) {
		http.NotFound(w, r)
		return
	}
	channel := strings.TrimSpace(r.FormValue("kanal"))
	sub := &repo.GazeteSubscription{
		SiteID:    site.ID,
		Email:     strings.TrimSpace(r.FormValue("email")),
		Phone:     strings.TrimSpace(r.FormValue("telefon")),
		Channel:   channel,
		Frequency: strings.TrimSpace(r.FormValue("siklik")),
		Status:    "aktif",
		OptInIP:   r.RemoteAddr,
	}
	if sub.Channel == "" {
		sub.Channel = "email"
	}
	if (sub.Channel == "whatsapp" && sub.Phone == "") || (sub.Channel != "whatsapp" && sub.Email == "") {
		s.flashRedirect(w, r, "/gazete", "error", "Geçerli bir e-posta veya telefon girin.")
		return
	}
	if _, err := repo.CreateGazeteSubscription(ctx, s.pool, sub); err != nil {
		s.flashRedirect(w, r, "/gazete", "error", "Abonelik kaydedilemedi.")
		return
	}
	s.audit(r, "gazete_abone_public", sub.Email+sub.Phone)
	s.flashRedirect(w, r, "/gazete", "info", "Aboneliğiniz alındı. Teşekkürler!")
}

// handleGazeteAbonelikIptal GET /gazete/abonelik/iptal/{token} — tek tıkla çıkış (KVKK).
func (s *Server) handleGazeteAbonelikIptal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSpace(r.PathValue("token"))
	sub, err := repo.GazeteSubscriptionByToken(ctx, s.pool, token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = repo.SetGazeteSubscriptionStatus(ctx, s.pool, sub.ID, "iptal")
	s.audit(r, "gazete_abonelik_iptal", token[:8])
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Abonelik İptal")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.GazetePublic = &gazetePublicView{Message: "Aboneliğiniz iptal edildi."}
	s.renderPublic(w, r, "gazete-arsiv", d, "no-store")
}
