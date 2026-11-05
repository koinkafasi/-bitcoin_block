package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// koyuTemaMetin: "Görünüm & Marka"da seçilen tek metin rengi iki temaya da
// uygulanıyordu. Kullanıcı siyah seçtiğinde koyu temada metin siyah kalıp
// okunmuyordu (ölçüldü: kontrast 1.2). Koyu tema için ton beyaza doğru
// açılmalı.
func TestKoyuTemaMetin(t *testing.T) {
	kart := [3]float64{39, 39, 42}
	sayfa := [3]float64{19, 19, 22}

	for _, girdi := range []string{"#000000", "#111827", "#1b1a18", "#0b0e14", "#333333"} {
		out := koyuTemaMetin(girdi)
		assert.NotEmptyf(t, out, "koyu tema tonu üretilmeli: %s", girdi)
		r, g, b, ok := hexRGB(out)
		assert.True(t, ok, "üretilen renk çözülemedi: %s", out)

		ok1 := kontrastOrani(r, g, b, kart[0], kart[1], kart[2])
		ok2 := kontrastOrani(r, g, b, sayfa[0], sayfa[1], sayfa[2])
		assert.GreaterOrEqualf(t, ok1, wcagNormalText,
			"%s -> %s kartta %.2f (eşik 4.5)", girdi, out, ok1)
		assert.GreaterOrEqualf(t, ok2, wcagNormalText,
			"%s -> %s sayfada %.2f (eşik 4.5)", girdi, out, ok2)
	}
}

// Zaten açık renkler değişmemeli (gereksiz beyazlatma görünümü bozar).
func TestKoyuTemaMetinZatenAcik(t *testing.T) {
	assert.Equal(t, "#ffffff", koyuTemaMetin("#ffffff"))
	assert.Equal(t, "#e6e9ef", koyuTemaMetin("#e6e9ef"))
}

// accentTextKoyu, koyuTemaMetin'e delege etmeli.
func TestAccentTextKoyuDelege(t *testing.T) {
	for _, h := range []string{"#e11d48", "#fa0000", "#16a34a", "#7c3aed"} {
		assert.Equal(t, koyuTemaMetin(h), accentTextKoyu(h))
		r, g, b, _ := hexRGB(accentTextKoyu(h))
		oran := kontrastOrani(r, g, b, 24, 24, 27)
		assert.GreaterOrEqualf(t, oran, wcagNormalText,
			"%s -> %s kartta %.2f", h, accentTextKoyu(h), oran)
	}
}
