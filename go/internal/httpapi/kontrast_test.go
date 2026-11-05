package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Kontrast (Faz 180, A-7).
//
// NEDEN: marka rengi kullanıcı seçimi (#fa0000). Üstüne beyaz yazı 4.15
// veriyordu; AA küçük metin eşiği 4.5. Marka rengini koyulaştırmak yanlış
// olurdu (başlık, bağlantı, grafik aynı tonu paylaşıyor). Bunun yerine
// yalnızca "üstünde beyaz yazı taşıyan dolgu" için türetilmiş ton üretiliyor.

func TestAccentFillEsigiGecer(t *testing.T) {
	// Gerçek marka rengi ve birkaç uç durum.
	for _, accent := range []string{
		"#fa0000", // gerçek marka (varsayılan)
		"#e11d48", // eski pembe-kırmızı
		"#d9532a", // pano sinyal turuncusu
		"#111111", // koyu marka: beyaz zaten geçer
		"#0b5fff", // mavi
		"#16a34a", // yeşil
		"#7c3aed", // mor
	} {
		fill := accentFill(accent)
		assert.NotEmpty(t, fill, "accentFill boş dönmemeli: %s", accent)

		r, g, b, ok := hexRGB(fill)
		assert.True(t, ok, "üretilen renk çözülemedi: %s", fill)
		oran := kontrastOrani(r, g, b, 255, 255, 255)
		assert.GreaterOrEqualf(t, oran, wcagNormalText,
			"%s için üretilen dolgu %s beyazla %.2f veriyor (eşik %.1f)",
			accent, fill, oran, wcagNormalText)
	}
}

// TestAccentFillMarkayaYakin: türetme, eşiği geçen EN AÇIK tonu seçmeli —
// yani markaya mümkün olduğunca yakın. Gereğinden koyu üretmek görünümü
// bozar.
func TestAccentFillMarkayaYakin(t *testing.T) {
	fill := accentFill("#fa0000")
	// Ölçüldü: #ef0000 (4.5% koyu). Daha koyu üretilirse marka gözle kayar.
	r, _, _, _ := hexRGB(fill)
	or, _, _, _ := hexRGB("#fa0000")
	assert.Greaterf(t, r, or*0.90,
		"dolgu markadan fazla uzaklaşmış: %s (marka kırmızı kanalı %.0f)", fill, or)

	rr, gg, bb, _ := hexRGB(fill)
	oran := kontrastOrani(rr, gg, bb, 255, 255, 255)
	assert.Lessf(t, oran, 4.9,
		"eşiğin çok üstünde (%.2f) — gereğinden koyu seçilmiş olabilir", oran)
}

// TestAccentFillZatenGecerseAynen: beyazla 4.5'i geçen marka rengi
// değiştirilmemeli.
func TestAccentFillZatenGecerseAynen(t *testing.T) {
	assert.Equal(t, "#111111", accentFill("#111111"))
	assert.Equal(t, "#000000", accentFill("#000000"))
	assert.Equal(t, "#7c3aed", accentFill("#7c3aed"))
}

func TestAccentFillGecersiz(t *testing.T) {
	for _, girdi := range []string{"", "#12", "#zzzzzz", "rgb(1,2,3)"} {
		assert.Emptyf(t, accentFill(girdi), "geçersiz girdi boş dönmeli: %q", girdi)
	}
	// Kısa yazım (#rgb) desteklenir; üstüne beyaz yazı geçmediği için
	// eşiği geçen tona indirilir.
	assert.Equal(t, "#ee0000", accentFill("#f00"))
}

// TestKontrastAraclari: yardımcıların kendisi doğru mu (bilinen değerlerle).
func TestKontrastAraclari(t *testing.T) {
	// Siyah/beyaz = 21:1 (WCAG tanımı).
	assert.InDelta(t, 21.0, kontrastOrani(0, 0, 0, 255, 255, 255), 0.1)
	// Aynı renk = 1:1.
	assert.InDelta(t, 1.0, kontrastOrani(120, 120, 120, 120, 120, 120), 0.001)
	// Simetrik.
	a := kontrastOrani(250, 0, 0, 255, 255, 255)
	b := kontrastOrani(255, 255, 255, 250, 0, 0)
	assert.InDelta(t, a, b, 1e-9)
}
