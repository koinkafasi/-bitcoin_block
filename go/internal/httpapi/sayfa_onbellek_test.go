package httpapi

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"theblock/site/internal/auth"
)

// sayfaTestSunucu önbellek + gzip zincirini kurar ve kaç kez render
// edildiğini sayar.
func sayfaTestSunucu(govde string, basliklar map[string]string) (*Server, http.Handler, *int32) {
	var sayac int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sayac, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		for k, v := range basliklar {
			w.Header().Set(k, v)
		}
		_, _ = w.Write([]byte(govde))
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	// Blok listesi "taze" işaretlenir: isIPBlocked DB'ye gitmesin (bu test
	// önbellek davranışını sınıyor, blok listesini değil).
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	return s, s.withSayfaOnbellek(s.withGzip(ic)), &sayac
}

func sayfaIste(h http.Handler, yol string, cerezler ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, yol, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	for _, c := range cerezler {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// TestSayfaOnbellekIsabetRenderTekrarlanmaz: ikinci istek önbellekten gelir,
// handler bir kez çalışır. Kazancın tamamı buna dayanıyor.
func TestSayfaOnbellekIsabetRenderTekrarlanmaz(t *testing.T) {
	govde := strings.Repeat("<p>kripto haber</p>", 50)
	_, h, sayac := sayfaTestSunucu(govde, nil)

	w1 := sayfaIste(h, "/")
	w2 := sayfaIste(h, "/")

	if n := atomic.LoadInt32(sayac); n != 1 {
		t.Fatalf("handler %d kez çalıştı, 1 beklenirdi", n)
	}
	if w1.Body.Len() != w2.Body.Len() {
		t.Fatalf("gövdeler farklı boyutta: %d vs %d", w1.Body.Len(), w2.Body.Len())
	}
	// İkisi de geçerli gzip olmalı.
	for i, w := range []*httptest.ResponseRecorder{w1, w2} {
		zr, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
		if err != nil {
			t.Fatalf("%d. yanıt gzip değil: %v", i+1, err)
		}
		coz, _ := io.ReadAll(zr)
		_ = zr.Close()
		if string(coz) != govde {
			t.Fatalf("%d. yanıt gövdesi bozuk", i+1)
		}
	}
}

// TestSayfaOnbellekOturumluIstekKisisel: oturum çerezi olan istek önbelleğe
// alınmaz ve önbellekten okunmaz (kullanıcı adı sızmasın).
func TestSayfaOnbellekOturumluIstekKisisel(t *testing.T) {
	_, h, sayac := sayfaTestSunucu("sayfa", nil)

	sayfaIste(h, "/") // anonim: önbelleğe girer
	oturum := &http.Cookie{Name: auth.CookieName, Value: "gecerli-oturum"}
	sayfaIste(h, "/", oturum) // oturumlu: önbelleği ATLAMALI
	sayfaIste(h, "/", oturum)

	if n := atomic.LoadInt32(sayac); n != 3 {
		t.Fatalf("handler %d kez çalıştı, 3 beklenirdi (anonim 1 + oturumlu 2)", n)
	}
}

// TestSayfaOnbellekIstekKimligiTekrarlanmaz: X-Request-Id önbellekten
// gelmemeli — gelseydi tüm istekler aynı kimliği taşırdı (log/tracing bozulur).
func TestSayfaOnbellekIstekKimligiTekrarlanmaz(t *testing.T) {
	_, h, _ := sayfaTestSunucu("sayfa", nil)

	sayfaIste(h, "/") // ıska: önbelleği doldurur
	w1 := sayfaIste(h, "/")
	w2 := sayfaIste(h, "/")
	id1, id2 := w1.Header().Get("X-Request-Id"), w2.Header().Get("X-Request-Id")
	if id1 == "" || id2 == "" {
		t.Fatalf("önbellekten gelen yanıtta istek kimliği boş: %q %q", id1, id2)
	}
	if id1 == id2 {
		t.Fatalf("iki önbellek isabeti aynı kimliği taşıyor: %q", id1)
	}
}

// TestSayfaOnbellekCerezYazanYanitSaklanmaz: Set-Cookie içeren yanıt
// önbelleğe girerse oturum çerezi herkese dağılırdı.
func TestSayfaOnbellekCerezYazanYanitSaklanmaz(t *testing.T) {
	_, h, sayac := sayfaTestSunucu("sayfa", map[string]string{"Set-Cookie": "tb_session=abc; Path=/"})
	sayfaIste(h, "/")
	sayfaIste(h, "/")
	if n := atomic.LoadInt32(sayac); n != 2 {
		t.Fatalf("Set-Cookie'li yanıt önbelleğe alınmış (handler %d kez)", n)
	}
}

// TestSayfaOnbellekHataSaklanmaz: 200 dışı yanıtlar önbelleğe girmez.
func TestSayfaOnbellekHataSaklanmaz(t *testing.T) {
	var n int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("hata"))
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	// Blok listesi "taze": isIPBlocked DB'ye gitmesin.
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	h := s.withSayfaOnbellek(s.withGzip(ic))

	sayfaIste(h, "/")
	sayfaIste(h, "/")
	if got := atomic.LoadInt32(&n); got != 2 {
		t.Fatalf("500 yanıtı önbelleğe alınmış (%d kez)", got)
	}
}

// TestSayfaOnbellekYolVeMetotFiltresi: yalnız herkese açık CD sayfaları ve
// yalnız GET önbelleklenir.
func TestSayfaOnbellekYolVeMetotFiltresi(t *testing.T) {
	evet := []string{"/", "/cd", "/markets", "/business", "/price", "/price/bitcoin",
		"/tag/bitcoin", "/videos", "/canli-yayin", "/podcasts", "/newsletters"}
	for _, y := range evet {
		if !sayfaYoluOnbelleklenir(y) {
			t.Errorf("%q önbelleklenmeliydi", y)
		}
	}
	hayir := []string{"/admin", "/admin/icerikler", "/api/x", "/media/thumb/1",
		"/kg/abc", "/giris", "/hesabim", "/arama", "/sitemap.xml",
		"/robots.txt", "/healthz", "/readyz", "/klasik", "/bilinmeyen-bolum"}
	for _, y := range hayir {
		if sayfaYoluOnbelleklenir(y) {
			t.Errorf("%q önbelleklenmemeliydi", y)
		}
	}

	// Metot filtresi
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if sayfaOnbellegeUygun(r) {
		t.Error("POST önbelleğe uygun olmamalı")
	}
	// no-cache başlığı
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Cache-Control", "no-cache")
	if sayfaOnbellegeUygun(r) {
		t.Error("no-cache isteği önbelleğe uygun olmamalı")
	}
	// HTMX/parça istekleri aynı yolda farklı gövde döndürebilir.
	r = httptest.NewRequest(http.MethodGet, "/markets?parca=1", nil)
	r.Header.Set("HX-Request", "true")
	if sayfaOnbellegeUygun(r) {
		t.Error("HX-Request isteği önbelleğe uygun olmamalı")
	}
}

// TestSayfaOnbellekGzipVeHamAyriSaklanir: gzip kabul etmeyen istemciye
// sıkıştırılmış bayt gitmemeli.
func TestSayfaOnbellekGzipVeHamAyriSaklanir(t *testing.T) {
	// .html içerik türü sıkıştırılabilir olmalı ki gzip yolu çalışsın.
	_, h, _ := sayfaTestSunucu(strings.Repeat("<p>x</p>", 40), nil)

	// önce gzip'li istek: önbelleğe gzip girdisi
	wg := sayfaIste(h, "/")

	// sonra ham istek (Accept-Encoding yok)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	wh := httptest.NewRecorder()
	h.ServeHTTP(wh, r)

	if enc := wh.Header().Get("Content-Encoding"); enc != "" {
		t.Fatalf("ham istemciye Content-Encoding: %q geldi", enc)
	}
	if !strings.Contains(wh.Body.String(), "<p>x</p>") {
		t.Fatalf("ham gövde bozuk: %q", wh.Body.String()[:min(60, wh.Body.Len())])
	}
	if enc := wg.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("gzip istemciye Content-Encoding = %q", enc)
	}
}

