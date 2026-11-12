package httpapi

import (
	"fmt"
	"html"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Bülten (spec 06 K-F / 01 §L6) ----

// handleBultenGET bülten abonelik formu + abone sayısı.
func (s *Server) handleBultenGET(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	d := s.baseView(locale, s.langBase(locale), s.siteName+" Bülten")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Digest = &digestData{
		Counts:   repo.CountSubscribers(ctx, s.pool, site.ID),
		FormPath: "/bulten",
	}
	digests, _ := repo.ListDigests(ctx, s.pool, site.ID, locale, "gunluk", 14)
	d.Digest.Digests = digests
	s.renderPublic(w, r, "bulten", d, cacheControlPublic)
}

// handleBultenPOST abonelik kaydı (double opt-in).
func (s *Server) handleBultenPOST(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	email := r.FormValue("email")
	sub, err := repo.CreateSubscriber(ctx, s.pool, site.ID, email, locale)
	if err != nil {
		s.flashRedirect(w, r, "/bulten", "error", "Geçerli bir e-posta girin.")
		return
	}
	s.audit(r, "bulten_abone", sub.Email)
	// Çift tıkla onay e-postası: SMTP yapılandırıldıysa gerçekten gönderilir.
	// Yapılandırılmamışsa onay bağlantısı yanıtta döner (geliştirme kolaylığı).
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	onayLink := scheme + "://" + r.Host + "/bulten/onay/" + sub.Token
	if cfg := s.smtpConfig(ctx); cfg.Configured() {
		body := `<p>Merhaba,</p><p>Bülten aboneliğinizi onaylamak için ` +
			`<a href="` + html.EscapeString(onayLink) + `">bu bağlantıya tıklayın</a>.</p>` +
			`<p>Bu e-postayı siz istemediyseniz görmezden gelebilirsiniz.</p>`
		if err := cfg.SendMail(sub.Email, "Bülten aboneliğinizi onaylayın", body); err == nil {
			s.writeJSON(w, http.StatusOK, map[string]any{
				"ok":    true,
				"mesaj": "Kayıt alındı. Onay e-postası gönderildi — gelen kutunuzu kontrol edin.",
				"durum": sub.Status,
			})
			return
		}
		s.logger.Warn("bülten onay e-postası gönderilemedi", "email", sub.Email)
	}
	if s.cfg.Debug {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"mesaj": "Kayıt alındı (geliştirme kipi). Onay bağlantısı: /bulten/onay/" + sub.Token,
			"durum": sub.Status,
		})
		return
	}
	s.logger.Error("bülten onayı gönderilemedi: SMTP yapılandırılmamış", "site", site.ID)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"mesaj": "Kaydınız alındı, ancak onay e-postası şu anda gönderilemiyor. " +
			"Aboneliğiniz onaylanmadan size bülten gönderilmez.",
		"durum": sub.Status,
	})
}

// handleBultenOnay çift kayıt onayı.
func (s *Server) handleBultenOnay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := r.PathValue("token")
	sub, err := repo.SubscriberByToken(ctx, s.pool, token)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	action := r.URL.Query().Get("islem")
	if action == "iptal" {
		_ = repo.UnsubscribeSubscriber(ctx, s.pool, token)
	} else {
		_ = repo.ConfirmSubscriber(ctx, s.pool, token)
	}
	locale := sub.Locale
	d := s.baseView(locale, s.langBase(locale), "Bülten")
	d.HomeURL = homeURL(locale)
	d.Theme = themeFromReq(r)
	msg := "Aboneliğiniz aktif edildi. Teşekkürler!"
	if action == "iptal" {
		msg = "Aboneliğiniz iptal edildi."
	}
	d.Digest = &digestData{FormPath: "/bulten", Done: true, DoneMsg: msg}
	s.renderPublic(w, r, "bulten", d, "no-store")
}

// ---- Günlük özet: /ping/YYYY-MM-DD (spec 06 K-E / 01 §L5) ----

