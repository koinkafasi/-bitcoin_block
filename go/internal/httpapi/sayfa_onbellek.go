package httpapi

import (
	"crypto/sha256"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"theblock/site/internal/auth"
)

// sayfaOnbellekTTL tam sayfa önbelleğinin ömrü.
//
// HTTP yanıtı zaten `max-age=60, s-maxage=300` ile önbellekleniyor; 10 sn'lik
// süreç içi katman kullanıcıya görünen tazeliği değiştirmez.
const sayfaOnbellekTTL = 10 * time.Second

// sayfaOnbellekMaxGovde önbelleğe alınacak en büyük gövde. Daha büyük yanıtlar
// (nadir) önbelleğe alınmaz: bellek sınırlı kalsın.
const sayfaOnbellekMaxGovde = 512 << 10 // 512 KB

// sayfaOnbellekMaxGiris toplam giriş sayısı sınırı (~40 MB üst sınır).
const sayfaOnbellekMaxGiris = 400

// sayfaOnbellek replays'te kopyalanacak başlıklar.
//
// BEYAZ LİSTE bilinçli: `X-Request-Id` istek başına üretilir ve önbellekten
// gelirse tüm istekler aynı kimliği taşır (log/tracing bozulur). `Set-Cookie`
// asla kopyalanmaz — önbelleklenmiş bir oturum çerezi herkese dağıtılırdı.
var sayfaOnbellekBasliklari = []string{
	"Content-Type",
	"Content-Encoding",
	"Content-Language",
	"Cache-Control",
	"Vary",
	"Content-Security-Policy",
	"Permissions-Policy",
	"Referrer-Policy",
	"Report-To",
	"Strict-Transport-Security",
	"X-Content-Type-Options",
	"X-Frame-Options",
}

// sayfaGirdi tek bir önbellek kaydı.
type sayfaGirdi struct {
	at     time.Time
	durum  int
	baslik map[string]string
	govde  []byte
}

// sayfaOnbellek tam sayfa (render + CSP + gzip) önbelleği.
//
// NEDEN EN DIŞTA: gzip ara katmanı da dışarıda kalırsa isabet eden istek
// render, şablon, CSP hash üretimi VE sıkıştırmayı birlikte atlar. Profil
// (9 Kasım 2026) gzip'i CPU'nun %31'i, şablon+CSP'yi ~%15'i olarak
// gösteriyordu; ikisi birlikte istek başına CPU'nun yarısı.
//
// YALNIZ ANONİM: gövdeye oturum durumu (kullanıcı adı, okunmamış bildirim),
// A/B varyantı gibi kişisel alanlar giriyor. `tb_session` çerezi olan istek
// hiç önbelleğe alınmaz ve önbellekten okunmaz.
type sayfaOnbellek struct {
	mu    sync.Mutex
	m     map[[32]byte]*sayfaGirdi
	sira  [][32]byte
	calls map[[32]byte]chan struct{}

	// sayaçlar: isabet/ıska görünürlüğü (pano ve testler için).
	isabet, iska atomic.Int64
}

func newSayfaOnbellek() *sayfaOnbellek {
	return &sayfaOnbellek{
		m:     map[[32]byte]*sayfaGirdi{},
		calls: map[[32]byte]chan struct{}{},
	}
}

// sayfaAnahtar istekten önbellek anahtarı üretir.
//
// Host DAHİL: çok siteli kurulumda aynı yol farklı siteye hizmet eder.
// Kodlama DAHİL: gzip'li ve ham gövde ayrı saklanır, aksi hâlde gzip
// kabul etmeyen istemciye sıkıştırılmış bayt gitmiş olurdu.
func sayfaAnahtar(host, yol, sorgu string, gzipIster bool) [32]byte {
	k := host + "\x00" + yol + "\x00" + sorgu + "\x00"
	if gzipIster {
		k += "g"
	} else {
		k += "i"
	}
	return sha256.Sum256([]byte(k))
}

func (p *sayfaOnbellek) al(anahtar [32]byte) (*sayfaGirdi, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	g, ok := p.m[anahtar]
	if !ok || time.Since(g.at) >= sayfaOnbellekTTL {
		return nil, false
	}
	return g, true
}

