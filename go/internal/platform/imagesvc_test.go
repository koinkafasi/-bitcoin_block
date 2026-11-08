package platform

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
)

// testJPEG küçük ama gerçek bir JPEG üretir (kırpma/ölçekleme yolu çalışsın).
func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("test jpeg: %v", err)
	}
	return buf.Bytes()
}

// TestResizeToWidthServisAdresiniKullanir: adres tanımlıyken görsel işi
// servise gider ve dönen baytlar aynen kullanılır.
func TestResizeToWidthServisAdresiniKullanir(t *testing.T) {
	var cagri int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cagri, 1)
		if r.Method != http.MethodPost {
			t.Errorf("metot = %s, POST bekleniyordu", r.Method)
		}
		q := r.URL.Query()
		if q.Get("format") != "jpeg" {
			t.Errorf("format = %q", q.Get("format"))
		}
		if q.Get("quality") != "82" {
			t.Errorf("quality = %q, yerel yolun q82'si bekleniyordu", q.Get("quality"))
		}
		wantW, _ := strconv.Atoi(q.Get("width"))
		if wantW != 400 {
			t.Errorf("width = %q", q.Get("width"))
		}
		// 800x400 girdi → ratio 2 olmalı (16:9 sabit değil).
		if got := q.Get("ratio"); got != "2" {
			t.Errorf("ratio = %q, 2 bekleniyordu", got)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("SERVIS-CIKTISI"))
	}))
	defer srv.Close()

	SetImageSvcURL(srv.URL)
	defer SetImageSvcURL("")

	out, err := ResizeToWidth(testJPEG(t, 800, 400), 400)
	if err != nil {
		t.Fatalf("ResizeToWidth: %v", err)
	}
	if string(out) != "SERVIS-CIKTISI" {
		t.Fatalf("servis çıktısı kullanılmadı: %q", out)
	}
	if atomic.LoadInt32(&cagri) != 1 {
		t.Fatalf("servis çağrı sayısı = %d", cagri)
	}
}

// TestImageSvcHataYerelYolaDuser: servis 500 dönerse görsel üretimi durmaz,
// yerel Go yolu devreye girer.
func TestImageSvcHataYerelYolaDuser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	SetImageSvcURL(srv.URL)
	defer SetImageSvcURL("")

	out, err := ResizeToWidth(testJPEG(t, 800, 400), 400)
	if err != nil {
		t.Fatalf("yerel yola düşmeliydi, hata: %v", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("çıktı JPEG değil: %v", err)
	}
	if cfg.Width != 400 {
		t.Fatalf("genişlik = %d, 400 bekleniyordu", cfg.Width)
	}
}

// TestImageSvcHatadanSonraTekrarDenenmez: hata sonrası devre açılır; ölü
// servis her görselde 5 sn zaman aşımı eklemesin (kazanç kaybı olurdu).
func TestImageSvcHatadanSonraTekrarDenenmez(t *testing.T) {
	var cagri int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cagri, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	SetImageSvcURL(srv.URL)
	defer SetImageSvcURL("")

	img := testJPEG(t, 800, 400)
	if _, err := ResizeToWidth(img, 400); err != nil {
		t.Fatalf("ilk çağrı yerel yola düşmeliydi: %v", err)
	}
	if _, err := ResizeToWidth(img, 400); err != nil {
		t.Fatalf("ikinci çağrı: %v", err)
	}
	if n := atomic.LoadInt32(&cagri); n != 1 {
		t.Fatalf("devre açıkken servise %d kez gidildi, 1 beklenirdi", n)
	}
}

// TestImageSvcPNGYereldeKalir: saydamlık Rust tarafında beyaz zemine
// çevrilmediği için PNG girdi servise gönderilmez.
func TestImageSvcPNGYereldeKalir(t *testing.T) {
	var cagri int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cagri, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	SetImageSvcURL(srv.URL)
	defer SetImageSvcURL("")

	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 64)...)
	if _, ok := resizeViaImageSvc(t.Context(), png, 100); ok {
		t.Fatal("PNG servise gönderilmemeliydi")
	}
	if atomic.LoadInt32(&cagri) != 0 {
		t.Fatalf("servise %d kez gidildi", cagri)
	}
}

// TestImageSvcKapaliVarsayilan: adres boşken hiçbir çağrı yapılmaz.
func TestImageSvcKapaliVarsayilan(t *testing.T) {
	SetImageSvcURL("")
	if ImageSvcURL() != "" {
		t.Fatal("adres boş olmalıydı")
	}
	if _, ok := resizeViaImageSvc(t.Context(), testJPEG(t, 800, 400), 400); ok {
		t.Fatal("servis kapalıyken kullanılmamalı")
	}
}
