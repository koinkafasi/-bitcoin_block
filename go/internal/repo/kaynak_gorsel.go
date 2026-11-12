package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	neturl "net/url"
	"strings"
)

func KaynakGorselHash(url string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(url)))
	return hex.EncodeToString(h[:16])
}

func KaynakGorselKaydet(ctx context.Context, pool DB, rawURL string) string {
	rawURL = GorselAdresiCoz(rawURL)
	if rawURL == "" {
		return ""
	}
	l := strings.ToLower(rawURL)
	if !strings.HasPrefix(l, "http://") && !strings.HasPrefix(l, "https://") {
		return ""
	}
	hash := KaynakGorselHash(rawURL)
	_, err := pool.Exec(ctx, `
		INSERT INTO source_images (hash, url) VALUES ($1, $2)
		ON CONFLICT (hash) DO UPDATE SET used_at = now()`, hash, rawURL)
	if err != nil {
		return ""
	}
	return hash
}

func GorselAdresiCoz(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// Yalnız bilinen görsel proxy'leri: her adresin sorgusundaki `url`
	// parametresini çözmek, meşru bir adresi bozabilirdi.
	if !strings.Contains(strings.ToLower(u.Host), "coindesk.com") ||
		!strings.Contains(u.Path, "/_next/image") {
		return rawURL
	}
	ic := u.Query().Get("url")
	if ic == "" {
		return rawURL
	}
	iu, err := neturl.Parse(ic)
	if err != nil || (iu.Scheme != "http" && iu.Scheme != "https") || iu.Host == "" {
		return rawURL
	}
	return iu.String()
}

// KaynakGorselURL hash'in kaynak adresini döndürür (yoksa "").
func KaynakGorselURL(ctx context.Context, pool DB, hash string) string {
	var u string
	if err := pool.QueryRow(ctx,
		`SELECT url FROM source_images WHERE hash = $1`, hash).Scan(&u); err != nil {
		return ""
	}
	return u
}

// KaynakGorselIsaretle gövde indirildiğinde boyut/tipi yazar.
func KaynakGorselIsaretle(ctx context.Context, pool DB, hash, contentType string, bytes int64) {
	_, _ = pool.Exec(ctx, `
		UPDATE source_images SET content_type = $2, bytes = $3, used_at = now()
		WHERE hash = $1`, hash, contentType, bytes)
}

// KaynakGorselYol web yolunu üretir: /kg/<hash>. Uzantı YOK — içerik tipi
// DB'den gelir; uzantı eklemek `image_url` değişince URL'i değiştirirdi.
func KaynakGorselYol(hash string) string {
	if hash == "" {
		return ""
	}
	return "/kg/" + hash
}
