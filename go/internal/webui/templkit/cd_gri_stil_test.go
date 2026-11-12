package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Siyah-beyaz gösterim CSS filtresiyle yapılıyor. Kural silinirse
// admin'deki işaret sessizce etkisiz kalır — bu yüzden teste bağlı.
func TestCDGriStiliVar(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-gri img {")
	require.GreaterOrEqual(t, i, 0, "gri görsel kuralı yok")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")+1]
	assert.Contains(t, blok, "filter: grayscale(1)")

	// Üzerine gelince renk geri geliyor.
	assert.Contains(t, css, ".cd-gri:hover img")
}
