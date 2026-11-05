package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Sürümsüz statik dosya bir yıl "immutable" önbelleklenirse, içeriği
// değiştiğinde mevcut kullanıcılara ASLA ulaşmaz. Bu bir kez oldu: ikon
// sprite'ı 41 ikondan 621'e çıkarıldı ama şablonlar onu sürümsüz çağırdığı
// için tarayıcılar 11 KB'lık eski kopyayı tuttu ve olmayan ikonlar sessizce
// boş kutu olarak basıldı — ne hata ne uyarı.
func TestSurumsuzStatikDosyaBayatlamiyor(t *testing.T) {
	// asset() ile gelen sürümlü adresler uzun ömürlü kalmalı.
	require.Contains(t, staticCacheControl("v=abc123"), "immutable",
		"sürümlü dosya uzun süre önbelleklenebilmeli")

	// Sürümsüz adres immutable OLMAMALI.
	for _, q := range []string{"", "foo=1"} {
		cc := staticCacheControl(q)
		require.NotContainsf(t, cc, "immutable",
			"sürümsüz istek (%q) immutable almamalı: içerik değişince kullanıcıya ulaşmaz", q)
		require.Contains(t, cc, "must-revalidate",
			"sürümsüz istek doğrulamaya zorlamalı")
	}
}

// Sprite şablonlarda sürümsüz çağrılıyor; bu testin varlık sebebi o.
// Çağrı biçimi sürümlüye dönerse test güncellenmeli, ama o zamana kadar
// politikanın sürümsüz yolu desteklediğinden emin olunmalı.
func TestSpriteSurumsuzYolaDayaniyor(t *testing.T) {
	cc := staticCacheControl("")
	require.NotEmpty(t, cc)
	require.NotContains(t, cc, "immutable")
}
