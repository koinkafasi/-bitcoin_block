package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Anketler (spec 02 §26 / 01 §H14) ----

type pollsData struct {
	Items []repo.PollItem
}

// pollView genel anket sayfası görünümü (yüzdeyle).
type pollView struct {
	ID      int64
	Title   string
	Spot    string
	Total   int
	Voted   bool
	Options []pollOptView
}

type pollOptView struct {
	ID    string
	Text  string
	Votes int
	Pct   int
	Mine  bool
}

// handlePollsAdmin panel: anket listesi + oluşturma.
func (s *Server) handlePollsAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildPollsData(r)
	if IsHTMX(r) {
		s.renderAnketlerPanelV2(w, r, d)
		return
	}
	s.renderAnketlerPageV2(w, r, d)
}

func (s *Server) buildPollsData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	items, _, _ := repo.ListContents(ctx, s.pool, repo.ListFilter{TypeKey: "anket", SiteID: sid, PerPage: 200, Page: 1})
	var list []repo.PollItem
	for _, c := range items {
		p, _ := repo.GetPoll(ctx, s.pool, c.ID)
		list = append(list, repo.PollItem{Content: c, Poll: p})
	}
	d := s.baseView("tr", "/admin", "Anketler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.AnketPanel = &pollsData{Items: list}
	return d
}

func (s *Server) renderPollsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildPollsData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderAnketlerPanelV2(w, r, d)
}

// handlePollCreate yeni anket içeriği + seçenekleri oluşturur.
func (s *Server) handlePollCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	title := strings.TrimSpace(r.FormValue("baslik"))
	if title == "" {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Soru (başlık) gerekli.")
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Soru (başlık) gerekli.")
		return
	}
	u := userFromContext(ctx)
	c := &repo.Content{
		TypeKey:       "anket",
		Status:        "yayinda",
		Title:         title,
		Spot:          strings.TrimSpace(r.FormValue("spot")),
		Detail:        textToBlocks(""),
		AuthorID:      u.ID,
		SiteID:        s.curSiteID(r),
		CommentStatus: "acik",
		Locale:        "tr",
	}
	if sl := platform.Slugify(title); sl != "" {
		c.Slug = optStr(sl)
	}
	id, err := repo.CreateContent(ctx, s.pool, c)
	if err != nil {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Oluşturulamadı: "+err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Oluşturulamadı: "+err.Error())
		return
	}
	opts := parseOptionLines(r.FormValue("secenekler"))
	if len(opts) < 2 {
		opts = []repo.PollOption{{Text: "Evet"}, {Text: "Hayır"}}
	}
	for i := range opts {
		opts[i].ID = "o" + rndSuffix()
	}
	if err := repo.SavePoll(ctx, s.pool, id, repo.Poll{Options: opts}); err != nil {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Seçenekler kaydedilemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Seçenekler kaydedilemedi.")
		return
	}
	s.audit(r, "anket_ekle", title)
	if IsHTMX(r) {
		s.renderPollsPanel(w, r, "info", "Anket oluşturuldu: "+title)
		return
	}
	s.flashRedirect(w, r, "/admin/anketler", "info", "Anket oluşturuldu: "+title)
}

// handlePollSave mevcut anketin soru/durum/seçeneklerini günceller.
func (s *Server) handlePollSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	if c, err := repo.GetContent(ctx, s.pool, id); err == nil {
		if !canEditContent(userFromContext(ctx), c) {
			if IsHTMX(r) {
				s.renderPollsPanel(w, r, "error", "Bu anketi düzenleme yetkiniz yok.")
				return
			}
			s.flashRedirect(w, r, "/admin/anketler", "error", "Bu anketi düzenleme yetkiniz yok.")
			return
		}
		if t := strings.TrimSpace(r.FormValue("baslik")); t != "" {
			c.Title = t
		}
		if st := r.FormValue("durum"); st == "yayinda" || st == "taslak" || st == "arsiv" {
			c.Status = st
		}
		_ = repo.UpdateContent(ctx, s.pool, &c)
	}
	ids := r.Form["oyun_id"]
	texts := r.Form["oyun_metin"]
	counts := r.Form["oyun_sayi"]
	var opts []repo.PollOption
	for i := range texts {
		t := strings.TrimSpace(texts[i])
		if t == "" {
			continue
		}
		oid := formAt(ids, i)
		if oid == "" {
			oid = "o" + rndSuffix()
		}
		opts = append(opts, repo.PollOption{ID: oid, Text: t, Votes: parseIntOr(formAt(counts, i))})
	}
	if err := repo.SavePoll(ctx, s.pool, id, repo.Poll{Options: opts}); err != nil {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Kaydedilemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Kaydedilemedi.")
		return
	}
	s.audit(r, "anket_guncelle", r.PathValue("id"))
	if IsHTMX(r) {
		s.renderPollsPanel(w, r, "info", "Anket güncellendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/anketler", "info", "Anket güncellendi.")
}

