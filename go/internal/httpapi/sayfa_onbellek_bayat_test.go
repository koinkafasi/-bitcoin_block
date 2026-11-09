package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// sayfaTTLKisa testler için TTL/bayat pencereyi kısaltır ve sonunda geri alır.
func sayfaTTLKisa(t *testing.T, ttl, bayat time.Duration) {
	t.Helper()
	eskiTTL, eskiBayat := sayfaOnbellekTTL, sayfaOnbellekBayatPencere
	sayfaOnbellekTTL, sayfaOnbellekBayatPencere = ttl, bayat
	t.Cleanup(func() {
		sayfaOnbellekTTL, sayfaOnbellekBayatPencere = eskiTTL, eskiBayat
	})
}

// TestSayfaOnbellekBayatHemenSunulurArkaPlandaYenilenir: TTL dolunca
// kullanıcı BEKLEMEZ — bayat gövde hemen gider, yenisi arka planda üretilir.
func TestSayfaOnbellekBayatHemenSunulurArkaPlandaYenilenir(t *testing.T) {
	sayfaTTLKisa(t, 20*time.Millisecond, 2*time.Second)

	var sayac int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&sayac, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "surum-%d %s", n, strings.Repeat("x", 100))
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	h := s.withSayfaOnbellek(ic)

	iste := func() string {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Body.String()
	}

	// 1) ilk istek: üretim
	if got := iste(); !strings.HasPrefix(got, "surum-1") {
		t.Fatalf("ilk yanıt = %q", got[:20])
	}
	// 2) TTL içinde: taze isabet, üretim YOK
	if got := iste(); !strings.HasPrefix(got, "surum-1") {
		t.Fatalf("taze isabet bekleniyordu, gelen %q", got[:20])
	}
	if n := atomic.LoadInt32(&sayac); n != 1 {
		t.Fatalf("taze yolda handler %d kez çalıştı", n)
	}

	// 3) TTL geçti (bayat pencerede): ESKİ gövde hemen dönmeli
	time.Sleep(40 * time.Millisecond)
	bas := time.Now()
	got := iste()
	gecen := time.Since(bas)
	if !strings.HasPrefix(got, "surum-1") {
		t.Fatalf("bayat gövde bekleniyordu, gelen %q", got[:20])
	}
	// Arka plan yenilemesi 20 ms sürseydi istek bloklanırdı; eşik bol tutuldu
	// ki yavaş CI'da yalancı kırmızı olmasın.
	if gecen > 500*time.Millisecond {
		t.Fatalf("bayat istek bloklandı: %v", gecen)
	}

	// 4) arka plan yenilemesi bitene kadar bekle, sonra yeni sürüm gelmeli
	son := time.Now().Add(2 * time.Second)
	for time.Now().Before(son) {
		if got := iste(); strings.HasPrefix(got, "surum-2") {
			return // başarılı
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("arka plan yenilemesi olmadı (sayac=%d)", atomic.LoadInt32(&sayac))
}

// TestSayfaOnbellekSingleflightTekUretim: önbellek boşken gelen eşzamanlı
// istekler TEK üretim yapar (TTL anındaki 50x CPU sıçramasını engeller).
func TestSayfaOnbellekSingleflightTekUretim(t *testing.T) {
	sayfaTTLKisa(t, time.Minute, time.Minute)

	var sayac int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sayac, 1)
		// Yavaş üretim: bekleyenlerin bekleme yolunu gerçekten sınamak için.
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strings.Repeat("<p>haber</p>", 50)))
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	h := s.withSayfaOnbellek(ic)

	const N = 20
	var wg sync.WaitGroup
	kodlar := make([]int, N)
	boylar := make([]int, N)
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			kodlar[i] = w.Code
			boylar[i] = w.Body.Len()
		}(i)
	}
	wg.Wait()

	if n := atomic.LoadInt32(&sayac); n != 1 {
		t.Fatalf("handler %d kez çalıştı, 1 beklenirdi (singleflight)", n)
	}
	for i := 0; i < N; i++ {
		if kodlar[i] != http.StatusOK {
			t.Fatalf("istek %d: kod %d", i, kodlar[i])
		}
		if boylar[i] != boylar[0] {
			t.Fatalf("istek %d: boyut %d != %d", i, boylar[i], boylar[0])
		}
	}
}

// TestSayfaOnbellekArkaPlanBagimsizCalisir: istek bağlamı iptal edilse bile
// arka plan yenilemesi tamamlanır (istemci gitse de önbellek tazelenir).
func TestSayfaOnbellekArkaPlanBagimsizCalisir(t *testing.T) {
	sayfaTTLKisa(t, 10*time.Millisecond, 2*time.Second)

	var sayac int32
	ic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&sayac, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "surum-%d", n)
	})
	s := &Server{sayfaCache: newSayfaOnbellek()}
	s.blockAt = time.Now()
	s.blockCache = map[string]bool{}
	h := s.withSayfaOnbellek(ic)

	// ilk üretim
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(httptest.NewRecorder(), r)

	time.Sleep(20 * time.Millisecond)

	// bayat istek: bağlam HEMEN iptal ediliyor
	ctx, iptal := context.WithCancel(context.Background())
	r2 := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	iptal()

	if w2.Code != http.StatusOK {
		t.Fatalf("bayat yanıt kodu %d", w2.Code)
	}
	// arka plan tamamlanmalı
	son := time.Now().Add(2 * time.Second)
	for time.Now().Before(son) {
		if atomic.LoadInt32(&sayac) >= 2 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("iptal edilen istek arka plan yenilemesini durdurdu")
}