func (p *sayfaOnbellek) koy(anahtar [32]byte, g *sayfaGirdi) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.sira) >= sayfaOnbellekMaxGiris {
		eski := p.sira[0]
		p.sira = p.sira[1:]
		delete(p.m, eski)
	}
	if _, varMi := p.m[anahtar]; !varMi {
		p.sira = append(p.sira, anahtar)
	}
	p.m[anahtar] = g
}

// temizle süresi dolmuş kayıtları atar (yalnız bellek sınırı için).
func (p *sayfaOnbellek) temizle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	kalan := p.sira[:0]
	for _, k := range p.sira {
		if g, ok := p.m[k]; ok && time.Since(g.at) < sayfaOnbellekTTL {
			kalan = append(kalan, k)
		} else {
			delete(p.m, k)
		}
	}
	p.sira = kalan
}

// sayfaOnbellegeUygun yol bu istek için tam sayfa önbelleğine uygun mu?
func sayfaOnbellegeUygun(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	// Oturum çerezi varsa kişiselleştirilmiş gövde üretilebilir.
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		return false
	}
	// Yetkilendirme başlığı taşıyan istek kişiseldir.
	if r.Header.Get("Authorization") != "" {
		return false
	}
	// HTMX/parça istekleri AYNI yolda FARKLI gövde döndürebilir (ör. panel
	// parçaları). Bugün herkese açık CD yollarında HX-Request'e bakan bir
	// dal yok, ama bu bir sözleşme değil: bir gün eklenirse önbellek tam
	// sayfayı parça sanıp dağıtırdı. Kapıyı şimdi kapatmak bedava.
	if r.Header.Get("HX-Request") != "" {
		return false
	}
	// İstemci açıkça tazeleme istiyorsa önbelleği atla.
	cc := strings.ToLower(r.Header.Get("Cache-Control"))
	if strings.Contains(cc, "no-cache") || strings.Contains(cc, "no-store") {
		return false
	}
	return sayfaYoluOnbelleklenir(r.URL.Path)
}

// sayfaYoluOnbelleklenir yalnız HERKESE AÇIK CD sayfaları.
//
// Yönetim paneli (/admin), API (/api), medya (/media, /kg), canlı akış
// (/canli), oturum yolları (/giris, /hesabim…) ve sitemap/robots gibi
// üretilen dosyalar bilinçli olarak DIŞARIDA: kişisel ya da ucuz.
func sayfaYoluOnbelleklenir(yol string) bool {
	if yol == "/" || yol == "/cd" {
		return true
	}
	// "/tag/" gibi SONU BÖLÜ olan önekler doğrudan; "/price" gibi tam
	// sözcük önekleri hem kendisi hem "/.../" alt yolları.
	for _, onek := range []string{"/tag/", "/video/"} {
		if strings.HasPrefix(yol, onek) {
			return true
		}
	}
	for _, tam := range []string{"/price", "/videos", "/canli-yayin",
		"/podcasts", "/sponsored-content", "/newsletters"} {
		if yol == tam || strings.HasPrefix(yol, tam+"/") {
			return true
		}
	}
	// Bölüm sayfaları (/markets, /business, /tech, /policy, /opinion,
	// /news-analysis, /latest-crypto-news, /focus, /research, /press-release)
	tekSegment := strings.TrimPrefix(yol, "/")
	if !strings.Contains(tekSegment, "/") {
		_, ok := cdSectionMeta[tekSegment]
		return ok
	}
	return false
}

// cacheWriter yanıtı yakalayan yazıcı.
type cacheWriter struct {
	http.ResponseWriter
	durum     int
	yazildi   bool
	govde     []byte
	tasma     bool
	basliklar http.Header
}

func (c *cacheWriter) Header() http.Header {
	if c.basliklar == nil {
		c.basliklar = c.ResponseWriter.Header()
	}
	return c.basliklar
}

func (c *cacheWriter) WriteHeader(code int) {
	if c.yazildi {
		return
	}
	c.yazildi = true
	c.durum = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *cacheWriter) Write(p []byte) (int, error) {
	if !c.yazildi {
		c.WriteHeader(http.StatusOK)
	}
	if !c.tasma {
		if len(c.govde)+len(p) > sayfaOnbellekMaxGovde {
			c.tasma = true
			c.govde = nil
		} else {
			c.govde = append(c.govde, p...)
		}
	}
	return c.ResponseWriter.Write(p)
}

