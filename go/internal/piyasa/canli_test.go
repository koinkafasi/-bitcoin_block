package piyasa

import (
	"context"
	"testing"
	"time"
)

// Canlı doğrulama: gerçek kaynaklardan gelen değerleri basar.
// `go test -run TestCanli -v` ile elle çalıştırılır; CI'da ağ olmayabilir.
func TestCanliPiyasa(t *testing.T) {
	if testing.Short() {
		t.Skip("canlı ağ testi")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	s, err := Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	t.Logf("kaynak=%s kalem=%d güncel=%s", s.Kaynak, len(s.Kalemler), s.Guncel.Format("15:04:05"))
	for _, k := range s.Kalemler {
		t.Logf("  %-14s %-14s %14s  %s%%  yon=%d",
			k.Anahtar, k.Kisa, FormatTR(k.Deger, k.Ondalik), FormatDegisim(k.Degisim), k.Yon)
	}
}
