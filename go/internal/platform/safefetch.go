package platform

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrGuvenliOlmayanAdres kullanıcı kaynaklı bir adresin SSRF kurallarına
// takıldığını bildirir.
var ErrGuvenliOlmayanAdres = errors.New("güvenli olmayan kaynak adresi")

func SafeFetchClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 20 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safeFetchDial(dialer),
		TLSHandshakeTimeout:   6 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableKeepAlives:     true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("çok fazla yönlendirme")
			}
			return ValidateOutboundURL(req.URL)
		},
	}
}

func safeFetchDial(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if port != "80" && port != "443" {
			return nil, fmt.Errorf("%w: port %s", ErrGuvenliOlmayanAdres, port)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("%w: adres çözümlenemedi", ErrGuvenliOlmayanAdres)
		}
		var secilen net.IP
		for _, ip := range ips {
			if OzelAdresMi(ip.IP) {
				return nil, fmt.Errorf("%w: özel/ayrılmış adres", ErrGuvenliOlmayanAdres)
			}
			if secilen == nil {
				secilen = ip.IP
			}
		}
		if secilen == nil {
			return nil, fmt.Errorf("%w: adres çözümlenemedi", ErrGuvenliOlmayanAdres)
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(secilen.String(), port))
	}
}

// OzelAdresMi loopback, özel, link-local, çoklu yayın ve ayrılmış
// (dokümantasyon/benchmark/CGNAT) adresleri işaretler.
func OzelAdresMi(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		switch {
		case v4[0] == 0: // 0.0.0.0/8
			return true
		case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127: // CGNAT 100.64/10
			return true
		case v4[0] == 192 && v4[1] == 0 && v4[2] == 0: // 192.0.0.0/24
			return true
		case v4[0] == 198 && (v4[1] == 18 || v4[1] == 19): // benchmark
			return true
		case v4[0] >= 240: // ayrılmış + broadcast
			return true
		}
	}
	return false
}

func ValidateOutboundURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("%w: boş adres", ErrGuvenliOlmayanAdres)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: yalnız http/https", ErrGuvenliOlmayanAdres)
	}
	if u.User != nil {
		return fmt.Errorf("%w: kullanıcı bilgisi içeren adres", ErrGuvenliOlmayanAdres)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("%w: alan adı yok", ErrGuvenliOlmayanAdres)
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("%w: yerel alan adı", ErrGuvenliOlmayanAdres)
	}
	if ip := net.ParseIP(host); ip != nil && OzelAdresMi(ip) {
		return fmt.Errorf("%w: özel ağ adresi", ErrGuvenliOlmayanAdres)
	}
	return nil
}

func DogrulaVeCek(ctx context.Context, rawURL string, timeout time.Duration) (*http.Response, error) {
	return DogrulaVeCekUA(ctx, rawURL, timeout, "TheBlockBot/1.0 (+https://theblock.com.tr)")
}

// DogrulaVeCekUA DogrulaVeCek'in UA'sı özelleştirilebilir sürümü.
func DogrulaVeCekUA(ctx context.Context, rawURL string, timeout time.Duration, ua string) (*http.Response, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("%w: ayrıştırılamadı", ErrGuvenliOlmayanAdres)
	}
	if err := ValidateOutboundURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	return SafeFetchClient(timeout).Do(req)
}
