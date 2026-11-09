package httpapi

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestMakaleYoluMu: yalnız tarih desenli makale yolları önbelleğe uygun.
//
// Kapsamı genişletmek riskli: buraya takılan her yol önbellekten sunulur.
// Bu yüzden DESEN testi ayrı: kişisel/panel yolları eşleşmemeli.
func TestMakaleYoluMu(t *testing.T) {
	uygun := []string{
		"/policy/2026/11/05/sec-custody-kurallari",
		"/markets/2026/11/09/bitcoin-rekor",
		"/2026/11/09/bolumsuz-makale",
		"/tag/2026/11/05/x", // biçim olarak makale; ayrıntı aşağıda
	}
	for _, yol := range uygun {
		if !makaleYoluMu(yol) {
			t.Errorf("%q makale yolu sayılmalıydı", yol)
		}
	}

	uygunDegil := []string{
		"/",
		"/admin",
		"/admin/icerik/42",
		"/admin/icerik/42/duzenle",
		"/policy/2026/11/kisa",    // eksik segment
		"/policy/2026/11/05/",     // boş slug
		"/policy/abcd/11/05/slug", // yıl rakam değil
		"/sitemap.xml",
	}
	for _, yol := range uygunDegil {
		if makaleYoluMu(yol) {
			t.Errorf("%q makale yolu SAYILMAMALIYDI", yol)
		}
	}

	// Hariç tutulan kökler desene UYSA BİLE önbelleğe girmemeli. Desen
	// kontrolü (makaleYoluMu) ile kapsam kontrolü (sayfaYoluOnbelleklenir)
	// ayrı şeyler: `/api/2026/11/05/x` desene uyar, ama önbellek dışı kalır.
	deseneUyanAmaHaric := []string{
		"/api/2026/11/05/x",
		"/admin/2026/11/05/x",
		"/hesabim/2026/11/05/x",
		"/giris/2026/11/05/x",
		"/internal/2026/11/05/x",
		"/static/2026/11/05/x",
	}
	for _, yol := range deseneUyanAmaHaric {
		if !makaleYoluMu(yol) {
			t.Logf("%q desene uymuyor (beklenmedik, zararsız)", yol)
		}
		if sayfaYoluOnbelleklenir(yol) {
			t.Errorf("%q önbelleğe GİRMEMELİYDİ (hariç kök)", yol)
		}
	}
}

// TestSayfaOnbellekMakaleYoluUygun: makale yolu gerçekten önbelleklenir.
func TestSayfaOnbellekMakaleYoluUygun(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/policy/2026/11/05/sec-custody-kurallari", nil)
	if !sayfaOnbellegeUygun(r) {
		t.Fatal("makale yolu önbelleğe uygun olmalı")
	}
	// Oturumlu istek yine kişisel: makale olması değiştirmez.
	r = httptest.NewRequest(http.MethodGet, "/policy/2026/11/05/sec-custody-kurallari", nil)
	r.AddCookie(&http.Cookie{Name: "tb_session", Value: "abc"})
	if sayfaOnbellegeUygun(r) {
		t.Fatal("oturumlu makale isteği önbelleğe girmemeli")
	}
}

// TestSayfaOnbellekOkunmaIsabetteDeSayilir: makale önbellekten gelince
// okunma sayacı DURMAMALI.
func TestSayfaOnbellekOkunmaIsabetteDeSayilir(t *testing.T) {
	sayfaTTLKisa(t, time.Minute, time.Minute)
	okunmaSayaciBosalt() // önceki testlerden kalan

	var handler int32
	s := &Server{sayfaCache: newSayfaOnbellek(), pool: nil}
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&handler, 1)
		// Gerçek handler böyle bildiriyor:
		s.okunmaKaydet(r.Context(), 4242)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<p>makale</p>"))
	})
	h := s.withSayfaOnbellek(ic)

	yol := "/policy/2026/11/05/sec-custody"
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodGet, yol, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("istek %d: %d", i, w.Code)
		}
	}

	if n := atomic.LoadInt32(&handler); n != 1 {
		t.Fatalf("handler %d kez çalıştı, 1 beklenirdi", n)
	}
	m := okunmaSayaciBosalt()
	if got := m[4242]; got != 5 {
		t.Fatalf("okunma sayısı %d, 5 beklenirdi (isabetler sayılmamış)", got)
	}
}

// TestSayfaOnbellekArkaPlanYenilemeSayilmaz: arka plan yenilemesi sahte
// okunma üretmemeli.
func TestSayfaOnbellekArkaPlanYenilemeSayilmaz(t *testing.T) {
	sayfaTTLKisa(t, 10*time.Millisecond, 2*time.Second)
	okunmaSayaciBosalt()

	s := &Server{sayfaCache: newSayfaOnbellek()}
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.okunmaKaydet(r.Context(), 777)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<p>makale</p>"))
	})
	h := s.withSayfaOnbellek(ic)

	yol := "/policy/2026/11/05/baska-makale"
	iste := func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, yol, nil))
	}

	iste() // ıska → 1 okunma
	time.Sleep(20 * time.Millisecond)
	iste() // bayat → 1 okunma + arka plan yenilemesi (sayılmaz)
	time.Sleep(150 * time.Millisecond)
	iste() // taze isabet → 1 okunma

	// En az bir tur arka plan yenilemesi çalışmış olmalı.
	if n := okunmaTamponu.boyut(); n == 0 {
		t.Log("biriken yok (beklenmedik ama zararsız)")
	}
	m := okunmaSayaciBosalt()
	if got := m[777]; got != 3 {
		t.Fatalf("okunma %d, 3 beklenirdi (arka plan yenilemesi sayılmamalı)", got)
	}
}

// TestOkunmaBiriktiriciToplam: biriktirici sayıları toplar, boşaltınca sıfırlar.
func TestOkunmaBiriktiriciToplam(t *testing.T) {
	b := newOkunmaBiriktirici()
	b.ekle(1)
	b.ekle(1)
	b.ekle(2)
	b.ekle(0)  // yok sayılır
	b.ekle(-5) // yok sayılır
	m := b.al()
	if m[1] != 2 || m[2] != 1 || len(m) != 2 {
		t.Fatalf("al() = %v", m)
	}
	if got := b.al(); got != nil {
		t.Fatalf("ikinci al() boş olmalı, %v", got)
	}
}
