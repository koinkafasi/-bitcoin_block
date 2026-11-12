package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTekKabuk_PanoKendiKabugunuCizmez: pano içeriği ortak kabuğun içinde
// yaşamalı. pn-app geri gelirse iki menü iki görünüm demektir.
func TestTekKabuk_PanoKendiKabugunuCizmez(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "pano.templ"))
	require.NoError(t, err)
	s := string(b)

	for _, yasak := range []string{"pn-app", "pn-side", "pn-top", "pn-brand"} {
		assert.NotContainsf(t, s, `"`+yasak+`"`,
			"pano şablonu kendi kabuğunu çiziyor (%s); ortak kabuk kullanılmalı", yasak)
	}
	assert.Contains(t, s, `class="pn-view"`,
		"pano içeriği .pn-view sarmalayıcısında olmalı")
}

func TestTekKabuk_PanoCSSKabukTasimaz(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "pano.css"))
	require.NoError(t, err)
	s := string(b)

	for _, yasak := range []string{
		".pn-app {", ".pn-side {", ".pn-top {", ".pn-main {",
		".pn-item {", ".pn-brand {", ".pn-search {",
	} {
		assert.NotContainsf(t, s, yasak,
			"pano.css kabuk kuralı taşıyor (%s); admin-pano.css'e ait", yasak)
	}

	// Renk değişkenleri .pn-view kapsamında olmalı: .pn-app üzerinde
	// kalsaydı (o öge artık yok) grafikler renksiz kalırdı.
	assert.Contains(t, s, ".pn-view {", "değişkenler .pn-view kapsamında tanımlanmalı")
	assert.Contains(t, s, "--pn-sig", "pano renk değişkeni eksik")
}

// TestHareketKatmani: geçişler tüm panelde ortak ve erişilebilir olmalı.
func TestHareketKatmani(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	s := string(css)

	// 1) Kademeli giriş animasyonu her panel sayfasında geçerli.
	assert.Contains(t, s, "@keyframes pn-gir")
	assert.Contains(t, s, ".admin-body > *", "animasyon tüm gövde çocuklarını kapsamalı")
	assert.Contains(t, s, "animation-delay", "kademeli gecikme olmalı")

	// 2) HTMX ile yerinde güncellenen paneller yumuşak geçmeli.
	assert.Contains(t, s, ".htmx-settling")
	assert.Contains(t, s, ".htmx-added")

	kural := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(s, " ")
	assert.NotContains(t, kural, "@view-transition",
		"tam sayfa görünüm geçişi geri gelmemeli — hap animasyonuyla çakışır")
	assert.NotContains(t, kural, "view-transition-name",
		"adlandırılmış görünüm geçişi öğesi kalmamalı")

	i := strings.Index(s, "prefers-reduced-motion: reduce) {\n\t.admin-body > *")
	require.Greater(t, i, 0, "hareket katmanının reduced-motion bloğu bulunamadı")
	son := s[i:]
	assert.Contains(t, son, ".admin-body > *", "giriş animasyonu hedefi durmalı")
	assert.Contains(t, son, "animation: none", "animasyonlar kapatılmalı")
	assert.Contains(t, son, ".htmx-settling", "HTMX geçişi reduced-motion'da durmalı")
	assert.Contains(t, son, ".table tbody tr", "tablo geçişi reduced-motion'da durmalı")
}

func TestHareketJS_TamSayfaGecisiYok(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// Yalnız yorumda geçebilir; ÇAĞRI olarak bulunmamalı.
	assert.NotContains(t, s, "document.startViewTransition(",
		"manuel tam sayfa görünüm geçişi çağrısı kalmamalı")
	require.Contains(t, s, "window.tbToast",
		"ortak balon sistemi tanımlı olmalı")
}
