package httpapi

// Kanonik kök adres.
//
// NEDEN: `cfg.BaseURL` tek bir sabit adres taşıyor (bugün demo.bitcosar.com).
// Site çok alan adlı çalışıyor: aynı içerik bitcosar.com, demo.bitcosar.com ve
// theblock.com.tr üzerinden servis edilebiliyor. Canonical/og:url yanlış
// alan adını gösterirse Google içeriği "başka sitede" sayar; canlıya
// geçildiğinde tüm indekslenmiş adresler demo'ya işaret ederdi.
//
// Kural: istek alan adı, sitenin kayıtlı alan adıyla (veya alt alan adıyla)
// eşleşiyorsa isteğin kendi adresi kanonik olur; aksi hâlde (127.0.0.1,
// iç ağ, bilinmeyen host) yapılandırmadaki adrese düşülür.

import (
	"net/http"
	"strings"
)

func (s *Server) kanonikKok(r *http.Request) string {
	varsayilan := strings.TrimRight(s.cfg.BaseURL, "/")
	if r == nil {
		return varsayilan
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	if host == "" {
		return varsayilan
	}
	alan := strings.ToLower(strings.TrimSpace(s.siteByDomain(r.Context(), host).Domain))
	if alan == "" {
		return varsayilan
	}
	if host != alan && !strings.HasSuffix(host, "."+alan) {
		return varsayilan
	}
	// www. alt alan adı kanonik değil: tek adres `bitcosar.com` olsun.
	if host == "www."+alan {
		return "https://" + alan
	}
	if host == alan {
		return "https://" + alan
	}
	sema := "https"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.") ||
		strings.HasPrefix(host, "[::1]") {
		sema = "http"
	}
	return sema + "://" + host
}
