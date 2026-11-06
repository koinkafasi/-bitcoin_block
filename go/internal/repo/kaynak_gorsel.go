package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	neturl "net/url"
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

// GorselAdresiCoz ara görsel proxy'sini gerçek CDN adresine indirger.
//
// SORUN (ölçüm, 6 Kasım 2026): CoinDesk görselleri Next.js görsel
// proxy'sinden geliyor:
//
//	https://www.coindesk.com/_next/image?url=<CDN adresi>&w=1200&q=75
//
// Proxy bot koruması uyguluyor: sunucudan indirme denemeleri **HTTP 429**
// dönüyordu (günlükte "kaynak görsel indirilemedi … HTTP 429"). Aynı
// görselin CDN adresi doğrudan **200** veriyor (153 KB JPEG).
//
// Bu yüzden `url=` parametresi çözülür ve CDN adresi kullanılır. Çözüm
// uygulanamazsa adres olduğu gibi döner (davranış değişmez).
//
// GÜVENLİK: yalnız `url` parametresinin kendisi alınır ve yalnız
// http(s) kabul edilir; `javascript:`/`data:` gibi şemalar elenir.
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
