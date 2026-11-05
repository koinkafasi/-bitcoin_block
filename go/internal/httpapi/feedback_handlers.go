package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Okuyucu geri bildirimi / içerik güvenliği (spec 06 L12) ----

// handleFeedbackPOST okuyucu şikâyetini kaydeder (JSON).
func (s *Server) handleFeedbackPOST(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	contentID := parseID(r.PathValue("id"))
	kind := r.FormValue("tur")
	if kind == "" {
		kind = "sikayet"
	}
	switch kind {
	case "sikayet", "yanlis", "etik", "reklam", "eksik", "diger":
	default:
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "geçersiz tür"})
		return
	}
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	detail := strings.TrimSpace(r.FormValue("aciklama"))
	if len([]rune(detail)) > 800 {
		detail = string([]rune(detail)[:800])
	}
	reporter := strings.TrimSpace(r.FormValue("bildiren"))
	id, err := repo.AddFeedback(ctx, s.pool, contentID, site.ID, kind, detail, reporter)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "hata": "kaydedilemedi"})
		return
	}
	s.logger.Info("okuyucu geri bildirimi", "kind", kind, "content", contentID)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": id,
		"mesaj": "Geri bildiriminiz alındı. Editör ekibimiz inceleyecek — teşekkürler.",
	})
}

// ---- Panel: geri bildirim kuyruğu ----

// handleFeedbackQueue panelde bekleyen bildirimleri listeler.
func (s *Server) handleFeedbackQueue(w http.ResponseWriter, r *http.Request) {
	d := s.buildFeedbackData(r)
	comp := templkit.GeriBildirimPanel(geriBildirimViewFrom(d.Feedback, d.Flash))
	if IsHTMX(r) {
		s.renderTemplPanel(w, r, comp)
		return
	}
	s.renderTemplPage(w, r, d, comp)
}

func (s *Server) buildFeedbackData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	status := r.URL.Query().Get("durum")
	items, _ := repo.ListFeedback(ctx, s.pool, sid, status, 200)
	logs, _ := repo.ListAssistantLogs(ctx, s.pool, sid, 30)
	d := s.baseView("tr", "/admin", "Geri Bildirimler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Feedback = &feedbackData{
		Items: items, Status: status, Logs: logs,
		Counts:  repo.CountFeedback(ctx, s.pool, sid),
		AsstCnt: repo.CountAssistantLogs(ctx, s.pool, sid),
	}
	return d
}

func (s *Server) renderFeedbackPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildFeedbackData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTemplPanel(w, r, templkit.GeriBildirimPanel(geriBildirimViewFrom(d.Feedback, d.Flash)))
}

// handleFeedbackResolve bildirimi sonuçlandırır.
func (s *Server) handleFeedbackResolve(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	st := r.FormValue("durum")
	switch st {
	case "incelendi", "cozuldu", "reddedildi":
	default:
		st = "incelendi"
	}
	if err := repo.ResolveFeedback(r.Context(), s.pool, id, userFromContext(r.Context()).ID, st, r.FormValue("cozum")); err != nil {
		if IsHTMX(r) {
			s.renderFeedbackPanel(w, r, "error", "Güncellenemedi")
			return
		}
		s.flashRedirect(w, r, "/admin/geri-bildirim", "error", "Güncellenemedi")
		return
	}
	s.audit(r, "geri_bildirim_coz", "id="+itoa(int(id))+" durum="+st)
	if IsHTMX(r) {
		s.renderFeedbackPanel(w, r, "info", "Bildirim güncellendi: "+st)
		return
	}
	s.flashRedirect(w, r, "/admin/geri-bildirim", "info", "Bildirim güncellendi: "+st)
}

// ---- AI asistan (spec 06 L12) ----

// handleAsistanGET asistan sayfasını açar (belki bir haber bağlamıyla).
func (s *Server) handleAsistanGET(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	var ctxTitle, ctxBody string
	if cid := parseID(r.URL.Query().Get("haber")); cid > 0 {
		if c, err := repo.GetContent(ctx, s.pool, cid); err == nil && c.Status == "yayinda" {
			ctxTitle = c.Title
			ctxBody = c.Spot
		}
	}
	d := s.baseView(locale, s.langBase(locale), "AI Asistan")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Assistant = &assistantData{ContextTitle: ctxTitle, ContextBody: ctxBody, Enabled: s.ai.Enabled()}
	s.renderPublic(w, r, "asistan", d, "no-store")
}

// handleAsistanPOST soruyu AI'ye gönderir, yanıtı döndürür ve loglar.
func (s *Server) handleAsistanPOST(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	q := strings.TrimSpace(r.FormValue("soru"))
	if q == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "soru gerekli"})
		return
	}
	if len([]rune(q)) > 600 {
		q = string([]rune(q)[:600])
	}
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	locale := langOf(r, s)

	system := "Sen theblock.com.tr haber platformunun asistanısın. Türkçe, kısa ve doğrusal cevap ver. " +
		"Haberle ilgili değilsen haber uydurma; genel bilgi ver ve kaynak öner."
	var msgs []ai.Msg
	if t := strings.TrimSpace(r.FormValue("baglam_baslik")); t != "" {
		system += " Kullanıcı şu haberi okuyor: \"" + t + "\"."
		if b := strings.TrimSpace(r.FormValue("baglam_spot")); b != "" {
			system += " Özet: " + b
		}
	}
	msgs = append(msgs, ai.Msg{Role: "system", Content: system}, ai.Msg{Role: "user", Content: q})

	answer, err := s.ai.Chat(ctx, msgs)
	provider := "mock"
	if p := s.ai.Providers(); len(p) > 0 {
		provider = p[0]
	}
	if err != nil {
		s.logger.Warn("asistan hatası", "error", err)
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "yanit": "Şu anda asistan servisine ulaşılamıyor. Sorunuzu daha sonra tekrar deneyebilirsiniz.",
			"sağlayici": "hata", "hata": err.Error(),
		})
		return
	}
	var userID *int64
	if u, e := s.currentUser(r); e == nil {
		userID = &u.ID
	}
	var contentID *int64
	if cid := parseID(r.FormValue("haber")); cid > 0 {
		contentID = &cid
	}
	_ = repo.AddAssistantLog(ctx, s.pool, repo.AssistantLog{
		SiteID: site.ID, ContentID: contentID, Question: q, Answer: answer,
		Provider: provider, UserID: userID, Locale: locale,
	})
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "yanit": answer, "sağlayici": provider})
}
