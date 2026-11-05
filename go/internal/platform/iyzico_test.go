package platform

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIyzicoSignatureDeterministic(t *testing.T) {
	sig1 := IyzicoSignature("secret", "rnd123", "/payment/x", `{"a":1}`)
	sig2 := IyzicoSignature("secret", "rnd123", "/payment/x", `{"a":1}`)
	sig3 := IyzicoSignature("secret", "rnd124", "/payment/x", `{"a":1}`)
	assert.Equal(t, sig1, sig2)
	assert.NotEqual(t, sig1, sig3)
	// IMZWSv2 base64 imza 44 karakter (32 bayt → 44 base64).
	assert.Len(t, sig1, 44)
}

func TestIyzicoInitializeAndRetrieve(t *testing.T) {
	var gotAuth, gotRnd string
	var initPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotAuth = r.Header.Get("Authorization")
		gotRnd = r.Header.Get("x-iyzi-rnd")
		switch {
		case strings.Contains(r.URL.Path, "initialize"):
			_ = json.Unmarshal(body, &initPayload)
			_, _ = w.Write([]byte(`{"status":"success","token":"tok-1","paymentPageUrl":"https://sandbox.iyzico/pay/tok-1"}`))
		case strings.Contains(r.URL.Path, "detail"):
			_, _ = w.Write([]byte(`{"status":"success","paymentStatus":"SUCCESS","paymentId":"pay-9","paidPrice":"150.0","conversationId":"conv-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := IyzicoConfig{APIKey: "key", SecretKey: "secret", BaseURL: srv.URL, Client: srv.Client(),
		CallbackURL: "https://site.test/abonelik/geri-donus"}
	require.True(t, cfg.Enabled())

	res, err := cfg.Initialize(context.Background(), IyzicoInitRequest{
		ConversationID: "conv-1", Price: 150, Currency: "TRY", BasketID: "filiz-monthly",
		Buyer: IyzicoBuyer{ID: "7", Name: "Ayşe", Surname: "Yılmaz", Email: "a@b.c",
			IdentityNumber: "12345678901", RegistrationAddr: "Adres", City: "İstanbul",
			Country: "Turkey", IP: "127.0.0.1"},
		Item: IyzicoItem{ID: "filiz", Name: "Filiz (aylık)", Category: "Abonelik", Price: 150, ItemType: "VIRTUAL"},
	})
	require.NoError(t, err)
	assert.Equal(t, "tok-1", res.Token)
	assert.Contains(t, res.PaymentPageURL, "tok-1")
	assert.True(t, strings.HasPrefix(gotAuth, "IYZWSv2 apiKey:key&randomKey:"))
	assert.Contains(t, gotAuth, "&signature:")
	assert.NotEmpty(t, gotRnd)
	assert.Equal(t, "conv-1", initPayload["conversationId"])
	assert.Equal(t, "150.00", initPayload["price"])
	buyer := initPayload["buyer"].(map[string]any)
	assert.Equal(t, "12345678901", buyer["identityNumber"])

	detail, err := cfg.Retrieve(context.Background(), "tok-1", "conv-1")
	require.NoError(t, err)
	assert.True(t, detail.Success())
	assert.Equal(t, "pay-9", detail.PaymentID)
	assert.Equal(t, 150.0, detail.PaidPrice)
}

func TestIyzicoTCKNZorunlu(t *testing.T) {
	cfg := IyzicoConfig{APIKey: "k", SecretKey: "s"}
	_, err := cfg.Initialize(context.Background(), IyzicoInitRequest{
		ConversationID: "c", Price: 10, Currency: "TRY",
		Buyer: IyzicoBuyer{Name: "A", Surname: "B", Email: "a@b.c"},
		Item:  IyzicoItem{ID: "x", Name: "x", Price: 10},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TCKN")
}

func TestIyzicoFailureStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"failure","errorMessage":"kart reddedildi"}`))
	}))
	defer srv.Close()
	cfg := IyzicoConfig{APIKey: "k", SecretKey: "s", BaseURL: srv.URL, Client: srv.Client()}
	_, err := cfg.Initialize(context.Background(), IyzicoInitRequest{
		ConversationID: "c", Price: 10, Currency: "TRY",
		Buyer: IyzicoBuyer{ID: "1", Name: "A", Surname: "B", Email: "a@b.c", IdentityNumber: "12345678901"},
		Item:  IyzicoItem{ID: "x", Name: "x", Price: 10},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kart reddedildi")
}

func TestNewConversationID(t *testing.T) {
	a := NewConversationID()
	b := NewConversationID()
	assert.NotEqual(t, a, b)
	assert.True(t, strings.HasPrefix(a, "tb-"))
}
