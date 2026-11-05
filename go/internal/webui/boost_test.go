package webui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kabuk korumalı gezinmenin (Faz 156) JS/şablon tarafı.

func TestBoostYanMenuVeSekmelerde(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	s := string(b)

	require.Contains(t, s, `hx-boost="true"`,
		"yan menüde boost olmalı")
	require.Contains(t, s, `hx-target=".admin-main"`,
		"hedef içerik bölgesi olmalı (yan menü sabit kalır)")
	require.Contains(t, s, `hx-select=".admin-main"`)
	require.Contains(t, s, `hx-push-url="true"`)

	// Head'de betik yükleyen sayfalar boost DIŞI olmalı: boost <head>'i
	// yenilemez, o betikler hiç yüklenmez ve sayfa ölü gelir.
	assert.Contains(t, s, `href="/admin" data-ws="pano" hx-boost="false"`,
		"pano kendi betiğini (pano.js) head'den yüklüyor; boost dışı olmalı")
	assert.Contains(t, s, `href="/admin/cikis" hx-boost="false"`,
		"çıkış boost'lanmamalı (oturum akışı tam yükleme)")
}

func TestBoostAgirSayfalarHaric(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// ws-tabs boost taşımalı
	assert.Contains(t, s, "nav.setAttribute('hx-boost', 'true')")
	assert.Contains(t, s, "nav.setAttribute('hx-target', '.admin-main')")

	// Ağır sayfalar (head betiği gerektirenler) listede olmalı
	for _, yol := range []string{
		"/admin/icerikler/yeni", "/admin/manset-editor", "/admin/ses-ayarlari",
	} {
		assert.Containsf(t, s, `'`+yol+`': 1`,
			"ağır sayfa boost dışı listesinde olmalı: %s", yol)
	}
}

func TestBoostGecmisVeKurulum(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// Geçmiş yalnız içeriği saklasın: yoksa geri tuşunda yan menü yeniden kurulur.
	assert.Contains(t, s, "hx-history-elt",
		"geçmiş .admin-main üzerinden saklanmalı")
	// Geri/ileri sonrası kurucular çalışmalı.
	assert.Contains(t, s, "htmx:historyRestore")
	// Boost sonrası yeniden kurulum.
	assert.Contains(t, s, "htmx:afterSettle")
	assert.Contains(t, s, "function tbAdminInit()")
	// Hata/giriş yönlenmesinde tam yükleme (boş ya da giriş sayfası
	// içeriğe yazılırsa panel bozulur).
	assert.Contains(t, s, "htmx:beforeSwap")
	assert.Contains(t, s, "shouldSwap = false")
}

func TestBoostTabloAraclariIdempotent(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// Önbellekten geri gelen DOM'da JS özellikleri kaybolur; ikinci kez
	// araç çubuğu eklenmemesi için DOM'a da bakılmalı.
	assert.Contains(t, s, "tbl.previousElementSibling")
	assert.Contains(t, s, "classList.contains('tbl-tools')")
}
