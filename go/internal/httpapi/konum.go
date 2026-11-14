package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// ZİYARETÇİ KONUMU — sunucu tarafı.
//
// İKİ KATMAN (bkz. migration 0130):
//  1) ÜLKE — Cloudflare `CF-IPCountry` başlığından, İZİN SORMADAN.
//     Her ziyaretçi için çalışır, maliyeti sıfır (ek ağ isteği yok).
//  2) GERÇEK KONUM — tarayıcı `navigator.geolocation`, kullanıcı izniyle.
//     Enlem/boylam gelir; "gerçek konum" budur ve panelde ayrı listelenir.
//
// ÜÇÜNCÜ TARAFA VERİ GİTMEZ: tersine coğrafi API çağrısı yok. Ülke zaten
// Cloudflare'in bizim isteğimize eklediği başlıkta geliyor.

const (
	// vidCookie ziyaretçi kimliği. Kişiyi DEĞİL tarayıcıyı tanır; oturum
	// kimliğiyle veya e-posta ile hiçbir yerde birleştirilmez.
	vidCookie = "tb_vid"
	// geoCookie kayıt sıklığını sınırlar: aynı ziyaretçi için saatte en
	// fazla bir DB yazımı. Değer: "<ULKE>|<unix>".
	geoCookie = "tb_geo"
	// konumCookie tarayıcının izin durumu: "1" = verdi, "0" = reddetti.
	// JS okuyabilsin diye HttpOnly DEĞİL; içinde kişisel veri yok.
	konumCookie = "tb_konum"

	geoKayitAralik = time.Hour
)

// ziyaretciID çerezdeki kimliği döndürür; yoksa (yaz=true ise) üretip yazar.
func (s *Server) ziyaretciID(w http.ResponseWriter, r *http.Request, yaz bool) string {
	if c, err := r.Cookie(vidCookie); err == nil && gecerliUUID(c.Value) {
		return c.Value
	}
	if !yaz {
		return ""
	}
	id := yeniUUID()
	if id == "" {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name: vidCookie, Value: id, Path: "/",
		MaxAge: 365 * 24 * 3600, HttpOnly: true,
		Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode,
	})
	return id
}

func gecerliUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// yeniUUID RFC 4122 v4 kimlik üretir.
//
// crypto/rand hatasında BOŞ döner — sabit bir kimlik üretmek tüm
// ziyaretçileri tek satıra düşürürdü (sessiz ve tehlikeli).
func yeniUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// cfUlke Cloudflare'in ülke başlığını doğrular.
//
// "XX" (bilinmiyor) ve "T1" (Tor) gerçek ülke değildir. Değer kullanıcı
// girdisi değil ama yine de katı doğrulanır: başlık bir gün farklı bir
// proxy'den gelirse tabloya çöp yazılmasın.
func cfUlke(r *http.Request) string {
	v := strings.ToUpper(strings.TrimSpace(r.Header.Get("CF-IPCountry")))
	if len(v) != 2 {
		return ""
	}
	for _, c := range v {
		if c < 'A' || c > 'Z' {
			return ""
		}
	}
	if v == "XX" || v == "T1" {
		return ""
	}
	return v
}

// maskeliIP panelde gösterilecek IP: son oktet(ler) gizlenir.
//
// NEDEN: yönetici "hangi ağdan geldi" bilgisini görsün ama tam IP
// gereksiz yere ekranda durmasın (ekran görüntüsü/paylaşım riski).
func maskeliIP(s string) string {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return s
	}
	if v4 := ip.To4(); v4 != nil {
		return itoa(int(v4[0])) + "." + itoa(int(v4[1])) + "." + itoa(int(v4[2])) + ".xxx"
	}
	parca := strings.Split(ip.String(), ":")
	if len(parca) >= 2 {
		return parca[0] + ":" + parca[1] + "::xxxx"
	}
	return "xxxx"
}

// konumKaydetIP ülke bilgisini (izinsiz, kaba) kaydeder.
//
// SIKLIK SINIRI: çerezde son kayıt zamanı tutulur; aynı ziyaretçi için
// saatte en fazla bir yazım. Aksi hâlde her sayfa görüntülemesi bir
// UPDATE demek olurdu (trafik = DB yükü).
func (s *Server) konumKaydetIP(w http.ResponseWriter, r *http.Request) {
	ulke := cfUlke(r)
	if ulke == "" {
		return
	}
	if c, err := r.Cookie(geoCookie); err == nil {
		if kod, ts, ok := strings.Cut(c.Value, "|"); ok && kod == ulke {
			if n, err := strconv.ParseInt(ts, 10, 64); err == nil &&
				time.Since(time.Unix(n, 0)) < geoKayitAralik {
				return // taze: yazma yok
			}
		}
	}
	id := s.ziyaretciID(w, r, true)
	if id == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: geoCookie, Value: ulke + "|" + strconv.FormatInt(time.Now().Unix(), 10),
		Path: "/", MaxAge: 30 * 24 * 3600, HttpOnly: true,
		Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode,
	})
	if err := repo.ZiyaretciKaydet(r.Context(), s.pool, repo.ZiyaretciKonum{
		VisitorID: id,
		IP:        clientIP(r),
		Ulke:      ulke,
		Kaynak:    "ip",
		UserAgent: kisalt(strings.TrimSpace(r.UserAgent()), 300),
	}); err != nil {
		s.logger.Warn("ziyaretçi konum (ip) kaydedilemedi", "error", err)
	}
}

