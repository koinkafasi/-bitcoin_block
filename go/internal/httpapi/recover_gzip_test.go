package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// NOT: Bu testler `s.disKatman` üzerinden ÜRETİMDEKİ ara katman sırasını
// sınar. Kendi zincirini kursaydı (s.withGzip(s.withRecover(...)) gibi),
// üretimdeki sıra bozulsa bile test geçerdi — bu tuzak bir kez yaşandı ve
// testin değersiz olduğu ölçümle anlaşıldı.

// TestPanicGzipCercevesiBozulmaz
//
// KRİTİK: withRecover, withGzip'in İÇİNDE olmalı. Dışında olsaydı panic
// sonrası yazılan düz metin, `Content-Encoding: gzip` başlığıyla birlikte
// HAM olarak giderdi ve istemci bozuk yanıt alırdı.
//
// En zor senaryo: başlık ve bir parça gövde yazıldıktan SONRA panic.
func TestPanicGzipCercevesiBozulmaz(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.disKatman(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<p>kısmi"))
		panic("ortada panik")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bir-yer", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rec, req)

	if enc := rec.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("Content-Encoding = %q, gzip beklenirdi", enc)
	}

	// Gövde geçerli gzip olmalı: çerçeve bozulmamış.
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gövde geçerli gzip değil (çerçeve bozuk): %v", err)
	}
	acik, err := io.ReadAll(zr)
	if err != nil {
		// Sıra yanlışsa tam burada patlar: gzip: invalid header
		t.Fatalf("gzip akışı bozuk (ara katman sırası yanlış olabilir): %v", err)
	}
	if len(acik) == 0 {
		t.Error("açılan gövde boş")
	}
	// Başlık zaten yazıldığı için tam hata sayfası EKLENMEZ; kısa işaret yeter.
	if !strings.Contains(string(acik), "kısmi") {
		t.Errorf("kısmi gövde kayboldu: %q", string(acik))
	}
}

// TestPanicYazilmamisGzipTamSayfa Başlık yazılmadan panic olursa gzip'li
// tam hata sayfası dönmeli ve açılabilmeli.
func TestPanicYazilmamisGzipTamSayfa(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.disKatman(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("hiç yazmadan panik")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sayfa", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d, beklenen 500", rec.Code)
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gövde geçerli gzip değil: %v", err)
	}
	acik, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip akışı bozuk: %v", err)
	}
	if !strings.Contains(string(acik), "Beklenmeyen bir hata") {
		t.Errorf("hata sayfası yok: %q", string(acik)[:minInt(80, len(acik))])
	}
}

// TestDisKatmanPanicYakalar üretim zincirinin panic'i gerçekten yakaladığını
// ve 500 döndürdüğünü doğrular (gzip istemeyen istemci).
func TestDisKatmanPanicYakalar(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.disKatman(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("patladı")
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Accept", "text/html")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d, beklenen 500 (panic yakalanmadı)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "patladı") {
		t.Error("panic metni istemciye sızdı")
	}
}
