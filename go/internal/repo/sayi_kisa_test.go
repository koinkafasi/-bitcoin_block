package repo

import "testing"

func TestSayiBicimKisa(t *testing.T) {
	for _, k := range []struct {
		deger    float64
		para     string
		beklenen string
	}{
		{1_635_401_465_856, "USD", "$1.64T"},
		{423_349_420_032, "TRY", "₺423.35B"},
		{5_065_410_084_864, "USD", "$5.07T"},
		{2_500_000_000, "USD", "$2.50B"},
		{3_400_000, "USD", "$3.40M"},
		{5_600, "USD", "$5.60K"},
		{845.5, "USD", "$845.50"},
		{120, "EUR", "€120.00"},
	} {
		if got := SayiBicimKisa(k.deger, k.para); got != k.beklenen {
			t.Errorf("%.0f %s → %q, beklenen %q", k.deger, k.para, got, k.beklenen)
		}
	}
	// Sıfır/negatif: değer yok ("—") — uydurma sayı basılmaz.
	for _, d := range []float64{0, -1} {
		if got := SayiBicimKisa(d, "USD"); got != "—" {
			t.Errorf("%.0f → %q, beklenen —", d, got)
		}
	}
}

// TestSayiBicimKisaUzunluk: 326 px raya sığacak kadar kısa olmalı.
func TestSayiBicimKisaUzunluk(t *testing.T) {
	enUzun := SayiBicimKisa(9_999_999_999_999_999, "TRY")
	if len([]rune(enUzun)) > 12 {
		t.Errorf("kısaltma çok uzun (%d karakter): %q", len([]rune(enUzun)), enUzun)
	}
}
