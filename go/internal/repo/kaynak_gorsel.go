package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Kaynak görseli kaydı — kendi origin'imizden servis için.
//
// Ayrıntı ve gerekçe: migrations/0116_kaynak_gorsel.up.sql
// (üçüncü taraf çerez → Lighthouse BP 73; dış CDN bağımlılığı).

// KaynakGorselHash bir görsel adresinin kısa parmak izini üretir.
//
// 32 hex karakter (128 bit) yeterli: çakışma olasılığı pratikte yok,
// URL'de kısa kalır. Aynı adres her zaman aynı anahtarı verir → CDN
// önbelleği kalıcı olur (`immutable`).
func KaynakGorselHash(url string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(url)))
	return hex.EncodeToString(h[:16])
}

// KaynakGorselKaydet adresi kaydeder (varsa dokunmaz). Dönüş: hash.
//
// Boş veya http(s) olmayan adres kaydedilmez ("" döner) — şablon o zaman
// görsel basmaz. Gövde indirme burada YAPILMAZ: çekim, görsel ilk
// istendiğinde olur (crawler turunu yavaşlatmamak için).
func KaynakGorselKaydet(ctx context.Context, pool DB, rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
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
