package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGecerliUUID: çerezden gelen kimlik doğrulanmalı.
//
// NEDEN: bozuk bir çerez değeri doğrudan SQL'e `$1::uuid` olarak gidiyor;
// doğrulanmazsa her istekte veritabanı hatası üretir (log seli).
func TestGecerliUUID(t *testing.T) {
	gecerli := []string{
		"d1ec26c8-9d81-46d4-a959-ad76c74f2333",
		"D1EC26C8-9D81-46D4-A959-AD76C74F2333",
	}
	for _, s := range gecerli {
		if !gecerliUUID(s) {
			t.Errorf("geçerli sayılmalıydı: %q", s)
		}
	}
	gecersiz := []string{
		"", "abc", "d1ec26c8-9d81-46d4-a959-ad76c74f233",
		"d1ec26c8x9d81-46d4-a959-ad76c74f2333",
		"'; DROP TABLE visitor_locations; --",
		"d1ec26c8-9d81-46d4-a959-ad76c74f233g", // g hex değil
	}
	for _, s := range gecersiz {
		if gecerliUUID(s) {
			t.Errorf("reddedilmeliydi: %q", s)
		}
	}
}

// TestYeniUUIDTekil: %1000 aynı kimlik üretilmemeli.
func TestYeniUUIDTekil(t *testing.T) {
	gorulen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := yeniUUID()
		if id == "" {
			t.Fatal("kimlik üretilemedi")
		}
		if gorulen[id] {
			t.Fatalf("yinelenen kimlik: %s", id)
		}
		gorulen[id] = true
	}
}

// TestCfUlke: yalnız gerçek ISO-2 ülke kodları kabul edilir.
func TestCfUlke(t *testing.T) {
	haller := map[string]string{
		"TR": "TR", "de": "DE", " US ": "US",
		"XX": "", "T1": "", "": "", "TUR": "", "T!": "", "12": "",
	}
	for girdi, beklenen := range haller {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if girdi != "" {
			r.Header.Set("CF-IPCountry", girdi)
		}
		if got := cfUlke(r); got != beklenen {
			t.Errorf("%q -> %q, beklenen %q", girdi, got, beklenen)
		}
	}
}

// TestMaskeliIP: panelde tam IP görünmemeli.
func TestMaskeliIP(t *testing.T) {
	haller := map[string]string{
		"88.11.22.33":      "88.11.22.xxx",
		"2001:db8:abcd::1": "2001:db8::xxxx",
		"bozuk":            "bozuk",
		"":                 "",
	}
	for girdi, beklenen := range haller {
		if got := maskeliIP(girdi); got != beklenen {
			t.Errorf("%q -> %q, beklenen %q", girdi, got, beklenen)
		}
	}
}

// TestYuvarla4: koordinat ~11 m'ye yuvarlanır, işaret korunur.
func TestYuvarla4(t *testing.T) {
	haller := map[float64]float64{
		41.0082376:  41.0082,
		28.9783589:  28.9784,
		-33.8688197: -33.8688,
		-0.00004:    -0.0,
		0:           0,
	}
	for girdi, beklenen := range haller {
		if got := yuvarla4(girdi); got != beklenen {
			t.Errorf("%v -> %v, beklenen %v", girdi, got, beklenen)
		}
	}
}

// TestKonumSayfasiMi: konum kaydı yalnız belge isteklerinde çalışmalı.
//
// NEDEN ÖNEMLİ: statik dosya/API isteklerinde de kayıt yapılırsa bir sayfa
// görüntülemesi 30+ DB yazımına dönüşür.
func TestKonumSayfasiMi(t *testing.T) {
	sayfa := []string{"/", "/markets", "/en/", "/tag/bitcoin", "/2026/11/13/haber",
		"/price/bitcoin", "/iletisim"}
	for _, p := range sayfa {
		if !konumSayfasiMi(p) {
			t.Errorf("sayfa sayılmalıydı: %q", p)
		}
	}
	sayfaDegil := []string{"/static/cd.css", "/media/a.jpg", "/kg/abc",
		"/api/konum", "/admin", "/admin/ziyaretciler", "/sitemap.xml",
		"/robots.txt", "/ads.txt", "/favicon.ico", "/manifest.webmanifest",
		"/canli/piyasa", "/canli/icerik/5"}
	for _, p := range sayfaDegil {
		if konumSayfasiMi(p) {
			t.Errorf("sayfa SAYILMAMALIYDI: %q", p)
		}
	}
}

// TestKonumDurumu: çerez durumunu doğru okur.
func TestKonumDurumu(t *testing.T) {
	haller := map[string]string{"1": "izin", "0": "red", "": "", "x": ""}
	for deger, beklenen := range haller {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if deger != "" {
			r.AddCookie(&http.Cookie{Name: konumCookie, Value: deger})
		}
		if got := konumDurumu(r); got != beklenen {
			t.Errorf("%q -> %q, beklenen %q", deger, got, beklenen)
		}
	}
}

// TestKonumAPIValidation: uç nokta aralık dışı/eksik veriyi reddetmeli.
//
// Sunucu bu değerleri doğrudan DB'ye yazıyor; doğrulama olmadan
// "enlem=9999" gibi çöp kayıtlar panele düşer.
func TestKonumAPIValidation(t *testing.T) {
	s := &Server{logger: testLogger(t), pool: nil}
	haller := []struct {
		govde string
		kod   int
	}{
		{`{"enlem":9999,"boylam":28}`, http.StatusBadRequest},
		{`{"enlem":41,"boylam":9999}`, http.StatusBadRequest},
		{`{"boylam":28}`, http.StatusBadRequest},
		{`{"enlem":41}`, http.StatusBadRequest},
		{`bozuk json`, http.StatusBadRequest},
	}
	for _, h := range haller {
		r := httptest.NewRequest(http.MethodPost, "/api/konum", strings.NewReader(h.govde))
		w := httptest.NewRecorder()
		s.konumAPI(w, r)
		if w.Code != h.kod {
			t.Errorf("%s -> %d, beklenen %d", h.govde, w.Code, h.kod)
		}
	}
}

// TestKonumAPIGetReddedilir: yalnız POST.
func TestKonumAPIGetReddedilir(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	w := httptest.NewRecorder()
	s.konumAPI(w, httptest.NewRequest(http.MethodGet, "/api/konum", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET -> %d, beklenen 405", w.Code)
	}
}

// TestZiyaretciCerezYazilir: ilk istekte kimlik üretilip çerezlenmeli.
func TestZiyaretciCerezYazilir(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	id := s.ziyaretciID(w, r, true)
	if !gecerliUUID(id) {
		t.Fatalf("geçerli kimlik üretilmedi: %q", id)
	}
	cerezler := w.Result().Cookies()
	var bulundu bool
	for _, c := range cerezler {
		if c.Name == vidCookie && c.Value == id {
			bulundu = true
			if !c.HttpOnly {
				t.Error("ziyaretçi çerezi HttpOnly olmalı")
			}
		}
	}
	if !bulundu {
		t.Fatalf("%s çerezi yazılmadı: %v", vidCookie, cerezler)
	}

	// İkinci istekte AYNI kimlik dönmeli (yeni üretilmemeli).
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.AddCookie(&http.Cookie{Name: vidCookie, Value: id})
	w2 := httptest.NewRecorder()
	if got := s.ziyaretciID(w2, r2, true); got != id {
		t.Fatalf("kimlik değişti: %q != %q", got, id)
	}
}
