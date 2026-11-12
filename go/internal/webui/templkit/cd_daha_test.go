package templkit

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// TestBolumParca: parça yalnız satır + nav basar, sayfa iskeleti YOK.
func TestBolumParca(t *testing.T) {
	s := CDStory{Title: "Başlık", Href: "/h/1", Image: "/kg/" + strings.Repeat("a", 32),
		Time: "1 saat önce", Spot: "Spot"}
	var sb strings.Builder
	require.NoError(t, CDBolumParca([]CDStory{s, s}, "/business", "/business?s=2").
		Render(context.Background(), &sb))
	h := sb.String()

	assert.NotContains(t, h, "<!doctype", "parça tam sayfa olmamalı")
	assert.NotContains(t, h, "<html", "parça tam sayfa olmamalı")
	assert.NotContains(t, h, "cd-topbar", "parça üst çubuk içermemeli")
	assert.NotContains(t, h, "cd-kat-more", "parça 'More From' başlığı içermemeli")

	// İki satır, her birinden ÖNCE ayırıcı (ekleme sırasında sıra korunur).
	assert.Equal(t, 2, strings.Count(h, `class="cd-kat-satir"`), "satır sayısı")
	assert.Equal(t, 2, strings.Count(h, "cd-kat-satir__cizgi"),
		"her satırdan önce ayırıcı olmalı (parça sona eklenir)")

	// Nav: prev + next, rel nitelikleri SEO için yaşar.
	require.Contains(t, h, `<nav class="cd-daha"`)
	assert.Contains(t, h, `href="/business" rel="prev"`)
	assert.Contains(t, h, `href="/business?s=2" rel="next"`)
	assert.Contains(t, h, "Daha fazla haber")
}

// TestBolumParcaSonSayfa: next yokken buton basılmaz (yalnız prev).
func TestBolumParcaSonSayfa(t *testing.T) {
	var sb strings.Builder
	require.NoError(t, CDBolumParca(nil, "/business?s=2", "").Render(context.Background(), &sb))
	h := sb.String()
	assert.NotContains(t, h, "cd-daha__btn", "son sayfada buton olmamalı")
	assert.Contains(t, h, "cd-daha__geri", "prev kalmalı (geri dönüş)")
}

// TestTagParca: etiket parçası etiket satır stiliyle (cdTagSatir).
func TestTagParca(t *testing.T) {
	s := CDStory{Title: "Başlık", Href: "/h/1", Image: "/kg/" + strings.Repeat("b", 32),
		Time: "2 saat önce"}
	var sb strings.Builder
	require.NoError(t, CDTagParca([]CDStory{s}, "", "/tag/x?s=2").Render(context.Background(), &sb))
	h := sb.String()

	assert.NotContains(t, h, "<!doctype")
	assert.Equal(t, 1, strings.Count(h, `class="cd-tag-satir"`), "etiket satır stili")
	assert.Equal(t, 1, strings.Count(h, "cd-tag-satir__cizgi"))
	assert.Contains(t, h, `href="/tag/x?s=2" rel="next"`)
	// Etiket parçası bölüm satırı BİÇİMEZ (stil ailesi karışmasın).
	assert.NotContains(t, h, `class="cd-kat-satir"`)
}

// TestDahaScripti: sayfa yalnız NextHref varken cd-daha.js yükler;
// JS yoksa bağlantı normal sayfa geçişi yapar (fallback yaşar).
func TestDahaScripti(t *testing.T) {
	v := ornekKategori()
	v.NextHref = "/business?s=2"
	h := kategoriCiz(t, v)
	assert.Contains(t, h, `/static/cd-daha.js?v=`, "script etiketi yok")
	assert.Contains(t, h, `rel="next"`, "JS'siz fallback için bağlantı kalmalı")

	v2 := ornekKategori()
	h2 := kategoriCiz(t, v2)
	assert.NotContains(t, h2, `/static/cd-daha.js`,
		"NextHref yokken script yüklenmemeli")
}

// TestDahaJSDosyasi: cd-daha.js embed'lenmiş, kilit davranışlar içeriyor
// ve CSP'e uygun (inline script YOK — harici dosya `script-src 'self'`).
func TestDahaJSDosyasi(t *testing.T) {
	b, err := webui.FS.ReadFile("static/cd-daha.js")
	require.NoError(t, err, "static/cd-daha.js embed edilmemiş")
	js := string(b)

	for _, parca := range []string{
		"IntersectionObserver", // otomatik yükleme (buton 600px yakınına gelince)
		"parca=1",              // parça ucu
		"credentials: 'same-origin'",
		"cd-daha__btn", // tıklama hedefi
	} {
		assert.Contains(t, js, parca, "cd-daha.js'te eksik davranış: %s", parca)
	}
	// Harici istek yok (CSP `script-src 'self'` korunmalı).
	assert.NotContains(t, js, "http://", "harici kaynak yüklenmemeli")
	assert.NotContains(t, js, "https://", "harici kaynak yüklenmemeli")
}
