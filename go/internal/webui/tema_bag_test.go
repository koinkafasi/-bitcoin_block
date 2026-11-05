package webui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tema bazlı token bağı — SON katman (Faz 183 eki).
//
// NEDEN: admin-shadcn.css'te `--text` tema bazlı bağlandıktan sonra bile
// yanlış çalışıyordu. Sebep yükleme sırası: **admin-pano.css daha sonra**
// yükleniyor ve sabit `--text: var(--sh-fg)` yazarak bağı eziyordu.
//
//	canlı ölçüm: --text-dark #858585 üretiliyor, ama --text yine #f2f1ee
//
// Aynı hata `--accent-text` için de vardı: AdminCSS tek bir `--accent-text`
// yazdığı için koyu temada da açık temanın tonu geçerliydi (2.9 kontrast).
// Artık AdminCSS yalnız `-acik`/`-dark` adlarını yazar; seçimi tema blokları
// yapar.

func cssDosya(t *testing.T, ad string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", ad))
	require.NoError(t, err)
	return string(b)
}

// TestTemaBagiSonKatmanda: admin-pano.css (son yüklenen katman) de tema
// bazlı metin rengi kullanmalı.
func TestTemaBagiSonKatmanda(t *testing.T) {
	s := cssDosya(t, "admin-pano.css")

	assert.Contains(t, s, "--text: var(--text-kullanici, var(--sh-fg))",
		"admin-pano açık temada kullanıcı metin rengini kullanmalı")
	assert.Contains(t, s, "--text: var(--text-dark, var(--sh-fg))",
		"admin-pano koyu temada açılmış tonu kullanmalı")

	// Sabit bağ kalmamalı: sonra yüklenen katmanda `--text: var(--sh-fg)`
	// olursa tema bağı sessizce kırılır.
	assert.NotContains(t, s, "\t--text: var(--sh-fg);",
		"sabit --text bağı tema bağını eziyor")
}

// TestAccentTextTemaAyrimi: AdminCSS düz `--accent-text` yazmamalı.
func TestAccentTextTemaAyrimi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "httpapi", "gorunum_handlers.go"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"--accent-text-acik:"`,
		"AdminCSS açık tema tonunu `-acik` adıyla yazmalı")
	assert.NotContains(t, s, `"--accent-text:"`,
		"düz --accent-text yazmak koyu temada da geçerli olur (ölçüldü: 2.9)")

	// Seçim admin-shadcn'de: iki tema da kendi değişkenini kullanmalı.
	c := cssDosya(t, "admin-shadcn.css")
	assert.Contains(t, c, "--accent-text: var(--accent-text-acik,",
		"açık tema `-acik` değerini kullanmalı")
	assert.Contains(t, c, "--accent-text: var(--accent-text-dark,",
		"koyu tema `-dark` değerini kullanmalı")
}

// TestKoyuTemaMetinTuretimi baglı: seçilen metin rengi koyu temada açılmalı.
func TestKoyuTemaMetinUretiliyor(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "httpapi", "gorunum_handlers.go"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"--text-kullanici:"`,
		"seçilen metin rengi ayrı adla yazılmalı (tek değer iki temaya uymaz)")
	assert.Contains(t, s, `"--text-dark:"`,
		"koyu tema için açılmış ton üretilmeli")
	assert.Contains(t, s, "koyuTemaMetin(",
		"açma işlemi ortak yardımcıdan gelmeli")
}
