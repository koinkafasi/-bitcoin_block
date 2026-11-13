package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// insanIstegi gerçek bir tarayıcı bağlantı tıklamasını taklit eder:
// Chrome'un gönderdiği başlıkların birebir küçük bir alt kümesi.
func insanIstegi(ip, adPath string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, adPath, nil)
	r.RemoteAddr = ip + ":54321"
	r.Host = "bitcosar.com"
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	r.Header.Set("Accept-Language", "tr-TR,tr;q=0.9,en;q=0.8")
	r.Header.Set("Referer", "https://bitcosar.com/markets")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Sec-Fetch-Mode", "navigate")
	r.Header.Set("Sec-Fetch-Dest", "document")
	r.Header.Set("Sec-Fetch-User", "?1")
	return r
}

// TestReklamTiklamaInsanSayilir: normal tarayıcı tıklaması SAYILIR.
// (Koruma meşru geliri kesmemeli — en kritik test.)
func TestReklamTiklamaInsanSayilir(t *testing.T) {
	k := newReklamKoruma()
	r := insanIstegi("203.0.113.10", "/api/ad-tikla/7")
	karar := k.reklamTiklamaDegerlendir(r, clientIP(r), 7)
	if !karar.Say {
		t.Fatalf("meşru tıklama sayılmadı: %q", karar.Neden)
	}
}

// TestReklamTiklamaBotReddedilir: script/araç tıklamaları sayılmaz.
func TestReklamTiklamaBotReddedilir(t *testing.T) {
	haller := []struct {
		ad    string
		kur   func(*http.Request)
		neden string
	}{
		{"curl UA", func(r *http.Request) { r.Header.Set("User-Agent", "curl/8.5.0") }, "otomasyon"},
		{"python", func(r *http.Request) { r.Header.Set("User-Agent", "python-requests/2.31") }, "otomasyon"},
		{"headless", func(r *http.Request) { r.Header.Set("User-Agent", "Mozilla/5.0 HeadlessChrome/131") }, "otomasyon"},
		{"puppeteer", func(r *http.Request) { r.Header.Set("User-Agent", "Mozilla/5.0 puppeteer") }, "otomasyon"},
		{"boş UA", func(r *http.Request) { r.Header.Del("User-Agent") }, "user-agent"},
		{"proxy via", func(r *http.Request) { r.Header.Set("Via", "1.1 squid") }, "proxy"},
		{"proxy forwarded", func(r *http.Request) { r.Header.Set("Forwarded", "for=1.2.3.4") }, "proxy"},
		{"x-real-ip", func(r *http.Request) { r.Header.Set("X-Real-IP", "1.2.3.4") }, "proxy"},
		{"uzun XFF", func(r *http.Request) { r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 3.3.3.3") }, "X-Forwarded-For"},
		{"fetch modu", func(r *http.Request) { r.Header.Set("Sec-Fetch-Mode", "cors") }, "sec-fetch-mode"},
		{"cors siteler arasi", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, "sec-fetch-site"},
		{"fetch dest", func(r *http.Request) { r.Header.Set("Sec-Fetch-Dest", "empty") }, "sec-fetch-dest"},
		{"kullanici yok", func(r *http.Request) { r.Header.Del("Sec-Fetch-User") }, "sec-fetch-user"},
		{"accept json", func(r *http.Request) { r.Header.Set("Accept", "application/json") }, "Accept"},
		{"yabanci referer", func(r *http.Request) { r.Header.Set("Referer", "https://tiklama-ciftligi.example/") }, "referer"},
	}
	for _, h := range haller {
		t.Run(h.ad, func(t *testing.T) {
			k := newReklamKoruma()
			r := insanIstegi("203.0.113.11", "/api/ad-tikla/7")
			h.kur(r)
			karar := k.reklamTiklamaDegerlendir(r, clientIP(r), 7)
			if karar.Say {
				t.Fatalf("%s: sayıldı ama sayılmamalıydı", h.ad)
			}
			if !strings.Contains(strings.ToLower(karar.Neden), strings.ToLower(h.neden)) {
				t.Errorf("%s: neden %q, %q beklenirdi", h.ad, karar.Neden, h.neden)
			}
		})
	}
}

// TestReklamTiklamaTekilleme: aynı IP aynı reklamı 30 dk içinde bir kez
// saydırır — tıklama sayısı şişmez.
func TestReklamTiklamaTekilleme(t *testing.T) {
	k := newReklamKoruma()
	ip := "203.0.113.20"
	ilk := k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/5"), ip, 5)
	if !ilk.Say {
		t.Fatalf("ilk tıklama sayılmalıydı: %q", ilk.Neden)
	}
	for i := 0; i < 5; i++ {
		k2 := k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/5"), ip, 5)
		if k2.Say {
			t.Fatalf("%d. tekrar sayıldı", i+2)
		}
		if !strings.Contains(k2.Neden, "tekrar") {
			t.Fatalf("neden %q", k2.Neden)
		}
	}
	// Farklı reklam sayılır (tekilleştirme reklam bazlı).
	if k3 := k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/6"), ip, 6); !k3.Say {
		t.Fatalf("farklı reklam sayılmalıydı: %q", k3.Neden)
	}
}

// TestReklamTiklamaHizSiniri: tek IP'den seri tıklama sınırlanır.
func TestReklamTiklamaHizSiniri(t *testing.T) {
	k := newReklamKoruma()
	ip := "203.0.113.30"
	sayilan := 0
	for i := 0; i < 20; i++ {
		// Her seferinde FARKLI reklam: tekilleştirme devreye girmesin,
		// yalnız hız sınırı ölçülsün.
		if k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/1"), ip, int64(100+i)).Say {
			sayilan++
		}
	}
	if sayilan != reklamIPTiklamaSinir {
		t.Fatalf("sayılan %d, sınır %d", sayilan, reklamIPTiklamaSinir)
	}
	// Farklı IP etkilenmez.
	if !k.reklamTiklamaDegerlendir(insanIstegi("203.0.113.31", "/api/ad-tikla/1"), "203.0.113.31", 1).Say {
		t.Fatal("farklı IP sınırdan etkilendi")
	}
}