// Unwrap ResponseController'ın alt yazıcıya erişmesini sağlar.
func (c *cacheWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

// withSayfaOnbellek herkese açık CD sayfalarını 10 sn süreç içi önbelleğe alır.
//
// Yerleşim: zincirin EN DIŞINDA (gzip ve HTMLCSP'nin üstünde). Böylece
// isabet eden istek render + CSP + gzip işlerinin TAMAMINI atlar.
func (s *Server) withSayfaOnbellek(next http.Handler) http.Handler {
	if s.sayfaCache == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sayfaOnbellegeUygun(r) {
			next.ServeHTTP(w, r)
			return
		}

		// BLOKLU IP KONTROLÜ — ÖNCE, İSABETTEN ÖNCE.
		//
		// Bu ara katman `withIPBlock`'un DIŞINDA duruyor (gzip'in de dışında
		// olması gerekiyor: saklanan gövde zaten sıkıştırılmış, tekrar
		// sıkıştırmak bozar). Dışarıda olmanın bedeli şu olurdu: bloklanmış
		// bir IP önbellekteki sayfayı 200 ile alırdı — bot tuzağının
		// (410 + otomatik blok) koruması delinirdi. Bu yüzden kontrol
		// burada TEKRAR ediliyor. `isIPBlocked` 30 sn'lik bellek önbelleği
		// kullanır: mutex + map bakışı, DB'ye gitmez.
		if ip := clientIP(r); ip != "" && s.isIPBlocked(r.Context(), ip) {
			w.Header().Set("X-Robots-Tag", "noindex")
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		gzipIster := acceptsGzip(r)
		anahtar := sayfaAnahtar(r.Host, r.URL.Path, r.URL.RawQuery, gzipIster)

		if g, ok := s.sayfaCache.al(anahtar); ok {
			s.sayfaCache.isabet.Add(1)
			bas := time.Now()
			// İstek kimliği YENİ üretilir: önbellekten gelen kimlik tüm
			// isteklerde aynı olurdu (log/tracing bozulurdu).
			rid := genID()
			w.Header().Set("X-Request-Id", rid)
			for _, ad := range sayfaOnbellekBasliklari {
				if v, varMi := g.baslik[ad]; varMi {
					w.Header().Set(ad, v)
				}
			}
			// Content-Length her zaman elimizdeki gövdeden hesaplanır:
			// gzip middleware uzunluğu sildiği için saklanan başlıkta
			// bulunmaz, ama replay'de kesin olarak biliyoruz.
			w.Header().Set("Content-Length", strconv.Itoa(len(g.govde)))
			w.WriteHeader(g.durum)
			_, _ = w.Write(g.govde)
			// İSABET LOGLANIR: ara katman atlandığı için withRequestID
			// çalışmıyor; log yazılmazsa trafik panolarda "kaybolmuş"
			// görünür. `cache:"hit"` alanı isabet olduğunu açıkça söyler.
			if s.logger != nil {
				s.logger.Info("http",
					"method", r.Method,
					"path", r.URL.Path,
					"status", g.durum,
					"dur_ms", time.Since(bas).Milliseconds(),
					"req_id", rid,
					"cache", "hit",
				)
			}
			return
		}

		s.sayfaCache.iska.Add(1)
		cw := &cacheWriter{ResponseWriter: w}
		next.ServeHTTP(cw, r)

		// Yalnız başarılı, çerez yazmayan, taşmamış yanıtlar saklanır.
		if cw.durum != http.StatusOK || cw.tasma || len(cw.govde) == 0 {
			return
		}
		if len(cw.Header().Values("Set-Cookie")) > 0 {
			return
		}
		// Yönlendirme/oturum yanıtları asla önbelleğe girmez.
		if cw.Header().Get("Location") != "" {
			return
		}

		baslik := make(map[string]string, len(sayfaOnbellekBasliklari)+1)
		for _, ad := range sayfaOnbellekBasliklari {
			if v := cw.Header().Get(ad); v != "" {
				baslik[ad] = v
			}
		}
		s.sayfaCache.koy(anahtar, &sayfaGirdi{
			at:     time.Now(),
			durum:  cw.durum,
			baslik: baslik,
			govde:  cw.govde,
		})
	})
}
