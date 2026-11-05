// SMS gönderim sağlayıcıları (Netgsm, Twilio).
//
// Twilio istemci deseni staticbackendhq/core (MIT lisanslı) sms.go örneğinden
// esinlenmiştir; Netgsm (Türkiye) adaptörü eklenmiştir.
package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SMSConfig SMS yapılandırması (ayarlar: settings section='sms').
type SMSConfig struct {
	Provider string // netgsm | twilio | (boş = kapalı)
	From     string // başlık (Netgsm msgheader) veya gönderen numara (Twilio)

	// Netgsm
	NetgsmUser     string
	NetgsmPassword string

	// Twilio
	TwilioSID   string
	TwilioToken string
}

// Enabled SMS gönderimi yapılandırılmış mı?
func (c SMSConfig) Enabled() bool {
	if c.Provider == "" {
		return false
	}
	switch c.Provider {
	case "netgsm":
		return c.NetgsmUser != "" && c.NetgsmPassword != ""
	case "twilio":
		return c.TwilioSID != "" && c.TwilioToken != "" && c.From != ""
	}
	return false
}

// Send tek bir SMS gönderir. to uluslararası biçimde olmalı (905…).
func (c SMSConfig) Send(ctx context.Context, to, body string) error {
	to = normalizeGSM(to)
	if to == "" {
		return errors.New("alıcı numara gerekli")
	}
	if !c.Enabled() {
		return errors.New("SMS yapılandırılmadı (Ayarlar → Dağıtım)")
	}
	client := &http.Client{Timeout: 25 * time.Second}
	switch c.Provider {
	case "netgsm":
		return c.sendNetgsm(ctx, client, to, body)
	case "twilio":
		return c.sendTwilio(ctx, client, to, body)
	}
	return fmt.Errorf("bilinmeyen SMS sağlayıcısı: %s", c.Provider)
}

// sendNetgsm Netgsm klasik HTTP uç noktasına gönderir.
// Başarı yanıtı "00"; diğer gövde kodları hata sayılır.
func (c SMSConfig) sendNetgsm(ctx context.Context, client *http.Client, to, body string) error {
	q := url.Values{}
	q.Set("usercode", c.NetgsmUser)
	q.Set("password", c.NetgsmPassword)
	q.Set("gsmno", to)
	q.Set("msg", body)
	if c.From != "" {
		q.Set("msgheader", c.From)
	}
	endpoint := "https://api.netgsm.com.tr/sms/send/get?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Netgsm isteği başarısız: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	code := strings.TrimSpace(string(raw))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Netgsm HTTP %d: %s", resp.StatusCode, code)
	}
	if code != "00" {
		return fmt.Errorf("Netgsm hatası: %s", code)
	}
	return nil
}

// sendTwilio Twilio Messages API'ye gönderir (Basic Auth + form).
func (c SMSConfig) sendTwilio(ctx context.Context, client *http.Client, to, body string) error {
	apiURL := "https://api.twilio.com/2010-04-01/Accounts/" + c.TwilioSID + "/Messages.json"
	v := url.Values{}
	v.Set("To", "+"+to)
	v.Set("From", c.From)
	v.Set("Body", body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.TwilioSID, c.TwilioToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Twilio isteği başarısız: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("Twilio hatası (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// normalizeGSM numarayı 90XXXXXXXXXX biçimine getirir (+90, 0090, 5…→905…).
func normalizeGSM(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "00")
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "90") && len(s) >= 12 {
		return s
	}
	if strings.HasPrefix(s, "0") {
		return "9" + s
	}
	if strings.HasPrefix(s, "5") {
		return "90" + s
	}
	return s
}
