package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMediaCryptoIkon Yerel ikon kütüphanesinin /media/crypto/ üzerinden
// doğru MIME, önbellek ve güvenlik başlıklarıyla sunulduğunu doğrular.
func TestMediaCryptoIkon(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/media/crypto/btc.svg?v=1a63530", nil)
	rr := httptest.NewRecorder()

	s.handleMedia(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "image/svg+xml", rr.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=31536000, immutable", rr.Header().Get("Cache-Control"))
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, rr.Body.String(), "<svg")
}

func TestMediaCryptoIkonYokVeYolGuvenligi(t *testing.T) {
	s := &Server{}
	cases := []string{
		"/media/crypto/yokboyle.svg",
		"/media/crypto/",
		"/media/crypto/nested/btc.svg",
		"/media/crypto/../btc.svg",
		"/media/crypto/..%2fbtc.svg",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		s.handleMedia(rr, req)
		assert.Equalf(t, http.StatusNotFound, rr.Code, "%s 404 dönmeli", path)
		assert.NotContainsf(t, rr.Body.String(), "<svg", "%s ikon basmamalı", path)
	}
}
