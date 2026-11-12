package repo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Okuyucuya bakan listeler "yayin" sıralamasını kullanır.
//
// NEDEN TEST: varsayılan sıra `updated_at DESC` ve bu admin kuyruğu için
// doğru. Ana sayfa ile bölüm listeleri yanlışlıkla varsayılana düşerse
// hata SESSİZ olur: sayfa çalışır, yalnız altı yıllık bir yazı sırf
// düzenlendiği için manşetin yanına çıkar (ana sayfa denetimi,
// 12 Kasım 2026 — bölüm blokları 2020 tarihli haber gösteriyordu).
func TestYayinSiralamasiTanimli(t *testing.T) {
	ifade := contentOrderBy("yayin")

	assert.NotEqual(t, contentOrderDefault, ifade,
		"'yayin' anahtarı beyaz listede yok, varsayılana düşüyor")
	assert.Contains(t, ifade, "published_at DESC",
		"yayın sırası published_at'e göre olmalı")
	assert.Contains(t, strings.ToUpper(ifade), "NULLS LAST",
		"yayın tarihi boş kayıt listeyi baştan tutmamalı")
}

func TestVarsayilanSiraHalaGuncellenme(t *testing.T) {
	// Admin kuyruğu bu davranışa dayanıyor: son dokunulan üstte.
	assert.Equal(t, "c.updated_at DESC", contentOrderBy("bilinmeyen-anahtar"))
	assert.Equal(t, "c.updated_at DESC", contentOrderBy("guncel"))
}
