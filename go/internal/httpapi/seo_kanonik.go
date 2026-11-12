package httpapi

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
	return "https://" + alan
}
