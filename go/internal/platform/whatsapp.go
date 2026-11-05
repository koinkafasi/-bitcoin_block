package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---- WhatsApp Business Cloud API (Meta) — gazete bülten dağıtımı ----
//
// PDF gönderimi onaylı şablon mesajı (document) ile yapılır; opt-in zorunludur.
// Yapılandırma settings('whatsapp'): token, phone_id, api_version, template.
// Anahtar yoksa sağlayıcı "yapılandırılmamış" döner; çağıran e-postaya düşer.

// WhatsAppConfig Meta Cloud API yapılandırması.
type WhatsAppConfig struct {
	Token      string
	PhoneID    string
	APIVersion string // varsayılan v21.0
	Template   string // onaylı şablon adı (document gönderimi için)
	HTTPClient *http.Client
}

// Configured API anahtarı ve telefon kimliği var mı?
func (c WhatsAppConfig) Configured() bool {
	return strings.TrimSpace(c.Token) != "" && strings.TrimSpace(c.PhoneID) != ""
}

// SendDocument alıcıya PDF dokümanı gönderir (link ile; herkese açık PDF URL'i).
// to E.164 formatında (örn. +905321234567). Meta şablon kategorisi: utility/marketing.
func (c WhatsAppConfig) SendDocument(to, bodyText, pdfURL, filename string) error {
	if !c.Configured() {
		return errors.New("WhatsApp yapılandırılmamış")
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("alıcı telefon boş")
	}
	if strings.TrimSpace(pdfURL) == "" {
		return errors.New("PDF adresi boş")
	}
	if filename == "" {
		filename = "gazete.pdf"
	}
	version := c.APIVersion
	if version == "" {
		version = "v21.0"
	}
	msg := map[string]any{
		"messaging_product": "whatsapp",
		"to":                normalizePhone(to),
		"type":              "document",
		"document": map[string]any{
			"link":     pdfURL,
			"filename": filename,
			"caption":  bodyText,
		},
	}
	// Onaylı şablon gereksinimi: şablon adı verilmişse template mesajı olarak gönder.
	if strings.TrimSpace(c.Template) != "" {
		msg["type"] = "template"
		msg["template"] = map[string]any{
			"name":     c.Template,
			"language": map[string]any{"code": "tr"},
			"components": []map[string]any{
				{"type": "header", "format": "document",
					"document": map[string]any{"link": pdfURL, "filename": filename}},
				{"type": "body", "parameters": []map[string]any{{"type": "text", "text": bodyText}}},
			},
		}
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", version, c.PhoneID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("WhatsApp API %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// normalizePhone basit E.164 sadeleştirme (boşluk, parantez, tire temizler).
func normalizePhone(p string) string {
	p = strings.TrimSpace(p)
	p = strings.NewReplacer(" ", "", "(", "", ")", "", "-", "", ".", "").Replace(p)
	if p != "" && !strings.HasPrefix(p, "+") {
		p = "+" + p
	}
	return p
}
