package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Görseller üç ayrı kaynaktan geliyor (kendi sunucumuz, i.ytimg.com,
// cdn.sanity.io). Ortak ton jetonu silinirse kartlar yeniden farklı
// "sıcaklıkta" görünür — sessiz bir gerileme, bu yüzden teste bağlı.
func TestCDGorselTonJetonuVar(t *testing.T) {
	css := cdCSS(t)
	require.Contains(t, css, "--cd-gorsel-ton:", "ortak ton jetonu yok")

	// Değerler KÜÇÜK olmalı: amaç uçları yaklaştırmak, görseli yeniden
	// renklendirmek değil. Agresif bir dönüşüm haber fotoğrafını bozar.
	i := strings.Index(css, "--cd-gorsel-ton:")
	satir := css[i:]
	satir = satir[:strings.Index(satir, ";")]
	assert.Contains(t, satir, "saturate(")
	assert.Contains(t, satir, "contrast(")
	assert.NotContains(t, satir, "grayscale", "ortak ton gri YAPMAMALI")
	assert.NotContains(t, satir, "sepia")
}

// Ton yalnız içerik görsellerine uygulanmalı; logo ve ikonlar hariç.
func TestCDGorselTonuIcerikGorselineUygulanir(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, "--cd-gorsel-ton: saturate")
	require.GreaterOrEqual(t, i, 0)

	// Kuralı kendi bildirimiyle bul: CSS'te başka `:where()` kullanımları
	// da var, ilkini almak yanlış bloğu ölçerdi.
	k := strings.Index(css, "filter: var(--cd-gorsel-ton);")
	require.GreaterOrEqual(t, k, 0, "ton kuralı yok")

	// Bildirimden GERİYE doğru seçiciyi oku.
	bas := strings.LastIndex(css[:k], "}")
	secici := css[bas+1 : k]

	// `:where()` ile düşük özgüllük: tek sınıflı istisna (örn. .cd-gri)
	// bu kuralı ezebilmeli.
	assert.Contains(t, secici, ":where(", "ton kuralı :where() ile yazılmalı")
	assert.Contains(t, secici, "__gorsel")
	assert.Contains(t, secici, "img")
}