func (s *Server) handlePingDay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	day, err := time.Parse("2006-01-02", r.PathValue("date"))
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	dg, err := repo.DigestByDate(ctx, s.pool, site.ID, day, locale, "gunluk")
	if err != nil {
		// gün henüz üretilmemişse şimdi üret (lazy)
		dg, err = repo.BuildDailyDigest(ctx, s.pool, site.ID, day, locale)
		if err != nil {
			s.handleNotFound(w, r)
			return
		}
	}
	items, _ := repo.RecentPublishedOn(ctx, s.pool, site.ID, locale, day)
	base := s.absBase(r)
	d := s.baseView(locale, s.langBase(locale), dg.Title)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Digest = &digestData{Day: dg, Items: items, Base: base}
	s.renderPublic(w, r, "ping", d, cacheControlPublic)
}

// handlePingIndex günlük özet arşivi.
func (s *Server) handlePingIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	list, _ := repo.ListDigests(ctx, s.pool, site.ID, locale, "gunluk", 60)
	base := s.absBase(r)
	d := s.baseView(locale, s.langBase(locale), "Günlük Özetler")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Digest = &digestData{Archive: list, Base: base}
	s.renderPublic(w, r, "ping-arsiv", d, cacheControlPublic)
}

// handlePodcast günlük podcast: özeti konuşmaya çevirir (espeak-ng + ffmpeg → MP3).
func (s *Server) handlePodcast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	day, err := time.Parse("2006-01-02", r.PathValue("date"))
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	dg, err := repo.DigestByDate(ctx, s.pool, site.ID, day, locale, "gunluk")
	if err != nil || dg == nil {
		dg, err = repo.BuildDailyDigest(ctx, s.pool, site.ID, day, locale)
		if err != nil {
			s.handleNotFound(w, r)
			return
		}
	}
	items, _ := repo.RecentPublishedOn(ctx, s.pool, site.ID, locale, day)
	if dg.AudioPath == nil || *dg.AudioPath == "" {
		// konuşma metnini kur
		var sb strings.Builder
		sb.WriteString(dg.Title + ". ")
		if dg.Summary != "" {
			sb.WriteString(dg.Summary + " ")
		}
		for i, it := range items {
			sb.WriteString(fmt.Sprintf("%d. %s. ", i+1, it.Title))
			if it.Spot != "" && len([]rune(it.Spot)) < 220 {
				sb.WriteString(it.Spot + " ")
			}
		}
		name := fmt.Sprintf("ozet-%s-%s", day.Format("20060102"), locale)
		rel, err := platform.SynthesizeSpeech(sb.String(), locale,
			filepath.Join(platform.StorageDir(), "media", "audio"), name)
		if err != nil {
			s.logger.Warn("podcast üretilemedi", "error", err)
			s.handleNotFound(w, r)
			return
		}
		stored, err := platform.SaveMediaFile(filepath.Join(platform.StorageDir(), "media", "audio", rel), rel)
		if err != nil {
			s.logger.Warn("podcast kaydedilemedi", "error", err)
			s.handleNotFound(w, r)
			return
		}
		dg.AudioPath = &stored
		if err := repo.SetDigestAudio(ctx, s.pool, *dg); err != nil {
			s.logger.Warn("podcast bağlanamadı", "error", err)
		}
	}
	base := s.absBase(r)
	d := s.baseView(locale, s.langBase(locale), "Podcast — "+dg.Title)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Digest = &digestData{Day: dg, Items: items, Base: base, AudioURL: publicMediaURL(dg.AudioPath)}
	s.renderPublic(w, r, "podcast", d, cacheControlPublic)
}

// handleAPIDigest Kantan uyumlu günlük özet API'si.

func (s *Server) handleAPIDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	list, err := repo.ListDigests(ctx, s.pool, site.ID, locale, "gunluk", 30)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false})
		return
	}
	type row struct {
		Date    string `json:"date"`
		Title   string `json:"title"`
		Summary string `json:"summary"`
		Count   int    `json:"post_count"`
		URL     string `json:"url"`
	}
	data := make([]row, 0, len(list))
	base := s.absBase(r)
	for _, dg := range list {
		data = append(data, row{Date: dg.Date.Format("2006-01-02"), Title: dg.Title, Summary: dg.Summary,
			Count: len(dg.ContentIDs), URL: base + "/ping/" + dg.Date.Format("2006-01-02")})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}
