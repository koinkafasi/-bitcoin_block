package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testLogger sessiz günlükçü (test çıktısını kirletmesin).
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestRecoverPanicYakalar handler panic'i istemciye 500 olarak dönmeli.
//
// Regresyon: bu ara katman yokken Go'nun net/http'si bağlantıyı SIFIRLIYOR,
// istemci 500 sayfası değil kopuk bağlantı görüyordu.
func TestRecoverPanicYakalar(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("kasıtlı test panic")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bir-yer", nil)
	req.Header.Set("Accept", "text/html")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d, beklenen 500", rec.Code)
	}
	govde := rec.Body.String()
	if strings.Contains(govde, "kasıtlı test panic") {
		t.Error("panic metni istemciye SIZDI")
	}
	if !strings.Contains(govde, "Beklenmeyen bir hata") {
		t.Errorf("hata sayfası dönmedi: %q", govde[:minInt(120, len(govde))])
	}
}

// TestRecoverAPIJSON API yollarında JSON dönmeli (HTML değil).
func TestRecoverAPIJSON(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("db patladı")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/trend-suggest", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, JSON beklenirdi", ct)
	}
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Errorf("JSON gövde beklenirdi: %q", rec.Body.String())
	}
}

// TestRecoverHTMXFragment HTMX isteğinde 500 + HX-Trigger toast dönmeli.
func TestRecoverHTMXFragment(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panel patladı")
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/icerikler", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d", rec.Code)
	}
	if trg := rec.Header().Get("HX-Trigger"); !strings.Contains(trg, "tbToast") {
		t.Errorf("HX-Trigger toast yok: %q", trg)
	}
	if !strings.Contains(rec.Body.String(), "flash error") {
		t.Errorf("fragment yok: %q", rec.Body.String())
	}
}

// TestRecoverNormalAkisDokunmaz panic yoksa yanıt AYNEN geçmeli.
func TestRecoverNormalAkisDokunmaz(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "1")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("çay"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("durum = %d, beklenen 418", rec.Code)
	}
	if rec.Header().Get("X-Test") != "1" {
		t.Error("başlık kayboldu")
	}
	if rec.Body.String() != "çay" {
		t.Errorf("gövde = %q", rec.Body.String())
	}
}

// TestRecoverErrAbortHandlerYenidenFirlatilir SSE iptali normal davranış
// olmalı; 500'e çevrilmemeli.
func TestRecoverErrAbortHandlerYenidenFirlatilir(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	h := s.withRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("ErrAbortHandler yeniden fırlatılmalıydı (net/http kendi işlesin)")
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/canli/icerik/1", nil))
}

// TestReadyzHazir SetReady(true) iken 200 dönmeli.
// DB ping'i httptest'te gerçek havuz gerektirdiği için bu test yalnız
// "draining" dalını ve durum makinesini kapsar.
func TestReadyzDurumMakinesi(t *testing.T) {
	SetReady(true)
	if !Ready() {
		t.Fatal("SetReady(true) sonrası Ready() false")
	}
	SetReady(false)
	if Ready() {
		t.Fatal("SetReady(false) sonrası Ready() true")
	}
	SetReady(true) // diğer testleri etkilemesin
}

// TestReadyzDraining 503 + "draining" dönmeli (havuz nil olsa bile
// draining dalı ping'e ULAŞMADAN döner).
func TestReadyzDraining(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	SetReady(false)
	defer SetReady(true)

	rec := httptest.NewRecorder()
	s.handleReadyz(rec, httptest.NewRequest("GET", "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("durum = %d, beklenen 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "draining") {
		t.Errorf("gövde = %q", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("readyz önbelleklenmemeli")
	}
}
