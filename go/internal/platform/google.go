package platform

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GoogleOAuth Google OAuth 2.0 (offline erişim) yardımcısı — AdSense/GA API'leri için.
type GoogleOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Configured istemci kimlik bilgileri yeterli mi?
func (g GoogleOAuth) Configured() bool {
	return strings.TrimSpace(g.ClientID) != "" && strings.TrimSpace(g.ClientSecret) != ""
}

// AuthURL kullanıcıyı yönlendirilecek onay URL'ini üretir (offline refresh token için).
func (g GoogleOAuth) AuthURL(state string, scopes ...string) string {
	q := url.Values{}
	q.Set("client_id", g.ClientID)
	q.Set("redirect_uri", g.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("include_granted_scopes", "true")
	q.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
}

// TokenSet OAuth token yanıtı.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// Exchange authorization code'u token'a çevirir.
func (g GoogleOAuth) Exchange(ctx context.Context, code string) (TokenSet, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("redirect_uri", g.RedirectURI)
	form.Set("grant_type", "authorization_code")
	return g.tokenRequest(ctx, form)
}

// Refresh refresh token ile yeni access token alır.
func (g GoogleOAuth) Refresh(ctx context.Context, refreshToken string) (TokenSet, error) {
	form := url.Values{}
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("grant_type", "refresh_token")
	return g.tokenRequest(ctx, form)
}

func (g GoogleOAuth) tokenRequest(ctx context.Context, form url.Values) (TokenSet, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return TokenSet{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenSet{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return TokenSet{}, fmt.Errorf("token ucu HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return TokenSet{}, err
	}
	if out.AccessToken == "" {
		return TokenSet{}, errors.New("access_token dönmedi")
	}
	return TokenSet{AccessToken: out.AccessToken, RefreshToken: out.RefreshToken, ExpiresIn: out.ExpiresIn}, nil
}

// GetJSON Bearer token ile GET; JSON gövdeyi döndürür (AdSense/GA okuma çağrıları için).
func GetJSON(ctx context.Context, accessToken, urlStr string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return body, resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---- Google servis hesabı (JWT RS256) — Indexing API için ----

type serviceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// ServiceAccountToken servis hesabı JSON'undan JWT imzalayıp erişim token'ı alır.
func ServiceAccountToken(ctx context.Context, saJSON []byte, scope string) (string, error) {
	var sa serviceAccountKey
	if err := json.Unmarshal(saJSON, &sa); err != nil {
		return "", err
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return "", errors.New("servis hesabı eksik (client_email/private_key)")
	}
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		// #nosec G101 -- "kimlik bilgisi" değil: Google OAuth2'nin HERKESE
		// AÇIK token uç noktası. Gömülü anahtar/parola yok.
		tokenURI = "https://oauth2.googleapis.com/token"
	}
	now := time.Now()
	claims := fmt.Sprintf(`{"iss":%q,"scope":%q,"aud":%q,"iat":%d,"exp":%d}`,
		sa.ClientEmail, scope, tokenURI, now.Unix(), now.Add(time.Hour).Unix())
	assertion, err := signRS256(sa.PrivateKey, claims)
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(body, &out)
	if out.AccessToken == "" {
		return "", errors.New("access_token dönmedi")
	}
	return out.AccessToken, nil
}

func signRS256(pemKey, claims string) (string, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return "", errors.New("private_key PEM çözümlenemedi")
	}
	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			key = rk
		}
	}
	if key == nil {
		if rk, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			key = rk
		}
	}
	if key == nil {
		return "", errors.New("RSA anahtarı çözümlenemedi")
	}
	b64 := base64.RawURLEncoding
	header := b64.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	signingInput := header + "." + b64.EncodeToString([]byte(claims))
	h := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + b64.EncodeToString(sig), nil
}

// GoogleIndexingPublish Indexing API'ye URL_UPDATED bildirir.
func GoogleIndexingPublish(ctx context.Context, accessToken, urlStr string) (int, error) {
	code, body, err := PostJSON(ctx, "https://indexing.googleapis.com/v3/urlNotifications:publish",
		map[string]any{"url": urlStr, "type": "URL_UPDATED"},
		map[string]string{"Authorization": "Bearer " + accessToken})
	if err != nil {
		return 0, err
	}
	if code >= 300 {
		return code, fmt.Errorf("indexing HTTP %d: %s", code, truncate(string(body), 200))
	}
	return code, nil
}
