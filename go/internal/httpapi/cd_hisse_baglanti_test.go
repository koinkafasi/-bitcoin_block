package httpapi

import "testing"

func TestHisseBorsaKodu(t *testing.T) {
	for giris, beklenen := range map[string]string{
		"THYAO":    "THYAO.IS",
		"thyao":    "THYAO.IS",
		" THYAO ":  "THYAO.IS",
		"THYAO.IS": "THYAO.IS",
		"XU100":    "XU100.IS",
		"":         "",
	} {
		if got := hisseBorsaKodu(giris); got != beklenen {
			t.Errorf("%q → %q, beklenen %q", giris, got, beklenen)
		}
	}
}
