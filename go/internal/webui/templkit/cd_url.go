package templkit

import "strings"

// cdMutlak göreli görsel yolunu mutlak adrese çevirir.
//
// NEDEN: Open Graph protokolü `og:image` için MUTLAK URL ister. Göreli
// `/kg/abc` değeriyle X/Facebook kartı görselsiz çıkıyordu (7 Kasım 2026
// ölçümü: ana sayfa ve makale). Kapak <img>'i göreli kalmalı (kendi
// origin'imizden servis ediliyor, srcset üretimi yola bağlı) — bu yüzden
// dönüşüm yalnız head'de yapılıyor.
func cdMutlak(u, base string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return u
	}
	return base + "/" + strings.TrimPrefix(u, "/")
}
