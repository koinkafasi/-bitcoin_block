package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- P4-7: Sosyal medya otomatik paylaşım (Faz 40: 8 kanal yönetimi) ----

// paylasim-kanallari: panelde bu sırada listelenir; kodlar ayar anahtarı ve
// LogShare platform adı için kullanılır.
var paylasimChannels = []struct{ Code, Ad, Aciklama string }{
	{"telegram", "Telegram", "Bot token + kanal/grup chat id (en kolay)."},
	{"facebook", "Facebook", "Sayfa erişim token'ı + sayfa id (Graph API)."},
	{"x", "X (Twitter)", "OAuth 1.0a: API key/secret + access token/secret."},
	{"linkedin", "LinkedIn", "Kişi token'ı + person URN (örn. AbCdEf123)."},
	{"bluesky", "Bluesky", "Kullanıcı adı + uygulama şifresi (app password)."},
	{"mastodon", "Mastodon", "Örnek adresi (https://örnek) + erişim token'ı."},
	{"whatsapp", "WhatsApp", "Meta Cloud API: token + phone number id + hedef numara."},
	{"instagram", "Instagram", "Meta Graph API: token + IG kullanıcı id (kapak görseli gerekir)."},
}

type shareData struct {
	Auto             bool
	Enabled          map[string]bool
	EnabledCount     int
	TelegramToken    string
	TelegramChat     string
	FacebookToken    string
	FacebookPage     string
	XKey             string
	XSecret          string
	XToken           string
	XTokenSecret     string
	LinkedInToken    string
	LinkedInURN      string
	BlueskyHandle    string
	BlueskyAppKey    string
	MastodonInstance string
	MastodonToken    string
	WhatsAppToken    string
	WhatsAppPhoneID  string
	WhatsAppTo       string
	InstagramToken   string
	InstagramUser    string
	WebhookURL       string
	Shares           []repo.ShareLog
	Contents         []repo.Content
}

func (s *Server) paylasimSetting(ctx context.Context, key string) string {
	b, err := repo.GetSetting(ctx, s.pool, "paylasim", key)
	if err != nil {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return strings.Trim(string(b), `"`)
}

func (s *Server) setPaylasimSetting(ctx context.Context, key, val string) {
	b, _ := json.Marshal(val)
	_ = repo.SetSetting(ctx, s.pool, "paylasim", key, b)
}

// paylasimEnabled kanal açık mı (etkin kümesi "enabled" JSON'unda saklanır).
func (s *Server) paylasimEnabled(ctx context.Context, code string) bool {
	b, err := repo.GetSetting(ctx, s.pool, "paylasim", "enabled")
	if err != nil {
		return false
	}
	var m map[string]bool
	if json.Unmarshal(b, &m) != nil {
		return false
	}
	return m[code]
}

func (s *Server) setPaylasimEnabled(ctx context.Context, code string, on bool) {
	m := map[string]bool{}
	if b, err := repo.GetSetting(ctx, s.pool, "paylasim", "enabled"); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	if m == nil {
		m = map[string]bool{}
	}
	m[code] = on
	b, _ := json.Marshal(m)
	_ = repo.SetSetting(ctx, s.pool, "paylasim", "enabled", b)
}

func (s *Server) handleShareAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildShareData(r)
	if IsHTMX(r) {
		s.renderPaylasimPanelV2(w, r, d)
		return
	}
	s.renderPaylasimPageV2(w, r, d)
}

func (s *Server) buildShareData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "Sosyal Paylaşım")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	items, _, _ := repo.ListContents(ctx, s.pool, repo.ListFilter{Status: "yayinda", SiteID: sid, PerPage: 100, Page: 1})
	shares, _ := repo.ListShares(ctx, s.pool, sid, 40)
	enabled := map[string]bool{}
	enabledCount := 0
	for _, ch := range paylasimChannels {
		enabled[ch.Code] = s.paylasimEnabled(ctx, ch.Code)
		if enabled[ch.Code] {
			enabledCount++
		}
	}
	d.Paylasim = &shareData{
		Auto:             s.paylasimSetting(ctx, "auto") == "1",
		Enabled:          enabled,
		EnabledCount:     enabledCount,
		TelegramToken:    s.paylasimSetting(ctx, "telegram_token"),
		TelegramChat:     s.paylasimSetting(ctx, "telegram_chat"),
		FacebookToken:    s.paylasimSetting(ctx, "facebook_token"),
		FacebookPage:     s.paylasimSetting(ctx, "facebook_page"),
		XKey:             s.paylasimSetting(ctx, "x_key"),
		XSecret:          s.paylasimSetting(ctx, "x_secret"),
		XToken:           s.paylasimSetting(ctx, "x_token"),
		XTokenSecret:     s.paylasimSetting(ctx, "x_token_secret"),
		LinkedInToken:    s.paylasimSetting(ctx, "linkedin_token"),
		LinkedInURN:      s.paylasimSetting(ctx, "linkedin_urn"),
		BlueskyHandle:    s.paylasimSetting(ctx, "bluesky_handle"),
		BlueskyAppKey:    s.paylasimSetting(ctx, "bluesky_appkey"),
		MastodonInstance: s.paylasimSetting(ctx, "mastodon_instance"),
		MastodonToken:    s.paylasimSetting(ctx, "mastodon_token"),
		WhatsAppToken:    s.paylasimSetting(ctx, "whatsapp_token"),
		WhatsAppPhoneID:  s.paylasimSetting(ctx, "whatsapp_phone_id"),
		WhatsAppTo:       s.paylasimSetting(ctx, "whatsapp_to"),
		InstagramToken:   s.paylasimSetting(ctx, "instagram_token"),
		InstagramUser:    s.paylasimSetting(ctx, "instagram_user"),
		WebhookURL:       s.paylasimSetting(ctx, "webhook_url"),
		Shares:           shares,
		Contents:         items,
	}
	return d
}

