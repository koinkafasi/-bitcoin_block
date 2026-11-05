package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

const pending2FACookie = "tb_2fa"

// pending2FAValue, şifre adımını geçen kullanıcı için HMAC imzalı ara çerez
// değeri üretir: "uid:exp:imza".
//
// Eski biçim yalnız "uid:exp" idi; imza olmadığı için saldırgan çerezi elle
// yazıp şifre adımını atlayabiliyordu (2FA'yı devre dışı bırakmadan kodu
// denemek serbest kalıyordu).
func (s *Server) pending2FAValue(uid, exp int64) string {
	payload := strconv.FormatInt(uid, 10) + ":" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, s.memberSecret())
	_, _ = mac.Write([]byte("2fa:" + payload))
	return payload + ":" + hex.EncodeToString(mac.Sum(nil))
}

// setPending2FA şifre doğrulanmış kullanıcıyı 5 dk'lık geçici işaretle işaretler.
func (s *Server) setPending2FA(w http.ResponseWriter, userID int64) {
	exp := time.Now().Add(5 * time.Minute).Unix()
	http.SetCookie(w, &http.Cookie{
		Name:     pending2FACookie,
		Value:    s.pending2FAValue(userID, exp),
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) readPending2FA(r *http.Request) int64 {
	c, err := r.Cookie(pending2FACookie)
	if err != nil || c.Value == "" {
		return 0
	}
	parts := strings.Split(c.Value, ":")
	if len(parts) != 3 {
		return 0
	}
	uid, err1 := strconv.ParseInt(parts[0], 10, 64)
	exp, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || exp < time.Now().Unix() {
		return 0
	}
	// İmza denetimi (sabit zamanlı).
	want := s.pending2FAValue(uid, exp)
	got := c.Value
	if len(want) != len(got) || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return 0
	}
	return uid
}

func (s *Server) clearPending2FA(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: pending2FACookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
}

// handle2FAGet 2FA yönetim sayfası (giriş yapılmış).
func (s *Server) handle2FAGet(w http.ResponseWriter, r *http.Request) {
	d := s.build2FAData(r)
	if IsHTMX(r) {
		s.renderTFAPanelV2(w, r, d)
		return
	}
	s.renderTFAV2(w, r, d)
}

func (s *Server) build2FAData(r *http.Request) *tmplData {
	ctx := r.Context()
	u, err := repo.GetUserByID(ctx, s.pool, userFromContext(ctx).ID)
	if err != nil {
		d := s.baseView("tr", "/admin", "2FA (TOTP)")
		d.User = ptr(userFromContext(ctx))
		d.Flash = s.consumeFlash(r)
		return d
	}
	d := s.baseView("tr", "/admin", "2FA (TOTP)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	str := ""
	if u.TwoFactorSecret == nil || *u.TwoFactorSecret == "" {
		sec, _ := auth.GenerateSecret()
		_ = repo.UpdateUser2FA(ctx, s.pool, u.ID, &sec, false)
		str = sec
	} else {
		str = *u.TwoFactorSecret
	}
	d.TOTPSecret = str
	d.TOTPEnabled = u.TwoFactorEnabled
	return d
}

func (s *Server) render2FAPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.build2FAData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTFAPanelV2(w, r, d)
}

func (s *Server) twoFAResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.render2FAPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/2fa", kind, text)
}

// handle2FAEnable kodu doğrulayıp etkinleştirir.
func (s *Server) handle2FAEnable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := userFromContext(ctx).ID
	code := strings.TrimSpace(r.FormValue("kod"))
	u, err := repo.GetUserByID(ctx, s.pool, uid)
	if err != nil || u.TwoFactorSecret == nil {
		s.twoFAResult(w, r, "error", "Kayıt yok.")
		return
	}
	if !auth.VerifyTOTP(*u.TwoFactorSecret, code, time.Now()) {
		s.twoFAResult(w, r, "error", "Kod hatalı.")
		return
	}
	_ = repo.UpdateUser2FA(ctx, s.pool, uid, u.TwoFactorSecret, true)
	s.audit(r, "2fa_kur", "")
	s.twoFAResult(w, r, "info", "2FA etkinleştirildi — artık girişte kod istenecek.")
}

