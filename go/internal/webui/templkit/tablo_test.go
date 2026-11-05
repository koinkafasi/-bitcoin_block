package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTabloBosSifirSutun: colspan="0" geçersizdir (tarayıcı yok sayar ve
// hücre hiçbir sütunu kaplamaz). Boş durum sessizce kırık tablo üretmesin.
func TestTabloBosSifirSutun(t *testing.T) {
	assert.Equal(t, 1, tabloBosSutun(0))
	assert.Equal(t, 1, tabloBosSutun(-3))
	assert.Equal(t, 9, tabloBosSutun(9))
	assert.Equal(t, 1, tabloBosSutun(1))
}
