package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// backlinkViewFrom iç link ağı verisi → templkit.BacklinkView.
func backlinkViewFrom(b *backlinkData, fl *flashMsg) templkit.BacklinkView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.BacklinkView{FlashKind: kind, FlashText: text}
	if b == nil {
		return v
	}
	v.Stats = templkit.BacklinkStat{
		Published:   fmt.Sprint(b.Stats.Published),
		WithCluster: fmt.Sprint(b.Stats.WithCluster),
		Orphan:      fmt.Sprint(b.Stats.Orphan),
		OrphanBad:   b.Stats.Orphan > 0,
	}
	for _, c := range b.Clusters {
		v.Clusters = append(v.Clusters, templkit.BacklinkKume{
			ID: strconv.FormatInt(c.ID, 10), Name: c.Name, Intent: c.Intent,
			Count: strconv.Itoa(c.ContentCount),
		})
	}
	for _, o := range b.Orphans {
		cat := "—"
		if o.CategoryName != nil && *o.CategoryName != "" {
			cat = *o.CategoryName
		}
		v.Orphans = append(v.Orphans, templkit.BacklinkOrphan{
			ID: strconv.FormatInt(o.ID, 10), Title: o.Title, Category: cat,
			Updated: o.UpdatedAt.Format("2006-01-02"),
		})
	}
	return v
}

// paylasimViewFrom sosyal paylaşım verisi → templkit.PaylasimView.
// Kanal kartları veri odaklı üretilir (8 kanal aynı desende).
func paylasimViewFrom(p *shareData, fl *flashMsg) templkit.PaylasimView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.PaylasimView{FlashKind: kind, FlashText: text}
	if p == nil {
		return v
	}
	v.EnabledCount = strconv.Itoa(p.EnabledCount)
	v.Auto = p.Auto
	v.WebhookURL = p.WebhookURL
	on := func(k string) bool { return p.Enabled[k] }

	v.Kanallar = []templkit.PaylasimKanal{
		{
			Key: "telegram", Name: "Telegram",
			Hint:    "Bot token + chat id (@kanal veya -100…; @BotFather).",
			Enabled: on("telegram"),
			Fields: []templkit.PaylasimAlan{
				{Name: "telegram_chat", Value: p.TelegramChat, Placeholder: "chat_id"},
				{Name: "telegram_token", Value: p.TelegramToken, Placeholder: "Bot token", Password: true},
			},
		},
		{
			Key: "facebook", Name: "Facebook",
			Hint:    "Sayfa erişim token'ı + sayfa id (Facebook Graph API).",
			Enabled: on("facebook"),
			Fields: []templkit.PaylasimAlan{
				{Name: "facebook_token", Value: p.FacebookToken, Placeholder: "Sayfa token'ı (EAAB…)"},
				{Name: "facebook_page", Value: p.FacebookPage, Placeholder: "Sayfa id"},
			},
		},
		{
			Key: "x", Name: "X (Twitter)",
			Hint:    "OAuth 1.0a — API key/secret + access token/secret (developer.twitter.com).",
			Enabled: on("x"),
			Fields: []templkit.PaylasimAlan{
				{Name: "x_key", Value: p.XKey, Placeholder: "API key"},
				{Name: "x_secret", Value: p.XSecret, Placeholder: "API secret"},
				{Name: "x_token", Value: p.XToken, Placeholder: "Access token"},
				{Name: "x_token_secret", Value: p.XTokenSecret, Placeholder: "Access token secret"},
			},
		},
		{
			Key: "linkedin", Name: "LinkedIn",
			Hint:    "Kişi token'ı + person URN (örn. abcdef123).",
			Enabled: on("linkedin"),
			Fields: []templkit.PaylasimAlan{
				{Name: "linkedin_token", Value: p.LinkedInToken, Placeholder: "Access token"},
				{Name: "linkedin_urn", Value: p.LinkedInURN, Placeholder: "Person id / URN"},
			},
		},
		{
			Key: "bluesky", Name: "Bluesky",
			Hint:    "Kullanıcı adı + uygulama şifresi (Ayarlar → Uygulama şifreleri).",
			Enabled: on("bluesky"),
			Fields: []templkit.PaylasimAlan{
				{Name: "bluesky_handle", Value: p.BlueskyHandle, Placeholder: "kullanıcı.bsky.social"},
				{Name: "bluesky_appkey", Value: p.BlueskyAppKey, Placeholder: "App password"},
			},
		},
		{
			Key: "mastodon", Name: "Mastodon",
			Hint:    "Örnek adresi + erişim token'ı (https://örnek).",
			Enabled: on("mastodon"),
			Fields: []templkit.PaylasimAlan{
				{Name: "mastodon_instance", Value: p.MastodonInstance, Placeholder: "https://mastodon.instance"},
				{Name: "mastodon_token", Value: p.MastodonToken, Placeholder: "Access token"},
			},
		},
		{
			Key: "whatsapp", Name: "WhatsApp",
			Hint:    "Meta Cloud API: token + phone number id + hedef numara.",
			Enabled: on("whatsapp"),
			Fields: []templkit.PaylasimAlan{
				{Name: "whatsapp_token", Value: p.WhatsAppToken, Placeholder: "System user token (EAAG…)", Password: true},
				{Name: "whatsapp_phone_id", Value: p.WhatsAppPhoneID, Placeholder: "Phone number id"},
				{Name: "whatsapp_to", Value: p.WhatsAppTo, Placeholder: "Hedef: 90XXXXXXXXXX"},
			},
		},
		{
			Key: "instagram", Name: "Instagram",
			Hint:    "Meta Graph API token + IG kullanıcı id (görsel paylaşımı kapak URL'siyle).",
			Enabled: on("instagram"),
			Fields: []templkit.PaylasimAlan{
				{Name: "instagram_token", Value: p.InstagramToken, Placeholder: "IG access token"},
				{Name: "instagram_user", Value: p.InstagramUser, Placeholder: "IG kullanıcı id"},
			},
		},
	}

	for _, c := range p.Contents {
		v.Contents = append(v.Contents, templkit.PaylasimIcerik{
			ID: strconv.FormatInt(c.ID, 10), Title: c.Title,
		})
	}
	for _, s := range p.Shares {
		v.Shares = append(v.Shares, templkit.PaylasimShare{
			Platform: s.Platform,
			OK:       s.Status == "gonderildi",
			Detail:   s.Detail,
			Time:     fmt.Sprint(s.CreatedAt),
		})
	}
	return v
}

