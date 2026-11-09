package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSPHashCacheAyniGovdeAyniBaslik: önbellek isabetinde başlık birebir
// aynı kalır — CSP'de tek bayt farkı tüm script'leri düşürür.
func TestCSPHashCacheAyniGovdeAyniBaslik(t *testing.T) {
	govde := `<!doctype html><html><head><style>.a{color:red}</style></head>
	<body><script>console.log("merhaba")</script></body></html>`

	w1 := httptest.NewRecorder()
	c1 := &cspWriter{w: w1, raporURL: "", hashes: newCSPHashCache()}
	c1.buf.WriteString(govde)
	c1.finish()

	w2 := httptest.NewRecorder()
	c2 := &cspWriter{w: w2, raporURL: "", hashes: c1.hashes}
	c2.buf.WriteString(govde)
	c2.finish()

	h1 := w1.Header().Get("Content-Security-Policy")
	h2 := w2.Header().Get("Content-Security-Policy")
	if h1 == "" || h1 != h2 {
		t.Fatalf("başlıklar farklı:\n1: %s\n2: %s", h1, h2)
	}
	// Hash gerçekten gövdeden mi geldi: script içeriği değişince başlık da
	// değişmeli.
	w3 := httptest.NewRecorder()
	c3 := &cspWriter{w: w3, raporURL: "", hashes: c1.hashes}
	c3.buf.WriteString(strings.Replace(govde, "merhaba", "dunya", 1))
	c3.finish()
	if h3 := w3.Header().Get("Content-Security-Policy"); h3 == h1 {
		t.Fatal("gövde değişti ama başlık aynı kaldı — önbellek yanlış anahtar kullanıyor")
	}
}

// TestCSPHashCacheKapasiteTahliye: sınır aşılınca en eski giriş düşer,
// önbellek süreç ömrü boyunca büyümez.
func TestCSPHashCacheKapasiteTahliye(t *testing.T) {
	hc := newCSPHashCache()
	hc.limit = 3
	for i := 0; i < 5; i++ {
		hc.Set(strings.Repeat("x", 100)+string(rune('a'+i)), "baslik")
	}
	if got := hc.Len(); got != 3 {
		t.Fatalf("uzunluk = %d, 3 beklenirdi", got)
	}
	// İlk iki girmiş olmalı (en yeniler kalır).
	if _, ok := hc.Get(strings.Repeat("x", 100) + "a"); ok {
		t.Fatal("en eski giriş tahliye edilmemiş")
	}
	if _, ok := hc.Get(strings.Repeat("x", 100) + "e"); !ok {
		t.Fatal("en yeni giriş kayboldu")
	}
}

// TestCSPHashCacheIskalamadaDegismez: önbellek kapalıyken (nil) davranış
// eskisiyle aynı — başlık yine üretilir.
func TestCSPHashCacheIskalamadaDegismez(t *testing.T) {
	govde := `<html><body><script>alert(1)</script></body></html>`
	w := httptest.NewRecorder()
	c := &cspWriter{w: w, raporURL: "", hashes: nil}
	c.buf.WriteString(govde)
	c.finish()
	if h := w.Header().Get("Content-Security-Policy"); h == "" || !strings.Contains(h, "script-src") {
		t.Fatalf("başlık üretilmedi: %q", h)
	}
}

// TestCSPHashCacheUretimMiddlewareIsabet: gerçek zincirde (withHTMLCSP)
// ikinci istek önbellekten gelir ve başlık aynıdır.
func TestCSPHashCacheUretimMiddlewareIsabet(t *testing.T) {
	var cagri int
	icerik := []byte(`<html><head><title>t</title></head><body><script>var a=1;</script></body></html>`)
	ust := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cagri++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(icerik)
	})

	s := &Server{cspHashes: newCSPHashCache()}
	katman := s.withHTMLCSP(ust)

	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	w1 := httptest.NewRecorder()
	katman.ServeHTTP(w1, r1)

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	katman.ServeHTTP(w2, r2)

	if w1.Header().Get("Content-Security-Policy") != w2.Header().Get("Content-Security-Policy") {
		t.Fatal("iki istekte başlıklar farklı")
	}
	if n := s.cspHashes.Len(); n != 1 {
		t.Fatalf("önbellekte %d giriş var, 1 beklenirdi", n)
	}
}
