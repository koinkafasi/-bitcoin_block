package httpapi

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"theblock/site/internal/auth"
)

// sayfaOnbellekTTL tam sayfa önbelleğinin TAZE ömrü.
//
// HTTP yanıtı zaten `max-age=60, s-maxage=300` ile önbellekleniyor; 10 sn'lik
// süreç içi katman kullanıcıya görünen tazeliği değiştirmez.
var sayfaOnbellekTTL = 10 * time.Second

// sayfaOnbellekBayatPencere taze ömür geçtikten sonra BAYAT gövdenin ne kadar
// daha sunulacağı.
//
// NEDEN: ilk sürüm TTL dolunca önbelleği tamamen ıskalıyordu; 50 eşzamanlı
// istek TTL anına denk gelirse hepsi aynı anda render + gzip yapıyordu
// (ölçüm: p-max 700 ms sıçrama). Artık süresi dolmuş gövde HEMEN sunulur,
// yenisini tek bir arka plan goroutine'i üretir — kullanıcı beklemez.
//
// 30 sn, HTTP `max-age=60`'ın altında: tarayıcı/kenar önbelleği zaten daha
// eskisini sunmuyor. Haber içeriği dakikalar mertebesinde değiştiği için
// görünür bir bayatlama yaratmıyor.
var sayfaOnbellekBayatPencere = 30 * time.Second

// sayfaOnbellekMaxGovde önbelleğe alınacak en büyük gövde.
const sayfaOnbellekMaxGovde = 512 << 10 // 512 KB

// sayfaOnbellekMaxGiris toplam giriş sayısı sınırı (~40 MB üst sınır).
const sayfaOnbellekMaxGiris = 400

// sayfaOnbellekArkaPlanButce arka plan yenilemesinin üst süresi.
const sayfaOnbellekArkaPlanButce = 20 * time.Second

// sayfaOnbellekBasliklari replays'te kopyalanacak başlıklar.
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
	// okunmaID: gövdeyi üreten handler'ın bildirdiği içerik kimliği (makale
	// sayfaları). İsabette okunma sayacı bununla artırılır; aksi hâlde
	// önbellek, okunma sayılarını tümden durdururdu.
	okunmaID int64
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

	// sayaçlar: isabet/bayat/ıska görünürlüğü.
	isabet, bayat, iska atomic.Int64
	arsivYenileme       atomic.Int64
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

