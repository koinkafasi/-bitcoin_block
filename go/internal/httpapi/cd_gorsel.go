package httpapi

import (
	"bytes"
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
	b, ct, ok := s.kaynakGorselAsil(r, hash)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.kaynakGorselSun(w, r, b, ct)
}

// kaynakGorselAsil orijinal baytları verir: diskte varsa oradan, yoksa
// indirip atomik yazdıktan sonra.
//
// İki uç da (orijinal ve ölçekli) buradan geçer — indirme, SSRF koruması ve
// DB işaretlemesi TEK yerde kalsın. Ayrıldığında biri güncellenmeden diğeri
// kalıyor; /kg/ ucunun güvenlik kapısı tam bu akışta.
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

// handleKaynakGorselOlcekli — kaynak görselin ölçeklenmiş hâli.
//
// GET /kg/{hash}/{genislik}
//
// NEDEN (ölçüm, 5 Kasım 2026): /kg/ önbellekliyordu ama BOYUTLANDIRMIYORDU.
// Ana sayfadaki beş görsel canlı ölçüldü: 708+262+245+106+65 KB = 1,39 MB;
// dokuz görselle ~2,5 MB. Oysa en geniş yuva 474x266 (lead 734x413) — yani
// 708 KB'lık bir PNG 292x164'lük karta giriyordu. Ölçülen variant/480
// ortalaması 27 KB; aynı sayfa ~250 KB'ye iniyor.
//
// Akış orijinalle aynı iskelet: diskte varsa sun, yoksa üret-yaz-sun.
// Ölçekli dosya orijinalin yanında durur, adı genişlikle ayrışır.
//
// Çıktı DAİMA JPEG (platform.ResizeToWidth öyle kodluyor). Bu bilinçli:
// kaynaklarda WebP/PNG karışık geliyor, tek biçim önbelleği ve Content-Type'ı
// basitleştiriyor. Animasyonlu GIF animasyonunu yitirir — haber kapaklarında
// animasyon yok, ölçüldü.
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

// kaynakGorselGenislikGecerli izin verilen genişlikler.
//
// Liste KAPALI ve bu bir güvenlik kararı: istemci serbest genişlik
// isteyebilseydi her istek yeni bir dosya üretirdi — /kg/<hash>/1,
// /kg/<hash>/2 … ile disk doldurulur ve her biri CPU yakar (ölçekleme
// pahalı). Değerler mediaSrcset ile aynı tutuluyor ki iki görsel sistemi
// ayrı listelere düşmesin.
func kaynakGorselGenislikGecerli(genislik int) bool {
	switch genislik {
	case 320, 480, 768, 960, 1200, 1600:
		return true
	}
	return false
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

// kaynakGorselKok depolama kökünü "hapishane" olarak açar (storage/kaynak).
//
// NEDEN os.Root (Go 1.24+): hash doğrulaması doğru ama SAVUNMA TEK KATMAN
// olmamalı. Doğrulama bir gün gevşetilirse (ya da çağrı yolu değişirse)
// `..` içeren bir ad storage dışına çıkabilirdi. os.Root çekirdek tarafından
// uygulanır: kökün dışına çıkan her işlem hata döner — kod ne yazarsa yazsın.
//
// (Ölçüm, 6 Kasım 2026: gosec G703 bu dosyada 5 bulgu verdi — taint analizi
// `gecerliKaynakGorselHash` doğrulamasını göremiyor. Bulgu "yanlış pozitif"
// olsa da işaret ettiği şey gerçek: yol güvenliği tek bir doğrulama
// fonksiyonuna bağlıydı. os.Root bunu yapısal garantiye çevirir.)
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
	defer kok.Close()
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
	defer kok.Close()
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
