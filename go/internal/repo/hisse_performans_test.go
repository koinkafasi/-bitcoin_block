package repo

import (
	"testing"
	"time"
)

// TestHissePerformans yüzdelerin seriden doğru türetildiğini kilitler.
func TestHissePerformans(t *testing.T) {
	simdi := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	gun := 24 * time.Hour
	nokta := func(gunOnce int, fiyat float64) HisseNokta {
		return HisseNokta{T: simdi.Add(-time.Duration(gunOnce) * gun).Unix(), C: fiyat}
	}
	seri := []HisseNokta{
		nokta(400, 100), // 1 yıldan eski — hiçbir döneme girmemeli
		nokta(360, 200), // 1 year
		nokta(20, 400),  // YTD + 1 month
		nokta(4, 500),   // 5 days
		nokta(1, 520),   // 24 hours yerine önceki kapanış kullanılır
		nokta(0, 550),
	}
	per := HissePerformans(seri, 500, simdi)
	if len(per) != 5 {
		t.Fatalf("dönem sayısı %d, beklenen 5", len(per))
	}
	want := map[string]float64{
		"24 hours": 10,   // 550/500-1
		"5 days":   10,   // 550/500-1
		"1 month":  37.5, // 550/400-1
		"YTD":      37.5,
		"1 year":   175, // 550/200-1
	}
	for _, d := range per {
		b, ok := want[d.Ad]
		if !ok {
			t.Errorf("beklenmeyen dönem %q", d.Ad)
			continue
		}
		if !d.Gecerli {
			t.Errorf("%s geçersiz olmamalıydı", d.Ad)
			continue
		}
		if fark := d.Yuzde - b; fark > 0.01 || fark < -0.01 {
			t.Errorf("%s: %.4f, beklenen %.4f", d.Ad, d.Yuzde, b)
		}
	}
}

// TestHissePerformansKisaGecmis yeterli nokta yoksa hücre geçersiz olmalı
// (sıfır yüzde uydurulmamalı).
func TestHissePerformansKisaGecmis(t *testing.T) {
	simdi := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	seri := []HisseNokta{
		{T: simdi.Add(-2 * 24 * time.Hour).Unix(), C: 100},
		{T: simdi.Unix(), C: 110},
	}
	per := HissePerformans(seri, 100, simdi)
	for _, d := range per {
		switch d.Ad {
		case "1 year", "YTD":
			if d.Gecerli {
				t.Errorf("%s geçerli olmamalıydı (seri kısa)", d.Ad)
			}
		}
	}
}

// TestHissePerformansBosSeri boş seride nil dönmeli.
func TestHissePerformansBosSeri(t *testing.T) {
	if got := HissePerformans(nil, 0, time.Now()); got != nil {
		t.Fatalf("boş seride %v döndü", got)
	}
}
