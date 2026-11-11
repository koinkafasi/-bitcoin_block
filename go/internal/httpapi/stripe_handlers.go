package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Stripe: Checkout + webhook → üyelik (paywall) ----

const memberCookieName = "tb_uye"

// stripeConfig ayarlardan (section=stripe) yapılandırmayı okur.
func (s *Server) stripeConfig(r *http.Request) platform.StripeConfig {
	get := func(key string) string {
		raw, err := repo.GetSetting(r.Context(), s.pool, "stripe", key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(strings.Trim(string(raw), `"`))
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	cfg := platform.StripeConfig{
		SecretKey:     get("secret_key"),
		WebhookSecret: get("webhook_secret"),
		PriceID:       get("price_id"),
		SuccessURL:    get("success_url"),
		CancelURL:     get("cancel_url"),
	}
	base := scheme + "://" + host
	if cfg.SuccessURL == "" {
		cfg.SuccessURL = base + "/uyelik/basarili"
	}
	if cfg.CancelURL == "" {
		cfg.CancelURL = base + "/uyelik/iptal"
	}
	return cfg
}

// handleUyelikOdeme POST /uyelik/odeme — Checkout oturumu açıp Stripe'a yollar.
func (s *Server) handleUyelikOdeme(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("eposta"))
	if email == "" {
		if u, err := s.currentUser(r); err == nil {
			email = u.Email
		}
	}
	if email == "" || !strings.Contains(email, "@") {
		http.Error(w, "Geçerli bir e-posta gerekli.", http.StatusBadRequest)
		return
	}
	cfg := s.stripeConfig(r)
	url, err := cfg.CreateCheckoutSession(r.Context(), nil, email)
	if err != nil {
		http.Error(w, "Ödeme başlatılamadı: "+err.Error(), http.StatusBadGateway)
		return
	}
	s.audit(r, "uyelik_odeme_baslat", email)
	http.Redirect(w, r, url, http.StatusFound)
}

// handleUyelikBasarili GET /uyelik/basarili — dönüş sayfası + üyelik çerezi.
// (Abonelik onayı webhook ile gelir; sayfa "onay bekleniyor" gösterir.)
func (s *Server) handleUyelikBasarili(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := strings.TrimSpace(r.FormValue("eposta"))
	if email != "" {
		s.setMemberCookie(w, email)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><title>Üyelik</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1"></head><body style="font-family:system-ui,sans-serif;max-width:560px;margin:48px auto;padding:0 16px">` +
		`<h1>🎉 Teşekkürler</h1><p>Ödemeniz alındı. Üyeliğiniz birkaç saniye içinde aktifleşir; ` +
		`bu sayfayı yenileyerek özel haberlere erişebilirsiniz.</p>` +
		`<p><a href="/">Anasayfaya dön</a></p></body></html>`))
}

// handleUyelikIptal GET /uyelik/iptal — iptal sayfası + çerez temizliği.
func (s *Server) handleUyelikIptal(w http.ResponseWriter, r *http.Request) {
	s.clearMemberCookie(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><title>Üyelik</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1"></head><body style="font-family:system-ui,sans-serif;max-width:560px;margin:48px auto;padding:0 16px">` +
		`<h1>İptal edildi</h1><p>Ödeme tamamlanmadı; üyelik oluşturulmadı.</p><p><a href="/">Anasayfaya dön</a></p></body></html>`))
}

