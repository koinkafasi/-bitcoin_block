package httpapi

import "testing"

// TestHisseBorsaKodu BIST kodunun Yahoo sembolüne çevrilmesini kilitler.
//
// NEDEN: `BorsaRow.Code` ".IS" EKİ OLMADAN saklanıyor (şeritte "THYAO"
// görünsün diye). Detay ucu ".IS" bekliyor. Bu dönüşüm unutulunca
// /price sayfasındaki 9 hisse bağlantısının 9'u da 404 verdi
// (ölçüldü, 9 Kasım 2026) — regresyon testi bunu yakalar.
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
