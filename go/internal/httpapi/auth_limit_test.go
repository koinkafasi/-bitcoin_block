package httpapi

import (
	"net/http/httptest"
	"testing"
)

// ---- clientIP: proxy başlıklarına güven sınırı ----

func TestClientIP_PublicRemoteIgnoresProxyHeaders(t *testing.T) {
	// Doğrudan bağlanan istemci X-Forwarded-For uyduramaz: aksi hâlde hız
	// sınırı ve bot tuzağı tek satırlık başlıkla atlatılırdı.
	r := httptest.NewRequest("POST", "/giris", nil)
	r.RemoteAddr = "203.0.113.9:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	r.Header.Set("CF-Connecting-IP", "5.6.7.8")

	if got := clientIP(r); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, beklenen gerçek kaynak 203.0.113.9", got)
	}
}

func TestClientIP_PrivateRemoteTrustsProxyHeaders(t *testing.T) {
	r := httptest.NewRequest("POST", "/giris", nil)
	r.RemoteAddr = "127.0.0.1:60000"
	r.Header.Set("CF-Connecting-IP", "198.51.100.7")
	if got := clientIP(r); got != "198.51.100.7" {
		t.Fatalf("CF başlığı güvenilmeliydi, clientIP = %q", got)
	}

	r2 := httptest.NewRequest("POST", "/giris", nil)
	r2.RemoteAddr = "10.1.2.3:60000"
	r2.Header.Set("X-Forwarded-For", "198.51.100.8, 10.1.2.3")
	if got := clientIP(r2); got != "198.51.100.8" {
		t.Fatalf("XFF'in ilk değeri alınmalıydı, clientIP = %q", got)
	}
}

func TestClientIP_NoHeaders(t *testing.T) {
	r := httptest.NewRequest("POST", "/giris", nil)
	r.RemoteAddr = "198.51.100.20:1234"
	if got := clientIP(r); got != "198.51.100.20" {
		t.Fatalf("clientIP = %q", got)
	}
}

// ---- collab uçları yalnız yerel/özel ağdan ----

func TestCollabYerelAg(t *testing.T) {
	cases := []struct {
		remote string
		xff    string
		want   bool
	}{
		{"127.0.0.1:5000", "", true},
		{"10.0.0.5:5000", "", true},
		{"192.168.1.5:5000", "", true},
		{"203.0.113.9:5000", "", false},
		// Dışarıdan gelen istek XFF ile iç ağ taklidi yapamaz.
		{"203.0.113.9:5000", "127.0.0.1", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("POST", "/internal/collab/1", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := collabYerelAğ(r); got != c.want {
			t.Errorf("collabYerelAğ(%s, xff=%q) = %v, beklenen %v", c.remote, c.xff, got, c.want)
		}
	}
}

// ---- kimlik doğrulama sayaçları ----

func TestAuthDenemeSayaci_IPVeHesap(t *testing.T) {
	ip := "198.51.100.77"
	hesap := "test@example.com"
	sifirla := func() {
		authDenemeleri.mu.Lock()
		authDenemeleri.m = map[string]*denemeSayaci{}
		authDenemeleri.mu.Unlock()
	}
	sifirla()
	t.Cleanup(sifirla)

	for i := 0; i < authHesapAzami; i++ {
		if !authHesapIzinli(ip, hesap) {
			t.Fatalf("%d. denemede erken kilit", i)
		}
		authBasarisiz(ip, hesap)
	}
	if authHesapIzinli(ip, hesap) {
		t.Fatal("hesap sınırı aşıldıktan sonra hâlâ izin veriliyor")
	}
	// Farklı IP aynı hesabı kilitlemez (hesap hedefli DoS koruması).
	if !authHesapIzinli("198.51.100.99", hesap) {
		t.Fatal("başka IP'den gelen meşru giriş engellendi")
	}
	// Başarılı giriş sayacı temizler.
	authBasarili(ip, hesap)
	if !authHesapIzinli(ip, hesap) {
		t.Fatal("başarılı girişten sonra sayaç sıfırlanmadı")
	}
}

func TestAuthDenemeSayaci_IPToplamSiniri(t *testing.T) {
	ip := "198.51.100.55"
	sifirla := func() {
		authDenemeleri.mu.Lock()
		authDenemeleri.m = map[string]*denemeSayaci{}
		authDenemeleri.mu.Unlock()
	}
	sifirla()
	t.Cleanup(sifirla)

	for i := 0; i < authIPAzami; i++ {
		authBasarisiz(ip, "kullanici"+string(rune('a'+i%26))+"@example.com")
	}
	if authIPIzinli(ip) {
		t.Fatal("IP toplam sınırı aşıldıktan sonra hâlâ izin veriliyor")
	}
	if !authIPIzinli("198.51.100.56") {
		t.Fatal("başka IP etkilenmemeli")
	}
}

func TestKayitSiniri(t *testing.T) {
	ip := "198.51.100.42"
	sifirla := func() {
		authDenemeleri.mu.Lock()
		authDenemeleri.m = map[string]*denemeSayaci{}
		authDenemeleri.mu.Unlock()
	}
	sifirla()
	t.Cleanup(sifirla)

	for i := 0; i < kayitAzami; i++ {
		if !kayitIzinli(ip) {
			t.Fatalf("%d. kayıtta erken kilit", i)
		}
		kayitSay(ip)
	}
	if kayitIzinli(ip) {
		t.Fatal("saatlik kayıt sınırı uygulanmadı")
	}
}
