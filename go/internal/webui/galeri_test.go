package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGazeteSablonGalerisi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "gazete.templ"))
	require.NoError(t, err)
	s := string(b)

	// Kart düzeni (tablo değil).
	assert.Contains(t, s, `class="gz-gal"`,
		"şablonlar ızgara olarak basılmalı")
	assert.Contains(t, s, `class="gz-gal__kart"`,
		"her şablon bir kart olmalı")
	assert.NotContains(t, s, `<th>Varsayılan</th>`,
		"eski tablo başlığı kalmamalı")

	// Galeriyle AYNI önizleme bileşeni kullanılmalı.
	assert.Contains(t, s, "@PageThumb(p)",
		"sayfa mini önizlemeleri galeriyle aynı bileşenden gelmeli")

	// Seçilebilirlik: "Bu Şablondan Sayı" karttan çalışmalı.
	assert.Contains(t, s, "Bu Şablondan Sayı",
		"kart şablonu seçmeye izin vermeli")

	// Sığma: ızgara auto-fill, önizlemeler yan yana.
	assert.Contains(t, s, "repeat(auto-fill, minmax(200px, 1fr))",
		"ızgara dar ekranda sarmalı (taşma olmamalı)")
	assert.Contains(t, s, ".gz-thumbs {",
		"önizleme şeridi tanımlı olmalı")

	// Tema: sabit açık zemin kullanılmamalı.
	assert.Contains(t, s, "background: var(--surface)",
		"kart zemini tema tokenından gelmeli")
}

// Şablon satırı veri taşıyıcısı önizlemeyi içermeli.
func TestGazeteSablonSatirOnizleme(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "gazete.templ"))
	require.NoError(t, err)
	s := string(b)

	i := strings.Index(s, "type GazeteSablonSatir struct {")
	require.Greater(t, i, 0)
	son := strings.Index(s[i:], "}")
	require.Greater(t, son, 0)
	yapi := s[i : i+son]

	for _, alan := range []string{"Pages", "SayfaSayisi", "YeniSayiURL"} {
		assert.Containsf(t, yapi, alan, "GazeteSablonSatir.%s alanı olmalı", alan)
	}

	// Handler önizlemeyi galeriyle aynı yardımcıdan doldurmalı.
	h, err := os.ReadFile(filepath.Join("..", "httpapi", "gazete_panel_view.go"))
	require.NoError(t, err)
	assert.Contains(t, string(h), "sablonPagePreview(t.Pages)",
		"panel, galeriyle aynı önizleme yardımcısını kullanmalı")
}
