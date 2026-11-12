package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecimBalonuKonumlanmis(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "icerikform.templ"))
	require.NoError(t, err)
	s := string(b)

	m := regexp.MustCompile(`\.be-bubble\{[^}]*\}`).FindString(s)
	require.NotEmpty(t, m, ".be-bubble kuralı bulunamadı")

	assert.Contains(t, m, "position:absolute",
		"balon konumlandırılmalı; yoksa sayfa akışında en alta düşer")
	assert.Contains(t, m, "visibility:hidden",
		"balon varsayılan gizli olmalı (TipTap satır içi stil ile gösterir)")
	assert.NotContains(t, m, "display:none",
		"display:none TipTap'in göstermesini engeller")
	assert.Contains(t, m, "width:max-content",
		"balon içeriği kadar geniş olmalı; tam genişlik hatası buydu")
	assert.Contains(t, m, "max-width:",
		"dar ekranda taşmamalı")
}
