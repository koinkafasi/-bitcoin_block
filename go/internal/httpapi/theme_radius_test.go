package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tema köşe yarıçapı: 0 açıkça kaydedildiğinde alan "0" göstermeli,
// boş bırakıldığında varsayılan (8px) korunmalı.
func TestThemeRadiusSifirGorunumu(t *testing.T) {
	assert.Equal(t, "", themeField(&ThemeConfig{}, "theme_radius"))
	assert.Equal(t, "0", themeField(&ThemeConfig{RadiusSet: true}, "theme_radius"))
	assert.Equal(t, "12", themeField(&ThemeConfig{Radius: 12}, "theme_radius"))
}
