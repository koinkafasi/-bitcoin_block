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
	if host == "localhost" || strings.HasPrefix(host, "localhost:") ||
		strings.HasPrefix(host, "127.") || host == "[::1]" || host == "::1" {
		return varsayilan
	}
	alan := strings.ToLower(strings.TrimSpace(s.siteByDomain(r.Context(), host).Domain))
	if alan == "" {
		return varsayilan
	}
	if host != alan && !strings.HasSuffix(host, "."+alan) {
		return varsayilan
	}
	// KANONİK TEK ADRES: sitenin kayıtlı alan adı. www ve demo gibi alt alan
	// adları aynı içeriği gösteriyor; her biri kendi adresini kanonik ilan
	// ederse Google'a "birbirinden bağımsız üç site" demiş oluruz ve dizin
	// bölünür. Önizleme (demo) trafiği de üretime işaret eder.
	return "https://" + alan
}
