// Stripe ödeme/abonelik entegrasyonu (SDK'sız; REST + webhook imza doğrulama).
//
// Webhook imza şeması ve olay modeli, staticbackendhq/core (MIT) stripe.go
// örneğindeki akıştan esinlenmiştir; kendi ayar/şifreleme kalıplarımıza göre
// yeniden yazılmıştır.
package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// StripeConfig Stripe yapılandırması (settings section='stripe').
type StripeConfig struct {
	SecretKey     string // sk_live_… / sk_test_… (Checkout için)
	WebhookSecret string // whsec_… (imza doğrulama için)
	PriceID       string // price_… (abonelik fiyatı)
	SuccessURL    string
	CancelURL     string
}

// Enabled Checkout oluşturma yapılandırması tam mı?
func (c StripeConfig) Enabled() bool {
	return c.SecretKey != "" && c.PriceID != ""
}

// VerifyWebhookSignature Stripe-Signature başlığını doğrular.
// Şema: t=<unix>,v1=<hmac-sha256(t + "." + payload, secret)>; tolerans 5 dk.
func VerifyWebhookSignature(payload []byte, header, secret string, now time.Time) error {
	if secret == "" {
		return errors.New("webhook sırrı tanımlı değil")
	}
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == "" || len(sigs) == 0 {
		return errors.New("imza başlığı eksik")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("imza zamanı geçersiz")
	}
	if d := now.Sub(time.Unix(sec, 0)); d > 5*time.Minute || d < -5*time.Minute {
		return errors.New("imza zamanı tolerans dışında")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(payload)))
	want := hex.EncodeToString(mac.Sum(nil))
	for _, s := range sigs {
		if hmac.Equal([]byte(want), []byte(s)) {
			return nil
		}
	}
	return errors.New("imza eşleşmedi")
}

// StripeEvent webhook olayından çıkarılan işlenmiş özet.
type StripeEvent struct {
	ID               string
	Type             string // checkout.session.completed | customer.subscription.updated | ...
	CustomerID       string
	SubscriptionID   string
	Email            string
	SubscriptionStat string // active | past_due | canceled | ...
	PeriodEnd        time.Time
}

// ParseStripeEvent webhook gövdesini özet yapıya çevirir.
func ParseStripeEvent(payload []byte) (StripeEvent, error) {
	var raw struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return StripeEvent{}, fmt.Errorf("geçersiz olay gövdesi: %w", err)
	}
	ev := StripeEvent{ID: raw.ID, Type: raw.Type}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw.Data.Object, &obj); err != nil {
		return ev, nil
	}
	str := func(k string) string {
		var s string
		_ = json.Unmarshal(obj[k], &s)
		return s
	}
	ev.CustomerID = str("customer")
	ev.SubscriptionID = str("subscription")
	if ev.SubscriptionID == "" && raw.Type != "checkout.session.completed" {
		ev.SubscriptionID = str("id")
	}
	ev.Email = str("customer_email")
	if ev.Email == "" {
		var md map[string]string
		if err := json.Unmarshal(obj["metadata"], &md); err == nil && md["email"] != "" {
			ev.Email = md["email"]
		}
	}
	ev.SubscriptionStat = str("status")
	// current_period_end bazı API sürümlerinde üst düzeydedir; yoksa 0 kalır.
	var pe int64
	_ = json.Unmarshal(obj["current_period_end"], &pe)
	if pe > 0 {
		ev.PeriodEnd = time.Unix(pe, 0)
	}
	return ev, nil
}

// CreateCheckoutSession abonelik Checkout oturumu açar ve yönlendirme
// adresini döndürür. email boş olabilir; metadata olarak taşınır.
func (c StripeConfig) CreateCheckoutSession(ctx context.Context, client *http.Client, email string) (string, error) {
	if !c.Enabled() {
		return "", errors.New("Stripe yapılandırılmadı (Üyelik → Stripe ayarları)")
	}
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	v := url.Values{}
	v.Set("mode", "subscription")
	v.Set("line_items[0][price]", c.PriceID)
	v.Set("line_items[0][quantity]", "1")
	v.Set("success_url", orDefault(c.SuccessURL, "https://example.com/uyelik/basarili"))
	v.Set("cancel_url", orDefault(c.CancelURL, "https://example.com/uyelik/iptal"))
	if email != "" {
		v.Set("customer_email", email)
		v.Set("metadata[email]", email)
		v.Set("subscription_data[metadata][email]", email)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v1/checkout/sessions", strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Stripe isteği başarısız: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("Stripe hatası (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("Stripe yanıtı çözülemedi: %w", err)
	}
	if out.URL == "" {
		return "", errors.New("Stripe checkout adresi boş döndü")
	}
	return out.URL, nil
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
