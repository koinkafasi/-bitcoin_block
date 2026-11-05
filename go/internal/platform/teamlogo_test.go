package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Küçük geçerli PNG: 1x1 saydam. Gerçek dosyalar 15–290 KB ama imza denetimi
// için bu yeterli.
func tinyPNG() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A},
		make([]byte, 120)...)
}

func TestValidTeamLogoKey(t *testing.T) {
	gecerli := []string{"e-432", "t-galatasaray", "e-132335", "a1"}
	for _, k := range gecerli {
		assert.True(t, ValidTeamLogoKey(k), "geçerli olmalıydı: %q", k)
	}
	// Yol geçişi ve gizli dosya denemeleri reddedilmeli.
	gecersiz := []string{"", "../etc/passwd", "..", "e-432/../x", "E-432", "e 432", "a.png", ".hidden", string(make([]byte, 70))}
	for _, k := range gecersiz {
		assert.False(t, ValidTeamLogoKey(k), "geçersiz olmalıydı: %q", k)
	}
}

func TestSaveTeamLogoVeOku(t *testing.T) {
	// StorageDir testte geçici dizine alınır (süreç geneli değişken).
	eski := storageDir
	storageDir = t.TempDir()
	defer func() { storageDir = eski }()

	assert.False(t, HasTeamLogo("e-432"), "başta logo olmamalı")
	assert.NoError(t, SaveTeamLogo("e-432", tinyPNG()))
	assert.True(t, HasTeamLogo("e-432"))

	f, st, err := OpenTeamLogo("e-432")
	assert.NoError(t, err)
	defer f.Close()
	assert.Greater(t, st.Size(), int64(100))
}

func TestSaveTeamLogoPNGOlmayaniReddeder(t *testing.T) {
	eski := storageDir
	storageDir = t.TempDir()
	defer func() { storageDir = eski }()

	// CDN'ler hata sayfasını da 200 ile döndürebiliyor; imza denetimi bunu
	// diske yazmadan yakalar.
	err := SaveTeamLogo("e-1", []byte("<html><body>403 Forbidden</body></html>"))
	assert.Error(t, err)
	assert.False(t, HasTeamLogo("e-1"))
}

func TestSaveTeamLogoYolGecisiniReddeder(t *testing.T) {
	eski := storageDir
	storageDir = t.TempDir()
	defer func() { storageDir = eski }()

	assert.Error(t, SaveTeamLogo("../kacak", tinyPNG()))
	_, err := os.Stat(filepath.Join(storageDir, "..", "kacak.png"))
	assert.True(t, os.IsNotExist(err), "dizin dışına dosya yazılmamalı")
}

func TestOpenTeamLogoYoksaHata(t *testing.T) {
	eski := storageDir
	storageDir = t.TempDir()
	defer func() { storageDir = eski }()

	_, _, err := OpenTeamLogo("e-yok")
	assert.ErrorIs(t, err, ErrTeamLogoNotFound)
}

func TestSaveTeamLogoUzerineYazar(t *testing.T) {
	eski := storageDir
	storageDir = t.TempDir()
	defer func() { storageDir = eski }()

	assert.NoError(t, SaveTeamLogo("e-2", tinyPNG()))
	buyuk := append(tinyPNG(), make([]byte, 500)...)
	assert.NoError(t, SaveTeamLogo("e-2", buyuk))
	_, st, err := OpenTeamLogo("e-2")
	assert.NoError(t, err)
	assert.Equal(t, len(buyuk), int(st.Size()))
}
