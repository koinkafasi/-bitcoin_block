package platform

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestOzelAdresMi SSRF adres sınıflandırması.
func TestOzelAdresMi(t *testing.T) {
	ozel := []string{
		"127.0.0.1", "127.1.2.3", "::1", "0.0.0.0",
		"10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1",
		"169.254.169.254", // bulut metadata
		"100.64.0.1",      // CGNAT
		"198.18.0.1",      // benchmark
		"192.0.0.1",
		"224.0.0.1", "255.255.255.255",
		"fc00::1", "fe80::1", "::",
	}
	for _, s := range ozel {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("test adresi ayrıştırılamadı: %s", s)
		}
		if !OzelAdresMi(ip) {
			t.Errorf("%s özel/ayrılmış sayılmalıydı", s)
		}
	}
	genel := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111"}
	for _, s := range genel {
		ip := net.ParseIP(s)
		if OzelAdresMi(ip) {
			t.Errorf("%s genel adres sayılmalıydı", s)
		}
	}
}

// TestValidateOutboundURL kullanıcı kaynaklı adres doğrulaması.
func TestValidateOutboundURL(t *testing.T) {
	reddedilen := []string{
		"http://127.0.0.1/",
		"http://localhost:8070/admin",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/",
		"http://[::1]/",
		"file:///etc/passwd",
		"gopher://example.com/",
		"http://user:pass@example.com/",
		"http://sunucu.local/",
		"http://intranet.internal/",
		"http://0.0.0.0/",
	}
	for _, raw := range reddedilen {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if err := ValidateOutboundURL(u); err == nil {
			t.Errorf("%q reddedilmeliydi", raw)
		} else if !errors.Is(err, ErrGuvenliOlmayanAdres) {
			t.Errorf("%q: beklenen ErrGuvenliOlmayanAdres, alınan %v", raw, err)
		}
	}
	kabul := []string{
		"https://theblock.com.tr/haber/1",
		"http://example.com/rss.xml",
		"https://8.8.8.8/",
	}
	for _, raw := range kabul {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("ayrıştırılamadı: %v", err)
		}
		if err := ValidateOutboundURL(u); err != nil {
			t.Errorf("%q kabul edilmeliydi: %v", raw, err)
		}
	}
}

// TestDogrulaVeCekLoopbackReddi gerçek bağlantı denemesi yapmadan erken
// reddedildiğini doğrular (iç ağa istek atılmaz).
func TestDogrulaVeCekLoopbackReddi(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:8070/healthz",
		"http://169.254.169.254/latest/meta-data/iam/security-credentials/",
	} {
		if _, err := DogrulaVeCek(context.Background(), raw, time.Second); err == nil {
			t.Errorf("%q için istek yapılmamalıydı", raw)
		} else if !strings.Contains(err.Error(), "güvenli olmayan") {
			t.Errorf("%q: beklenen SSRF hatası, alınan %v", raw, err)
		}
	}
}

// TestSafeFetchClientYonlendirmeSayisi yönlendirme zincirinde 3'ten sonra
// durduğunu (döngüsel yönlendirme koruması) doğrular.
func TestSafeFetchClientYonlendirmeSayisi(t *testing.T) {
	c := SafeFetchClient(time.Second)
	if c.CheckRedirect == nil {
		t.Fatal("CheckRedirect tanımlı olmalı (yönlendirme ile SSRF atlatılabilir)")
	}
	via := make([]*http.Request, 3)
	for i := range via {
		via[i] = &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	}
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	if err := c.CheckRedirect(req, via); err == nil {
		t.Fatal("3 yönlendirmeden sonra durmalıydı")
	}
	// İç adrese yönlendirme reddedilmeli.
	ic := &http.Request{URL: &url.URL{Scheme: "http", Host: "127.0.0.1"}}
	if err := c.CheckRedirect(ic, nil); err == nil {
		t.Fatal("loopback'e yönlendirme reddedilmeliydi")
	}
}
