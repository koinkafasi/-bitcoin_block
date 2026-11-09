package repo

import (
	"testing"
	"time"
)

// TestTarihTRAyAdlari GERÇEK ay adını yazar.
//
// BU TESTİN YAKALADIĞI HATA (ölçüldü, 9 Kasım 2026): biçim
// `day.Format("2 Ocak 2006")` idi; Go'da ay belirteci "Jan"/"January"
// olduğu için "Ocak" DÜZ METİN olarak basılıyordu ve Kasım'da yayınlanan
// içerik "6 Ocak 2026" görünüyordu. Ay adı eşlemesi bozulursa test düşer.
func TestTarihTRAyAdlari(t *testing.T) {
	for ay := time.January; ay <= time.December; ay++ {
		tarih := time.Date(2026, ay, 15, 0, 0, 0, 0, time.UTC)
		got := TarihTRUzun(tarih)
		beklenen := "15 " + AyAdlariTR[int(ay)-1] + " 2026"
		if got != beklenen {
			t.Errorf("%s → %q, beklenen %q", ay, got, beklenen)
		}
		// AY ADI GERÇEKTEN DEĞİŞMELİ: eski hata tüm aylarda "Ocak" basıyordu.
		if ay != time.January && got == TarihTRUzun(time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("%s ayında da Ocak yazıyor", ay)
		}
	}
}

// TestTarihTRKasim regresyonun tam yaşandığı ay (Kasım).
func TestTarihTRKasim(t *testing.T) {
	kasim := time.Date(2026, time.November, 6, 20, 40, 0, 0, time.UTC)
	if got := TarihTRUzun(kasim); got != "6 Kasım 2026" {
		t.Errorf("TarihTRUzun = %q, beklenen \"6 Kasım 2026\"", got)
	}
	if got := TarihTRSaat(kasim); got != "6 Kasım 2026 20:40 UTC" {
		t.Errorf("TarihTRSaat = %q", got)
	}
	// Saat iki haneli (tek haneli saat "9:05" değil "09:05").
	tek := time.Date(2026, time.May, 7, 9, 5, 0, 0, time.UTC)
	if got := TarihTRSaat(tek); got != "7 Mayıs 2026 09:05 UTC" {
		t.Errorf("tek haneli saat = %q", got)
	}
}

// TestAyTRAralik ay aralığı dışında boş döner (panik yok).
func TestAyTRAralik(t *testing.T) {
	if got := AyTR(time.Month(0)); got != "" {
		t.Errorf("ay 0 → %q", got)
	}
	if got := AyTR(time.Month(13)); got != "" {
		t.Errorf("ay 13 → %q", got)
	}
}