// pushViewFrom web push verisi → templkit.PushView.
func pushViewFrom(p *pushData, fl *flashMsg) templkit.PushView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.PushView{FlashKind: kind, FlashText: text}
	if p == nil {
		return v
	}
	v.Count = fmt.Sprint(p.Count)
	v.Subject = p.Subject
	v.LastSent = fmt.Sprint(p.LastSent)
	v.PublicKey = p.PublicKey
	for _, s := range p.Subs {
		v.Subs = append(v.Subs, templkit.PushAbone{
			Endpoint: s.Endpoint, EndpointShort: s.EndpointShort,
			UserAgent: s.UserAgent, LastSeen: fmt.Sprint(s.LastSeen),
		})
	}
	return v
}

// ---- Render yardımcıları ----

func (s *Server) renderBacklinkPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.BacklinkPanel(backlinkViewFrom(d.Backlink, d.Flash)))
}

func (s *Server) renderBacklinkPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.BacklinkPanel(backlinkViewFrom(d.Backlink, d.Flash)))
}

func (s *Server) renderPaylasimPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.PaylasimPanel(paylasimViewFrom(d.Paylasim, d.Flash)))
}

func (s *Server) renderPaylasimPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.PaylasimPanel(paylasimViewFrom(d.Paylasim, d.Flash)))
}

func (s *Server) renderPushPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.PushPanel(pushViewFrom(d.Push, d.Flash)))
}

func (s *Server) renderPushPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.PushPanel(pushViewFrom(d.Push, d.Flash)))
}
