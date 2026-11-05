package platform

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// SecretBox AES-GCM ile gizli değer şifreleme (R2/S3 anahtarları vb.).
// Düz metin asla saklanmaz; sadece "v1:<base64 nonce|ciphertext>" biçimi.
type SecretBox struct {
	aead cipher.AEAD
}

// NewSecretBox anahtardan (en az 16 bayt) kutu üretir; kısa anahtar SHA-256 ile genişletilir.
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) == 0 {
		return nil, errors.New("gizli anahtar boş")
	}
	if len(key) < 32 {
		sum := sha256.Sum256(key)
		key = sum[:]
	}
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead}, nil
}

// Seal düz metni şifreler.
func (b *SecretBox) Seal(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := b.aead.Seal(nonce, nonce, []byte(plain), nil)
	return "v1:" + base64.StdEncoding.EncodeToString(out), nil
}

// Open şifreli değeri çözer.
func (b *SecretBox) Open(sealed string) (string, error) {
	if sealed == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(trimPrefixV1(sealed))
	if err != nil {
		return "", fmt.Errorf("şifreli değer çözülemedi: %w", err)
	}
	ns := b.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("şifreli değer bozuk")
	}
	plain, err := b.aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func trimPrefixV1(s string) string {
	const p = "v1:"
	if len(s) > len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

// RandomHex n baytlık rastgele değeri hex olarak döndürür (gizli anahtar üretimi).
func RandomHex(n int) string {
	return randHex(n)
}
