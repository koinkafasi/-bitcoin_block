package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/platform"
)

// /takim-logo/{key}.png ucu — logolar kendi altyapımızdan servis edilir.

func pngBytes() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 300)...)
}

func withLogos(t *testing.T) {
	t.Helper()
	old := platform.StorageDir()
	platform.SetStorageDir(t.TempDir())
	t.Cleanup(func() { platform.SetStorageDir(old) })
}

// serveLogo isteği gerçek yönlendirici üzerinden gönderir; böylece joker
// karakterin `/takim-logo/{key}` deseniyle gerçekten eşleştiği de doğrulanır.
func serveLogo(t *testing.T, target string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /takim-logo/{key}", s.handleTeamLogo)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func TestTeamLogoUcuLogoyuDoner(t *testing.T) {
	withLogos(t)
	assert.NoError(t, platform.SaveTeamLogo("e-432", pngBytes()))

	rec := serveLogo(t, "/takim-logo/e-432.png", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, teamLogoCacheHeader, rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
	assert.Greater(t, rec.Body.Len(), 100)
}

func TestTeamLogoUcuYoksa404(t *testing.T) {
	withLogos(t)
	rec := serveLogo(t, "/takim-logo/e-yok.png", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTeamLogoUcu304Doner(t *testing.T) {
	withLogos(t)
	assert.NoError(t, platform.SaveTeamLogo("e-997", pngBytes()))

	ilk := serveLogo(t, "/takim-logo/e-997.png", nil)
	assert.Equal(t, http.StatusOK, ilk.Code)
	etag := ilk.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	ikinci := serveLogo(t, "/takim-logo/e-997.png", map[string]string{"If-None-Match": etag})
	assert.Equal(t, http.StatusNotModified, ikinci.Code)
	assert.Equal(t, 0, ikinci.Body.Len())
}

func TestTeamLogoUcuYolGecisiniReddeder(t *testing.T) {
	withLogos(t)
	// Sunucu yol geçişi denemelerini 404 ile kapatmalı; dizin dışına çıkmamalı.
	for _, hedef := range []string{
		"/takim-logo/..%2f..%2fetc%2fpasswd.png",
		"/takim-logo/.hidden.png",
		"/takim-logo/BUYUK.png",
	} {
		rec := serveLogo(t, hedef, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code, "hedef: %s", hedef)
	}
}

func TestTeamLogoUcuUzantisizDaCalisir(t *testing.T) {
	withLogos(t)
	assert.NoError(t, platform.SaveTeamLogo("t-galatasaray", pngBytes()))
	// Uzantı zorunlu değil: eski/yanlış bağlantılar da çalışsın.
	rec := serveLogo(t, "/takim-logo/t-galatasaray", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}