// al kaydı ve tazelik durumunu döndürür.
//
//	varMi=false        → hiç yok (senkron üretim gerekir)
//	varMi=true, taze   → hemen sun
//	varMi=true, !taze  → bayat sun + arka planda yenile
//	                        (bayat pencere aşıldıysa yine senkron üretim)
func (p *sayfaOnbellek) al(anahtar [32]byte) (g *sayfaGirdi, taze bool, varMi bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.m[anahtar]
	if !ok {
		return nil, false, false
	}
	yas := time.Since(e.at)
	if yas < sayfaOnbellekTTL {
		return e, true, true
	}
	if yas < sayfaOnbellekTTL+sayfaOnbellekBayatPencere {
		return e, false, true
	}
	return nil, false, false // çok eski: yeniden üret
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

// kilit tek-uçuş (singleflight) kilidi.
//
// lider=true → çağıran üretimi yapar ve birak() çağırmalıdır.
// lider=false → bekle kanalı kapanana kadar beklenir.
func (p *sayfaOnbellek) kilit(anahtar [32]byte) (lider bool, bekle chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, varMi := p.calls[anahtar]; varMi {
		return false, c
	}
	c := make(chan struct{})
	p.calls[anahtar] = c
	return true, c
}

func (p *sayfaOnbellek) birak(anahtar [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, varMi := p.calls[anahtar]; varMi {
		delete(p.calls, anahtar)
		close(c)
	}
}

// SetSayfaOnbellekSuresi TTL ve bayat pencereyi ayarlar (ölçüm/işletim için).
//
// Sıfır veya negatif değer yok sayılır: yanlış yapılandırma önbelleği
// kapatmasın (bayat pencere TTL'den kısa olamaz, kırpılır).
func SetSayfaOnbellekSuresi(ttl, bayat time.Duration) {
	if ttl <= 0 {
		return
	}
	if bayat <= 0 {
		bayat = 3 * ttl
	}
	sayfaOnbellekTTL = ttl
	sayfaOnbellekBayatPencere = bayat
}

// SayfaOnbellekSuresi yapılandırılmış değerleri döndürür (log/tanı).
func SayfaOnbellekSuresi() (ttl, bayat time.Duration) {
	return sayfaOnbellekTTL, sayfaOnbellekBayatPencere
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

// sayfaOnbellekHaricOnekler makale desenine benzese bile ASLA önbelleğe
// girmeyecek kökler.
//
// NEDEN: makale deseni (4-5 segment + tarih) geniş; `/api/2026/11/05/x`
// gibi bir yol da desene uyuyor. Bugün `/api` mux'ta ayrı çözülüyor (yani
// pratikte buraya düşmez), ama rota değişirse kişisel/veri yolu önbelleğe
// girerdi. Bu liste bir sözleşme değil, emniyet kemeri.
var sayfaOnbellekHaricOnekler = []string{
	"/api/", "/admin", "/media/", "/kg/", "/canli/", "/giris", "/hesabim",
	"/static/", "/internal/", "/webhooks/", "/amp/", "/sitemap", "/indexnow/",
	"/sw.js", "/robots.txt", "/ads.txt",
}

// sayfaYoluOnbelleklenir yalnız HERKESE AÇIK CD sayfaları.
//
// Yönetim paneli (/admin), API (/api), medya (/media, /kg), canlı akış
// (/canli), oturum yolları (/giris, /hesabim…) ve sitemap/robots gibi
// üretilen dosyalar bilinçli olarak DIŞARIDA: kişisel ya da ucuz.
func sayfaYoluOnbelleklenir(yol string) bool {
	for _, o := range sayfaOnbellekHaricOnekler {
		if strings.HasPrefix(yol, o) {
			return false
		}
	}
	if yol == "/" || yol == "/cd" {
		return true
	}
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
	// MAKALE sayfaları — bir haber sitesinde trafiğin çoğu burası.
	// Ölçüm (9 Kasım 2026): makale 20-100 ms (cache dışı) → önbellekte
	// 0,4 ms. Tarih deseni zorunlu: rastgele çok segmentli yollar
	// (panel, hesap, API) buraya takılmasın.
	return makaleYoluMu(yol)
}

// makaleYoluMu /{bolum}/{YYYY}/{MM}/{DD}/{slug} ve /{YYYY}/{MM}/{DD}/{slug}
// desenlerini tanır.
//
// SON SEGMENT bilinçli olarak denetlenmez (slug serbest metin); kapsamı
// daraltan şey tarih üçlüsü. Desen dışı her yol önbellek DIŞINDA kalır.
func makaleYoluMu(yol string) bool {
	s := strings.Split(strings.Trim(yol, "/"), "/")
	switch len(s) {
	case 5: // bölüm + tarih + slug
		s = s[1:] // bölüm adını at → [YYYY MM DD slug]
	case 4: // tarih + slug
	default:
		return false
	}
	if s[3] == "" {
		return false
	}
	return len(s[0]) == 4 && sayiMi(s[0]) && sayiMi(s[1]) && sayiMi(s[2])
}

// sayiMi s boş değilse ve yalnız rakamlardan oluşuyorsa true.
func sayiMi(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

// yazmayanWriter arka plan yenilemesinde gövdeyi istemciye değil boşluğa
// yazar; yalnız başlıkları tutar (cacheWriter zaten gövdeyi yakalıyor).
type yazmayanWriter struct{ h http.Header }

func (w *yazmayanWriter) Header() http.Header {
	if w.h == nil {
		w.h = http.Header{}
	}
	return w.h
}
func (w *yazmayanWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *yazmayanWriter) WriteHeader(int)             {}

// withSayfaOnbellek herkese açık CD sayfalarını önbelleğe alır.
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
		cache := s.sayfaCache

		if g, taze, varMi := cache.al(anahtar); varMi {
			if taze {
				cache.isabet.Add(1)
			} else {
				cache.bayat.Add(1)
			}
			// OKUNMA İSABETTE DE SAYILIR. Makale gövdesi önbellekten gelince
			// handler çalışmaz; sayaç burada artırılmazsa popülerlik verisi
			// (counters_hit/hit_daily) önbellek açıkken neredeyse dururdu.
			if g.okunmaID != 0 {
				okunmaTamponu.ekle(g.okunmaID)
			}
			s.sayfaSun(w, r, g)
			if !taze {
				// BAYAT SUNULDU: kullanıcı beklemedi. Yenileme arka planda,
				// tek bir goroutine'de (singleflight) yapılır.
				s.sayfaArkaPlandaYenile(next, r, anahtar, g.okunmaID)
			}
			return
		}

		// HİÇ YOK: tek lider üretir, eşzamanlı diğer istekler bekler.
		// Beklemeden hepsi render etseydi TTL anında N kat CPU harcanırdı
		// (ölçüm: 50 eşzamanlıda p-max 700 ms sıçrama).
		cache.iska.Add(1)
		lider, bekle := cache.kilit(anahtar)
		if lider {
			defer cache.birak(anahtar)
			cw := &cacheWriter{ResponseWriter: w}
			// Handler'a sayaç işareti ver: hangi içeriğin okunduğunu
			// bildirsin, sayımı biz yapalım (isabette de aynı yol).
			isaret := &okunmaIsareti{}
			r = r.WithContext(context.WithValue(r.Context(), okunmaIsaretAnahtari{}, isaret))
			next.ServeHTTP(cw, r)
			if g, ok := sayfaGirdiCikar(cw); ok {
				g.okunmaID = isaret.al()
				if g.okunmaID != 0 {
					okunmaTamponu.ekle(g.okunmaID)
				}
				cache.koy(anahtar, g)
			}
			return
		}

		select {
		case <-bekle:
		case <-r.Context().Done():
			return
		}
		if g, _, varMi := cache.al(anahtar); varMi {
			s.sayfaSun(w, r, g)
			return
		}
		// Lider saklanabilir bir yanıt üretmedi (ör. 500): doğrudan render.
		next.ServeHTTP(w, r)
	})
}

// sayfaSun kaydı istemciye yazar.
func (s *Server) sayfaSun(w http.ResponseWriter, r *http.Request, g *sayfaGirdi) {
	// İstek kimliği YENİ üretilir: önbellekten gelen kimlik tüm isteklerde
	// aynı olurdu (log/tracing bozulurdu).
	rid := genID()
	w.Header().Set("X-Request-Id", rid)
	for _, ad := range sayfaOnbellekBasliklari {
		if v, varMi := g.baslik[ad]; varMi {
			w.Header().Set(ad, v)
		}
	}
	// Content-Length her zaman elimizdeki gövdeden hesaplanır: gzip
	// middleware uzunluğu sildiği için saklanan başlıkta bulunmaz.
	w.Header().Set("Content-Length", strconv.Itoa(len(g.govde)))
	w.WriteHeader(g.durum)
	_, _ = w.Write(g.govde)

	// İSABET LOGLANIR: ara katman atlandığı için withRequestID çalışmıyor;
	// log yazılmazsa trafik panolarda "kaybolmuş" görünür.
	if s.logger != nil {
		s.logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", g.durum,
			"dur_ms", 0,
			"req_id", rid,
			"cache", "hit",
		)
	}
}

