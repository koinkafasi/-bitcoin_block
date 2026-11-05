package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnsplashSearchWithoutKey(t *testing.T) {
	t.Setenv("UNSPLASH_ACCESS_KEY", "")
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/medya/unsplash?q=haber", nil)
	s.handleUnsplashSearch(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("durum = %d, beklenen 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "UNSPLASH_ACCESS_KEY") {
		t.Fatalf("yanıt anahtar uyarısı içermeli: %s", rec.Body.String())
	}
}

func TestUnsplashSearchRequiresQuery(t *testing.T) {
	t.Setenv("UNSPLASH_ACCESS_KEY", "test-key")
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/medya/unsplash", nil)
	s.handleUnsplashSearch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("durum = %d, beklenen 400", rec.Code)
	}
}
