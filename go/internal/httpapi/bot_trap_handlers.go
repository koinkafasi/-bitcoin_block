package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// ---- Bot tuzağı (honeypot) + bloklu IP katmanı — spec 06 K-I ----

// isTrapPath: robots.txt'te Disallow + insandan gizli linkler. Yalnız robots'u
// çiğneyen botlar buraya gelir → tespit + otomatik blok.
func isTrapPath(p string) bool {
	p = strings.TrimRight(p, "/")
	switch p {
	case "/arsiv/dizin", "/gizli-dizin", "/tum-haberler-dizin":
		return true
	}
	return false
}

// handleBotTrap tuzağa düşen isteği loglar, (uygunsa) IP'yi bloklar ve 410 döner.
func (s *Server) handleBotTrap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := clientIP(r)
	_ = repo.LogBotTrap(ctx, s.pool, ip, r.UserAgent(), r.URL.Path, r.Referer())

	// otomatik blok: public IP + smoke/allowlist değilse
	if ip != "" && !isPrivateIP(ip) && !s.isAllowlistedIP(ctx, ip) && r.Header.Get("X-Tb-Smoke") == "" {
		if err := repo.BlockIP(ctx, s.pool, ip, "bot_tuzagi: "+r.URL.Path); err == nil {
			s.invalidateIPBlock()
		}
	}

	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, "410 Gone", http.StatusGone)
}

// withIPBlock bloklu IP'lerin isteklerini erken reddeder (403).
func (s *Server) withIPBlock(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip != "" && s.isIPBlocked(r.Context(), ip) {
			w.Header().Set("X-Robots-Tag", "noindex")
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isIPBlocked blok listesini 30 sn bellek önbelleğiyle kontrol eder.
func (s *Server) isIPBlocked(ctx context.Context, ip string) bool {
	// private IP'ler asla bloklanmaz (güvenlik)
	if isPrivateIP(ip) {
		return false
	}
	s.blockMu.Lock()
	defer s.blockMu.Unlock()
	if time.Since(s.blockAt) > 30*time.Second {
		if set, err := repo.BlockedIPSet(ctx, s.pool); err == nil {
			s.blockCache = set
			s.blockAt = time.Now()
		}
	}
	return s.blockCache[ip]
}

// invalidateIPBlock önbelleği düşürür (blok/çözümde).
func (s *Server) invalidateIPBlock() {
	s.blockMu.Lock()
	s.blockAt = time.Time{}
	s.blockMu.Unlock()
}

// isAllowlistedIP ayarlardaki (seo/blok_izinli) izinli IP listesini kontrol eder.
func (s *Server) isAllowlistedIP(ctx context.Context, ip string) bool {
	list := s.seoSetting(ctx, "blok_izinli")
	if list == "" {
		return false
	}
	for _, it := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' }) {
		if strings.TrimSpace(it) == ip {
			return true
		}
	}
	return false
}

// isPrivateIP loopback/özel/link-local adresleri işaretler (asla bloklanmaz).
func isPrivateIP(ip string) bool {
	p := net.ParseIP(ip)
	if p == nil {
		return true // geçersiz IP → güvenli tarafta kal
	}
	return p.IsLoopback() || p.IsPrivate() || p.IsLinkLocalUnicast() || p.IsLinkLocalMulticast() || p.IsUnspecified()
}

// clientIP tanımı auth_limit.go'dadır: proxy başlıklarına yalnız yerel/özel
// ağdan gelen isteklerde güvenilir.
