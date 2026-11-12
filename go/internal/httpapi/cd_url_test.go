package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

func TestMakaleURLTarihliDesen(t *testing.T) {
	yayin := time.Date(2026, 11, 5, 9, 14, 0, 0, time.UTC)
	slug := "bitcoin-80-bin-dolara-yukseldi"
	kat := "markets"
	c := repo.Content{ID: 42, Slug: &slug, Locale: "tr", PublishedAt: &yayin, CategorySlug: &kat}

	got := makaleURL(c)
	assert.Equal(t, "/markets/2026/11/05/bitcoin-80-bin-dolara-yukseldi", got)
}

func TestMakaleURLBolumsuz(t *testing.T) {
	yayin := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)
	slug := "test-haberi"
	c := repo.Content{ID: 7, Slug: &slug, Locale: "tr", PublishedAt: &yayin}
	assert.Equal(t, "/2026/01/02/test-haberi", makaleURL(c))
}

func TestMakaleURLTarihYoksaEskiBicim(t *testing.T) {
	slug := "taslak-haber"
	c := repo.Content{ID: 9, Slug: &slug, Locale: "tr"}
	// Yayın tarihi yok: tarihli adres üretilemez. Kırık bağlantı yerine
	// eski (çalışan) biçime düşer.
	assert.Equal(t, "/taslak-haber-9", makaleURL(c))
	assert.Equal(t, "/taslak-haber-9", publicArticleURL("tr", slug, 9))
}

func TestMakaleURLIngilizce(t *testing.T) {
	yayin := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	slug := "bitcoin-hits-80k"
	kat := "markets"
	c := repo.Content{ID: 1, Slug: &slug, Locale: "en", PublishedAt: &yayin, CategorySlug: &kat}
	assert.Equal(t, "/en/markets/2026/11/05/bitcoin-hits-80k", makaleURL(c))
}

// TestMakaleTarihDeseni: desen çözümleme. Rastgele yollar makale sanılmamalı.
func TestMakaleTarihDeseni(t *testing.T) {
	vakalar := []struct {
		yol   []string
		slug  string
		okMu  bool
		neden string
	}{
		{[]string{"markets", "2026", "11", "05", "haber"}, "haber", true, "tam desen"},
		{[]string{"2026", "11", "05", "haber"}, "haber", true, "bölümsüz"},
		{[]string{"kategori", "haber"}, "", false, "tarih yok"},
		{[]string{"markets", "2026", "13", "05", "haber"}, "", false, "ay 13 olamaz"},
		{[]string{"markets", "2026", "11", "32", "haber"}, "", false, "gün 32 olamaz"},
		{[]string{"markets", "1999", "11", "05", "haber"}, "", false, "yıl aralığı dışı"},
		{[]string{"markets", "26", "11", "05", "haber"}, "", false, "yıl 4 hane değil"},
		{[]string{"markets", "2026", "11", "05", ""}, "", false, "slug boş"},
	}
	for _, v := range vakalar {
		slug, ok := makaleTarihDeseni(v.yol)
		assert.Equal(t, v.okMu, ok, "%s: %v", v.neden, v.yol)
		if v.okMu {
			assert.Equal(t, v.slug, slug, v.neden)
		}
	}
}
