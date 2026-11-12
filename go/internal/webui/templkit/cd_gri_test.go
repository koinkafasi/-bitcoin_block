package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCdGriSinif(t *testing.T) {
	// Kapalıyken sınıf eklenmez: mevcut kartların işaretlemesi değişmez.
	assert.Equal(t, "cd-kat-lead__gorsel", cdGriSinif("cd-kat-lead__gorsel", false))
	// Açıkken temel sınıfın YANINA eklenir, yerine geçmez — yoksa
	// görselin yerleşimi (en-boy, kırpma) bozulurdu.
	assert.Equal(t, "cd-kat-lead__gorsel cd-gri", cdGriSinif("cd-kat-lead__gorsel", true))
	// Temel sınıf yoksa baştaki boşluk bırakılmaz.
	assert.Equal(t, "cd-gri", cdGriSinif("", true))
	assert.Equal(t, "", cdGriSinif("", false))
}

// Sınıf adı `cd-grid` ile karışmamalı: ölçüm sırasında `cd-gri`
// aramak `cd-grid` içinde de eşleşip yanlış pozitif vermişti.
// Burada sınıfın tam adı sabitleniyor.
func TestCdGriSinifAdi(t *testing.T) {
	s := cdGriSinif("x", true)
	assert.Contains(t, s, "cd-gri")
	assert.NotContains(t, s, "cd-grid")
}
