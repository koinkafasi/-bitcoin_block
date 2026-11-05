package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// ---- Google ile giriş (okuyucu) ----
//
// Akış deseni staticbackendhq/core oauth.go örneğinden esinlenmiştir;
// goth bağımlılığı olmadan, yalnız Google için yeniden yazılmıştır.
// Güvenlik: yalnız 'Üye' rolündeki hesaplar Google ile giriş yapabilir;
// editoryal roller parola + 2FA'ya bağlı kalır.

const googleStateCookie = "tb_gstate"

// googleRedirectCookie, OAuth turu sırasında dönüş hedefini taşır. Değer
// yalnız "hesabim"/"abonelik" anahtarlarından biri olabilir; çerez
// imzasızdır ama readerRedirect bilinmeyen değeri "/" yapar (açık
// yönlendirme yok).
const googleRedirectCookie = "tb_gredir"

// googleDonusAnahtari yalnız bilinen hedef anahtarlarını kabul eder.
func googleDonusAnahtari(q string) string {
	switch q {
	case "abonelik", "hesabim":
		return q
	default:
		return ""
	}
}

// googleConfig ayarlardan (section=google) OAuth yapılandırmasını okur.
func (s *Server) googleConfig(r *http.Request) (clientID, clientSecret string) {
	get := func(key string) string {
		raw, err := repo.GetSetting(r.Context(), s.pool, "google", key)
		if err != nil {
			return ""
		}
		return strings.Trim(strings.TrimSpace(string(raw)), `"`)
	}
	return get("client_id"), get("client_secret")
}

func (s *Server) googleEnabled(r *http.Request) bool {
	id, sec := s.googleConfig(r)
	return id != "" && sec != ""
}

