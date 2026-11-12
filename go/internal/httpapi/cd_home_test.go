package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrGoreliZaman(t *testing.T) {
	simdi := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	vakalar := []struct {
		gecen time.Duration
		bekle string
	}{
		{10 * time.Second, "az önce"},
		{59 * time.Second, "az önce"},
		{1 * time.Minute, "1 dakika önce"},
		{59 * time.Minute, "59 dakika önce"},
		{60 * time.Minute, "1 saat önce"},
		{23*time.Hour + 59*time.Minute, "23 saat önce"},
		{24 * time.Hour, "1 gün önce"},
		{72 * time.Hour, "3 gün önce"},
	}
	for _, v := range vakalar {
		got := trGoreliZaman(simdi.Add(-v.gecen), simdi)
		assert.Equal(t, v.bekle, got, "geçen: %s", v.gecen)
	}
}

// TestTrGoreliZamanGelecek: sunucu saati kaynağınkinden gerideyse negatif
// süre çıkar; "az önce" dönmeli ( "-3 dakika önce" değil).
func TestTrGoreliZamanGelecek(t *testing.T) {
	simdi := time.Now()
	assert.Equal(t, "az önce", trGoreliZaman(simdi.Add(5*time.Minute), simdi))
}
