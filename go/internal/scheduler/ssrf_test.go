package scheduler

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"theblock/site/internal/platform"
)

// Varsayılan istemci SSRF korumalı olmalı ve loopback'i reddetmeli.
func TestTrendHTTPClientVarsayilanKorumali(t *testing.T) {
	c := trendHTTPClient()
	if c == nil {
		t.Fatal("istemci nil")
	}
	if c.Timeout <= 0 {
		t.Error("zaman aşımı olmalı")
	}

	// 127.0.0.1 üzerinde dinleyen bir sunucu: korumalı istemci bağlanmamalı.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("yerel dinleyici açılamadı: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close() // bağlantı kabul edilirse test başarısız olur
		}
	}()

	adres := "http://" + ln.Addr().String() + "/rss"
	_, err = platform.DogrulaVeCek(context.Background(), adres, 2*time.Second)
	if err == nil {
		t.Fatalf("SSRF korumalı istemci loopback adresini çekti: %s", adres)
	}
}

// Varsayılan istemci genel bir adresi engellememeli (aşırı katılık kontrolü).
// Ağ erişimi olmayan ortamda yalnız doğrulama katmanı sınanır.
func TestValidateOutboundURLGenelAdreslereIzinVerir(t *testing.T) {
	for _, adres := range []string{
		"https://trends.google.com/trending/rss?geo=TR",
		"http://rss.example.com/feed",
		"https://news.example.org/rss.xml",
	} {
		u, err := url.Parse(adres)
		if err != nil {
			t.Fatalf("%s ayrıştırılamadı: %v", adres, err)
		}
		if err := platform.ValidateOutboundURL(u); err != nil {
			t.Errorf("%s reddedildi: %v", adres, err)
		}
	}
}

// İç ağ / metadata / loopback adresleri KAYDETME anında reddedilmeli.
func TestValidateOutboundURLEngeller(t *testing.T) {
	kasalar := []string{
		"http://127.0.0.1:8070/admin",
		"http://localhost/rss",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/feed",
		"http://192.168.1.1/",
		"http://172.16.0.1/",
		"http://[::1]/",
		"file:///etc/passwd",
		"ftp://example.com/feed",
		"http://user:pass@example.com/feed",
		"http://sunucu.local/feed",
		"http://servis.internal/feed",
		"",
	}
	for _, adres := range kasalar {
		u, err := url.Parse(adres)
		if err != nil {
			continue // ayrıştırılamayan adres zaten kabul edilmez
		}
		if err := platform.ValidateOutboundURL(u); err == nil {
			t.Errorf("%q kabul edildi; SSRF kuralı reddetmeliydi", adres)
		}
	}
}