// TestReklamGosterimSiniri: gösterim pikseli de şişirilemez.
func TestReklamGosterimSiniri(t *testing.T) {
	k := newReklamKoruma()
	ip := "203.0.113.40"
	izin := 0
	for i := 0; i < reklamGosterimSinir+20; i++ {
		if k.reklamGosterimIzinli(insanIstegi(ip, "/api/ad-gorselleme?id=1"), ip) {
			izin++
		}
	}
	if izin != reklamGosterimSinir {
		t.Fatalf("izin %d, sınır %d", izin, reklamGosterimSinir)
	}
	// Bot hiç izin almaz.
	if k.reklamGosterimIzinli(insanIstegi("203.0.113.41", "/x"), "") {
		t.Fatal("IP boşken izin verildi")
	}
}

// TestReklamPencereSifirlanir: pencere dolunca (süre geçince) yeniden sayılır.
func TestReklamPencereSifirlanir(t *testing.T) {
	k := newReklamKoruma()
	ip := "203.0.113.50"
	for i := 0; i < reklamIPTiklamaSinir; i++ {
		k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/1"), ip, int64(200+i))
	}
	// Pencereyi elle geçmiş say.
	k.mu.Lock()
	if p := k.tiklamalar[ip]; p != nil {
		p.baslangic = time.Now().Add(-2 * reklamIPPencere)
	}
	k.mu.Unlock()
	if !k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/1"), ip, 999).Say {
		t.Fatal("pencere sıfırlanmadı")
	}
}

// TestReklamKorumaTemizlik: süresi geçmiş kayıtlar bellekten atılır
// (sızıntı yok). Temizlik İKİ haritayı da süpürür: pencere (1 dk) ve
// tekilleştirme (30 dk) — bu yüzden ikisi de yaşlandırılır.
func TestReklamKorumaTemizlik(t *testing.T) {
	k := newReklamKoruma()
	for i := 0; i < 100; i++ {
		ip := "198.51.100." + itoa(i%250)
		k.reklamTiklamaDegerlendir(insanIstegi(ip, "/api/ad-tikla/1"), ip, int64(i))
	}
	k.mu.Lock()
	k.sonTemizlik = time.Now().Add(-2 * time.Minute)
	for ip, p := range k.tiklamalar {
		p.baslangic = time.Now().Add(-2 * reklamIPPencere)
		k.tiklamalar[ip] = p
	}
	for a, ts := range k.sonTiklama {
		k.sonTiklama[a] = ts.Add(-2 * reklamTekilleme)
	}
	k.mu.Unlock()

	// Bir istek temizliği tetikler.
	k.reklamTiklamaDegerlendir(insanIstegi("203.0.113.60", "/api/ad-tikla/1"), "203.0.113.60", 1)

	k.mu.Lock()
	tik, son := len(k.tiklamalar), len(k.sonTiklama)
	k.mu.Unlock()
	if tik > 1 {
		t.Fatalf("tıklama pencereleri temizlenmedi: %d kaldı", tik)
	}
	if son > 1 {
		t.Fatalf("tekilleştirme kayıtları temizlenmedi: %d kaldı", son)
	}
}