// handleGirisGoogle GET /giris/google — Google yetkilendirmesine yönlendirir.
func (s *Server) handleGirisGoogle(w http.ResponseWriter, r *http.Request) {
	clientID, _ := s.googleConfig(r)
	if clientID == "" {
		http.Error(w, "Google girişi yapılandırılmadı (Üyelik → Google ayarları).", http.StatusServiceUnavailable)
		return
	}
	state, err := s.newGoogleState(w)
	if err != nil {
		http.Error(w, "Oturum durumu üretilemedi.", http.StatusInternalServerError)
		return
	}
	// Şifreli girişte olduğu gibi Google dönüşü de hedefi hatırlar.
	http.SetCookie(w, &http.Cookie{
		Name:     googleRedirectCookie,
		Value:    googleDonusAnahtari(r.URL.Query().Get("redirect")),
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	redirect := scheme + "://" + r.Host + "/giris/google/callback"
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirect)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("prompt", "select_account")
	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/v2/auth?"+q.Encode(), http.StatusFound)
}

// handleGirisGoogleCallback GET /giris/google/callback — kodu doğrular,
// kullanıcıyı bulur/oluşturur ve oturum açar.
func (s *Server) handleGirisGoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		http.Error(w, "Google girişi iptal edildi: "+e, http.StatusBadRequest)
		return
	}
	if !s.verifyGoogleState(r, q.Get("state")) {
		http.Error(w, "Geçersiz oturum durumu (state).", http.StatusBadRequest)
		return
	}
	clientID, clientSecret := s.googleConfig(r)
	if clientSecret == "" {
		http.Error(w, "Google girişi yapılandırılmadı.", http.StatusServiceUnavailable)
		return
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	redirect := scheme + "://" + r.Host + "/giris/google/callback"

	email, err := s.googleExchange(r.Context(), clientID, clientSecret, redirect, q.Get("code"))
	if err != nil {
		s.logger.Warn("google kod değişimi", "error", err)
		http.Error(w, "Google girişi doğrulanamadı.", http.StatusBadGateway)
		return
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		http.Error(w, "Google hesabından e-posta alınamadı.", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	uid, ok, _ := repo.FindUserByEmailID(ctx, s.pool, email)
	if !ok {
		// Yeni okuyucu hesabı (parolasız; giriş Google üzerinden).
		var err error
		uid, err = repo.CreateReaderUser(ctx, s.pool, email, email)
		if err != nil {
			s.logger.Error("google kullanıcı kaydı", "error", err)
			http.Error(w, "Hesap oluşturulamadı.", http.StatusInternalServerError)
			return
		}
	} else {
		// Editoryal roller parola + 2FA kullanmalı; Google girişi reddedilir.
		var role string
		_ = s.pool.QueryRow(ctx, `SELECT role FROM users WHERE id=$1`, uid).Scan(&role)
		if auth.StaffRoles[role] {
			http.Error(w, "Yönetici/editör hesapları Google ile giriş yapamaz; lütfen şifre ile giriş yapın.", http.StatusForbidden)
			return
		}
	}

	sessID, err := auth.CreateSessionScoped(ctx, s.pool, uid, r.UserAgent(), auth.ScopeUye)
	if err != nil {
		http.Error(w, "Oturum açılamadı.", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    sessID,
		Path:     "/",
		MaxAge:   int(auth.SessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	s.audit(r, "google_giris", email)

	// Premium ise çerez de yaz (paywall oturumsuz tanısın diye).
	if uid2, ok2, _ := repo.FindUserByEmailID(ctx, s.pool, email); ok2 {
		var m string
		if err := s.pool.QueryRow(ctx, `SELECT membership FROM users WHERE id=$1`, uid2).Scan(&m); err == nil && m == "premium" {
			s.setMemberCookie(w, email)
		}
	}
	hedef := "/"
	if c, err := r.Cookie(googleRedirectCookie); err == nil {
		hedef = readerRedirect(c.Value)
		// Tek kullanımlık: dönüş hedefi bir sonraki girişe taşınmasın.
		http.SetCookie(w, &http.Cookie{
			Name:     googleRedirectCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   s.secureCookie(),
			SameSite: http.SameSiteLaxMode,
		})
	}
	http.Redirect(w, r, hedef, http.StatusFound)
}

// googleExchange yetkilendirme kodunu token'la değiştirip e-postayı döndürür.
func (s *Server) googleExchange(ctx context.Context, clientID, clientSecret, redirect, code string) (string, error) {
	if code == "" {
		return "", errors.New("kod eksik")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", redirect)
	form.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("token uç noktası HTTP %d", resp.StatusCode)
	}
	var tok struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", err
	}
	if tok.IDToken == "" {
		return "", errors.New("id_token yok")
	}
	return googleEmailFromIDToken(tok.IDToken)
}

// googleEmailFromIDToken id_token JWT'sinden e-posta alanını okur.
// İmza doğrulaması Google'ın token uç noktasının yanıtına dayanır
// (kod değişimi yapılmadan id_token üretilemez).
func googleEmailFromIDToken(idToken string) (string, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", errors.New("geçersiz id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}
	return claims.Email, nil
}

// ---- state: CSRF korumalı yetkilendirme durumu ----

// newGoogleState rastgele state üretir, HMAC'li çerezde saklar.
func (s *Server) newGoogleState(w http.ResponseWriter) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(b[:])
	http.SetCookie(w, &http.Cookie{
		Name:     googleStateCookie,
		Value:    s.signGoogleState(raw),
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	return raw, nil
}

func (s *Server) signGoogleState(raw string) string {
	mac := hmac.New(sha256.New, s.memberSecret())
	mac.Write([]byte(googleStateCookie + ":" + raw))
	return raw + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyGoogleState çerezdeki state ile sorgudakini eşler.
func (s *Server) verifyGoogleState(r *http.Request, state string) bool {
	if state == "" {
		return false
	}
	c, err := r.Cookie(googleStateCookie)
	if err != nil || c.Value == "" {
		return false
	}
	want := s.signGoogleState(state)
	return hmac.Equal([]byte(want), []byte(c.Value))
}