// TestSayfaOnbellekCSPKorunur: replay'de CSP başlığı aynen gitmeli (aksi
// hâlde önbellekten gelen sayfada satır içi script'ler engellenir).
func TestSayfaOnbellekCSPKorunur(t *testing.T) {
	const csp = "default-src 'self'; script-src 'self' 'sha256-abc='"
	_, h, _ := sayfaTestSunucu("sayfa", map[string]string{"Content-Security-Policy": csp})

	w1 := sayfaIste(h, "/")
	w2 := sayfaIste(h, "/")
	if w1.Header().Get("Content-Security-Policy") != csp {
		t.Fatalf("1. yanıt CSP = %q", w1.Header().Get("Content-Security-Policy"))
	}
	if w2.Header().Get("Content-Security-Policy") != csp {
		t.Fatalf("önbellekten gelen yanıt CSP = %q", w2.Header().Get("Content-Security-Policy"))
	}
}

func TestSayfaOnbellekBlokluIPIsabetteDeReddedilir(t *testing.T) {
	govde := strings.Repeat("<p>haber</p>", 20)
	var sayac int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sayac, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(govde))
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	// Blok listesi hazır: blockAt taze olduğu için DB'ye gidilmez.
	s.blockCache = map[string]bool{"203.0.113.7": true}
	s.blockAt = time.Now()
	h := s.withSayfaOnbellek(s.withGzip(ic))

	// Önce temiz IP ile önbelleği doldur.
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.Header.Set("Accept-Encoding", "gzip")
	r1.RemoteAddr = "198.51.100.1:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("temiz IP: %d", w1.Code)
	}
	if n := atomic.LoadInt32(&sayac); n != 1 {
		t.Fatalf("handler %d kez çalıştı, 1 beklenirdi", n)
	}

	// Aynı sayfa, BLOKLU IP: önbellekte olmasına rağmen 403 dönmeli.
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("Accept-Encoding", "gzip")
	r2.RemoteAddr = "203.0.113.7:5555"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("bloklu IP %d aldı, 403 beklenirdi (önbellek bloğu ATLAMIŞ)", w2.Code)
	}
	if w2.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("403 yanıtında Cache-Control = %q", w2.Header().Get("Cache-Control"))
	}
}
