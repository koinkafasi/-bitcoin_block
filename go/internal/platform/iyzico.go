package platform

import (
	"bytes"
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
	"strconv"
	"strings"
	"time"
)

// IyzicoConfig iyzico bağlantı ayarları.
type IyzicoConfig struct {
	APIKey      string
	SecretKey   string
	Sandbox     bool
	CallbackURL string
	Client      *http.Client
	// BaseURL test/özel ortam için taban adresi (boşsa sandbox/production).
	BaseURL string
}

// IyzicoBuyer alıcı bilgileri (TCKN zorunlu).
type IyzicoBuyer struct {
	ID               string
	Name             string
	Surname          string
	Email            string
	IdentityNumber   string
	RegistrationAddr string
	City             string
	Country          string
	IP               string
}

// IyzicoItem sepet satırı.
type IyzicoItem struct {
	ID       string
	Name     string
	Category string
	Price    float64
	ItemType string // PHYSICAL | VIRTUAL
}

// IyzicoInitRequest hosted checkout başlatma isteği.
type IyzicoInitRequest struct {
	ConversationID string
	Price          float64
	Currency       string
	BasketID       string
	Buyer          IyzicoBuyer
	Item           IyzicoItem
}

// IyzicoInitResult başlatma yanıtı.
type IyzicoInitResult struct {
	Token          string
	PaymentPageURL string
	CheckoutForm   string
	Raw            []byte
}

// IyzicoResult ödeme doğrulama sonucu.
type IyzicoResult struct {
	Status         string // success | failure
	PaymentStatus  string // SUCCESS | FAILURE
	PaymentID      string
	PaidPrice      float64
	ConversationID string
	ErrorMessage   string
	Raw            []byte
}

// IyzicoSuccess ödemenin gerçekten alındığını bildirir.
func (r IyzicoResult) Success() bool {
	return r.Status == "success" && strings.EqualFold(r.PaymentStatus, "SUCCESS")
}

// Enabled yapılandırmanın kullanılabilir olduğunu bildirir.
func (c IyzicoConfig) Enabled() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.SecretKey) != ""
}

func (c IyzicoConfig) baseURL() string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	if c.Sandbox {
		return "https://sandbox-api.iyzipay.com"
	}
	return "https://api.iyzipay.com"
}

