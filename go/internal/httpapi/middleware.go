package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxSiteKey
)

// withSite isteğin host'undan site kimliğini çözüp bağlama koyar (çok-site izolasyonu).
func (s *Server) withSite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site := s.siteByDomain(r.Context(), r.Host)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxSiteKey, site.ID)))
	})
}

// currentUser çerezden oturumu çözüp kullanıcıyı döndürür.
func (s *Server) currentUser(r *http.Request) (auth.User, error) {
	c, err := r.Cookie(auth.CookieName)
	if err != nil || c.Value == "" {
		return auth.User{}, auth.ErrNoSession
	}
	return auth.SessionUser(r.Context(), s.pool, c.Value)
}

// userFromContext korumalı handler içinde kullanıcıyı döndürür.
func userFromContext(ctx context.Context) auth.User {
	u, _ := ctx.Value(ctxKeyUser).(*auth.User)
	if u == nil {
		return auth.User{}
	}
	return *u
}

// ---- flash (tek seferlik mesaj; çerezle taşınır, base64url ile ASCII-güvenli) ----

const flashCookie = "tb_flash"

func (s *Server) flash(w http.ResponseWriter, r *http.Request, kind, text string) {
	v := base64.RawURLEncoding.EncodeToString([]byte(sanitizeFlash(kind + "|" + text)))
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: v, Path: "/", MaxAge: 60, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode})
}

// consumeFlash çerezi okuyup siler ve flashMsg döndürür.
func (s *Server) consumeFlash(r *http.Request) *flashMsg {
	c, err := r.Cookie(flashCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil
	}
	return &flashMsg{Kind: parts[0], Text: parts[1]}
}

// secureCookie, çerezlere Secure bayrağının eklenip eklenmeyeceğini söyler.
// Sitenin genel adresi HTTPS ise oturum/üyelik çerezleri yalnız HTTPS
// üzerinden gönderilir; HTTP üzerinden çalışan yerel kurulumlar bozulmaz.
func (s *Server) secureCookie() bool { return isHTTPS(s.cfg.BaseURL) }

// ayniSiteYolu, Referer gibi dışarıdan gelebilen bir adresi yalnız kendi
// host'umuzdaysa site-içi göreli yola indirger. Geçersiz/çapraz kaynak
// adreslerde boş döner; çağıran kendi güvenli varsayılanına düşer.
// (Açık yönlendirme: saldırgan bağlantısına tıklayan kullanıcı kendi
// sitemizden saldırgan sitesine yönlendirilmemeli.)
func (s *Server) ayniSiteYolu(r *http.Request, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := neturl.Parse(raw); err == nil && u.Host != "" {
		if !strings.EqualFold(u.Host, r.Host) {
			return ""
		}
		p := u.EscapedPath()
		if p == "" {
			p = "/"
		}
		if u.RawQuery != "" {
			p += "?" + u.RawQuery
		}
		return p
	}
	return sameSitePath(raw)
}

func (s *Server) clearFlash(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode})
}

// flashRedirect mesaj set edip yönlendirir.
func (s *Server) flashRedirect(w http.ResponseWriter, r *http.Request, url, kind, text string) {
	s.flash(w, r, kind, text)
	http.Redirect(w, r, url, http.StatusFound)
}

func sanitizeFlash(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r < 32 {
			return ' '
		}
		return r
	}, s)
}

// render şablonu sahip isimle basar.
func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data *tmplData) {
	if data != nil && strings.HasPrefix(r.URL.Path, "/admin") {
		data.Embed = r.URL.Query().Get("embed") == "1"
		// Ağır istemci paketleri yalnız ihtiyaç duyan sayfaya
		// (bkz. adminassets.go).
		data.Scripts = adminScriptsFor(r.URL.Path)
		// Kullanıcı arayüz tercihleri (Faz 173): sunucu cihazlar arası
		// doğruluk kaynağı, localStorage anlık önbellek.
		data.Tercihler = s.tercihlerJSON(r)
		g := gorunumLoad(r.Context(), s.pool)
		data.AdminCSS = g.CSS()
		data.AdminHNav = g.Navbar == "yatay"
		if data.Gorunum == nil {
			gg := g
			data.Gorunum = &gg
		}
		if sites, err := repo.ListSites(r.Context(), s.pool); err == nil {
			data.Sites = sites
			cur := s.curSiteID(r)
			for i := range sites {
				if sites[i].ID == cur {
					data.CurSite = &sites[i]
					break
				}
			}
			if data.CurSite == nil && len(sites) > 0 {
				data.CurSite = &sites[0]
			}
		}
		sid := s.curSiteID(r)
		pc, _ := repo.CountPendingComments(r.Context(), s.pool, sid)
		fb := repo.CountFeedback(r.Context(), s.pool, sid)["bekliyor"]
		tr, _ := repo.CountUnseenTrendAlerts(r.Context(), s.pool, sid)
		data.Bekleyen = &pendingData{Comments: pc, Feedback: fb, Trend: tr}
	}
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Error("şablon basılamadı", "template", name, "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ---- requestID + yapısal log ----

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = genID()
		}
		w.Header().Set("X-Request-ID", rid)
		start := time.Now()
		lw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		s.logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lw.status,
			"dur_ms", time.Since(start).Milliseconds(),
			"req_id", rid,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush SSE ve uzun ömürlü yanıtların ara katmandan geçmesini sağlar.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap, http.ResponseController'ın alttaki bağlantıya erişmesini sağlar
// (SSE için write deadline temizliği).
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func genID() string {
	// Tahmin edilebilir olmamalı: X-Request-ID dışa dönen bir değerdir.
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand başarısızsa (pratikte olmaz) zaman tabanlı yedek.
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
