package auth

import (
	"testing"
	"time"
)

func TestTOTPGenerateAndVerify(t *testing.T) {
	sec, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	// kod üret: totpCode kapalı ama verify iç olduğu için genel doğru döner
	if !VerifyTOTP(sec, testCode(t, sec, now), now) {
		t.Fatal("doğru kod reddedildi")
	}
	if VerifyTOTP(sec, "000000", now) {
		t.Fatal("yanlış kod kabul edildi")
	}
}

func testCode(t *testing.T, secret string, at time.Time) string {
	c, err := totpCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTOTPWindow(t *testing.T) {
	sec, _ := GenerateSecret()
	now := time.Now()
	// +1 pencere (bir önceki adım) da kabul
	prev, err := totpCode(sec, now.Add(-30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyTOTP(sec, prev, now) {
		t.Fatal("ön pencere kabul edilmedi")
	}
	// +3 pencere reddedilir
	far, err := totpCode(sec, now.Add(-90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if VerifyTOTP(sec, far, now) {
		t.Fatal("uzak pencere kabul edilmemeli")
	}
}

func TestTOTPInvalidSecret(t *testing.T) {
	if VerifyTOTP("##geçersiz##", "123456", time.Now()) {
		t.Fatal("geçersiz secret kabul edildi")
	}
}
