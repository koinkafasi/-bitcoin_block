package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

const (
	kaynakGorselMaxBayt = 8 << 20
	kaynakGorselTimeout = 12 * time.Second
)

// ogSabitTarih varsayılan OG görselinin Last-Modified değeri.
// Sabit: içerik derlemede belirleniyor, sunucu her açılışta değişmemeli
// (değişirse tarayıcı/CDN önbelleği gereksiz tazelenir).
var ogSabitTarih = time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)

func gecerliKaynakGorselHash(h string) bool {
	return len(h) == 32 && yalnizHex(h)
}

func (s *Server) handleKaynakGorsel(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimSpace(r.PathValue("hash"))
	if !gecerliKaynakGorselHash(hash) {
		http.NotFound(w, r)
		return
	}
	b, ct, ok := s.kaynakGorselAsil(r, hash)
	if !ok {
		// Hash VERİTABANINDA var ama kaynak görsel indirilemiyor (hotlink
		// 403, zaman aşımı, SSRF koruması). 404 yerine markalı yer tutucu
		// sunuyoruz: `og:image` bu adrese işaret ediyor ve sosyal botlar
		// 404'ü kalıcı önbelleğe alıp bir daha denemiyordu.
		if repo.KaynakGorselURL(r.Context(), s.pool, hash) != "" {
			s.varsayilanGorselSun(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	s.kaynakGorselSun(w, r, b, ct)
}

// varsayilanGorselSun markalı OG kartını görsel yerine sunar.
func (s *Server) varsayilanGorselSun(w http.ResponseWriter, r *http.Request) {
	b := varsayilanOGPng()
	if len(b) == 0 {
		http.NotFound(w, r)
		return
	}
	// Kısa önbellek: kaynak görsel sonradan indirilebilir hâle gelebilir,
	// o yüzden `immutable` DEĞİL.
	w.Header().Set("Cache-Control", "public, max-age=3600")
	s.kaynakGorselSun(w, r, b, "image/png")
}

func (s *Server) kaynakGorselAsil(r *http.Request, hash string) ([]byte, string, bool) {
	if b, ct, ok := kaynakGorselOku(hash, ""); ok {
		return b, ct, true
	}

	kaynak := repo.KaynakGorselURL(r.Context(), s.pool, hash)
	if kaynak == "" {
		return nil, "", false
	}

	b, ct, err := kaynakGorselIndir(r, kaynak)
	if err != nil {
		s.logger.Warn("kaynak görsel indirilemedi", "hash", hash, "error", err)
		// Görsel gelmezse yarım sayfa bırakmayalım: 404 → tarayıcı
		// `<img>` için yer tutucu gösterir, düzen bozulmaz.
		return nil, "", false
	}
	kaynakGorselYaz(hash, "", b)
	repo.KaynakGorselIsaretle(r.Context(), s.pool, hash, ct, int64(len(b)))
	return b, ct, true
}

func (s *Server) handleKaynakGorselOlcekli(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimSpace(r.PathValue("hash"))
	if !gecerliKaynakGorselHash(hash) {
		http.NotFound(w, r)
		return
	}
	genislik, err := strconv.Atoi(r.PathValue("genislik"))
	if err != nil || !kaynakGorselGenislikGecerli(genislik) {
		http.NotFound(w, r)
		return
	}

	vYol := "-" + strconv.Itoa(genislik)
	if b, _, ok := kaynakGorselOku(hash, vYol); ok {
		s.kaynakGorselSun(w, r, b, "image/jpeg")
		return
	}

	asil, ct, ok := s.kaynakGorselAsil(r, hash)
	if !ok {
		// Ölçekli istekte de aynı kural: hash tanımlıysa 404 yerine
		// markalı yer tutucu (bkz. handleKaynakGorsel).
		if repo.KaynakGorselURL(r.Context(), s.pool, hash) != "" {
			s.varsayilanGorselSun(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}

	kucuk, err := platform.ResizeToWidth(asil, genislik)
	if err != nil {
		// Kodlama desteklenmiyorsa orijinali ver: büyük ama görünür.
		// Boş kare bırakmaktan iyidir.
		s.logger.Warn("kaynak görsel ölçeklenemedi",
			"hash", hash, "genislik", genislik, "error", err)
		s.kaynakGorselSun(w, r, asil, ct)
		return
	}
	// Yazma başarısız olsa bile yanıtı döndür; sonraki istek yeniden üretir.
	kaynakGorselYaz(hash, vYol, kucuk)
	s.kaynakGorselSun(w, r, kucuk, "image/jpeg")
}

func kaynakGorselGenislikGecerli(genislik int) bool {
	switch genislik {
	case 320, 480, 768, 960, 1200, 1600:
		return true
	}
	return false
}

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
	return kaynakGorselIndirCtx(r.Context(), rawURL)
}

// kaynakGorselIndirCtx istekten BAĞIMSIZ indirme.
//
// Önden indirme işçisinin elinde `*http.Request` yok; tek ihtiyacı olan
// şey zaten istek bağlamıydı. İki yol aynı doğrulama ve sınırları
// kullansın diye gövde buraya alındı — ayrı bir indirici yazılsaydı
// SVG reddi, boyut sınırı ve SSRF doğrulaması iki yerde ayrı ayrı
// bakılırdı.
func kaynakGorselIndirCtx(ctx context.Context, rawURL string) ([]byte, string, error) {
	resp, err := platform.DogrulaVeCekUA(ctx, rawURL, kaynakGorselTimeout,
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

func kaynakGorselKok() (*os.Root, error) {
	kok := filepath.Join(platform.StorageDir(), "kaynak")
	if err := os.MkdirAll(kok, 0o755); err != nil {
		return nil, err
	}
	return os.OpenRoot(kok)
}

// kaynakGorselGoreli kökten göreli ad: "<ilk2>/<ad>".
// Ad yalnız doğrulanmış hash ve (varsa) "-<genişlik>" son ekinden oluşur.
func kaynakGorselGoreli(hash, ek string) string {
	return filepath.Join(hash[:2], hash+ek)
}

func kaynakGorselOku(hash, ek string) ([]byte, string, bool) {
	kok, err := kaynakGorselKok()
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = kok.Close() }()
	b, err := kok.ReadFile(kaynakGorselGoreli(hash, ek))
	if err != nil || len(b) == 0 {
		return nil, "", false
	}
	return b, http.DetectContentType(b), true
}

// kaynakGorselYaz atomik yazar: yarım dosya sunulmaz.
func kaynakGorselYaz(hash, ek string, b []byte) {
	kok, err := kaynakGorselKok()
	if err != nil {
		return
	}
	defer func() { _ = kok.Close() }()
	if err := kok.MkdirAll(hash[:2], 0o755); err != nil {
		return
	}
	ad := kaynakGorselGoreli(hash, ek)
	tmp := ad + ".tmp"
	if err := kok.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	if err := kok.Rename(tmp, ad); err != nil {
		_ = kok.Remove(tmp)
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
