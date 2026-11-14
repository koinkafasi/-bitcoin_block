package httpapi

import (
	"net/http"
	"strings"
)

// withKonum ziyaretçinin ÜLKE bilgisini kaydeder (izinsiz, kaba katman).
//
// NEDEN MIDDLEWARE, NEDEN HANDLER İÇİNDE DEĞİL: tek yerde olmalı; her
// sayfa handler'ına bir satır eklemek unutulmaya açık. Zincirde mux'tan
// hemen önce duruyor, yani gerçek istek yolu (dil öneki soyulduktan
// sonra) görünüyor.
//
// MALİYET: yalnız BELGE isteklerinde çalışır ve çerezle saatte bir
// yazıma sınırlanır (bkz. konumKaydetIP). Statik dosya, API, admin ve
// besleme istekleri tamamen atlanır — bunlar sayfa görüntülemesi değil.
func (s *Server) withKonum(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && konumSayfasiMi(r.URL.Path) {
			// Yanıt yazılmadan ÖNCE çağrılmalı: Set-Cookie başlığı
			// gövde yazıldıktan sonra etkisiz olur.
			s.konumKaydetIP(w, r)
		}
		next.ServeHTTP(w, r)
	})
}

// konumSayfasiMi yolun sayfa görüntülemesi olup olmadığını söyler.
//
// Beyaz liste değil KARA liste: yeni sayfa eklendiğinde unutulup konum
// kaydı durmasın. Yalnız sayfa sayılmayan yollar dışarıda.
func konumSayfasiMi(p string) bool {
	if p == "" {
		p = "/"
	}
	// TAM eşleşen kökler: "/admin/" öneki "/admin"i yakalamaz
	// (test yakaladı: /admin kayda giriyordu).
	for _, tam := range []string{"/admin", "/api", "/internal", "/media", "/static"} {
		if p == tam {
			return false
		}
	}
	for _, onek := range []string{
		"/static/", "/media/", "/kg/", "/api/", "/admin/", "/internal/",
		"/takim-logo/", "/og-varsayilan", "/favicon", "/manifest",
	} {
		if strings.HasPrefix(p, onek) {
			return false
		}
	}
	// Beslemeler ve haritalar sayfa değil; tarayıcı değil bot okur.
	for _, sonek := range []string{".xml", ".txt", ".png", ".ico", ".webmanifest", ".json"} {
		if strings.HasSuffix(p, sonek) {
			return false
		}
	}
	// SSE/canlı uçlar.
	if p == "/canli/piyasa" || strings.HasPrefix(p, "/canli/icerik/") {
		return false
	}
	return true
}
