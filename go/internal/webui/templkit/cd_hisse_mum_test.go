package templkit

import (
	"math"
	"testing"
)

// TestCDMumlar Geometri: fitil uçları, gövde sınırları ve viewBox içinde kalma.
func TestCDMumlar(t *testing.T) {
	ac := []float64{10, 12, 9}
	yu := []float64{13, 14, 11}
	du := []float64{9, 11, 8}
	ka := []float64{12, 11, 10}
	m := CDMumlar(ac, yu, du, ka)
	if len(m) != 3 {
		t.Fatalf("mum sayısı %d, beklenen 3", len(m))
	}
	for i, x := range m {
		if x.YUst > x.YAlt {
			t.Errorf("%d. mumda fitil ters: üst=%.1f alt=%.1f", i, x.YUst, x.YAlt)
		}
		if x.YGovdeUst > x.YGovdeAlt {
			t.Errorf("%d. mumda gövde ters", i)
		}
		if x.YUst < 0 || x.YAlt > GrafikH {
			t.Errorf("%d. mum viewBox dışında: %.1f–%.1f", i, x.YUst, x.YAlt)
		}
		if x.Genislik < 1 {
			t.Errorf("%d. mum genişliği çok küçük: %.2f", i, x.Genislik)
		}
	}
	// Yükselen mum (kapanış > açılış) Up olmalı.
	if !m[0].Up {
		t.Errorf("10→12 yükseliş, Up beklenirdi")
	}
	if m[1].Up {
		t.Errorf("12→11 düşüş, Down beklenirdi")
	}
	// Fitil gövdeyi KAPSAMALI (yüksek ≥ gövde üstü).
	if m[0].YUst > m[0].YGovdeUst+0.01 {
		t.Errorf("fitil gövdenin üstünde değil")
	}
	if math.Abs(m[0].YAlt-m[0].YGovdeAlt) > 0.01 && m[0].YAlt < m[0].YGovdeAlt {
		t.Errorf("fitil gövdenin altını kapsamıyor")
	}
}

// TestCDMumlarTekNokta 2'den az noktada mum üretilmez (grafik zaten yok).
func TestCDMumlarTekNokta(t *testing.T) {
	if got := CDMumlar([]float64{1}, []float64{1}, []float64{1}, []float64{1}); got != nil {
		t.Fatalf("tek noktada %d mum döndü", len(got))
	}
}

// TestCDMumlarBosDeger sıfır kapanışlı barlar atlanmalı (borsa kapalı).
func TestCDMumlarBosDeger(t *testing.T) {
	ac := []float64{10, 0, 11}
	yu := []float64{12, 0, 13}
	du := []float64{9, 0, 10}
	ka := []float64{11, 0, 12}
	m := CDMumlar(ac, yu, du, ka)
	if len(m) != 2 {
		t.Fatalf("mum sayısı %d, beklenen 2", len(m))
	}
}
