package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Dil önekli üye adresleri gerçek kullanıcı akışıdır: şablonlar
// `{{.LangBase}}/giris` gibi hedefler üretir ve bu yollar /tr/... olabilir.
// Önek soyulmazsa giriş/kayıt/ödeme formları 404/405 döner.

func TestLocaleAliasPath(t *testing.T) {
	supported := func(l string) bool { return l == "tr" || l == "en" }
	cases := []struct {
		in       string
		wantPath string
		wantLang string
		wantOK   bool
	}{
		{"/tr/giris", "/giris", "tr", true},
		{"/en/kayit", "/kayit", "en", true},
		{"/tr/abonelik", "/abonelik", "tr", true},
		{"/tr/abonelik/odeme", "/abonelik/odeme", "tr", true},
		{"/tr/hesabim/abonelik/iptal", "/hesabim/abonelik/iptal", "tr", true},
		{"/tr/giris/google/callback", "/giris/google/callback", "tr", true},
		{"/tr/cikis", "/cikis", "tr", true},
		// Soyulmamalı: bilinmeyen dil, üye olmayan yol, önek yok.
		{"/xx/giris", "/xx/giris", "", false},
		{"/tr/kategori/gundem", "/tr/kategori/gundem", "", false},
		{"/tr/arama", "/tr/arama", "", false},
		{"/giris", "/giris", "", false},
		{"/tr", "/tr", "", false},
		{"/tr/", "/tr/", "", false},
		{"/ping/giris", "/ping/giris", "", false},
		{"/", "/", "", false},
	}
	for _, c := range cases {
		gotPath, gotLang, gotOK := localeAliasPath(c.in, supported)
		assert.Equal(t, c.wantOK, gotOK, "ok: %s", c.in)
		assert.Equal(t, c.wantPath, gotPath, "path: %s", c.in)
		assert.Equal(t, c.wantLang, gotLang, "lang: %s", c.in)
	}
}

func TestLocaleAliasHandlerMetoduKorur(t *testing.T) {
	var gotPath, gotLang, gotMethod string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotLang = r.PathValue("lang")
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	h := localeAliasHandler(next, func(l string) bool { return l == "tr" })

	req := httptest.NewRequest(http.MethodPost, "/tr/kayit", strings.NewReader("a=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "/kayit", gotPath)
	assert.Equal(t, "tr", gotLang)
	assert.Equal(t, http.MethodPost, gotMethod, "POST korunmalı (aksi halde 405)")

	// Önek yoksa dokunulmaz.
	req2 := httptest.NewRequest(http.MethodGet, "/kayit", nil)
	h.ServeHTTP(httptest.NewRecorder(), req2)
	assert.Equal(t, "/kayit", gotPath)

	// Üye olmayan sayfa dokunulmaz.
	req3 := httptest.NewRequest(http.MethodGet, "/tr/kategori/gundem", nil)
	h.ServeHTTP(httptest.NewRecorder(), req3)
	assert.Equal(t, "/tr/kategori/gundem", gotPath)
}

func TestLocaleAliasMiddlewareBundleNilPanikYapmaz(t *testing.T) {
	// i18n bundle nil ise (birim testleri) middleware çökmemeli.
	s := &Server{}
	h := s.localeAliasMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/tr/giris", nil)
	rec := httptest.NewRecorder()
	assert.NotPanics(t, func() { h.ServeHTTP(rec, req) })
}
