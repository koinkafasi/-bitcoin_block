package research

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var errBlockedURL = errors.New("güvenli olmayan kaynak adresi")

// defaultClient SSRF korumalı HTTP istemcisidir: loopback/özel/link-local
// adreslere, metadata IP'lerine ve 80/443 dışı portlara çıkmaz.
func defaultClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 20 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safeDialContext(dialer),
		TLSHandshakeTimeout:   6 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
		DisableKeepAlives:     true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   9 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("çok fazla yönlendirme")
			}
			return validateURL(req.URL)
		},
	}
}

func safeDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if port != "80" && port != "443" {
			return nil, fmt.Errorf("%w: port %s", errBlockedURL, port)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		var chosen net.IP
		for _, ip := range ips {
			if blockedIP(ip.IP) {
				return nil, fmt.Errorf("%w: özel adres", errBlockedURL)
			}
			if chosen == nil {
				chosen = ip.IP
			}
		}
		if chosen == nil {
			return nil, fmt.Errorf("%w: adres çözülemedi", errBlockedURL)
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(chosen.String(), port))
	}
}

func blockedIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	// CGNAT 100.64.0.0/10 ve benchmark 198.18.0.0/15.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return true
		}
	}
	return false
}

func validateURL(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errBlockedURL
	}
	if u.User != nil || u.Hostname() == "" {
		return errBlockedURL
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if u.Scheme == "http" {
			port = "80"
		}
	}
	if port != "80" && port != "443" {
		return fmt.Errorf("%w: port", errBlockedURL)
	}
	return nil
}

var (
	reScript = regexp.MustCompile(`(?is)<script.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style.*?</style>`)
	reTag    = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpace  = regexp.MustCompile(`\s+`)
)

// FetchText kaynağı güvenli biçimde çeker, HTML etiketlerini temizler ve
// modele verilecek düz metni döndürür. Hata durumunda içerik uydurulmaz.
func FetchText(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if err := validateURL(u); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "theblock-research/1.0 (+https://theblock.com.tr)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9")
	resp, err := defaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("kaynak HTTP %d", resp.StatusCode)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "text/plain") &&
		!strings.Contains(ct, "application/xhtml") {
		return "", fmt.Errorf("desteklenmeyen içerik türü: %s", ct)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	text := string(body)
	if strings.Contains(ct, "html") {
		text = reScript.ReplaceAllString(text, " ")
		text = reStyle.ReplaceAllString(text, " ")
		text = reTag.ReplaceAllString(text, " ")
	}
	text = html.UnescapeString(text)
	text = reSpace.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("kaynak metni boş")
	}
	r := []rune(text)
	if len(r) > 6000 {
		text = string(r[:6000])
	}
	return text, nil
}
