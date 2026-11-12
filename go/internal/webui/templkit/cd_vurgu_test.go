package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bölüm başlıklarındaki sarı vurgu bandı. Kural silinirse başlıklar
// sade metne döner ve kimse fark etmez — bu yüzden teste bağlı.
func TestCDVurguStiliVar(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-vurgu {")
	require.GreaterOrEqual(t, i, 0, "başlık vurgu kuralı yok")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")+1]

	// `display: inline` zorunlu: blok elemanda gradient tüm kutuyu
	// kaplar, vurgu değil arka plan olur.
	assert.Contains(t, blok, "display: inline")

	// Bant ölçüsü `em` olmalı: yüzde tabanlı değer font boyutuna
	// ölçeklenmiyor, 14px başlıkta metni yutuyordu.
	assert.Contains(t, blok, "em", "bant ölçüsü em tabanlı olmalı")
	assert.NotContains(t, blok, "background-size: 100% 0.38em",
		"eski (çok ince) değer geri gelmiş")

	// Satır kırılmasında bant her satırda sürmeli.
	assert.Contains(t, blok, "box-decoration-break: clone")
}

// background-position'ın ÜÇ değerli yazımı geçersizdir: tarayıcı
// sessizce `0% 0%`e düşer ve bant metnin ÜSTÜNDE kalır. Hata ekranda
// görülene kadar fark edilmiyor — dört değerli yazım sabitleniyor.
func TestCDVurguKonumuDortDegerli(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-vurgu {")
	require.GreaterOrEqual(t, i, 0)
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")+1]

	j := strings.Index(blok, "background-position:")
	require.GreaterOrEqual(t, j, 0, "bant konumu tanımsız")
	satir := blok[j:]
	satir = satir[:strings.Index(satir, ";")]

	assert.Contains(t, satir, "bottom", "bant ALTTAN hizalanmalı")
	// "left 0 bottom 0.08em" → 4 parça. Üç parçalı yazım geçersiz.
	parca := strings.Fields(strings.TrimPrefix(satir, "background-position:"))
	assert.GreaterOrEqual(t, len(parca), 4,
		"üç değerli background-position geçersizdir, tarayıcı yok sayar: %q", satir)
}
