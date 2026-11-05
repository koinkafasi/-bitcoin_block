// package platform — dış servis çağrıları.
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PostForm application/x-www-form-urlencoded gövde gönderir (20 sn timeout).
func PostForm(ctx context.Context, urlStr string, v url.Values, headers map[string]string) (int, []byte, error) {
	return PostRaw(ctx, urlStr, "application/x-www-form-urlencoded", []byte(v.Encode()), headers)
}

// SendMastodon Mastodon örneğine gönderi atar (Bearer token).
func SendMastodon(ctx context.Context, instance, token, text string) (int, error) {
	if instance == "" || token == "" {
		return 0, fmt.Errorf("mastodon instance/token eksik")
	}
	u := strings.TrimRight(instance, "/") + "/api/v1/statuses"
	form := url.Values{"status": {text}, "visibility": {"public"}}
	code, body, err := PostForm(ctx, u, form, map[string]string{"Authorization": "Bearer " + token})
	if err != nil {
		return code, err
	}
	if code >= 300 {
		return code, fmt.Errorf("mastodon HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

// SendBluesky Bluesky'e text + bağlantı kartıyla gönderi atar (handle + uygulama şifresi).
func SendBluesky(ctx context.Context, handle, appPassword, text, link string) (int, error) {
	if handle == "" || appPassword == "" {
		return 0, fmt.Errorf("bluesky handle/appkey eksik")
	}
	api := "https://bsky.social/xrpc/"
	// 1) oturum aç
	sc, body, err := PostJSON(ctx, api+"com.atproto.server.createSession", map[string]any{
		"identifier": handle, "password": appPassword,
	}, nil)
	if err != nil || sc >= 300 {
		return sc, fmt.Errorf("bluesky giriş HTTP %d: %s", sc, truncate(string(body), 160))
	}
	var sess struct {
		AccessJwt string `json:"accessJwt"`
		Did       string `json:"did"`
		Handle    string `json:"handle"`
	}
	if json.Unmarshal(body, &sess) != nil || sess.AccessJwt == "" {
		return sc, fmt.Errorf("bluesky oturum JSON'u bozuk")
	}
	auth := map[string]string{"Authorization": "Bearer " + sess.AccessJwt}
	// 2) gönderi kaydı (app.bsky.feed.post) + dış bağlantı kartı
	type externalLink struct {
		URI   string `json:"uri"`
		Title string `json:"title"`
	}
	record := map[string]any{
		"$type": "app.bsky.feed.post",
		"text":  shortText(text, 300),
	}
	if link != "" {
		record["embed"] = map[string]any{
			"$type":    "app.bsky.embed.external",
			"external": externalLink{URI: link, Title: firstLine(text)},
		}
		record["facets"] = []any{}
	}
	sc, body, err = PostJSON(ctx, api+"com.atproto.repo.createRecord", map[string]any{
		"repo": sess.Did, "collection": "app.bsky.feed.post", "record": record,
	}, auth)
	if err != nil || sc >= 300 {
		return sc, fmt.Errorf("bluesky gönderi HTTP %d: %s", sc, truncate(string(body), 200))
	}
	return sc, nil
}

// SendFacebook sayfa duvarına linkli gönderi bırakır (Page token + sayfa id).
func SendFacebook(ctx context.Context, token, pageID, text, link string) (int, error) {
	if token == "" || pageID == "" {
		return 0, fmt.Errorf("facebook token/sayfa eksik")
	}
	u := "https://graph.facebook.com/v18.0/" + pageID + "/feed"
	code, body, err := PostJSON(ctx, u, map[string]any{
		"message": shortText(text, 1000), "link": link, "access_token": token,
	}, nil)
	if err != nil || code >= 300 {
		return code, fmt.Errorf("facebook HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

// SendX X (Twitter) v1.1 status/update — OAuth 1.0a imzalı.
func SendX(ctx context.Context, akey, asecret, token, tokSecret, text string) (int, error) {
	if akey == "" || asecret == "" || token == "" || tokSecret == "" {
		return 0, fmt.Errorf("x anahtarları eksik (key/secret/token/token_secret)")
	}
	u := "https://api.twitter.com/1.1/statuses/update.json"
	params := map[string]string{"status": text}
	authHeader := oauth1Header("POST", u, params, akey, asecret, token, tokSecret)
	code, body, err := PostForm(ctx, u, url.Values{"status": {text}}, map[string]string{"Authorization": authHeader})
	if err != nil || code >= 300 {
		return code, fmt.Errorf("x HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

// SendLinkedIn kişi (urn) profiline ugcPosts olarak paylaşır (Bearer token).
func SendLinkedIn(ctx context.Context, token, urn, text, link string) (int, error) {
	if token == "" || urn == "" {
		return 0, fmt.Errorf("linkedin token/urn eksik")
	}
	if !strings.Contains(urn, "urn:li:") {
		urn = "urn:li:person:" + strings.TrimPrefix(urn, "person:")
	}
	u := "https://api.linkedin.com/v2/ugcPosts"
	bodyPayload := map[string]any{
		"author":         "urn:li:person:" + strings.TrimPrefix(urn, "urn:li:person:"),
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": map[string]any{
				"shareCommentary":    map[string]any{"text": shortText(text, 1300)},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]any{"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"},
	}
	if link != "" {
		bodyPayload = map[string]any{
			"author":         "urn:li:person:" + strings.TrimPrefix(urn, "urn:li:person:"),
			"lifecycleState": "PUBLISHED",
			"specificContent": map[string]any{
				"com.linkedin.ugc.ShareContent": map[string]any{
					"shareCommentary":    map[string]any{"text": shortText(text, 1300)},
					"shareMediaCategory": "ARTICLE",
					"media": []any{map[string]any{
						"status": "READY", "originalUrl": link,
						"title": map[string]any{"text": firstLine(text)}, "description": map[string]any{"text": secondLine(text)},
					}},
				},
			},
			"visibility": map[string]any{"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"},
		}
	}
	code, body, err := PostJSON(ctx, u, bodyPayload, map[string]string{"Authorization": "Bearer " + token, "X-Restli-Protocol-Version": "2.0.0"})
	if err != nil || code >= 300 {
		return code, fmt.Errorf("linkedin HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

// SendWhatsApp Meta Cloud API ile belirli numaraya mesaj atar.
func SendWhatsApp(ctx context.Context, token, phoneID, to, text string) (int, error) {
	if token == "" || phoneID == "" || to == "" {
		return 0, fmt.Errorf("whatsapp token/phone_id/hedef eksik")
	}
	u := "https://graph.facebook.com/v18.0/" + phoneID + "/messages"
	code, body, err := PostJSON(ctx, u, map[string]any{
		"messaging_product": "whatsapp", "to": to,
		"type": "text", "text": map[string]any{"body": shortText(text, 1000)},
	}, map[string]string{"Authorization": "Bearer " + token})
	if err != nil || code >= 300 {
		return code, fmt.Errorf("whatsapp HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

// SendInstagram Meta Graph API — tek veya çoklu görsel paylaşır (kapak URL'si gerekir).
func SendInstagram(ctx context.Context, token, igUser, imageURL, text string) (int, error) {
	if token == "" || igUser == "" {
		return 0, fmt.Errorf("instagram token/kullanıcı eksik")
	}
	if imageURL == "" {
		return 0, fmt.Errorf("instagram görsel (ironi kapak URL'si) gerekli")
	}
	caption := shortText(firstLine(text), 2200)
	// 1) medya konteyneri oluştur
	u := "https://graph.facebook.com/v18.0/" + igUser + "/media"
	code, body, err := PostJSON(ctx, u, map[string]any{
		"image_url": imageURL, "caption": caption, "access_token": token,
	}, nil)
	if err != nil || code >= 300 {
		return code, fmt.Errorf("instagram medya HTTP %d: %s", code, truncate(string(body), 200))
	}
	var m struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &m) != nil || m.ID == "" {
		return code, fmt.Errorf("instagram konteyner id alınamadı")
	}
	// 2) yayınla
	code, body, err = PostJSON(ctx, u+"/publish", map[string]any{"creation_id": m.ID, "access_token": token}, nil)
	if err != nil || code >= 300 {
		return code, fmt.Errorf("instagram yayın HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}

func shortText(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "\n"); i >= 0 {
		if i > 0 {
			return strings.TrimSpace(s[:i])
		}
	}
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:40])
	}
	return s
}

func secondLine(s string) string {
	parts := strings.SplitN(strings.TrimSpace(s), "\n", 2)
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(shortText(parts[1], 300))
	}
	return shortText(s, 300)
}

var _ = http.MethodPost
