package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// handleLoginGet giriş formu.
func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	// Yalnız personel oturumu panelin giriş ekranını atlar. Üye oturumu
	// (Google/parola ile açılmış) buraya yönlenirse 403'e düşmemeli.
	if u, err := s.currentUser(r); err == nil && auth.StaffRoles[u.Role] {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	d := s.baseView("tr", "/admin", "Giriş")
	d.Flash = s.consumeFlash(r)
	s.render(w, r, "login", d)
}

// handleLoginPost kimlik doğrulama + oturum.
func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	pass := r.FormValue("sifre")
	ip := clientIP(r)

	// Panel girişinde hız sınırı yoktu: parola sınırsız denenebiliyordu.
	if !authIPIzinli(ip) || !authHesapIzinli(ip, email) {
		s.flashRedirect(w, r, "/admin/giris", "error", authDenemeReddi)
		return
	}

	u, err := repo.GetUserByEmail(r.Context(), s.pool, email)
	if err != nil || u.Status != "aktif" {
		authBasarisiz(ip, email)
		s.flashRedirect(w, r, "/admin/giris", "error", "E-posta veya şifre hatalı.")
		return
	}
	if !auth.VerifyPassword(u.PasswordHash, pass) {
		authBasarisiz(ip, email)
		s.flashRedirect(w, r, "/admin/giris", "error", "E-posta veya şifre hatalı.")
		return
	}
	authBasarili(ip, email)

	if u.TwoFactorEnabled {
		s.setPending2FA(w, u.ID)
		s.flashRedirect(w, r, "/admin/giris/2fa", "info", "Şifre doğru — 2FA kodunu girin.")
		return
	}

	sessID, err := auth.CreateSessionScoped(r.Context(), s.pool, u.ID, r.UserAgent(), auth.ScopePanel)
	if err != nil {
		s.flashRedirect(w, r, "/admin/giris", "error", "Oturum açılamadı.")
		return
	}

	cookie := &http.Cookie{
		Name:     auth.CookieName,
		Value:    sessID,
		Path:     "/",
		MaxAge:   int(auth.SessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	next := sameSitePath(r.URL.Query().Get("sonra"))
	if next == "" {
		next = "/admin"
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && !sameOriginRequest(r) {
		s.logger.Warn("cikis: çapraz kaynak GET reddedildi",
			"referer", r.Referer(), "remote", r.RemoteAddr)
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		_ = auth.DeleteSession(r.Context(), s.pool, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
	// Üyelik çerezi de temizlenir: iki çerez ayrı kapılardan yazılıyor ama
	// tek bir "oturum" algısı oluşturuyor.
	s.clearMemberCookie(w)
	if IsHTMX(r) {
		// htmx yanıtı .admin-main'i değiştirir; çıkış tam sayfa gezinme ister.
		w.Header().Set("HX-Redirect", "/admin/giris")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/admin/giris", http.StatusFound)
}

func isHTTPS(baseURL string) bool {
	return len(baseURL) >= 8 && strings.EqualFold(baseURL[:8], "https://")
}

// navCategories üst menü kategorileri (isteğin sitesine göre).
func (s *Server) navCategories(ctx context.Context) []repo.Category {
	all := s.cachedCategories(ctx, siteIDFromCtx(ctx))
	if all == nil {
		return nil
	}
	var out []repo.Category
	for _, c := range all {
		if c.ParentID == nil && c.InHeader && c.Status == "yayinda" && len(c.Name) > 0 {
			out = append(out, c)
		}
		// haberpi menüsü: üstte 6 kategori + "Diğer" açılır menüsünde 8
		if len(out) >= 14 {
			break
		}
	}
	return out
}

var _ = time.Now
