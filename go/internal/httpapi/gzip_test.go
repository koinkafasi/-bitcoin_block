package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipCompressesHTML(t *testing.T) {
	s := &Server{}
	h := s.withGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("<p>performans</p>", 500))
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")
	zr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer zr.Close()
	body, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Contains(t, string(body), "performans")
}

func TestGzipSkipsSSE(t *testing.T) {
	s := &Server{}
	h := s.withGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: ping\ndata: {}\n\n")
	}))
	req := httptest.NewRequest(http.MethodGet, "/canli/icerik/1", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Contains(t, rec.Body.String(), "event: ping")
}

// TestGzipCompressesJavaScript: `.js` dosyaları Go 1.20'den beri
// `text/javascript` olarak sunuluyor (WHATWG). Sıkıştırma listesi yalnız
// `application/javascript` bildiği için panelin HİÇBİR JS dosyası
// sıkıştırılmıyordu — editor.js 771 KB ham gidiyordu (gzip'li hâli 240 KB).
//
// Gerçek sunum yolu üzerinden ölçülüyor: http.FileServer + text/javascript,
// yani üretimde olan şeyin aynısı.
func TestGzipCompressesJavaScript(t *testing.T) {
	// Go'nun .js için ne ürettiği sürüme göre değişebilir; ikisini de dene.
	for _, ct := range []string{"text/javascript; charset=utf-8", "application/javascript"} {
		t.Run(ct, func(t *testing.T) {
			s := &Server{}
			h := s.withGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", ct)
				_, _ = io.WriteString(w, strings.Repeat("function f(){return 1};", 500))
			}))
			req := httptest.NewRequest(http.MethodGet, "/static/editor.js", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"),
				"%s sıkıştırılmıyor", ct)
			assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")
		})
	}
}

// TestCompressibleContentType: liste tabloda tutuluyor, sessizce daralmasın.
func TestCompressibleContentType(t *testing.T) {
	sikisan := []string{
		"text/html; charset=utf-8", "text/css", "text/plain", "text/xml",
		"text/javascript; charset=utf-8", "application/javascript",
		"application/json", "image/svg+xml", "application/manifest+json",
	}
	for _, ct := range sikisan {
		assert.Truef(t, compressibleContentType(ct), "%q sıkışmalı", ct)
	}
	sikismayan := []string{
		"text/event-stream", "image/png", "image/jpeg", "image/webp",
		"font/woff2", "video/mp4", "application/pdf", "application/octet-stream",
	}
	for _, ct := range sikismayan {
		assert.Falsef(t, compressibleContentType(ct), "%q sıkışmamalı", ct)
	}
}
