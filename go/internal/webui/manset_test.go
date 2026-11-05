package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Manşet editörü tuvali kuralları (Faz 178).
//
// NEDEN: sayfada `.me-*` sınıfları için HİÇ kural yoktu. Tarayıcıda ölçüldü:
//
//	.me-canvaswrap  position:static   (kaplamanın referansı yok)
//	#me-grid-overlay height:0, background-image:none
//	sonuç: "Izgara" düğmesi hiçbir şey çizmiyordu
//
// JS `backgroundSize` yazıyor ve `.on` sınıfını çeviriyordu; eksik olan
// yalnızca CSS'ti. Kural sayfaya özel olduğu için manset.templ içinde.
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

// TestMansetIzgaraKimligi: onay kutusu ile kaplama AYNI id'yi taşımamalı.
// Taşıdığında `getElementById` ilk eşleşmeyi (onay kutusunu) döndürür ve
// `backgroundSize` kutuya yazılır — kaplama hiç boyanmaz (Faz 177'de bulundu).
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
