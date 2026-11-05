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

// ---- P1-8: Web push (bildirim) ----

// pushSubView panodaki abone satırı (gizli anahtarlar görünmez).
type pushSubView struct {
	Endpoint      string
	EndpointShort string
	UserAgent     string
	LastSeen      string
}

type pushData struct {
	Count     int64
	PublicKey string
	Subject   string
	LastSent  string
	Subs      []pushSubView
}

// vapidKeys ayarlardan VAPID anahtarlarını okur; yoksa üretip kaydeder.
func (s *Server) vapidKeys(ctx context.Context) platform.VAPIDKeys {
	pub := s.pushSetting(ctx, "vapid_public")
	priv := s.pushSetting(ctx, "vapid_private")
	if pub == "" || priv == "" {
		if k, err := platform.GenerateVAPIDKeys(); err == nil {
			s.setPushSetting(ctx, "vapid_public", k.Public)
			s.setPushSetting(ctx, "vapid_private", k.Private)
			return k
		}
	}
	return platform.VAPIDKeys{Public: pub, Private: priv}
}

func (s *Server) pushSetting(ctx context.Context, key string) string {
	b, err := repo.GetSetting(ctx, s.pool, "push", key)
	if err != nil {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return strings.Trim(string(b), `"`)
}

func (s *Server) setPushSetting(ctx context.Context, key, val string) {
	b, _ := json.Marshal(val)
	_ = repo.SetSetting(ctx, s.pool, "push", key, b)
}

func (s *Server) pushSubject(ctx context.Context) string {
	if v := s.pushSetting(ctx, "subject"); v != "" {
		return v
	}
	return "mailto:info@theblock.com.tr"
}

// handleScroll makale kaydırma derinliğini artırır (P5 "scroll_view_count" eşdeğeri).
func (s *Server) handleScroll(w http.ResponseWriter, r *http.Request) {
	if id := parseID(r.PathValue("id")); id > 0 {
		_ = repo.IncrementScroll(r.Context(), s.pool, id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePushVAPID tarayıcıya VAPID açık anahtarını verir.
func (s *Server) handlePushVAPID(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"publicKey": s.vapidKeys(r.Context()).Public})
}

type pushSubPayload struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// handlePushSubscribe aboneliği kaydeder.
func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var p pushSubPayload
	if json.NewDecoder(r.Body).Decode(&p) != nil || p.Endpoint == "" || p.Keys.P256dh == "" || p.Keys.Auth == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "geçersiz abonelik"})
		return
	}
	site := s.siteByDomain(ctx, r.Host)
	if err := repo.SavePushSub(ctx, s.pool, site.ID, p.Endpoint, p.Keys.P256dh, p.Keys.Auth, r.UserAgent()); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePushUnsubscribe aboneliği siler.
func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var p pushSubPayload
	if json.NewDecoder(r.Body).Decode(&p) == nil && p.Endpoint != "" {
		_ = repo.DeletePushSub(r.Context(), s.pool, p.Endpoint)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSW service worker betiğini sunar (kök kapsamda).
func (s *Server) handleSW(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Service-Worker-Allowed", "/")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(serviceWorkerJS))
}

// handlePushAdmin panel.
func (s *Server) handlePushAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildPushData(r)
	if IsHTMX(r) {
		s.renderPushPanelV2(w, r, d)
		return
	}
	s.renderPushPageV2(w, r, d)
}

func (s *Server) buildPushData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	n, _ := repo.CountPushSubs(ctx, s.pool, sid)
	subs, _ := repo.ListPushSubs(ctx, s.pool, sid, 200)
	view := make([]pushSubView, 0, len(subs))
	for _, sb := range subs {
		short := sb.Endpoint
		if len(short) > 42 {
			short = "…" + short[len(short)-42:]
		}
		view = append(view, pushSubView{
			Endpoint:      sb.Endpoint,
			EndpointShort: short,
			UserAgent:     sb.UserAgent,
			LastSeen:      sb.LastSeen.In(time.Local).Format("02.01 15:04"),
		})
	}
	d := s.baseView("tr", "/admin", "Bildirimler (Web Push)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Push = &pushData{Count: n, PublicKey: s.vapidKeys(ctx).Public, Subject: s.pushSubject(ctx), LastSent: s.pushSetting(ctx, "last_sent"), Subs: view}
	return d
}

func (s *Server) renderPushPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildPushData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderPushPanelV2(w, r, d)
}

func (s *Server) pushResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderPushPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/bildirim", kind, text)
}

// handlePushSubDelete aboneliği endpoint'ten siler (Faz 43).
func (s *Server) handlePushSubDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	endpoint := strings.TrimSpace(r.FormValue("endpoint"))
	if endpoint == "" {
		s.pushResult(w, r, "error", "Abonelik verisi yok.")
		return
	}
	if err := repo.DeletePushSub(ctx, s.pool, endpoint); err != nil {
		s.pushResult(w, r, "error", "Silinemedi.")
		return
	}
	ep := endpoint
	if len(ep) > 40 {
		ep = ep[:40] + "…"
	}
	s.audit(r, "push_sub_sil", "endpoint kısalt: "+ep)
	s.pushResult(w, r, "info", "Abonelik silindi.")
}

// handlePushSend tüm abonelere test bildirimi gönderir.
func (s *Server) handlePushSend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	subs, _ := repo.ListPushSubs(ctx, s.pool, sid, 500)
	if len(subs) == 0 {
		s.pushResult(w, r, "info", "Kayıtlı abone yok.")
		return
	}
	title := firstNonEmpty(strings.TrimSpace(r.FormValue("baslik")), s.siteName)
	body := firstNonEmpty(strings.TrimSpace(r.FormValue("metin")), "Yeni içerik yayında!")
	payload, _ := json.Marshal(map[string]any{"title": title, "body": body, "url": "/"})
	keys := s.vapidKeys(ctx)
	subject := s.pushSubject(ctx)
	ok, fail := 0, 0
	for _, sub := range subs {
		code, _, err := platform.SendWebPush(ctx, platform.PushSubscription{Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth}, payload, keys, subject)
		if err != nil || code >= 300 {
			fail++
			if code == 404 || code == 410 {
				_ = repo.DeletePushSub(ctx, s.pool, sub.Endpoint) // süresi geçmiş abonelik
			}
			continue
		}
		ok++
	}
	s.setPushSetting(ctx, "last_sent", time.Now().Format("2006-01-02 15:04"))
	s.audit(r, "push_gonder", itoa(ok)+" gönderildi / "+itoa(fail)+" hata")
	msg := itoa(ok) + " bildirim gönderildi."
	if fail > 0 {
		msg += " " + itoa(fail) + " hata (404/410 abonelikleri temizlendi)."
	}
	s.pushResult(w, r, "info", msg)
}

const serviceWorkerJS = `self.addEventListener('install', e => self.skipWaiting());
self.addEventListener('activate', e => e.waitUntil(self.clients.claim()));
self.addEventListener('push', e => {
  let d = {};
  try { d = e.data ? e.data.json() : {}; } catch (_) { d = {}; }
  const title = d.title || 'theblock';
  e.waitUntil(self.registration.showNotification(title, {
    body: d.body || '',
    data: { url: d.url || '/' }
  }));
});
self.addEventListener('notificationclick', e => {
  e.notification.close();
  const url = (e.notification.data && e.notification.data.url) || '/';
  e.waitUntil(self.clients.openWindow(url));
});
`
