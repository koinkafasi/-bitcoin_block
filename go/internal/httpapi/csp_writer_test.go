package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSPWriterDurumKodu CSP tamponlayıcısı handler'ın bildirdiği durum
// kodunu KAYBETMEMELİ. Regresyon: ilk uygulamada WriteHeader yalnız tampona
// yazılıyor, alttaki yazıcıya hiç geçmiyordu; sonuç: 204/302/404 yanıtları
// 200 olarak dönüyordu.
func TestCSPWriterDurumKodu(t *testing.T) {
	cases := []struct {
		ad      string
		handler http.HandlerFunc
		want    int
		govde   string
	}{
		{
			ad: "204 gövdesiz",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			want: http.StatusNoContent,
		},
		{
			ad: "404 metinli",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("yok"))
			},
			want:  http.StatusNotFound,
			govde: "yok",
		},
		{
			ad: "302 yönlendirme",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/admin/giris")
				w.WriteHeader(http.StatusFound)
			},
			want: http.StatusFound,
		},
		{
			ad: "200 HTML",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte("<html><script>x()</script></html>"))
			},
			want:  http.StatusOK,
			govde: "<script>x()</script>",
		},
		{
			ad: "500 hata",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("<html>hata</html>"))
			},
			want:  http.StatusInternalServerError,
			govde: "hata",
		},
	}

	for _, c := range cases {
		t.Run(c.ad, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s := &Server{}
			h := s.withHTMLCSP(c.handler)
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

			if rec.Code != c.want {
				t.Fatalf("durum kodu = %d, beklenen %d (CSP tamponu kodu kaybetti)", rec.Code, c.want)
			}
			if c.govde != "" && !strings.Contains(rec.Body.String(), c.govde) {
				t.Errorf("gövde beklenen parçayı içermiyor: %q", rec.Body.String())
			}
		})
	}
}

// TestCSPWriterHTMLHash CSP başlığı HTML yanıtta gövdeyle TUTARLI olmalı.
func TestCSPWriterHTMLHash(t *testing.T) {
	rec := httptest.NewRecorder()
	s := &Server{}
	html := `<html><script>var a=1;</script><script><\/script></html>`
	h := s.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("CSP başlığı yok")
	}
	want := hashOf("var a=1;")
	if !strings.Contains(csp, want) {
		t.Errorf("gövdedeki script'in hash'i başlıkta yok\n hash: %s\n csp: %s", want, csp)
	}
	if strings.Contains(csp, "nonce-") {
		t.Error("önbellek güvenliği için nonce kullanılmamalı")
	}
}

// TestCSPWriterHTMLOlmeyan HTML olmayan yanıtlara CSP hash'i eklenmemeli.
func TestCSPWriterHTMLOlmeyan(t *testing.T) {
	rec := httptest.NewRecorder()
	s := &Server{}
	h := s.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api", nil))
	if strings.Contains(rec.Header().Get("Content-Security-Policy"), "sha256-") {
		t.Errorf("JSON yanıtına script hash'i eklenmemeli: %s", rec.Header().Get("Content-Security-Policy"))
	}
}