// handle2FADisable kodu doğrulayıp kapatır.
func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := userFromContext(ctx).ID
	code := strings.TrimSpace(r.FormValue("kod"))
	u, err := repo.GetUserByID(ctx, s.pool, uid)
	if err != nil || u.TwoFactorSecret == nil || !u.TwoFactorEnabled {
		s.twoFAResult(w, r, "error", "2FA açık değil.")
		return
	}
	if !auth.VerifyTOTP(*u.TwoFactorSecret, code, time.Now()) {
		s.twoFAResult(w, r, "error", "Kod hatalı.")
		return
	}
	_ = repo.UpdateUser2FA(ctx, s.pool, uid, nil, false)
	s.audit(r, "2fa_kapat", "")
	s.twoFAResult(w, r, "info", "2FA kapatıldı.")
}

// handle2FALoginGet ikinci adım: kod girişi.
func (s *Server) handle2FALoginGet(w http.ResponseWriter, r *http.Request) {
	if s.readPending2FA(r) == 0 {
		http.Redirect(w, r, "/admin/giris", http.StatusFound)
		return
	}
	d := s.baseView("tr", "/admin", "2FA Girişi")
	d.Flash = s.consumeFlash(r)
	s.render(w, r, "2falogin", d)
}

// handle2FALoginPost kodu doğrular ve gerçek oturumu açar.
func (s *Server) handle2FALoginPost(w http.ResponseWriter, r *http.Request) {
	uid := s.readPending2FA(r)
	if uid == 0 {
		s.flashRedirect(w, r, "/admin/giris", "error", "Oturum bekleme süresi doldu.")
		return
	}
	u, err := repo.GetUserByID(r.Context(), s.pool, uid)
	if err != nil || !u.TwoFactorEnabled || u.TwoFactorSecret == nil {
		s.clearPending2FA(w)
		s.flashRedirect(w, r, "/admin/giris", "error", "Geçersiz durum.")
		return
	}
	// Kaba kuvvet koruması: 6 haneli kod, 5 dakikalık pencerede sınırsız
	// denenebiliyordu. Şifre adımıyla aynı ilke: IP ve (IP+hesap) sayaçları.
	ip := clientIP(r)
	if !authIPIzinli(ip) || !authHesapIzinli(ip, u.Email) {
		s.clearPending2FA(w)
		s.flashRedirect(w, r, "/admin/giris", "error", authDenemeReddi)
		return
	}
	if limited, rateErr := repo.LoginRateLimited(r.Context(), s.pool, u.Email, ip, 8); rateErr != nil {
		s.logger.Warn("2fa hız sınırı okunamadı", "error", rateErr)
	} else if limited {
		s.clearPending2FA(w)
		s.flashRedirect(w, r, "/admin/giris", "error", authDenemeReddi)
		return
	}
	if !auth.VerifyTOTP(*u.TwoFactorSecret, r.FormValue("kod"), time.Now()) {
		authBasarisiz(ip, u.Email)
		_ = repo.RecordLoginAttempt(r.Context(), s.pool, u.Email, ip, false)
		s.flashRedirect(w, r, "/admin/giris/2fa", "error", "Kod hatalı.")
		return
	}
	authBasarili(ip, u.Email)
	_ = repo.RecordLoginAttempt(r.Context(), s.pool, u.Email, ip, true)
	sessID, err := auth.CreateSessionScoped(r.Context(), s.pool, uid, r.UserAgent(), auth.ScopePanel)
	if err != nil {
		s.flashRedirect(w, r, "/admin/giris/2fa", "error", "Oturum açılamadı.")
		return
	}
	cookie := &http.Cookie{Name: auth.CookieName, Value: sessID, Path: "/", MaxAge: int(auth.SessionLifetime.Seconds()), HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, cookie)
	s.clearPending2FA(w)
	http.Redirect(w, r, "/admin", http.StatusFound)
}
