package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCDAnalizDilimSerpistirir(t *testing.T) {
	kartlar := []CDAnalizKart{
		{Baslik: "1"}, {Baslik: "2"}, {Baslik: "3"},
		{Baslik: "4"}, {Baslik: "5"}, {Baslik: "6"},
		{Baslik: "7"},
	}
	// En yeni üç yazı üç AYRI kolona düşmeli; blok blok bölmek en yeniyi
	// tek kolonda yığardı.
	k0 := cdAnalizDilim(kartlar, 0)
	k1 := cdAnalizDilim(kartlar, 1)
	k2 := cdAnalizDilim(kartlar, 2)

	require.Len(t, k0, 3)
	require.Len(t, k1, 2)
	require.Len(t, k2, 2)
	assert.Equal(t, "1", k0[0].Baslik)
	assert.Equal(t, "2", k1[0].Baslik)
	assert.Equal(t, "3", k2[0].Baslik)
	assert.Equal(t, "4", k0[1].Baslik)
	assert.Equal(t, "7", k0[2].Baslik)
}

func TestCDAnalizDilimSinirDisi(t *testing.T) {
	kartlar := []CDAnalizKart{{Baslik: "1"}}
	assert.Nil(t, cdAnalizDilim(kartlar, 3))
	assert.Nil(t, cdAnalizDilim(kartlar, -1))
	assert.Nil(t, cdAnalizDilim(nil, 0))
}

func TestCDAnalizModulTipiKayitli(t *testing.T) {
	// Tip listede yoksa `cdModuller` bloğu düşürür ve modül sayfa
	// tasarımcısına eklendiğinde sessizce kaybolur.
	assert.True(t, CDModulGecerli("cd_analiz"))
}

func TestCDAnalizRozetGercekSayi(t *testing.T) {
	assert.Equal(t, "son 7 günde 25 analiz", cdAnalizRozet(25))
}
