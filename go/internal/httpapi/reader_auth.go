package httpapi

import (
	"context"
	"errors"
	"html"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// readerRedirect yalnız bilinen hedef anahtarlarını kabul eder (open-redirect
// koruması; edirnegercek deseni).
func readerRedirect(q string) string {
	switch q {
	case "abonelik":
		return "/abonelik"
	case "hesabim":
		return "/hesabim"
	default:
		return "/"
	}
}

// handleUyeGiris GET /giris — giriş formu.
func (s *Server) handleUyeGiris(w http.ResponseWriter, r *http.Request) {
	if u, err := s.currentUser(r); err == nil && u.ID > 0 {
		http.Redirect(w, r, "/hesabim", http.StatusFound)
		return
	}
	d := s.publicView(r, "Giriş Yap")
	d.UyeMode = "giris"
	d.UyeRedirect = readerRedirect(r.URL.Query().Get("redirect"))
	s.renderUye(w, r, d)
}

// handleUyeGirisPost POST /giris — kimlik doğrulama.
func (s *Server) handleUyeGirisPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if !s.botTrapOK(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	login := strings.TrimSpace(r.FormValue("login"))
	pass := r.FormValue("password")
	if login == "" || pass == "" {
		s.uyeFormError(w, r, "giris", "E-posta/kullanıcı adı ve şifre gerekli.", login)
		return
	}
	ip := clientIP(r)
	limited, rateErr := repo.LoginRateLimited(r.Context(), s.pool, login, ip, 8)
	if rateErr != nil {
		s.logger.Warn("giriş hız sınırı okunamadı", "error", rateErr)
	}
	if limited || !authIPIzinli(ip) || !authHesapIzinli(ip, login) {
		s.uyeFormError(w, r, "giris", "Çok fazla deneme yaptınız. Lütfen 10 dakika sonra tekrar deneyin.", login)
		return
	}
	id, hash, role, status, err := repo.FindUserForLogin(r.Context(), s.pool, login)
	if err != nil || !auth.VerifyPassword(hash, pass) {
		authBasarisiz(ip, login)
		_ = repo.RecordLoginAttempt(r.Context(), s.pool, login, ip, false)
		s.uyeFormError(w, r, "giris", "Kullanıcı adı/e-posta veya şifre hatalı.", login)
		return
	}
	if status != "aktif" {
		s.uyeFormError(w, r, "giris", "Hesabınız aktif değil.", login)
		return
	}
	// Editoryal roller parola girişini /admin/giris üzerinden yapar.
	if auth.StaffRoles[role] {
		http.Redirect(w, r, "/admin/giris", http.StatusFound)
		return
	}
	authBasarili(ip, login)
	_ = repo.RecordLoginAttempt(r.Context(), s.pool, login, ip, true)
	if err := s.uyeOpenSession(w, r, id); err != nil {
		s.uyeFormError(w, r, "giris", "Oturum açılamadı. Lütfen tekrar deneyin.", login)
		return
	}
	http.Redirect(w, r, readerRedirect(r.FormValue("redirect")), http.StatusFound)
}

// uyeKayitAcik üye kaydının açık olup olmadığını döndürür (varsayılan: açık).
func (s *Server) uyeKayitAcik(ctx context.Context) bool {
	raw := s.cachedSetting(ctx, "uyelik", "kayit_acik")
	if len(raw) == 0 {
		return true
	}
	switch strings.ToLower(strings.Trim(strings.TrimSpace(string(raw)), `"`)) {
	case "0", "false", "kapali", "hayir":
		return false
	}
	return true
}

// handleUyeKayit GET /kayit — kayıt formu.
func (s *Server) handleUyeKayit(w http.ResponseWriter, r *http.Request) {
	if u, err := s.currentUser(r); err == nil && u.ID > 0 {
		http.Redirect(w, r, "/hesabim", http.StatusFound)
		return
	}
	// KONUM KAPISI (14 Kasım 2026): kayıt da konum izni olmadan açılmıyor
	// (kullanıcı isteği). Giriş (POST /giris) ENGELLENMİYOR: mevcut üye
	// her zaman hesabına girebilmeli.
	if s.konumKapiGerekli(r) {
		s.konumKapiSayfa(w, r)
		return
	}
	d := s.publicView(r, "Kayıt Ol")
	d.UyeMode = "kayit"
	if !s.uyeKayitAcik(r.Context()) {
		d.UyeMode = "kayit-kapali"
	}
	d.UyeRedirect = readerRedirect(r.URL.Query().Get("redirect"))
	s.renderUye(w, r, d)
}

// handleUyeKayitPost POST /kayit — yeni okuyucu hesabı.
func (s *Server) handleUyeKayitPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if !s.botTrapOK(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	// Sunucu tarafı kapı: JS kapalı/doğrudan POST ile atlanamasın.
	if s.konumKapiGerekli(r) {
		http.Redirect(w, r, s.langBase(langOf(r, s))+"/kayit", http.StatusFound)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	pass := r.FormValue("password")
	pass2 := r.FormValue("password2")
	terms := r.FormValue("terms")
	if !s.uyeKayitAcik(r.Context()) {
		s.uyeFormError(w, r, "kayit", "Yeni üye kayıtları şu anda kapalı.", name)
		return
	}
	switch {
	case name == "" || len(name) > 120:
		s.uyeFormError(w, r, "kayit", "Ad soyad gerekli (en fazla 120 karakter).", name)
		return
	case email == "" || !strings.Contains(email, "@"):
		s.uyeFormError(w, r, "kayit", "Geçerli bir e-posta gerekli.", name)
		return
	case len(pass) < 8:
		s.uyeFormError(w, r, "kayit", "Şifre en az 8 karakter olmalı.", name)
		return
	case pass != pass2:
		s.uyeFormError(w, r, "kayit", "Şifreler eşleşmiyor.", name)
		return
	case terms != "1":
		s.uyeFormError(w, r, "kayit", "Kullanım Koşulları'nı onaylamanız gerekiyor.", name)
		return
	}
	// Hız sınırı: kayıt denemeleri de sayılır.
	ip := clientIP(r)
	if !kayitIzinli(ip) {
		s.uyeFormError(w, r, "kayit", "Çok fazla kayıt denemesi. Lütfen daha sonra tekrar deneyin.", name)
		return
	}
	kayitSay(ip)
	limited, rateErr := repo.LoginRateLimited(r.Context(), s.pool, email, ip, 5)
	if rateErr != nil {
		s.logger.Warn("kayıt hız sınırı okunamadı", "error", rateErr)
	}
	if limited {
		s.uyeFormError(w, r, "kayit", "Çok fazla deneme yaptınız. Lütfen daha sonra tekrar deneyin.", name)
		return
	}
	hash, err := auth.HashPassword(pass)
	if err != nil {
		s.uyeFormError(w, r, "kayit", "Hesap oluşturulamadı.", name)
		return
	}
	id, err := repo.CreateReader(r.Context(), s.pool, name, email, hash)
	if err != nil {
		if errors.Is(err, repo.ErrEmailKayitli) {
			s.uyeFormError(w, r, "kayit", "Bu e-posta zaten kayıtlı. Giriş yapmayı deneyin.", name)
			return
		}
		s.uyeFormError(w, r, "kayit", "Hesap oluşturulamadı. Lütfen tekrar deneyin.", name)
		return
	}
	_ = repo.RecordLoginAttempt(r.Context(), s.pool, email, ip, true)
	// E-posta doğrulama bağlantısı gönder (başarısızlık oturumu engellemez).
	s.sendVerifyEmail(r, id, email)
	if err := s.uyeOpenSession(w, r, id); err != nil {
		s.uyeFormError(w, r, "kayit", "Hesap oluşturuldu ancak oturum açılamadı. Lütfen giriş yapın.", name)
		return
	}
	http.Redirect(w, r, readerRedirect(r.FormValue("redirect")), http.StatusFound)
}

// handleUyeSifremiUnuttum GET /sifremi-unuttum.
func (s *Server) handleUyeSifremiUnuttum(w http.ResponseWriter, r *http.Request) {
	d := s.publicView(r, "Şifremi Unuttum")
	d.UyeMode = "sifremi-unuttum"
	s.renderUye(w, r, d)
}

// handleUyeSifremiUnuttumPost POST /sifremi-unuttum — e-posta ile sıfırlama
// bağlantısı (varlık sızıntısını önlemek için yanıt her zaman aynıdır).
func (s *Server) handleUyeSifremiUnuttumPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if !s.botTrapOK(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if email != "" {
		if id, _, _, _, err := repo.FindUserForLogin(r.Context(), s.pool, email); err == nil && id > 0 {
			if token, err := repo.NewPasswordResetToken(r.Context(), s.pool, id); err == nil {
				s.sendMail(email, "Şifre sıfırlama",
					"<p>Şifrenizi sıfırmak için bağlantıya tıklayın (30 dakika geçerli):</p>"+
						"<p><a href=\""+html.EscapeString(s.absURL(r, "/sifre-sifirla?token="+token))+"\">Şifremi Sıfırla</a></p>")
			}
		}
	}
	d := s.publicView(r, "Şifremi Unuttum")
	d.UyeMode = "sifremi-unuttum"
	d.UyeInfo = "E-posta adresiniz kayıtlıysa sıfırlama bağlantısı gönderildi."
	s.renderUye(w, r, d)
}

// handleUyeSifreSifirla GET /sifre-sifirla?token=…
func (s *Server) handleUyeSifreSifirla(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if !repo.PasswordResetTokenGecerli(r.Context(), s.pool, token) {
		s.renderSifreBaglantiGecersiz(w, r)
		return
	}
	d := s.publicView(r, "Şifre Sıfırla")
	d.UyeMode = "sifre-sifirla"
	d.UyeToken = token
	s.renderUye(w, r, d)
}

// handleUyeSifreSifirlaPost POST /sifre-sifirla.
func (s *Server) handleUyeSifreSifirlaPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token := strings.TrimSpace(r.FormValue("token"))
	pass := r.FormValue("password")
	pass2 := r.FormValue("password2")
	fail := func(msg string) {
		d := s.publicView(r, "Şifre Sıfırla")
		d.UyeMode = "sifre-sifirla"
		d.UyeToken = token
		d.UyeError = msg
		s.renderUye(w, r, d)
	}
	if token == "" {
		fail("Bağlantı eksik.")
		return
	}
	if len(pass) < 8 {
		fail("Şifre en az 8 karakter olmalı.")
		return
	}
	if pass != pass2 {
		fail("Şifreler eşleşmiyor.")
		return
	}
	uid, err := repo.ConsumePasswordResetToken(r.Context(), s.pool, token)
	if err != nil {
		s.renderSifreBaglantiGecersiz(w, r)
		return
	}
	hash, err := auth.HashPassword(pass)
	if err != nil {
		fail("Şifre değiştirilemedi.")
		return
	}
	if err := repo.SetPassword(r.Context(), s.pool, uid, hash); err != nil {
		fail("Şifre değiştirilemedi.")
		return
	}
	d := s.publicView(r, "Şifre Sıfırlandı")
	d.UyeMode = "sifre-tamam"
	s.renderUye(w, r, d)
}

// handleUyeCikis POST /cikis — oturumu kapatır.
func (s *Server) handleUyeCikis(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		_ = auth.DeleteSession(r.Context(), s.pool, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
	s.clearMemberCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleUyeOnay GET /e-posta-onay?token=… — e-posta doğrulama.
func (s *Server) handleUyeOnay(w http.ResponseWriter, r *http.Request) {
	d := s.publicView(r, "E-posta Doğrulama")
	if _, err := repo.VerifyEmail(r.Context(), s.pool, strings.TrimSpace(r.URL.Query().Get("token"))); err != nil {
		d.UyeInfo = "Bağlantı geçersiz veya süresi dolmuş."
	} else {
		d.UyeInfo = "E-posta adresiniz doğrulandı. Teşekkürler!"
		d.UyeOK = true
	}
	d.UyeMode = "onay"
	s.renderUye(w, r, d)
}

// ---- yardımcılar ----

// uyeOpenSession oturum açar ve üye çerezini tazeler.
func (s *Server) uyeOpenSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	sess, err := auth.CreateSessionScoped(r.Context(), s.pool, userID, r.UserAgent(), auth.ScopeUye)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    sess,
		Path:     "/",
		MaxAge:   int(auth.SessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	_ = repo.TouchLastLogin(r.Context(), s.pool, userID)
	return nil
}

// uyeFormError formu hata mesajıyla yeniden basar.
func (s *Server) uyeFormError(w http.ResponseWriter, r *http.Request, mode, msg, name string) {
	d := s.publicView(r, "Üyelik")
	d.UyeMode = mode
	d.UyeError = msg
	d.UyeName = name
	d.UyeRedirect = readerRedirect(r.FormValue("redirect"))
	s.renderUye(w, r, d)
}

// publicView genel site sayfası verisi üretir.
func (s *Server) publicView(r *http.Request, title string) *tmplData {
	locale := langOf(r, s)
	return s.baseView(locale, s.langBase(locale), title)
}

// sendMail genel amaçlı e-posta gönderimi (SMTP yapılandırması yoksa sessiz).
func (s *Server) sendMail(to, subject, htmlBody string) {
	cfg := s.smtpConfig(context.Background())
	if cfg.Host == "" || to == "" {
		return
	}
	_ = cfg.SendMail(to, subject, htmlBody)
}

// renderUye üyelik sayfalarını (giriş/kayıt/şifre) yazar.
// uyeKur giriş illüstrasyonundaki bir kahramanın kur rozeti.
type uyeKur struct {
	// Mate illüstrasyondaki karakterin anahtarı (data-mate ile aynı).
	Mate string
	Sym  string
	// Fiyat biçimlenmiş metin ($74.520 gibi); şablon sayı biçimlemez.
	Fiyat string
	// Degisim 24 saatlik yüzde — işaretiyle birlikte metin (+1,4%).
	Degisim string
	// Yukari rozetin yeşil mi kırmızı mı çizileceğini söyler.
	Yukari bool
}

func (s *Server) uyeKurlari(ctx context.Context) map[string]uyeKur {
	istenen := map[string]string{
		"eth": "ETH",
		"sol": "SOL",
		"btc": "BTC",
		"bnb": "BNB",
	}
	satirlar := repo.Fiyatlar(ctx, s.pool)
	if len(satirlar) == 0 {
		return nil
	}
	indeks := make(map[string]repo.FiyatSatiri, len(satirlar))
	for _, sat := range satirlar {
		indeks[strings.ToUpper(sat.Sym)] = sat
	}
	out := make(map[string]uyeKur, len(istenen))
	for mate, sym := range istenen {
		sat, ok := indeks[sym]
		if !ok || !sat.HasData || sat.Price <= 0 {
			continue
		}
		isaret := "+"
		if sat.Change < 0 {
			isaret = "−" // U+2212: eksi işareti, kısa çizgi değil
		}
		out[mate] = uyeKur{
			Mate:    mate,
			Sym:     sym,
			Fiyat:   repo.FiyatBicim(sat.Price), // $ işaretini biçimleyici koyar
			Degisim: isaret + strconv.FormatFloat(math.Abs(sat.Change), 'f', 2, 64) + "%",
			Yukari:  sat.Change >= 0,
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Server) renderUye(w http.ResponseWriter, r *http.Request, d *tmplData) {
	name := "uye-giris"
	switch d.UyeMode {
	case "kayit":
		name = "uye-kayit"
	case "kayit-kapali":
		name = "uye-kayit"
	case "sifremi-unuttum":
		name = "uye-sifremi-unuttum"
	case "sifre-sifirla":
		name = "uye-sifre-sifirla"
	case "sifre-tamam":
		name = "uye-sifre-tamam"
	case "onay":
		name = "uye-onay"
	}
	// Üyelik sayfaları arama sonucu olmamalı: form ve hata metinleri
	// dizine girmesin. Sayfa içi bağlantılar yine izlenir (follow).
	d.Robots = "noindex,follow"
	d.GoogleEnabled = s.googleEnabled(r)
	d.UyeKayitAcik = s.uyeKayitAcik(r.Context())
	d.UyeFormTS = time.Now().Format(time.RFC3339)
	d.UyeKurlar = s.uyeKurlari(r.Context())
	s.renderPublic(w, r, name, d, "no-store")
}

func (s *Server) renderSifreBaglantiGecersiz(w http.ResponseWriter, r *http.Request) {
	d := s.publicView(r, "Şifre Sıfırla")
	d.UyeMode = "sifremi-unuttum"
	d.UyeError = "Bağlantı geçersiz veya süresi dolmuş. Yeni bir sıfırlama bağlantısı isteyin."
	s.renderUye(w, r, d)
}

// memberInitial avatar için adın ilk harfini (büyük) döndürür.
func memberInitial(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	r := []rune(name)
	return strings.ToUpper(string(r[0]))
}

// botTrapOK honeypot + zaman tuzağı kontrolü (bot ise istek yutulur).
func (s *Server) botTrapOK(r *http.Request) bool {
	if strings.TrimSpace(r.FormValue("website")) != "" { // gizli alan dolu = bot
		return false
	}
	if ts := r.FormValue("form_ts"); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil && time.Since(t) < 2*time.Second {
			return false
		}
	}
	return true
}

// sendVerifyEmail e-posta doğrulama bağlantısı gönderir.
func (s *Server) sendVerifyEmail(r *http.Request, userID int64, email string) {
	token, err := repo.SetEmailVerifyToken(r.Context(), s.pool, userID)
	if err != nil {
		return
	}
	s.sendMail(email, "E-posta adresinizi doğrulayın",
		"<p>Hesabınızı doğrulamak için bağlantıya tıklayın:</p>"+
			"<p><a href=\""+s.absURL(r, "/e-posta-onay?token="+token)+"\">E-postamı Doğrula</a></p>")
}

// absURL istekten mutlak adres üretir.
func (s *Server) absURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}