func (s *Server) renderSharePanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildShareData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderPaylasimPanelV2(w, r, d)
}

func (s *Server) shareResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderSharePanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/paylasim", kind, text)
}

func (s *Server) handleShareSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	set := func(key, val string) { s.setPaylasimSetting(ctx, key, strings.TrimSpace(val)) }
	set("telegram_token", r.FormValue("telegram_token"))
	set("telegram_chat", r.FormValue("telegram_chat"))
	set("facebook_token", r.FormValue("facebook_token"))
	set("facebook_page", r.FormValue("facebook_page"))
	set("x_key", r.FormValue("x_key"))
	set("x_secret", r.FormValue("x_secret"))
	set("x_token", r.FormValue("x_token"))
	set("x_token_secret", r.FormValue("x_token_secret"))
	set("linkedin_token", r.FormValue("linkedin_token"))
	set("linkedin_urn", r.FormValue("linkedin_urn"))
	set("bluesky_handle", r.FormValue("bluesky_handle"))
	set("bluesky_appkey", r.FormValue("bluesky_appkey"))
	set("mastodon_instance", r.FormValue("mastodon_instance"))
	set("mastodon_token", r.FormValue("mastodon_token"))
	set("whatsapp_token", r.FormValue("whatsapp_token"))
	set("whatsapp_phone_id", r.FormValue("whatsapp_phone_id"))
	set("whatsapp_to", r.FormValue("whatsapp_to"))
	set("instagram_token", r.FormValue("instagram_token"))
	set("instagram_user", r.FormValue("instagram_user"))
	set("webhook_url", r.FormValue("webhook_url"))
	auto := "0"
	if r.FormValue("auto") != "" {
		auto = "1"
	}
	s.setPaylasimSetting(ctx, "auto", auto)
	for _, ch := range paylasimChannels {
		s.setPaylasimEnabled(ctx, ch.Code, r.FormValue("cn_"+ch.Code) != "")
	}
	s.audit(r, "paylasim_ayar", "sosyal")
	s.shareResult(w, r, "info", "Paylaşım ayarları kaydedildi.")
}

// handleShareNow seçili içeriği hemen paylaşır.
func (s *Server) handleShareNow(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.FormValue("icerik"))
	sent, errs := s.shareContent(r.Context(), s.curSiteID(r), id)
	s.audit(r, "paylasim_gonder", itoa(sent)+" platform")
	if len(errs) > 0 {
		s.shareResult(w, r, "error", itoa(sent)+" gönderildi; hatalar: "+strings.Join(errs, "; "))
		return
	}
	s.shareResult(w, r, "info", itoa(sent)+" platforma paylaşıldı.")
}

// handleShareTest tek kanalın test gönderimini yapar (loglanır).
func (s *Server) handleShareTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	code := strings.ToLower(strings.TrimSpace(r.FormValue("kanal")))
	msg := "🧪 theblock sosyal medya testi — panelden yapılandırma doğrulandı. " + strings.TrimRight(s.cfg.BaseURL, "/")
	err := s.channelSend(ctx, sid, code, msg, "", "")
	if err != nil {
		s.audit(r, "paylasim_test", code+" hata")
		s.shareResult(w, r, "error", code+": "+err.Error())
		return
	}
	s.audit(r, "paylasim_test", code+" ok")
	s.shareResult(w, r, "info", code+" test gönderimi başarılı.")
}

