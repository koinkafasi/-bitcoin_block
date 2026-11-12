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
	// Gri dönüşüm, sitenin ortak ton jetonuyla BİRLİKTE yazılır: tek
	// `filter` kazandığı için yalnız `grayscale(1)` yazılsaydı gri
	// görseller ortak kontrast ayarını kaybederdi.
	assert.Contains(t, blok, "grayscale(1)")
	assert.Contains(t, blok, "var(--cd-gorsel-ton)",
		"gri görsel ortak tonu da uygulamalı")

	// Üzerine gelince renk geri geliyor.
	assert.Contains(t, css, ".cd-gri:hover img")
}
