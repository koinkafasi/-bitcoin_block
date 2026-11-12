package httpapi

import (
	"context"
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

type bookmarkPreview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Site        string `json:"site"`
	Image       string `json:"image"`
}

var (
	bookmarkTitleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	bookmarkMetaRe  = regexp.MustCompile(`(?is)<meta\s+[^>]*>`)
	bookmarkAttrRe  = regexp.MustCompile(`(?is)([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*("([^"]*)"|'([^']*)')`)
	bookmarkTagRe   = regexp.MustCompile(`(?is)<[^>]+>`)
	bookmarkSpaceRe = regexp.MustCompile(`\s+`)
)

// handleBookmarkPreview fetches OpenGraph metadata for the bookmark card.
// It intentionally rejects private/loopback targets to avoid SSRF.
func (s *Server) handleBookmarkPreview(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	target, err := validateBookmarkURL(raw)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()
	// #nosec G704 -- media_url.go ile aynı gerekçe: validateBookmarkURL +
	// safeRemoteHTTPClient (her bağlantıda DNS yeniden çözülür, özel ağ
	// reddedilir, çözülen IP'ye doğrudan bağlanılır).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bağlantı geçersiz"})
		return
	}
	req.Header.Set("User-Agent", "TheBlockBot/1.0 (+https://theblock.com.tr)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	client := safeRemoteHTTPClient(7 * time.Second)
	// #nosec G704 -- bkz. yukarıdaki NewRequestWithContext gerekçesi:
	// hedef doğrulandı, istemci yönlendirmede yeniden doğrular.
	resp, err := client.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "sayfa getirilemedi"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "sayfa yanıt vermedi"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "sayfa okunamadı"})
		return
	}
	preview := parseBookmarkHTML(data, resp.Request.URL)
	if preview.Title == "" {
		preview.Title = preview.Site
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bookmark": preview})
}

func safeRemoteHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if port != "80" && port != "443" {
				return nil, fmt.Errorf("yalnızca 80/443 portları destekleniyor")
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil || len(ips) == 0 {
				return nil, fmt.Errorf("alan adı çözümlenemedi")
			}
			for _, item := range ips {
				if !isPublicBookmarkIP(item.IP) {
					return nil, fmt.Errorf("özel ağ adresleri desteklenmiyor")
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("çok fazla yönlendirme")
			}
			if _, err := validateBookmarkURL(next.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

func validateBookmarkURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("geçerli bir adres girin")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("yalnızca http/https adresleri destekleniyor")
	}
	if u.User != nil {
		return nil, fmt.Errorf("kullanıcı bilgisi içeren adresler desteklenmiyor")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".local") {
		return nil, fmt.Errorf("yerel adresler desteklenmiyor")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicBookmarkIP(ip) {
			return nil, fmt.Errorf("özel ağ adresleri desteklenmiyor")
		}
		return u, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("alan adı çözümlenemedi")
	}
	for _, ip := range ips {
		if !isPublicBookmarkIP(ip) {
			return nil, fmt.Errorf("özel ağ adresleri desteklenmiyor")
		}
	}
	return u, nil
}

func isPublicBookmarkIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}

func parseBookmarkHTML(body []byte, base *url.URL) bookmarkPreview {
	out := bookmarkPreview{Site: base.Hostname()}
	text := string(body)
	metas := map[string]string{}
	for _, tag := range bookmarkMetaRe.FindAllString(text, -1) {
		attrs := map[string]string{}
		for _, m := range bookmarkAttrRe.FindAllStringSubmatch(tag, -1) {
			value := m[3]
			if value == "" {
				value = m[4]
			}
			attrs[strings.ToLower(m[1])] = html.UnescapeString(value)
		}
		key := strings.ToLower(attrs["property"])
		if key == "" {
			key = strings.ToLower(attrs["name"])
		}
		if key != "" && attrs["content"] != "" {
			metas[key] = strings.TrimSpace(attrs["content"])
		}
	}
	if m := bookmarkTitleRe.FindStringSubmatch(text); len(m) > 1 {
		out.Title = cleanBookmarkText(m[1])
	}
	if v := metas["og:title"]; v != "" {
		out.Title = v
	} else if v := metas["twitter:title"]; v != "" {
		out.Title = v
	}
	out.Description = metas["og:description"]
	if out.Description == "" {
		out.Description = metas["twitter:description"]
	}
	if out.Description == "" {
		out.Description = metas["description"]
	}
	out.Description = cleanBookmarkText(out.Description)
	if v := metas["og:site_name"]; v != "" {
		out.Site = v
	}
	out.Image = metas["og:image"]
	if out.Image == "" {
		out.Image = metas["twitter:image"]
	}
	if out.Image != "" {
		if iu, err := url.Parse(out.Image); err == nil {
			out.Image = base.ResolveReference(iu).String()
		}
	}
	out.URL = base.String()
	out.Title = truncateBookmarkRunes(out.Title, 160)
	out.Description = truncateBookmarkRunes(out.Description, 360)
	out.Site = truncateBookmarkRunes(cleanBookmarkText(out.Site), 100)
	return out
}

func cleanBookmarkText(v string) string {
	v = html.UnescapeString(bookmarkTagRe.ReplaceAllString(v, " "))
	return strings.TrimSpace(bookmarkSpaceRe.ReplaceAllString(v, " "))
}

func truncateBookmarkRunes(v string, n int) string {
	r := []rune(v)
	if len(r) <= n {
		return v
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
