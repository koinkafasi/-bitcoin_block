package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Bülten e-posta gönderimi (spec 03 P4-8, 06 K-F) ----

type bultenData struct {
	Counts   map[string]int
	SMTP     platform.SMTPConfig
	SubCount int
}

// smtpConfig ayarlardan (section=mail) SMTP yapılandırmasını okur.
func (s *Server) smtpConfig(ctx context.Context) platform.SMTPConfig {
	gs, _ := repo.GetSettings(ctx, s.pool, "mail")
	str := func(k string) string {
		b, ok := gs[k]
		if !ok {
			return ""
		}
		var v string
		if json.Unmarshal(b, &v) == nil {
			return v
		}
		return strings.Trim(string(b), `"`)
	}
	port := 587
	if p := str("smtp_port"); p != "" {
		if n := parseIntOr(p); n > 0 {
			port = n
		}
	}
	return platform.SMTPConfig{
		Host:        str("smtp_host"),
		Port:        port,
		User:        str("smtp_user"),
		Pass:        str("smtp_pass"),
		From:        str("smtp_from"),
		FromName:    str("smtp_from_name"),
		ImplicitTLS: str("smtp_tls") == "1",
	}
}

// handleBultenAdmin bülten paneli (abone sayaçları + SMTP ayarları + gönderim).
func (s *Server) handleBultenAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildBultenData(r)
	if IsHTMX(r) {
		s.renderBultenPanelV2(w, r, d)
		return
	}
	s.renderBultenPageV2(w, r, d)
}

func (s *Server) buildBultenData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "Bülten (E-posta)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	counts := repo.CountSubscribers(ctx, s.pool, sid)
	d.Bulten = &bultenData{Counts: counts, SMTP: s.smtpConfig(ctx), SubCount: counts["aktif"]}
	return d
}

func (s *Server) renderBultenPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildBultenData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderBultenPanelV2(w, r, d)
}

func (s *Server) bultenResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderBultenPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/bulten", kind, text)
}

// handleBultenSave SMTP ayarlarını kaydeder.
func (s *Server) handleBultenSave(w http.ResponseWriter, r *http.Request) {
	set := func(k, v string) {
		b, _ := json.Marshal(v)
		_ = repo.SetSetting(r.Context(), s.pool, "mail", k, b)
	}
	set("smtp_host", strings.TrimSpace(r.FormValue("smtp_host")))
	set("smtp_port", strings.TrimSpace(r.FormValue("smtp_port")))
	set("smtp_user", strings.TrimSpace(r.FormValue("smtp_user")))
	if p := r.FormValue("smtp_pass"); p != "" { // boş bırakılırsa mevcut korunur
		set("smtp_pass", p)
	}
	set("smtp_from", strings.TrimSpace(r.FormValue("smtp_from")))
	set("smtp_from_name", strings.TrimSpace(r.FormValue("smtp_from_name")))
	tls := "0"
	if r.FormValue("smtp_tls") != "" {
		tls = "1"
	}
	set("smtp_tls", tls)
	s.audit(r, "bulten_ayar", "smtp")
	s.bultenResult(w, r, "info", "SMTP ayarları kaydedildi.")
}

// handleBultenTest tek adrese test e-postası gönderir.
func (s *Server) handleBultenTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := s.smtpConfig(ctx)
	if !cfg.Configured() {
		s.bultenResult(w, r, "error", "SMTP yapılandırılmamış (host + gönderen gerekli).")
		return
	}
	to := strings.TrimSpace(r.FormValue("eposta"))
	if to == "" {
		to = cfg.From
	}
	subject, body := s.digestEmail(ctx, s.curSiteID(r))
	if err := cfg.SendMail(to, subject, body); err != nil {
		s.logger.Error("bülten test gönderimi", "error", err)
		s.bultenResult(w, r, "error", "Gönderilemedi: "+err.Error())
		return
	}
	s.audit(r, "bulten_test", to)
	s.bultenResult(w, r, "info", "Test e-postası gönderildi: "+to)
}

// handleBultenSend onaylı tüm abonelere günlük özeti gönderir (en fazla 200/istek).
func (s *Server) handleBultenSend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	cfg := s.smtpConfig(ctx)
	if !cfg.Configured() {
		s.bultenResult(w, r, "error", "SMTP yapılandırılmamış (host + gönderen gerekli).")
		return
	}
	subs, err := repo.ActiveSubscribers(ctx, s.pool, sid, 200)
	if err != nil {
		s.bultenResult(w, r, "error", "Aboneler okunamadı.")
		return
	}
	if len(subs) == 0 {
		s.bultenResult(w, r, "info", "Onaylı abone yok.")
		return
	}
	subject, body := s.digestEmail(ctx, sid)
	sent, failed := 0, 0
	for _, sub := range subs {
		if err := cfg.SendMail(sub.Email, subject, body); err != nil {
			failed++
			continue
		}
		sent++
	}
	s.audit(r, "bulten_gonder", itoa(sent)+" gönderildi / "+itoa(failed)+" hata")
	msg := itoa(sent) + " e-posta gönderildi."
	if failed > 0 {
		msg += " " + itoa(failed) + " hata."
	}
	s.bultenResult(w, r, "info", msg)
}

// digestEmail gönderilecek HTML gövdeyi (son yayınlanan içerikler) üretir.
func (s *Server) digestEmail(ctx context.Context, siteID int64) (string, string) {
	items, _ := repo.RecentPublished(ctx, s.pool, siteID, "tr", 8)
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,Helvetica,sans-serif;max-width:640px;margin:0 auto">`)
	b.WriteString(`<h2 style="border-bottom:2px solid #eee;padding-bottom:8px">` + htmlEscape(s.siteAdi()) + ` — Günlük Özet</h2>`)
	if len(items) == 0 {
		b.WriteString(`<p>Bugün için içerik bulunmuyor.</p>`)
	} else {
		b.WriteString(`<ul style="line-height:1.7">`)
		for _, it := range items {
			url := base + publicArticleURL(it.Locale, it.DisplaySlug(), it.ID)
			b.WriteString(`<li><a href="` + url + `" style="color:#1a5fb4;text-decoration:none">` + htmlEscape(it.Title) + `</a></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<p style="color:#888;font-size:12px;margin-top:20px">` + htmlEscape(s.siteAdi()) + ` — bu e-postayı bülten aboneliğiniz için alıyorsunuz.</p>`)
	b.WriteString(`</div>`)
	return s.siteAdi() + " — Günlük Özet", b.String()
}