func (c IyzicoConfig) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// IyzicoSignature iyzico v2 imzasını üretir (test edilebilir).
func IyzicoSignature(secretKey, randomKey, uriPath, body string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(randomKey + uriPath + body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// iyzicoRandom 8 karakterlik rastgele anahtar üretir (iyzico x-iyzi-rnd).
func iyzicoRandom() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (c IyzicoConfig) post(ctx context.Context, path string, payload any, out any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	rnd, err := iyzicoRandom()
	if err != nil {
		return nil, err
	}
	sig := IyzicoSignature(c.SecretKey, rnd, path, string(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-iyzi-rnd", rnd)
	req.Header.Set("Authorization", fmt.Sprintf("IYZWSv2 apiKey:%s&randomKey:%s&signature:%s", c.APIKey, rnd, sig))
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("iyzico HTTP %d: %s", resp.StatusCode, iyzicoTruncate(string(raw), 300))
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, err
		}
	}
	return raw, nil
}

func iyzicoTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Initialize hosted checkout sayfasını başlatır.
func (c IyzicoConfig) Initialize(ctx context.Context, in IyzicoInitRequest) (IyzicoInitResult, error) {
	if !c.Enabled() {
		return IyzicoInitResult{}, errors.New("iyzico yapılandırılmadı")
	}
	if strings.TrimSpace(in.Buyer.IdentityNumber) == "" {
		return IyzicoInitResult{}, errors.New("TCKN zorunlu")
	}
	price := strconv.FormatFloat(in.Price, 'f', 2, 64)
	addr := map[string]string{
		"contactName": in.Buyer.Name + " " + in.Buyer.Surname,
		"city":        in.Buyer.City,
		"country":     in.Buyer.Country,
		"address":     in.Buyer.RegistrationAddr,
	}
	payload := map[string]any{
		"locale":              "tr",
		"conversationId":      in.ConversationID,
		"price":               price,
		"paidPrice":           price,
		"currency":            in.Currency,
		"basketId":            in.BasketID,
		"paymentGroup":        "PRODUCT",
		"callbackUrl":         c.CallbackURL,
		"enabledInstallments": []int{1, 2, 3, 6, 9},
		"buyer": map[string]string{
			"id":                  in.Buyer.ID,
			"name":                in.Buyer.Name,
			"surname":             in.Buyer.Surname,
			"email":               in.Buyer.Email,
			"identityNumber":      in.Buyer.IdentityNumber,
			"registrationAddress": in.Buyer.RegistrationAddr,
			"city":                in.Buyer.City,
			"country":             in.Buyer.Country,
			"ip":                  in.Buyer.IP,
		},
		"shippingAddress": addr,
		"billingAddress":  addr,
		"basketItems": []map[string]string{{
			"id":        in.Item.ID,
			"name":      in.Item.Name,
			"category1": in.Item.Category,
			"itemType":  in.Item.ItemType,
			"price":     price,
		}},
	}
	var res struct {
		Status              string `json:"status"`
		ErrorCode           string `json:"errorCode"`
		ErrorMessage        string `json:"errorMessage"`
		Token               string `json:"token"`
		PaymentPageURL      string `json:"paymentPageUrl"`
		CheckoutFormContent string `json:"checkoutFormContent"`
	}
	raw, err := c.post(ctx, "/payment/iyzipay/checkoutform/initialize/auth/ecom", payload, &res)
	if err != nil {
		return IyzicoInitResult{Raw: raw}, err
	}
	if res.Status != "success" {
		msg := res.ErrorMessage
		if msg == "" {
			msg = "iyzico ödeme sayfası açılamadı"
		}
		return IyzicoInitResult{Raw: raw}, errors.New(msg)
	}
	return IyzicoInitResult{
		Token:          res.Token,
		PaymentPageURL: res.PaymentPageURL,
		CheckoutForm:   res.CheckoutFormContent,
		Raw:            raw,
	}, nil
}

// Retrieve callback token'ı ile ödemeyi doğrular.
func (c IyzicoConfig) Retrieve(ctx context.Context, token, conversationID string) (IyzicoResult, error) {
	if !c.Enabled() {
		return IyzicoResult{}, errors.New("iyzico yapılandırılmadı")
	}
	payload := map[string]string{
		"locale":         "tr",
		"conversationId": conversationID,
		"token":          token,
	}
	var res struct {
		Status         string `json:"status"`
		PaymentStatus  string `json:"paymentStatus"`
		PaymentID      string `json:"paymentId"`
		PaidPrice      any    `json:"paidPrice"`
		ConversationID string `json:"conversationId"`
		ErrorMessage   string `json:"errorMessage"`
	}
	raw, err := c.post(ctx, "/payment/iyzipay/checkoutform/auth/ecom/detail", payload, &res)
	if err != nil {
		return IyzicoResult{Raw: raw}, err
	}
	out := IyzicoResult{
		Status:         res.Status,
		PaymentStatus:  res.PaymentStatus,
		PaymentID:      res.PaymentID,
		ConversationID: res.ConversationID,
		ErrorMessage:   res.ErrorMessage,
		Raw:            raw,
	}
	switch v := res.PaidPrice.(type) {
	case string:
		out.PaidPrice, _ = strconv.ParseFloat(v, 64)
	case float64:
		out.PaidPrice = v
	}
	return out, nil
}

// IyzicoSubscriptionTier satın alınacak katman (fiyat/dönem).
type IyzicoSubscriptionTier struct {
	Slug  string
	Name  string
	Price float64
	Cycle string // monthly | annual
}

// NewConversationID ödeme kaydı için kimlik üretir.
func NewConversationID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("tb-%d", time.Now().UnixNano())
	}
	return "tb-" + hex.EncodeToString(b)
}
