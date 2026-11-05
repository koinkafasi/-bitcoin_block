package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// handleKaynakGorsel — kaynak görselini KENDİ origin'imizden servis eder.
//
// GET /kg/{hash}
//
// NEDEN (ölçüm, 5 Kasım 2026): /markets sayfasında Lighthouse
// best-practices 73 idi. Neden bizim işaretlememiz değil, kaynak CDN'inin
// yazdığı ÜÇÜNCÜ TARAF ÇEREZ:
//
//	third-party-cookies → cdn.decrypt.co/...png
//	inspector-issues    → Cookie
//
// Ayrıca her görüntülemede dış DNS/TLS yeniden kuruluyordu (LCP 2,0 s).
//
// Akış:
//  1. hash → DB'den kaynak adres (crawler yazdı)
//  2. diskte varsa doğrudan sun (edge/CDN önbelleği: immutable)
//  3. yoksa indir (SSRF korumalı istemci), doğrula, atomik yaz, sun
//
// GÜVENLİK: yalnız http(s); SafeFetchClient (özel IP reddi + yönlendirme
// doğrulaması); 8 MB üst sınır; içerik tipi image/* olmalı; SVG kabul
// EDİLMEZ (script taşıyabilir).
//
// TELİF: bu bir teknik önbellektir; makalede kaynak atfı ve kaynağa bağlantı
// korunur. Kaynak kaldırma talebinde satır silinir (repo.DeleteSourceImage).
const (
	kaynakGorselMaxBayt = 8 << 20
	kaynakGorselTimeout = 12 * time.Second
)

// gecerliKaynakGorselHash hash biçimini doğrular: 32 karakter, yalnız hex.
//
// Uzunluk ve karakter kontrolü TEK yerde: ikisi ayrıldığında biri
// güncellenmeden diğeri kalıyor ve yol aşımına kapı açılıyor
// (test bunu yakaladı — `yalnizHex("")` true dönüyordu).
func gecerliKaynakGorselHash(h string) bool {
	return len(h) == 32 && yalnizHex(h)
}

func (s *Server) handleKaynakGorsel(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimSpace(r.PathValue("hash"))
	if !gecerliKaynakGorselHash(hash) {
		http.NotFound(w, r)
		return
	}

	yol := kaynakGorselDiskYolu(hash)
	if b, ct, ok := kaynakGorselOku(yol); ok {
		s.kaynakGorselSun(w, r, b, ct)
		return
	}

	kaynak := repo.KaynakGorselURL(r.Context(), s.pool, hash)
	if kaynak == "" {
		http.NotFound(w, r)
		return
	}

	b, ct, err := kaynakGorselIndir(r, kaynak)
	if err != nil {
		s.logger.Warn("kaynak görsel indirilemedi", "hash", hash, "error", err)
		// Görsel gelmezse yarım sayfa bırakmayalım: 404 → tarayıcı
		// `<img>` için yer tutucu gösterir, düzen bozulmaz.
		http.NotFound(w, r)
		return
	}
	kaynakGorselYaz(yol, b)
	repo.KaynakGorselIsaretle(r.Context(), s.pool, hash, ct, int64(len(b)))
	s.kaynakGorselSun(w, r, b, ct)
}

// kaynakGorselSun gövdeyi uzun önbellekle sunar.
//
// `immutable` bilinçli: URL içerik adresinin parmak izidir, içerik
// değişmez. 1 yıl tarayıcı + CDN önbelleği.
func (s *Server) kaynakGorselSun(w http.ResponseWriter, r *http.Request, b []byte, ct string) {
	if ct == "" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Kaynak görseli kendi origin'imizden sunulduğu için CSP `img-src 'self'`
	// yeterli olur; ek izin gerekmez.
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(b))
}

// kaynakGorselIndir görseli SSRF korumalı istemciyle indirir.
func kaynakGorselIndir(r *http.Request, rawURL string) ([]byte, string, error) {
	resp, err := platform.DogrulaVeCekUA(r.Context(), rawURL, kaynakGorselTimeout,
		"Mozilla/5.0 (compatible; bitcosar/1.0; +https://bitcosar.com)")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", errHTTP(resp.StatusCode)
	}
	ct := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	// SVG reddedilir: içinde script taşıyabilir ve <img> ile gömülünce
	// bazı tarayıcılarda çalışır. Kaynaklarda SVG kapak yok.
	if ct != "" && (!strings.HasPrefix(ct, "image/") || ct == "image/svg+xml") {
		return nil, "", errHTTP(http.StatusUnsupportedMediaType)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, kaynakGorselMaxBayt+1))
	if err != nil {
		return nil, "", err
	}
	if len(b) > kaynakGorselMaxBayt {
		return nil, "", errHTTP(http.StatusRequestEntityTooLarge)
	}
	if len(b) == 0 {
		return nil, "", errHTTP(http.StatusNoContent)
	}
	return b, ct, nil
}

// kaynakGorselDiskYolu dağıtık dosya sistemi dostu yol üretir.
// İlk iki hex karakter alt dizin: tek dizinde binlerce dosya olmasın.
func kaynakGorselDiskYolu(hash string) string {
	return filepath.Join(platform.StorageDir(), "kaynak", hash[:2], hash)
}

func kaynakGorselOku(yol string) ([]byte, string, bool) {
	b, err := os.ReadFile(yol)
	if err != nil || len(b) == 0 {
		return nil, "", false
	}
	return b, http.DetectContentType(b), true
}

// kaynakGorselYaz atomik yazar: yarım dosya sunulmaz.
func kaynakGorselYaz(yol string, b []byte) {
	if err := os.MkdirAll(filepath.Dir(yol), 0o755); err != nil {
		return
	}
	tmp := yol + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, yol); err != nil {
		_ = os.Remove(tmp)
	}
}

func yalnizHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// errHTTP küçük yardımcı (http hata metni).
type httpHata int

func (e httpHata) Error() string { return "HTTP " + itoa(int(e)) }

func errHTTP(kod int) error { return httpHata(kod) }
