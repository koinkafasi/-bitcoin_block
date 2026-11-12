package repo

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderByGuvenliSecenekler(t *testing.T) {
	cases := map[string]string{
		"":       "c.updated_at DESC",
		"guncel": "c.updated_at DESC",
		"skor":   "c.ai_score DESC NULLS LAST, c.updated_at DESC",
		"eski":   "c.updated_at ASC",
		"baslik": "c.title ASC",
	}
	for in, want := range cases {
		assert.Equal(t, want, contentOrderBy(in), "girdi: %q", in)
	}
}

func TestOrderByBilinmeyenDegerVarsayilanaDuser(t *testing.T) {
	for _, in := range []string{
		"bilinmeyen",
		"c.title; DROP TABLE contents",
		"1=1",
		"skor; DELETE FROM contents",
		"ai_score DESC--",
		"SKOR", // büyük harf de beyaz listede değil
		" skor",
	} {
		got := contentOrderBy(in)
		assert.Equal(t, "c.updated_at DESC", got, "girdi: %q varsayılana düşmeliydi", in)
	}
}

var takmaAdRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.`)

func TestOrderByIfadeleriSabitVeZararsiz(t *testing.T) {
	for key := range contentOrderOptions {
		expr := contentOrderOptions[key]
		assert.NotContains(t, expr, ";", "sıralama ifadesinde noktalı virgül olamaz: %q", key)
		assert.NotContains(t, expr, "--", "sıralama ifadesinde yorum olamaz: %q", key)
		assert.NotContains(t, expr, "/*", "sıralama ifadesinde blok yorum olamaz: %q", key)

		adlar := takmaAdRe.FindAllStringSubmatch(expr, -1)
		assert.NotEmpty(t, adlar, "ifade hiçbir sütuna dokunmuyor: %q -> %q", key, expr)
		for _, m := range adlar {
			assert.Equal(t, "c", m[1],
				"ifade içerik tablosu dışına çıkıyor: %q -> %q içinde %q", key, expr, m[1])
		}
	}
}

// Skor eşiği: 0 ve negatif değer filtre uygulamaz (editör eşiği kapatmış
// demektir); 1..100 arası uygulanır, 100 üstü 100'e sabitlenir.
func TestNormalizeMinScore(t *testing.T) {
	cases := map[int]int{
		-5:  0,
		0:   0,
		1:   1,
		55:  55,
		100: 100,
		101: 100,
		999: 100,
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeMinScore(in), "girdi: %d", in)
	}
}