// konumAPI POST /api/konum — tarayıcının izinle gönderdiği gerçek konum.
//
// GÜVENLİK:
//
//	· Gövde 1 KB ile sınırlı (gerçek yük ~100 bayt).
//	· Koordinat aralıkları katı denetlenir (enlem ±90, boylam ±180).
//	· Koordinat 4 ondalığa YUVARLANIR (~11 m) — kapı numarası değil.
//	· Ziyaretçi çerezi yoksa istek reddedilir: anonim konum tutmuyoruz.
//	· Yalnız POST (önbellek/CSRF yüzeyi küçük).
func (s *Server) konumAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var g struct {
		Enlem     *float64 `json:"enlem"`
		Boylam    *float64 `json:"boylam"`
		DogrulukM *int     `json:"dogruluk_m"`
		Sehir     string   `json:"sehir"`
		Ulke      string   `json:"ulke"`
		Red       bool     `json:"red"`
	}
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, "geçersiz gövde", http.StatusBadRequest)
		return
	}

	id := s.ziyaretciID(w, r, true)
	if id == "" {
		http.Error(w, "ziyaretçi kimliği üretilemedi", http.StatusInternalServerError)
		return
	}

	// REDDEDİLDİ: yalnız ülke bilgisini tazele, koordinat yazma.
	if g.Red {
		s.konumCerezYaz(w, r, "0")
		s.konumKaydetIP(w, r)
		konumJSON(w, http.StatusOK, map[string]any{"ok": true, "durum": "red"})
		return
	}

	if g.Enlem == nil || g.Boylam == nil {
		http.Error(w, "enlem/boylam gerekli", http.StatusBadRequest)
		return
	}
	if *g.Enlem < -90 || *g.Enlem > 90 || *g.Boylam < -180 || *g.Boylam > 180 {
		http.Error(w, "koordinat aralık dışı", http.StatusBadRequest)
		return
	}
	enlem := yuvarla4(*g.Enlem)
	boylam := yuvarla4(*g.Boylam)
	var dogruluk *int
	if g.DogrulukM != nil && *g.DogrulukM >= 0 && *g.DogrulukM <= 100000 {
		d := *g.DogrulukM
		dogruluk = &d
	}
	ulke := cfUlke(r)
	if ulke == "" {
		ulke = temizUlke(g.Ulke)
	}

	s.konumCerezYaz(w, r, "1")
	if err := repo.ZiyaretciKaydet(r.Context(), s.pool, repo.ZiyaretciKonum{
		VisitorID: id,
		IP:        clientIP(r),
		Ulke:      ulke,
		Sehir:     kisalt(strings.TrimSpace(g.Sehir), 80),
		Enlem:     &enlem,
		Boylam:    &boylam,
		DogrulukM: dogruluk,
		Kaynak:    "tarayici",
		Izin:      true,
		UserAgent: kisalt(strings.TrimSpace(r.UserAgent()), 300),
	}); err != nil {
		s.logger.Error("ziyaretçi konum (tarayıcı) kaydedilemedi", "error", err)
		http.Error(w, "kaydedilemedi", http.StatusInternalServerError)
		return
	}
	konumJSON(w, http.StatusOK, map[string]any{"ok": true, "durum": "izin"})
}

func (s *Server) konumCerezYaz(w http.ResponseWriter, r *http.Request, deger string) {
	http.SetCookie(w, &http.Cookie{
		Name: konumCookie, Value: deger, Path: "/",
		MaxAge: 30 * 24 * 3600,
		Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode,
	})
}

func konumJSON(w http.ResponseWriter, kod int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(kod)
	_ = json.NewEncoder(w).Encode(v)
}

// yuvarla4 4 ondalığa yuvarlar (~11 m).
func yuvarla4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func temizUlke(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 {
		return ""
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return ""
		}
	}
	return s
}

// konumZorunlu panel kapısının açık olup olmadığını söyler.
//
// Ayar: baglantilar|konum_zorunlu. Varsayılan AÇIK (kullanıcı isteği:
// "lokasyon açmadan üyelik paneline giremiyordu"). Yönetici kapıyı
// kapatabilir — kapalıyken konum yine toplanır, ama panel engellenmez.
func (s *Server) konumZorunlu(ctx context.Context) bool {
	return strings.TrimSpace(string(s.cachedSetting(ctx, "baglantilar", "konum_zorunlu"))) != "0"
}

// konumDurumu isteğin konum durumunu döndürür: "izin", "red" veya "".
func konumDurumu(r *http.Request) string {
	if c, err := r.Cookie(konumCookie); err == nil {
		switch c.Value {
		case "1":
			return "izin"
		case "0":
			return "red"
		}
	}
	return ""
}
