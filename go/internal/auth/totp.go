package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"strings"
	"time"
)

// TOTP (RFC 6238) — parolasız, kendi başına çalışır (QR sonra).
// Secret: base32 (padding'siz); kod: 6 hane, 30 sn adım, ±1 pencere.

// GenerateSecret 20 baytlık rastgele TOTP secret üretir (base32, padding'siz).
func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := base32.StdEncoding.EncodeToString(b)
	return strings.TrimRight(s, "="), nil
}

// totpCode belirtilen zamanda 6 haneli kod üretir.
func totpCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	// #nosec G115 -- Unix zaman damgası 1970 sonrası için pozitif; uint64'e
	// dönüşüm negatif değer üretmez (TOTP sayacı protokolü uint64 ister).
	counter := uint64(t.Unix()) / 30
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return sprintf6(code % 1000000), nil
}

func sprintf6(v uint32) string {
	s := itoaU(v)
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}

func itoaU(v uint32) string {
	if v == 0 {
		return "0"
	}
	var b [10]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// VerifyTOTP verilen kodu geçerli ±1 zaman penceresinde doğrular.
func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	for i := int64(-1); i <= 1; i++ {
		c, err := totpCode(secret, now.Add(time.Duration(i)*30*time.Second))
		if err == nil && c == code {
			return true
		}
	}
	return false
}
