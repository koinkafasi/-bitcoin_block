package repo

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Triyaj sıralaması (U1).
//
// Durum tespiti (28 Ekim 2026): 39.698 içeriğin 39.206'sı taslak. Admin
// içerik listesi `ORDER BY c.updated_at DESC` ile sabitti; editör havuzun
// en üstünde her zaman "en son bot kaydı"nı görüyordu. Hangi taslağın
// yayına değdiğini anlamanın yolu yoktu.
//
// Çözüm ayrı bir ekran DEĞİL — mevcut liste genişletildi. İkinci bir içerik
// listesi, zaten şikâyet konusu olan "aynı işi yapan iki alan" sorununu
// büyütürdü.
//
// Sıralama ifadesi SQL'e doğrudan giriyor; bu yüzden **beyaz liste**
// zorunlu. Bilinmeyen değer varsayılana düşer, asla sorguya yazılmaz.

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

// Beyaz listedeki hiçbir ifade kullanıcı girdisi taşımamalı: tamamı sabit.
// takmaAdRe ifadedeki tablo referanslarını yakalar (ör. "c.title").
// EXTRACT(... ) gibi fonksiyon çağrıları nokta içermediği için eşleşmez.
var takmaAdRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.`)

func TestOrderByIfadeleriSabitVeZararsiz(t *testing.T) {
	for key := range contentOrderOptions {
		expr := contentOrderOptions[key]
		assert.NotContains(t, expr, ";", "sıralama ifadesinde noktalı virgül olamaz: %q", key)
		assert.NotContains(t, expr, "--", "sıralama ifadesinde yorum olamaz: %q", key)
		assert.NotContains(t, expr, "/*", "sıralama ifadesinde blok yorum olamaz: %q", key)

		// Asıl kural: ifade YALNIZ içerik tablosuna dokunmalı.
		//
		// Önce "c. ile başlamalı" diye bakılıyordu; bu ucuz bir vekildi ve
		// yalnız ilk referansı denetliyordu. Öncelik ifadesi parantezle
		// başladığı için takıldı. Artık ifadedeki HER tablo referansı
		// denetleniyor — ilkini değil hepsini kapsadığı için bu daha güçlü.
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