// shareContent içeriği etkin platformlara gönderir; gönderilen platform sayısını döndürür.
func (s *Server) shareContent(ctx context.Context, siteID, contentID int64) (int, []string) {
	c, err := repo.GetContent(ctx, s.pool, contentID)
	if err != nil {
		return 0, []string{"içerik bulunamadı"}
	}
	url := strings.TrimRight(s.cfg.BaseURL, "/") + makaleURL(c)
	text := c.Title + "\n" + c.Spot + "\n" + url
	cover := coverImageURL(ctx, s, &c)
	cid := c.ID
	sent := 0
	var errs []string

	// Nötr webhook (JSON) — test edilebilir; ayrıca kanallar gibi loglanır.
	if wh := s.paylasimSetting(ctx, "webhook_url"); wh != "" {
		code, body, perr := platform.PostJSON(ctx, wh, map[string]any{
			"title": c.Title, "url": url, "spot": c.Spot, "site_id": siteID, "content_id": contentID,
		}, nil)
		if perr != nil {
			_ = repo.LogShare(ctx, s.pool, siteID, &cid, "webhook", "hata", perr.Error())
			errs = append(errs, "webhook: "+perr.Error())
		} else if code >= 300 {
			_ = repo.LogShare(ctx, s.pool, siteID, &cid, "webhook", "hata", "HTTP "+itoa(code))
			errs = append(errs, "webhook HTTP "+itoa(code))
		} else {
			sent++
			_ = repo.LogShare(ctx, s.pool, siteID, &cid, "webhook", "gonderildi", "HTTP "+itoa(code)+" "+string(body))
		}
	}

	// Sekiz sosyal kanal: etkinleştirilmişler içinden yapılandırması tam olanlara git.
	for _, ch := range paylasimChannels {
		if !s.paylasimEnabled(ctx, ch.Code) {
			continue
		}
		if err := s.channelSend(ctx, siteID, ch.Code, text, url, cover); err != nil {
			_ = repo.LogShare(ctx, s.pool, siteID, &cid, ch.Code, "hata", err.Error())
			errs = append(errs, ch.Code+": "+err.Error())
			continue
		}
		sent++
		_ = repo.LogShare(ctx, s.pool, siteID, &cid, ch.Code, "gonderildi", "")
	}
	if sent == 0 && len(errs) == 0 {
		errs = append(errs, "etkin+yapılandırılmış platform yok")
	}
	return sent, errs
}

// channelSend tek kanala gönderi atar; yapılandırma eksikse hata döner.
func (s *Server) channelSend(ctx context.Context, siteID int64, code, text, link, cover string) error {
	st := func(key string) string { return s.paylasimSetting(ctx, key) }
	switch code {
	case "telegram":
		return platform.SendTelegram(ctx, st("telegram_token"), st("telegram_chat"), text)
	case "facebook":
		if link == "" {
			return errConfig("facebook linki yok")
		}
		code, err := platform.SendFacebook(ctx, st("facebook_token"), st("facebook_page"), text, link)
		return codeErr("facebook", code, err)
	case "x":
		_, err := platform.SendX(ctx, st("x_key"), st("x_secret"), st("x_token"), st("x_token_secret"), text)
		return err
	case "linkedin":
		_, err := platform.SendLinkedIn(ctx, st("linkedin_token"), st("linkedin_urn"), text, link)
		return err
	case "bluesky":
		_, err := platform.SendBluesky(ctx, st("bluesky_handle"), st("bluesky_appkey"), text, link)
		return err
	case "mastodon":
		_, err := platform.SendMastodon(ctx, st("mastodon_instance"), st("mastodon_token"), text)
		return err
	case "whatsapp":
		_, err := platform.SendWhatsApp(ctx, st("whatsapp_token"), st("whatsapp_phone_id"), st("whatsapp_to"), text)
		return err
	case "instagram":
		_, err := platform.SendInstagram(ctx, st("instagram_token"), st("instagram_user"), cover, text)
		return err
	}
	return errConfig("bilinmeyen kanal: " + code)
}

func errConfig(msg string) error { return &cfgErr{msg} }

type cfgErr struct{ m string }

func (e *cfgErr) Error() string { return e.m }

func codeErr(name string, code int, err error) error {
	if err != nil {
		return err
	}
	if code >= 300 {
		return &cfgErr{name + " HTTP " + itoa(code)}
	}
	return nil
}

// coverImageURL içerik kapağının mutlak adresi (Instagram için; yoksa "").
func coverImageURL(ctx context.Context, s *Server, c *repo.Content) string {
	if c == nil || c.CoverMediaID == nil {
		return ""
	}
	m, err := repo.GetMedia(ctx, s.pool, *c.CoverMediaID)
	if err != nil || m.Path == "" {
		return ""
	}
	return strings.TrimRight(s.cfg.BaseURL, "/") + "/" + strings.TrimLeft(m.Path, "/")
}

// autoShare yayınlanan içeriği otomatik paylaşır (ayar açıksa).
func (s *Server) autoShare(ctx context.Context, siteID, contentID int64) {
	if s.paylasimSetting(ctx, "auto") != "1" {
		return
	}
	go func() {
		// İstek bağlamı KASITLI kullanılmıyor: HTTP yanıtı dönünce
		// r.Context() iptal olur ve bu iş yarıda kesilirdi. Ama çıplak
		// Background() de doğru değil — iptal ve süre sınırı olmadan
		// goroutine sunucu kapanırken asılı kalabiliyor (gosec G118).
		// Sınırlı bir bağlam ikisini de çözüyor.
		bg, iptal := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
		defer iptal()
		_, _ = s.shareContent(bg, siteID, contentID)
	}()
}
