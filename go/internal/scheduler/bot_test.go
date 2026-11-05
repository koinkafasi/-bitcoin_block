package scheduler

import (
	"testing"
	"time"
)

// E-10'un ilk adımı: arka plan işlerinde test tabanı yoktu (1.224 satır,
// 0 test). Bu dosya bot poller'ının en riskli kararını — ölü feed'e ne kadar
// ara verileceğini — saf fonksiyon olarak korur.

func TestBackoffDelayIlkHatadaEnAzOnBesDakika(t *testing.T) {
	// Panelde 5 dakika ayarlıysa bile ölü feed 5 dakikada bir denenmez.
	if got := backoffDelay(5*time.Minute, 1); got != 15*time.Minute {
		t.Fatalf("beklenen 15m, alınan %s", got)
	}
}

func TestBackoffDelayBasarisizliktaIkiyeKatlanir(t *testing.T) {
	base := 15 * time.Minute
	want := []time.Duration{15 * time.Minute, 30 * time.Minute, 60 * time.Minute, 120 * time.Minute}
	for i, w := range want {
		if got := backoffDelay(base, i+1); got != w {
			t.Fatalf("fails=%d için beklenen %s, alınan %s", i+1, w, got)
		}
	}
}

func TestBackoffDelayAltiSaatteDurur(t *testing.T) {
	if got := backoffDelay(15*time.Minute, 20); got != 6*time.Hour {
		t.Fatalf("üst sınır 6 saat olmalı, alınan %s", got)
	}
}

func TestBackoffDelayTabanAralikBuyukseKorunur(t *testing.T) {
	if got := backoffDelay(45*time.Minute, 1); got != 45*time.Minute {
		t.Fatalf("taban aralık korunmalı, alınan %s", got)
	}
}

func TestBackoffDelaySifirHataTabaniDondurur(t *testing.T) {
	if got := backoffDelay(7*time.Minute, 0); got != 7*time.Minute {
		t.Fatalf("hata yoksa taban dönmeli, alınan %s", got)
	}
}