// handlePollDelete anketi siler.
func (s *Server) handlePollDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil || !canEditContent(userFromContext(r.Context()), c) {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Bu anketi silme yetkiniz yok.")
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Bu anketi silme yetkiniz yok.")
		return
	}
	if err := repo.DeleteContent(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderPollsPanel(w, r, "error", "Silinemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/anketler", "error", "Silinemedi.")
		return
	}
	s.audit(r, "anket_sil", r.PathValue("id"))
	if IsHTMX(r) {
		s.renderPollsPanel(w, r, "info", "Anket silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/anketler", "info", "Anket silindi.")
}

// handlePollsPublic genel anket sayfası.
func (s *Server) handlePollsPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	d := s.baseView(locale, s.langBase(locale), "Anketler")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	items, _ := repo.ListPolls(ctx, s.pool, site.ID, 50)
	// oy durumuna göre görünüm üret
	var views []pollView
	for _, it := range items {
		my := ""
		if ck, err := r.Cookie(oyCookieName(it.Content.ID)); err == nil {
			my = ck.Value
		}
		tot := it.Poll.Total()
		pv := pollView{ID: it.Content.ID, Title: it.Content.Title, Spot: it.Content.Spot, Total: tot, Voted: my != ""}
		for _, o := range it.Poll.Options {
			pct := 0
			if tot > 0 {
				pct = int(float64(o.Votes)*100/float64(tot) + 0.5)
			}
			pv.Options = append(pv.Options, pollOptView{ID: o.ID, Text: o.Text, Votes: o.Votes, Pct: pct, Mine: o.ID == my})
		}
		views = append(views, pv)
	}
	d.Anketler = views
	s.renderPublic(w, r, "anketler", d, "no-store")
}

// handlePollVote oy kaydeder ve geri yönlendirir.
func (s *Server) handlePollVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	_ = r.ParseForm()
	oid := r.FormValue("secenek")
	site := s.siteByDomain(ctx, r.Host)
	if id <= 0 || oid == "" {
		http.Redirect(w, r, s.langBase(langOf(r, s))+"/anketler", http.StatusFound)
		return
	}
	// yalnız yayındaki anket
	if c, err := repo.GetContent(ctx, s.pool, id); err != nil || c.Status != "yayinda" || c.SiteID != site.ID {
		http.Redirect(w, r, s.langBase(langOf(r, s))+"/anketler", http.StatusFound)
		return
	}
	if _, err := repo.VotePoll(ctx, s.pool, id, oid); err != nil {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/anketler", "error", "Oy kaydedilemedi.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oyCookieName(id), Value: oid, Path: "/", MaxAge: 31536000, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, s.langBase(langOf(r, s))+"/anketler", http.StatusFound)
}

// ---- yardımcılar ----

func oyCookieName(id int64) string { return "oy_" + itoa(int(id)) }

// parseOptionLines her satırı bir seçenek yapar ("Metin | oy" biçimini destekler).
func parseOptionLines(v string) []repo.PollOption {
	var out []repo.PollOption
	for _, ln := range strings.Split(v, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		text := ln
		votes := 0
		if i := strings.LastIndex(ln, "|"); i > 0 {
			text = strings.TrimSpace(ln[:i])
			votes = parseIntOr(strings.TrimSpace(ln[i+1:]))
		}
		if text == "" {
			continue
		}
		out = append(out, repo.PollOption{Text: text, Votes: votes})
	}
	return out
}

func formAt(s []string, i int) string {
	if i >= 0 && i < len(s) {
		return strings.TrimSpace(s[i])
	}
	return ""
}
