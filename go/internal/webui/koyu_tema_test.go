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

func cssOku(t *testing.T, ad string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", ad))
	require.NoError(t, err)
	return string(b)
}

// TestKoyuTemaMetinTonlari: pano.css koyu bloğunda metin tonları tanımlı ve
// açık temadakinden FARKLI olmalı (koyu zeminde açık ton gerekir).
func TestKoyuTemaMetinTonlari(t *testing.T) {
	s := cssOku(t, "pano.css")

	bas := strings.Index(s, `html[data-theme="dark"] .pn-view`)
	require.Greater(t, bas, 0, "koyu tema bloğu bulunamadı")
	son := strings.Index(s[bas:], "}")
	require.Greater(t, son, 0)
	koyuBlok := s[bas : bas+son]

	for _, token := range []string{"--pn-sig-fg:", "--pn-good-fg:", "--pn-bad:"} {
		assert.Containsf(t, koyuBlok, token,
			"koyu temada %s tanımlı olmalı (metin eşiği 4.5)", token)
	}
}

// TestTehlikeDolgusu: `--danger` açık kırmızı; beyaz yazıyla 4.5 geçmiyor.
// Rozet dolgusu için ayrı ton olmalı ve kullanılmalı.
func TestTehlikeDolgusu(t *testing.T) {
	s := cssOku(t, "admin-shadcn.css")
	assert.Contains(t, s, "--danger-fill:",
		"beyaz yazılı tehlike rozeti için koyu dolgu tonu tanımlı olmalı")

	// Şablonlarda `var(--danger);color:#fff` doğrudan kullanılmamalı.
	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		ic := string(b)
		assert.NotContainsf(t, ic, "background:var(--danger);color:#fff",
			"%s: beyaz yazılı dolgu --danger-fill kullanmalı", filepath.Base(f))
	}
}

// TestKoyuTemaMetinRengiBagli: AdminCSS `--text` yazmamalı; tema bazlı
// değişkenlere bırakmalı. Aksi hâlde koyu temada siyah metin kalır.
func TestKoyuTemaMetinRengiBagli(t *testing.T) {
	s := cssOku(t, "admin-shadcn.css")

	// Açık tema: kullanıcı seçimi, yoksa tema rengi.
	assert.Contains(t, s, "--text: var(--text-kullanici, var(--sh-fg));",
		"açık temada kullanıcı metin rengi kullanılmalı")
	// Koyu tema: açılmış ton, yoksa tema rengi.
	assert.Contains(t, s, "--text: var(--text-dark, var(--sh-fg));",
		"koyu temada açılmış metin tonu kullanılmalı")
}

// TestBeToolbarTemayaBagli: içerik formunun araç çubuğu sabit açık zemindi;
// koyu temada bembeyaz kalıp düğme yazıları okunmuyordu.
func TestBeToolbarTemayaBagli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "icerikform.templ"))
	require.NoError(t, err)
	s := string(b)

	m := regexp.MustCompile(`\.be-toolbar\{[^}]*\}`).FindString(s)
	require.NotEmpty(t, m, ".be-toolbar kuralı bulunamadı")
	assert.NotContains(t, m, "background:var(--bg2,#f7f7f6)",
		"sabit açık zemin koyu temada okunmuyor")
	assert.Contains(t, m, "var(--surface-2",
		"araç çubuğu tema yüzeyini kullanmalı")
}
