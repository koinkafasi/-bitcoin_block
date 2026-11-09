package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/webui/templkit"
)

// YENİ TASARIM MODÜLLERİ — PALET/KABUL SÖZLEŞMESİ (9 Kasım 2026).
//
// NEDEN TEST: ana sayfanın yeni tasarımı `cd_home.templ` içinde sabit
// sıradaydı; modüler yönetime geçirildi. İki liste birbirinden koparsa
// tasarımcıda görünen blok sessizce basılmaz (ya da tersi: şablonun
// tanıdığı tip palette hiç çıkmaz). Bu test iki listeyi birbirine bağlar.
func TestCDModulTipleriPaletle(t *testing.T) {
	palet := typeKeys(builderTypesFor("home"))
	for _, tip := range templkit.CDModulTipleri {
		assert.Contains(t, palet, tip, "modül palette yok: %s", tip)
	}
}

// TestCDModulGecerli: tanınmayan tip kabul edilmez (yazım kayması).
func TestCDModulGecerli(t *testing.T) {
	assert.True(t, templkit.CDModulGecerli("cd_hero"))
	assert.True(t, templkit.CDModulGecerli("cd_bloklar"))
	assert.False(t, templkit.CDModulGecerli("cd_yok"))
	assert.False(t, templkit.CDModulGecerli("hero"))
	assert.False(t, templkit.CDModulGecerli(""))
}
