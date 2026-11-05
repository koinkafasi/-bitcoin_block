package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Kabuk korumalı gezinme (Faz 156) testleri.
//
// Arka plan: her gezinmede 8 CSS + 12 JS (24 istek) yeniden iniyor ve
// admin-ui.js baştan koşuyordu. Sunucu 8–29 ms; yavaşlık yeniden
// yüklemeden geliyordu. Çözüm: yan menü DOM'da kalır, yalnız .admin-main
// hx-boost ile değişir.
//
// İki sessiz tuzak ölçümle bulundu ve burada kilitleniyor:
//  1. Sunucu HX-Request görünce PANEL PARÇASI döndürüyordu; boost'a da aynı
//     parça gidince hedefte .admin-main bulunmuyor ve swap BOŞ kalıyordu
//     (sayfa bembeyaz). Boost artık "parça isteği" sayılmıyor.
//  2. htmx geçmişi <body> üzerinden saklıyordu; geri tuşunda yan menü de
//     yeniden kuruluyordu. hx-history-elt ile yalnız .admin-main saklanıyor.

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
