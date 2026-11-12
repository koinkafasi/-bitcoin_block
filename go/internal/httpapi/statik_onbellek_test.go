package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestSpriteSurumsuzYolaDayaniyor(t *testing.T) {
	cc := staticCacheControl("")
	require.NotEmpty(t, cc)
	require.NotContains(t, cc, "immutable")
}
