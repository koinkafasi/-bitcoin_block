package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHTMXBoostParçaDeğil(t *testing.T) {
	// Normal parça isteği (hx-get + hx-target): parça dönmeli.
	r := httptest.NewRequest("GET", "/admin/dosyalar", nil)
	r.Header.Set("HX-Request", "true")
	assert.True(t, IsHTMX(r), "hx-get parça isteği HTMX sayılmalı")

	// Boost: SAYFA gezinmesi — tam sayfa dönmeli.
	r2 := httptest.NewRequest("GET", "/admin/dosyalar", nil)
	r2.Header.Set("HX-Request", "true")
	r2.Header.Set("HX-Boosted", "true")
	assert.False(t, IsHTMX(r2),
		"boost'lu gezinme parça isteği sayılmamalı: tam sayfa dönmezse "+
			"hedefte .admin-main bulunmaz ve swap boş kalır")

	// Ne boost ne htmx: normal tarayıcı isteği.
	r3 := httptest.NewRequest("GET", "/admin/dosyalar", nil)
	assert.False(t, IsHTMX(r3))
}
