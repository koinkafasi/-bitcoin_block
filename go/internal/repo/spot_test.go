package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpotEsitBaslik(t *testing.T) {
	vakalar := []struct {
		spot, baslik string
		esit         bool
		neden        string
	}{
		{
			"Federal Reserve Board issues enforcement actions with former employee",
			"Federal Reserve Board issues enforcement actions with former employee",
			true, "birebir aynı",
		},
		{
			"federal reserve board issues ENFORCEMENT actions",
			"Federal Reserve Board Issues Enforcement Actions",
			true, "harf büyüklüğü farkı",
		},
		{
			"Federal Reserve Board, issues enforcement actions!",
			"Federal Reserve Board issues enforcement actions",
			true, "noktalama farkı",
		},
		{
			"Bitcoin yükseldi",
			"Bitcoin yükseldi ve piyasa hareketlendi, analistler değerlendirdi",
			false, "kısa spot önek olsa da tekrar sayılmaz (40 rune eşiği)",
		},
		{
			"Bitcoin 80 bin doların üzerine çıktı ve piyasa hareketlendi",
			"Bitcoin 80 bin doların üzerine çıktı ve piyasa hareketlendi, analistler yorumladı",
			true, "uzun spot başlığın öneki",
		},
		{
			"Tamamen farklı bir spot metni burada",
			"Başka bir başlık",
			false, "farklı metinler",
		},
		{"", "Başlık", false, "boş spot tekrar değil"},
	}
	for _, v := range vakalar {
		assert.Equal(t, v.esit, spotEsitBaslik(v.spot, v.baslik), v.neden)
	}
}

// TestKaynakGorselHashKarsilastirma: aynı adres aynı anahtar, farklı adres
// farklı anahtar. CDN önbelleği buna dayanıyor (immutable).
func TestKaynakGorselHashKarsilastirma(t *testing.T) {
	a := KaynakGorselHash("https://cdn.example.com/a.jpg")
	b := KaynakGorselHash("https://cdn.example.com/a.jpg")
	c := KaynakGorselHash("https://cdn.example.com/b.jpg")

	assert.Equal(t, a, b, "aynı adres aynı hash")
	assert.NotEqual(t, a, c, "farklı adres farklı hash")
	assert.Len(t, a, 32, "32 hex karakter")
	assert.True(t, yalnizHexTest(a), "yalnız hex")
}

// yalnizHexTest test yardımcısı (httpapi'deki fonksiyonun eşi).
func yalnizHexTest(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
