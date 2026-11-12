package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMansetTuvalKurallari(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "manset.templ"))
	require.NoError(t, err)
	s := string(b)

	// Kaplama, canvas'ın üstüne mutlak konumlanmalı: referans .me-canvaswrap.
	assert.Contains(t, s, ".me-canvaswrap", "canvas sarmalayıcı kuralı olmalı")
	assert.Contains(t, s, "position: relative", "sarmalayıcı konumlandırma bağlamı olmalı")

	// Kaplama kuralı: mutlak + tam kaplama + arka plan deseni.
	i := strings.Index(s, ".me-grid {")
	require.Greater(t, i, 0, ".me-grid kuralı bulunamadı")
	son := strings.Index(s[i:], "}")
	require.Greater(t, son, 0)
	kural := s[i : i+son]
	assert.Contains(t, kural, "position: absolute")
	assert.Contains(t, kural, "background-image", "ızgara deseni olmalı")
	assert.Contains(t, kural, "inset: 0", "canvas'ı tam kaplamalı")

	// Varsayılan gizli, .on ile görünür.
	assert.Contains(t, s, ".me-grid.on", "görünürlük sınıfı olmalı")
}

func TestMansetIzgaraKimligi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "manset.templ"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `id="me-grid"`, "onay kutusu kimliği korunmalı (JS bağlıyor)")
	assert.Contains(t, s, `id="me-grid-overlay"`, "kaplama ayrı kimlik taşımalı")

	// JS de aynı ayrımı kullanmalı.
	js, err := os.ReadFile(filepath.Join("static", "manset.js"))
	require.NoError(t, err)
	j := string(js)
	assert.Contains(t, j, `getElementById("me-grid-overlay")`,
		"kaplama ölçekleme kodu ayrı kimliği kullanmalı")
	assert.Contains(t, j, `$("#me-grid")`,
		"onay kutusu dinleyicisi kendi kimliğini kullanmalı")
}
