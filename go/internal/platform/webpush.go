package platform

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---- Web Push (VAPID / RFC 8291) — P1-8 ----

// VAPIDKeys base64url (raw) kodlu VAPID anahtar çifti.
type VAPIDKeys struct {
	Public  string
	Private string
}

// GenerateVAPIDKeys P-256 VAPID anahtar çifti üretir (public=65B, private=32B, base64url).
func GenerateVAPIDKeys() (VAPIDKeys, error) {
	d, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return VAPIDKeys{}, err
	}
	pub := elliptic.Marshal(elliptic.P256(), d.X, d.Y)
	return VAPIDKeys{
		Public:  base64.RawURLEncoding.EncodeToString(pub),
		Private: base64.RawURLEncoding.EncodeToString(pad32(d.D.Bytes())),
	}, nil
}

func pad32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func unb64(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	return base64.RawURLEncoding.DecodeString(s)
}

// PushSubscription tarayıcı abonelik bilgisi.
type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// SendWebPush tek bir aboneye şifreli web push gönderir; HTTP durum kodunu döndürür.
func SendWebPush(ctx context.Context, sub PushSubscription, payload []byte, keys VAPIDKeys, subject string) (int, []byte, error) {
	u, err := url.Parse(sub.Endpoint)
	if err != nil {
		return 0, nil, err
	}
	audience := u.Scheme + "://" + u.Host
	jwt, err := vapidJWT(audience, subject, keys.Private)
	if err != nil {
		return 0, nil, err
	}
	body, err := encryptWebPush(sub.P256dh, sub.Auth, payload)
	if err != nil {
		return 0, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, strings.NewReader(string(body)))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", "86400")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+keys.Public)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, rb, nil
}

// vapidJWT ES256 imzalı VAPID JWT üretir.
func vapidJWT(audience, subject, privateB64 string) (string, error) {
	raw, err := unb64(privateB64)
	if err != nil || len(raw) != 32 {
		return "", errors.New("geçersiz VAPID özel anahtarı")
	}
	curve := elliptic.P256()
	priv := new(ecdsa.PrivateKey)
	priv.Curve = curve
	priv.D = new(big.Int).SetBytes(raw)
	priv.X, priv.Y = curve.ScalarBaseMult(raw)

	header := `{"typ":"JWT","alg":"ES256"}`
	claims, _ := json.Marshal(map[string]any{
		"aud": audience,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": subject,
	})
	b64 := base64.RawURLEncoding
	signingInput := b64.EncodeToString([]byte(header)) + "." + b64.EncodeToString(claims)
	h := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
	if err != nil {
		return "", err
	}
	sig := append(pad32(r.Bytes()), pad32(s.Bytes())...)
	return signingInput + "." + b64.EncodeToString(sig), nil
}

// encryptWebPush RFC 8291 (aes128gcm) gövdesini üretir.
func encryptWebPush(p256dhB64, authB64 string, payload []byte) ([]byte, error) {
	uaPubBytes, err := unb64(p256dhB64)
	if err != nil {
		return nil, fmt.Errorf("p256dh çözümlenemedi: %w", err)
	}
	uaPub, err := ecdh.P256().NewPublicKey(uaPubBytes)
	if err != nil {
		return nil, fmt.Errorf("p256dh geçersiz: %w", err)
	}
	auth, err := unb64(authB64)
	if err != nil {
		return nil, fmt.Errorf("auth çözümlenemedi: %w", err)
	}
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := as.ECDH(uaPub)
	if err != nil {
		return nil, err
	}
	asPub := as.PublicKey().Bytes() // 65

	// IKM = HKDF-Expand(HKDF-Extract(auth, shared), "WebPush: info\0" || ua_pub || as_pub, 32)
	ikmInfo := append([]byte("WebPush: info\x00"), uaPubBytes...)
	ikmInfo = append(ikmInfo, asPub...)
	prkKey := hkdfExtract(auth, shared)
	ikm := hkdfExpand(prkKey, ikmInfo, 32)

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	prk := hkdfExtract(salt, ikm)
	cek := hkdfExpand(prk, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce := hkdfExpand(prk, []byte("Content-Encoding: nonce\x00"), 12)

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt := append(append([]byte{}, payload...), 0x02) // son kayıt ayırıcısı
	ct := gcm.Seal(nil, nonce, pt, nil)

	var out []byte
	out = append(out, salt...)
	var rs [4]byte
	binary.BigEndian.PutUint32(rs[:], 4096)
	out = append(out, rs[:]...)
	// #nosec G115 -- Web Push protokolü: auth secret uzunluğu 16 bayttır
	// (RFC 8291). Tek baytlık alan bu sabit değeri taşır.
	out = append(out, byte(len(asPub)))
	out = append(out, asPub...)
	out = append(out, ct...)
	return out, nil
}

func hkdfExtract(salt, ikm []byte) []byte {
	if len(salt) == 0 {
		salt = make([]byte, sha256.Size)
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write(ikm)
	return mac.Sum(nil)
}

func hkdfExpand(prk, info []byte, length int) []byte {
	var out []byte
	var t []byte
	for i := byte(1); len(out) < length; i++ {
		mac := hmac.New(sha256.New, prk)
		mac.Write(t)
		mac.Write(info)
		mac.Write([]byte{i})
		t = mac.Sum(nil)
		out = append(out, t...)
	}
	return out[:length]
}