// sayfaArkaPlandaYenile bayat kaydı tek bir arka plan goroutine'inde tazeler.
//
// Bağlam: istek bağlamı istek bitince İPTAL OLUR. Arka plan işi ona
// bağlanırsa yarıda kalır; bu yüzden WithoutCancel kullanılır ve ayrı bir
// bütçe verilir.
func (s *Server) sayfaArkaPlandaYenile(next http.Handler, r *http.Request, anahtar [32]byte, okunmaID int64) {
	cache := s.sayfaCache
	lider, _ := cache.kilit(anahtar)
	if !lider {
		return // yenileme zaten uçuşta
	}
	cache.arsivYenileme.Add(1)

	go func() {
		defer cache.birak(anahtar)
		// Bağlam: istek bağlamı istek bitince İPTAL OLUR. Arka plan işi ona
		// bağlanırsa yarıda kalır; WithoutCancel + ayrı bütçe kullanılır.
		ctx, iptal := context.WithTimeout(context.WithoutCancel(r.Context()), sayfaOnbellekArkaPlanButce)
		defer iptal()
		// İsteğin kopyası: başlıklar (Accept-Encoding) korunur, gövde yok.
		// Sayım kapatılır: yenileme kullanıcı okuması değildir.
		kopya := r.Clone(okunmaSayilmasin(ctx))
		kopya.Header.Set("X-Request-Id", "arka-plan-yenileme")
		cw := &cacheWriter{ResponseWriter: &yazmayanWriter{}}
		next.ServeHTTP(cw, kopya)
		if g, ok := sayfaGirdiCikar(cw); ok {
			// okunmaID korunur: sonraki isabetler yine sayılsın.
			g.okunmaID = okunmaID
			cache.koy(anahtar, g)
		}
	}()
}

// sayfaGirdiCikar yakalanan yanıttan önbellek kaydı üretir.
//
// SAKLANMAZ: 200 dışı, taşmış, boş, çerez yazan veya yönlendiren yanıtlar.
// Çerez yazan bir yanıt saklanırsa oturum çerezi herkese dağılırdı.
func sayfaGirdiCikar(cw *cacheWriter) (*sayfaGirdi, bool) {
	if cw.durum != http.StatusOK || cw.tasma || len(cw.govde) == 0 {
		return nil, false
	}
	if len(cw.Header().Values("Set-Cookie")) > 0 {
		return nil, false
	}
	if cw.Header().Get("Location") != "" {
		return nil, false
	}
	baslik := make(map[string]string, len(sayfaOnbellekBasliklari))
	for _, ad := range sayfaOnbellekBasliklari {
		if v := cw.Header().Get(ad); v != "" {
			baslik[ad] = v
		}
	}
	return &sayfaGirdi{
		at:     time.Now(),
		durum:  cw.durum,
		baslik: baslik,
		govde:  cw.govde,
	}, true
}

// temizle tüm sayfa önbelleğini boşaltır.
//
// NEDEN: marka/logo gibi HER SAYFADA görünen bir ayar değiştiğinde
// yalnız kabuk (cdShellCache) değil, TAM SAYFA önbelleği de eskir.
// Ölçüm (11 Kasım 2026): logo kaydedildikten sonra ana sayfa 4 saniye
// boyunca eski logoyu gösterdi — sayfa önbelleği TTL'i 10 sn.
func (p *sayfaOnbellek) temizle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m = map[[32]byte]*sayfaGirdi{}
	p.sira = nil
}