// handleStripeWebhook POST /webhooks/stripe — ödeme/abonelik olayları.
func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		http.Error(w, "gövde okunamadı", http.StatusBadRequest)
		return
	}
	cfg := s.stripeConfig(r)
	if err := platform.VerifyWebhookSignature(body, r.Header.Get("Stripe-Signature"), cfg.WebhookSecret, time.Now()); err != nil {
		s.logger.Warn("stripe webhook imzası", "error", err)
		http.Error(w, "imza doğrulanamadı", http.StatusBadRequest)
		return
	}
	ev, err := platform.ParseStripeEvent(body)
	if err != nil {
		http.Error(w, "geçersiz olay", http.StatusBadRequest)
		return
	}
	// İdempotens: daha önce İŞLENMİŞ olay tekrar çalışmaz. Kayıt yalnızca
	// işlem BAŞARIYLA bittikten sonra atılır; hata durumunda Stripe'ın tekrar
	// deneyebilmesi için olay "görüldü" sayılmaz.
	if ev.ID != "" {
		var seen bool
		_ = s.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM stripe_events WHERE id=$1)`, ev.ID).Scan(&seen)
		if seen {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	processedOK := false
	defer func() {
		if processedOK && ev.ID != "" {
			bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = repo.MarkStripeEventSeen(bg, s.pool, ev.ID)
		}
	}()

	ctx := r.Context()
	switch ev.Type {
	case "checkout.session.completed":
		if ev.Email == "" {
			break
		}
		uid, ok, _ := repo.FindUserByEmailID(ctx, s.pool, ev.Email)
		if !ok {
			var err error
			uid, err = repo.CreateReaderUser(ctx, s.pool, ev.Email, ev.Email)
			if err != nil {
				s.logger.Error("stripe okuyucu kaydı", "error", err)
				http.Error(w, "kullanıcı oluşturulamadı", http.StatusInternalServerError)
				return
			}
		}
		_ = repo.SetUserStripeIDs(ctx, s.pool, uid, ev.CustomerID, ev.SubscriptionID)
		until := ev.PeriodEnd
		var up *time.Time
		if !until.IsZero() {
			up = &until
		}
		_ = repo.SetUserMembershipByStripe(ctx, s.pool, uid, "premium", up)
		s.logger.Info("stripe üyelik açıldı", "uid", uid, "event", ev.ID)

	case "customer.subscription.updated", "customer.subscription.deleted":
		uid, ok, _ := repo.FindUserByStripeCustomer(ctx, s.pool, ev.CustomerID)
		if !ok && ev.Email != "" {
			uid, ok, _ = repo.FindUserByEmailID(ctx, s.pool, ev.Email)
		}
		if !ok {
			break // bizim kullanıcımız değil
		}
		plan := ""
		if ev.Type == "customer.subscription.updated" &&
			(ev.SubscriptionStat == "active" || ev.SubscriptionStat == "trialing") {
			plan = "premium"
		}
		var up *time.Time
		if plan == "premium" && !ev.PeriodEnd.IsZero() {
			t := ev.PeriodEnd
			up = &t
		}
		_ = repo.SetUserMembershipByStripe(ctx, s.pool, uid, plan, up)
		if ev.SubscriptionID != "" {
			_ = repo.SetUserStripeIDs(ctx, s.pool, uid, "", ev.SubscriptionID)
		}
		s.logger.Info("stripe üyelik güncellendi", "uid", uid, "plan", plan, "event", ev.Type)

	default:
		s.logger.Info("stripe: işlenmeyen olay", "type", ev.Type)
	}
	processedOK = true
	w.WriteHeader(http.StatusOK)
}

// ---- Üyelik çerezi (imzalı; e-posta ile premium doğrulama) ----

// memberCookieValue e-posta için imzalı çerez değeri üretir: email.hmac.
func (s *Server) memberCookieValue(email string) string {
	mac := hmac.New(sha256.New, s.memberSecret())
	mac.Write([]byte(memberCookieName + ":" + strings.ToLower(email)))
	return strings.ToLower(email) + "." + hex.EncodeToString(mac.Sum(nil))
}

// memberCookieEmail çerez geçerliyse e-postayı döndürür.
func (s *Server) memberCookieEmail(r *http.Request) string {
	c, err := r.Cookie(memberCookieName)
	if err != nil || c.Value == "" {
		return ""
	}
	email, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return ""
	}
	mac := hmac.New(sha256.New, s.memberSecret())
	mac.Write([]byte(memberCookieName + ":" + strings.ToLower(email)))
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return ""
	}
	return email
}

func (s *Server) setMemberCookie(w http.ResponseWriter, email string) {
	http.SetCookie(w, &http.Cookie{
		Name:     memberCookieName,
		Value:    s.memberCookieValue(email),
		Path:     "/",
		MaxAge:   30 * 24 * 3600,
		HttpOnly: true,
		Secure:   s.secureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearMemberCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: memberCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
}

// memberSecret çerez imzalama gizli anahtarı (settings 'guvenlik'.'anahtarından
// türetilir; anahtar yoksa üretilip kaydedilir).
func (s *Server) memberSecret() []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := repo.GetSetting(ctx, s.pool, "guvenlik", "anahtar")
	key := strings.TrimSpace(string(raw))
	if err != nil || key == "" {
		key = platform.RandomHex(32)
		_ = repo.SetSetting(ctx, s.pool, "guvenlik", "anahtar", []byte(key))
	}
	sum := sha256.Sum256([]byte("uye:" + key))
	return sum[:]
}
