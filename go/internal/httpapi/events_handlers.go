package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Etkinlikler (spec 01 §B2, 02 §207) ----

type eventsData struct {
	Events []repo.Event
	Cats   []repo.Category
}

// handleEventsGet panel: etkinlik listesi + ekleme formu.
func (s *Server) handleEventsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildEventsData(r)
	if IsHTMX(r) {
		s.renderEtkinliklerPanel(w, r, d)
		return
	}
	s.renderEtkinliklerPage(w, r, d)
}

func (s *Server) buildEventsData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "Etkinlikler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	evs, _ := repo.ListEvents(ctx, s.pool, sid, "", 200)
	cats, _ := repo.ListCategories(ctx, s.pool, sid)
	d.EtkinlikPanel = &eventsData{Events: evs, Cats: cats}
	return d
}

func (s *Server) renderEventsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildEventsData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderEtkinliklerPanel(w, r, d)
}

func (s *Server) eventResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderEventsPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/etkinlikler", kind, text)
}

func (s *Server) handleEventCreate(w http.ResponseWriter, r *http.Request) {
	e := eventFromForm(r, s)
	if e.Title == "" || e.StartAt.IsZero() {
		s.eventResult(w, r, "error", "Başlık ve başlangıç tarihi gerekli.")
		return
	}
	if _, err := repo.CreateEvent(r.Context(), s.pool, e); err != nil {
		s.logger.Error("etkinlik ekleme", "error", err)
		s.eventResult(w, r, "error", "Eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "etkinlik_ekle", e.Title)
	s.eventResult(w, r, "info", "Etkinlik eklendi: "+e.Title)
}

func (s *Server) handleEventUpdate(w http.ResponseWriter, r *http.Request) {
	e := eventFromForm(r, s)
	e.ID = parseID(r.PathValue("id"))
	if e.Title == "" || e.StartAt.IsZero() {
		s.eventResult(w, r, "error", "Başlık ve başlangıç tarihi gerekli.")
		return
	}
	if err := repo.UpdateEvent(r.Context(), s.pool, e); err != nil {
		s.eventResult(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "etkinlik_guncelle", e.Title)
	s.eventResult(w, r, "info", "Etkinlik güncellendi.")
}

func (s *Server) handleEventDelete(w http.ResponseWriter, r *http.Request) {
	if err := repo.DeleteEvent(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.eventResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "etkinlik_sil", r.PathValue("id"))
	s.eventResult(w, r, "info", "Etkinlik silindi.")
}

// handleEventDraft etkinlikten haber taslağı oluşturur ve düzenleme ekranına yönlendirir.
func (s *Server) handleEventDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	e, err := repo.GetEvent(ctx, s.pool, parseID(r.PathValue("id")))
	if err != nil {
		s.eventResult(w, r, "error", "Etkinlik bulunamadı.")
		return
	}
	u := userFromContext(ctx)
	body := e.Description
	if e.Location != "" {
		body += "\nYer: " + e.Location
	}
	body += "\nTarih: " + e.StartAt.Format("02.01.2006 15:04")
	c := &repo.Content{
		TypeKey:       "haber",
		Status:        "taslak",
		Title:         e.Title,
		Spot:          e.Description,
		Detail:        textToBlocks(body),
		AuthorID:      u.ID,
		SiteID:        s.curSiteID(r),
		CommentStatus: "acik",
		Locale:        "tr",
		SourceURL:     optStr(e.URL),
	}
	if e.CategoryID > 0 {
		cid := e.CategoryID
		c.CategoryID = &cid
	}
	if sl := platform.Slugify(e.Title); sl != "" {
		c.Slug = optStr(sl)
	}
	id, err := repo.CreateContent(ctx, s.pool, c)
	if err != nil {
		s.eventResult(w, r, "error", "Taslak oluşturulamadı: "+err.Error())
		return
	}
	s.audit(r, "etkinlik_haber_taslak", e.Title)
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), "info", "Etkinlikten taslak oluşturuldu.")
}

// handleEventsPublic genel etkinlik takvimi sayfası + hatırlatma şeridi.
func (s *Server) handleEventsPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Etkinlikler")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	site := s.siteByDomain(ctx, r.Host)
	etkinlikler, _ := repo.UpcomingEvents(ctx, s.pool, site.ID, 90, 60)

	rows := make([]templkit.CDEtkinlik, 0, len(etkinlikler))
	for _, e := range etkinlikler {
		row := templkit.CDEtkinlik{
			Baslik: e.Title, Zaman: e.StartAt.Format("02.01.2006 15:04"),
			Yer: e.Location, Aciklama: e.Description, URL: e.URL,
		}
		if e.CategoryName != nil {
			row.Kategori = *e.CategoryName
		}
		rows = append(rows, row)
	}
	v := templkit.CDEtkinliklerPage{
		Shell:           s.cdShellCtx(ctx, r, ""),
		Title:           templkit.CDT(locale, "Etkinlikler") + " — " + s.siteName,
		Desc:            "Kripto ve finans dünyasından yaklaşan etkinlikler, konferanslar ve takvim.",
		Canonical:       strings.TrimRight(s.cfg.BaseURL, "/") + "/etkinlikler",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
		Locale:          s.cdYerel(r),
		Etkinlikler:     rows,
		PageCategory:    "static_page",
		ContentCategory: "events",
	}
	w.Header().Set("Cache-Control", cacheControlPublic)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templkit.CDEtkinliklerPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("etkinlikler sayfası render", "error", err)
	}
}

// eventFromForm form alanlarını Event yapısına çevirir.
func eventFromForm(r *http.Request, s *Server) *repo.Event {
	e := &repo.Event{
		SiteID:      s.curSiteID(r),
		Title:       strings.TrimSpace(r.FormValue("baslik")),
		Description: strings.TrimSpace(r.FormValue("aciklama")),
		Location:    strings.TrimSpace(r.FormValue("yer")),
		URL:         strings.TrimSpace(r.FormValue("url")),
		StartAt:     parseEventTime(r.FormValue("baslangic")),
		RemindDays:  parseIntList(r.FormValue("hatirlatma")),
		Status:      firstNonEmpty(r.FormValue("durum"), "aktif"),
	}
	if v := parseID(r.FormValue("kategori")); v > 0 {
		e.CategoryID = v
	}
	if t := parseEventTime(r.FormValue("bitis")); !t.IsZero() {
		e.EndAt = &t
	}
	if u := userFromContext(r.Context()); u.ID > 0 {
		id := u.ID
		e.CreatedByID = &id
	}
	return e
}

// parseEventTime datetime-local (2006-01-02T15:04) ve tarih biçimlerini çözer.
func parseEventTime(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseIntList(v string) []int {
	var out []int
	for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// eventWhen etkinliğin ne zaman olduğunu insan diline çevirir ("Bugün"/"Yarın"/"X gün sonra").
func eventWhen(t time.Time) string {
	now := time.Now()
	days := int(t.Sub(now).Hours() / 24)
	switch {
	case t.Before(now):
		return "Geçti"
	case days == 0 && t.YearDay() == now.YearDay():
		return "Bugün"
	case days == 0 || days == 1:
		return "Yarın"
	default:
		return itoa(days) + " gün sonra"
	}
}
