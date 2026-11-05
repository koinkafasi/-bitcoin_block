package repo

import (
	"strings"
	"testing"
)

func TestComputeSEO(t *testing.T) {
	title := strings.Repeat("x", 50) // 50 → ideal (40-70)
	spot := strings.Repeat("y", 140) // 140 → ideal (120-160)
	word := 450                      // 300-600 ideal
	h2 := 3                          // 2-4 ideal
	cover := true
	imgs := 2
	tags := 4
	bold := 4
	italic := 2

	r := ComputeSEO(title, spot, word, h2, cover, imgs, tags, bold, italic)
	if r.Score < 80 {
		t.Errorf("ideal girdi düşük skor verdi: %d", r.Score)
	}
	if len(r.Parts) != 10 {
		t.Errorf("10 kriter beklenir, %d", len(r.Parts))
	}
}

func TestComputeSEOWeak(t *testing.T) {
	// tüm kriterler kötü (kelime sayısı <4 olduğu için paragraf kriteri de tutmaz) → düşük skor
	r := ComputeSEO("kısa", "", 2, 0, false, 0, 0, 0, 0)
	if r.Score != 5 && r.Score != 0 {
		t.Errorf("zayıf girdi yaklaşık 0 olmalıydı, %d", r.Score)
	}
}
