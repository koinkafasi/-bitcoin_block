// Package httpapi Go web/API HTTP sunucusu: genel site (SSR + edge cache) ve
// yönetim paneli (auth + içerik CRUD + medya + AI + raporlama).
package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/aieditor"
	"theblock/site/internal/auth"
	"theblock/site/internal/config"
	"theblock/site/internal/i18n"
	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
	"theblock/site/internal/searchsvc"
	"theblock/site/internal/transcribe"
	"theblock/site/internal/tts"
	"theblock/site/internal/webui"
	"theblock/site/internal/webui/templkit"
)

type Server struct {
	cfg      config.Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	i18n     *i18n.Bundle
	ai       *ai.Client
	aiEditor *aieditor.Service
	tts      *tts.Client
	// spell isteğe bağlı sözlük tabanlı yazım denetimidir (hunspell).
	spell *quality.ImlaDenetleyici
	// asrSettings konuşma tanıma ayarlarının kalıcı deposu (panelden
	// düzenlenebilir; .env yalnız varsayılan).
	asrSettings *transcribe.SettingsStore
	tmpl        *template.Template
	mux         *http.ServeMux

	siteName         string
	defaultLangCache string
	search           *searchsvc.Client

	i18nMu     sync.Mutex // çeviri paketi yeniden yüklenirken (Faz 45)
	mediaMu    sync.RWMutex
	mediaCache map[int64]string

	adMu    sync.Mutex
	adCache map[int64]adCacheEntry

	redirMu    sync.Mutex
	redirCache map[int64]redirCacheEntry

	blockMu    sync.Mutex
	blockCache map[string]bool
	blockAt    time.Time

	siteMu    sync.Mutex
	siteCache map[string]siteEntry

	assetVer string

	// içerik/ayar önbelleği (menü+footer+marka tekrarlarını azaltır)
	siteCfgMu   sync.Mutex
	catsCache   map[int64]catsCacheEntry
	kvCache     map[string]kvCacheEntry
	ytAyarCache ytAyarCacheEntry

	// sistemsel üretim (blok kapak şablonu — Faz 3) için kullanılacak kullanıcı
	sysMu     sync.Mutex
	sysUserID int64

	// CD (CoinDesk) yüzeyinin süreç içi önbellekleri. TTL `cdCacheTTL`.
	// Kabuk verisi her sayfada aynı; liste verisi ise aynı sayfa adresine
	// gelen eşzamanlı isteklerde aynı. HTTP yanıtı zaten s-maxage=300 ile
	// önbellekte; bu katman yalnız DB'yi korur.
	cdShellCache *ttlcache[cdShellData]
	cdHomeCache  *ttlcache[[]repo.Content]
	cdListeCache *ttlcache[cdListeSonuc]
	// kategoriCache slug → kategori kimliği (ana sayfa aynı 4 slug'ı her
	// render'da çeviriyordu).
	kategoriCache *ttlcache[int64]
	// ytListeCache YouTube listeleri, TÜR bazında (ana sayfa aynı listeyi
	// 6 kez çekiyordu; bkz. yt_liste.go).
	ytListeCache *ttlcache[[]repo.YoutubeKayit]

	// cspHashes gövde → CSP başlığı önbelleği (bkz. csp_hashcache.go).
	cspHashes *cspHashCache

	// sayfaCache tam sayfa önbelleği (render + CSP + gzip birlikte).
	// Yalnız anonim istekler; bkz. sayfa_onbellek.go.
	sayfaCache *sayfaOnbellek
}

// siteIDFromCtx bağlamdan site kimliğini döndürür (yoksa 1).
func siteIDFromCtx(ctx context.Context) int64 {
	if v, ok := ctx.Value(ctxSiteKey).(int64); ok && v > 0 {
		return v
	}
	return 1
}

const cacheControlPublic = "public, max-age=60, s-maxage=300"

// NewServer HTTP sunucusu kurar.
func NewServer(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool, bundle *i18n.Bundle, aiclient *ai.Client) *http.Server {
	srv, _ := NewServerApp(cfg, logger, pool, bundle, aiclient)
	return srv
}

// NewServerApp HTTP sunucusunu kurar ve uygulama örneğini de döndürür
// (gazete gibi arka plan işleri için).
func NewServerApp(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool, bundle *i18n.Bundle, aiclient *ai.Client) (*http.Server, *Server) {
	s := &Server{
		cfg:        cfg,
		logger:     logger,
		pool:       pool,
		i18n:       bundle,
		ai:         aiclient,
		aiEditor:   aieditor.NewService(pool, aiclient, logger),
		spell:      quality.YeniImlaDenetleyici(cfg.HunspellBin, cfg.HunspellDict),
		tts:        tts.New(cfg.OpenAIKey, cfg.OpenAIBase, cfg.TTSModel, cfg.TTSVoice),
		mux:        http.NewServeMux(),
		siteName:   "theblock.com.tr",
		search:     searchsvc.NewClient(os.Getenv("SEARCH_SVC_URL")),
		mediaCache: map[int64]string{},
		adCache:    map[int64]adCacheEntry{},
		redirCache: map[int64]redirCacheEntry{},
		blockCache: map[string]bool{},
		siteCache:  map[string]siteEntry{},
		assetVer:   strconv.FormatInt(time.Now().Unix(), 36),

		cdShellCache:  newTTLCache[cdShellData](cdCacheTTL),
		cdHomeCache:   newTTLCache[[]repo.Content](cdCacheTTL),
		cdListeCache:  newTTLCache[cdListeSonuc](cdCacheTTL),
		kategoriCache: newTTLCache[int64](cdCacheTTL),
		ytListeCache:  newTTLCache[[]repo.YoutubeKayit](ytListeTTL),
		cspHashes:     newCSPHashCache(),
		sayfaCache:    newSayfaOnbellek(),
	}
	s.siteName = s.initialSiteName()
	// Statik varlık sürümünü templkit'e de ver: cd_head.templ `?v=` ile
	// tarayıcı önbelleğini kırıyor. Sabit "1" kalınca tasarım değişiklikleri
	// eski önbellekten servis ediliyordu (6 Kasım 2026'da yaşandı).
	templkit.VarlikSurum = s.assetVer

	funcs := templateFuncs()
	funcs["mediaURL"] = s.mediaURL
	funcs["mediaThumbURL"] = s.mediaThumbURL
	funcs["mediaVariantURL"] = s.mediaVariantURL
	funcs["mediaSrcset"] = s.mediaSrcset
	funcs["asset"] = s.assetURL
	funcs["urlAbs"] = urlAbs
	funcs["catIcon"] = catIcon
	s.tmpl = template.Must(template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html"))

	s.routes()

	return &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      s.withSayfaOnbellek(s.disKatman(s.withRequestID(s.withSecurityHeaders(s.withSameOrigin(s.withIPBlock(s.withSite(s.withHTMLCSP(s.localeAliasMiddleware(s.mux))))))))),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, s
}

// disKatman en dış iki ara katmanı sarar: gzip ve panic kurtarma.
//
// SIRA KRİTİK — withRecover, withGzip'in İÇİNDE olmalı:
//
// gzip ara katmanı `Content-Encoding: gzip` başlığını yazar ve gövdeyi
// sıkıştırıcıya yönlendirir. Panic sonrası düz metni DOĞRUDAN ham yazıcıya
// yazsaydık, istemci "gzip" başlığıyla sıkıştırılmamış bayt alırdı; yani
// bozuk yanıt (ölçüm: `io.ReadAll` → `gzip: invalid header`). İçeride
// kalınca hata yanıtı da sıkıştırıcıdan geçer ve çerçeve tutarlı kalır.
//
// Bu fonksiyon ayrı duruyor ki testler ÜRETİMDEKİ sırayı sınasın
// (recover_gzip_test.go). Test kendi zincirini kursaydı, üretimdeki sıra
// bozulsa bile test geçerdi — bu tuzağa bir kez düşüldü.
func (s *Server) disKatman(ic http.Handler) http.Handler {
	return s.withGzip(s.withRecover(ic))
}

// assetURL statik varlığa sürüm ekler (tarayıcı/edge cache kırıcı).
func (s *Server) assetURL(name string) string {
	return "/static/" + name + "?v=" + s.assetVer
}

// initialSiteName ayarlardan site başlığını okur (yoksa varsayılan).
func (s *Server) initialSiteName() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b, err := repo.GetSetting(ctx, s.pool, "genel", "site_baslik")
	if err != nil {
		return s.siteName
	}
	// {"tr":"..."} veya "str" desteklenir
	var obj map[string]string
	if json.Unmarshal(b, &obj) == nil {
		for _, v := range obj {
			if v != "" {
				return v
			}
		}
	}
	var plain string
	if json.Unmarshal(b, &plain) == nil && plain != "" {
		return plain
	}
	return s.siteName
}

// uyeRoute üye/abonelik rotalarını kaydeder (yalnız öneksiz).
//
// Dil önekli adresler (ör. /tr/giris) için mux'tan ÖNCE çalışan
// localeAliasMiddleware öneki soyar; böylece ServeMux'ta `{lang}` joker'i
// açmak gerekmez (`/{lang}/x` ile `/ping/{date}` gibi kalıplar Go mux'ta
// çakışıyor ve sunucu panic ile açılmıyordu).
func (s *Server) uyeRoute(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
}

// uyePrefixes dil öneki soyulacak üye/abonelik yol kökleri.
var uyePrefixes = []string{
	"giris", "kayit", "sifremi-unuttum", "sifre-sifirla", "e-posta-onay",
	"cikis", "abonelik", "hesabim", "uyelik",
}

// localeAliasPath "/<dil>/<üye-yolu>" biçimindeki yolu öneksiz hâle getirir.
// Eşleşme yoksa ok=false döner. saf fonksiyon — birim testi kolay olsun diye
// middleware'den ayrıdır.
func localeAliasPath(path string, supported func(string) bool) (yeni, lang string, ok bool) {
	if len(path) < 4 || path[0] != '/' {
		return path, "", false
	}
	seg, tail, found := strings.Cut(path[1:], "/")
	if !found || tail == "" || !supported(seg) {
		return path, "", false
	}
	first, _, _ := strings.Cut(tail, "/")
	for _, pre := range uyePrefixes {
		if first == pre {
			return "/" + tail, seg, true
		}
	}
	return path, "", false
}

// localeAliasMiddleware "/<dil>/<üye-yolu>" isteğini "/<üye-yolu>" olarak
// yeniden yazar. Metot, gövde ve çerezler korunur; yalnız üye/abonelik
// yolları etkilenir (diğer sayfalar handlePublicSegment ile çalışmaya
// devam eder).
func (s *Server) localeAliasMiddleware(next http.Handler) http.Handler {
	supported := func(code string) bool {
		return s.i18n != nil && s.i18n.IsSupported(code)
	}
	return localeAliasHandler(next, supported)
}

// localeAliasHandler, test edilebilir çekirdek: desteklenen dil kontrolü
// dışarıdan verilir.
func localeAliasHandler(next http.Handler, supported func(string) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if yeni, lang, ok := localeAliasPath(r.URL.Path, supported); ok {
			r2 := r.Clone(r.Context())
			r2.URL.Path = yeni
			r2.URL.RawPath = ""
			r2.SetPathValue("lang", lang)
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) routes() {
	// Genel site: tek catch-all ile yol segmentlerine göre dağıtım yapılır
	// (ServeMux'ta `/{lang}/...` jokerleri literal kalıplarla çakışabildiği için).
	//
	// NOT: Sitenin kanonik adresleri dil önekli olabilir (ör. /tr/...). Bu
	// yüzden aşağıdaki üye/abonelik rotaları hem öneksiz hem `/{lang}/…`
	// biçiminde kaydedilir; aksi halde şablonlardaki `{{.LangBase}}/giris`
	// gibi form hedefleri 404/405 dönerdi (gerçek kullanıcı akışı kırılırdı).
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)
	s.mux.HandleFunc("POST /csp-rapor", s.handleCSPReport)
	s.mux.HandleFunc("GET /{$}", s.handleCDHome)
	// Eski (theblock) düzeni karşılaştırma için korunur.
	s.mux.HandleFunc("GET /klasik", s.handleHome)
	s.mux.HandleFunc("GET /cd", s.handleCDHome)
	// CoinDesk fiyat sayfası (§14).
	s.mux.HandleFunc("GET /price", s.handleCDPrice)
	s.mux.HandleFunc("GET /price/{coin}", s.handleCDCoin)
	// Hisse detayı — CoinDesk'te AYRI şablon (§10.6). İki segment olduğu
	// için /price/{coin} ile çakışmaz.
	s.mux.HandleFunc("GET /price/stock/{sembol}", s.handleCDHisse)
	// Hisse bölümünün kökü: 404 değil, tek kanonik dizine yönlendirir.
	s.mux.HandleFunc("GET /price/stock", s.handleCDHisseDizin)
	// Mega menünün bağlandığı ek sayfalar (hepsi gerçek; boşsa dürüst
	// boş durum basar). Ölçüm: bu dört yol menüde 404 veriyordu.
	s.mux.HandleFunc("GET /videos", s.handleCDVideos)
	s.mux.HandleFunc("GET /videos/shorts", s.handleCDShorts)
	s.mux.HandleFunc("GET /video/{id}", s.handleCDVideoWatch)
	// Canlı yayın sayfası (YouTube kanalına bağlanır, bkz. cd_video.go).
	s.mux.HandleFunc("GET /canli-yayin", s.handleCDCanli)
	s.mux.HandleFunc("GET /podcasts", s.handleCDPodcasts)
	s.mux.HandleFunc("GET /sponsored-content", s.handleCDSponsored)
	s.mux.HandleFunc("GET /newsletters", s.handleCDNewsletters)
	// Etiket sayfaları (§4 /tag/{tag}).
	s.mux.HandleFunc("GET /tag/{slug}", s.handleCDTag)
	// Kaynak görselleri KENDİ origin'imizden (üçüncü taraf çerez/BP sorunu).
	s.mux.HandleFunc("GET /kg/{hash}", s.handleKaynakGorsel)
	// Ölçekli kaynak görsel; genişlik kapalı listeden (bkz. cd_gorsel.go).
	s.mux.HandleFunc("GET /kg/{hash}/{genislik}", s.handleKaynakGorselOlcekli)
	// CoinDesk bölüm adresleri (§4): /markets, /business, /tech, /policy …
	// Tek segment; slug closure ile sabitlenir (mux PathValue literal
	// desende boş döner — bu tuzak canlı testte yakalandı).
	//
	// DİKKAT: /videos ve /podcasts burada YOK — onların kendi handler'ları
	// var (tür bazlı liste). İkisini birden kaydetmek mux'ta panic üretir
	// ("conflicts with pattern"), ölçüldü: 53 kez yeniden başlama döngüsü.
	for _, b := range []string{"markets", "business", "tech", "policy", "opinion",
		"news-analysis", "latest-crypto-news", "focus", "research",
		"press-release", "learn"} {
		s.mux.HandleFunc("GET /"+b, s.cdBolumHandler(b))
	}
	s.mux.HandleFunc("GET /robots.txt", s.handleRobots)
	// Takım logoları kendi altyapımızdan servis edilir (bkz. teamlogo_handlers.go).
	s.mux.HandleFunc("GET /takim-logo/{key}", s.handleTeamLogo)
	s.mux.HandleFunc("GET /manifest.webmanifest", s.handleManifest)
	s.mux.HandleFunc("GET /manifest.json", s.handleManifest)
	// favicon.ico: tarayıcılar ve botlar KÖKTEN ister; rota yoksa her sayfa
	// yüklemesinde 404 üretiyordu (ölçüldü, 6 Kasım 2026). ICO üretmek
	// yerine PNG'nin aynısı sunuluyor — tarayıcılar MIME'a bakıyor.
	s.mux.HandleFunc("GET /favicon.ico", s.handleFavicon)
	s.mux.HandleFunc("GET /ads.txt", s.handleAdsTxt)
	s.mux.HandleFunc("GET /sitemap.xml", s.handleSitemap)
	s.mux.HandleFunc("GET /sitemap-news.xml", s.handleSitemapNews)
	// CoinDesk'in BÖLÜNMÜŞ sitemap ailesi (7 Kasım 2026): indeks + sabit
	// sayfalar + haber dilimleri + fiyat sayfaları. Tek dosyalık
	// /sitemap.xml yukarıda korunuyor (BİK resmî ilanları orada).
	s.mux.HandleFunc("GET /sitemap-index.xml", s.handleSitemapIndex)
	// Go mux joker'i segment ORTASINDA kabul etmiyor (`articles-{n}.xml`
	// panikliyor); tek joker alınıp dosya adı handler'da çözülüyor.
	s.mux.HandleFunc("GET /sitemaps/{dosya}", s.handleSitemapsDosya)
	// Haber sitemap indeksi (CoinDesk: /arc/outboundfeeds/news-sitemap-index).
	s.mux.HandleFunc("GET /news-sitemap-index", s.handleNewsSitemapIndex)
	s.mux.HandleFunc("GET /rss.xml", s.handleFeed)
	s.mux.HandleFunc("GET /rss", s.handleFeed)
	s.mux.HandleFunc("GET /rss-listesi", s.handleRSSList)
	s.mux.HandleFunc("GET /gundem.xml", s.handleTrendRSS)
	s.mux.HandleFunc("GET /api/trend-suggest", s.handleTrendSuggest)
	s.mux.HandleFunc("GET /api/icerik/{id}/canli", s.handleContentLiveJSON)
	s.mux.HandleFunc("GET /canli/icerik/{id}", s.handleContentLiveSSE)
	s.mux.HandleFunc("GET /internal/collab/{id}", s.handleCollabGet)
	s.mux.HandleFunc("POST /internal/collab/{id}", s.handleCollabSave)
	s.mux.HandleFunc("GET /{rest}", s.handlePublicSegment)
	s.mux.HandleFunc("GET /{lang}/{rest...}", s.handlePublicSegment)
	s.mux.HandleFunc("POST /hits/", s.handleHit)
	s.mux.HandleFunc("GET /api/hava/{slug}", s.handleAPIHava)
	s.mux.HandleFunc("GET /api/trends", s.handleAPITrends)
	s.mux.HandleFunc("GET /api/digest", s.handleAPIDigest)
	s.mux.HandleFunc("GET /bulten", s.handleBultenGET)
	s.mux.HandleFunc("POST /bulten", s.handleBultenPOST)
	s.mux.HandleFunc("GET /bulten/onay/{token}", s.handleBultenOnay)
	s.mux.HandleFunc("GET /ping", s.handlePingIndex)
	s.mux.HandleFunc("GET /ping/{date}", s.handlePingDay)
	s.mux.HandleFunc("GET /podcast/{date}", s.handlePodcast)
	s.mux.HandleFunc("POST /api/bookmarks/{id}", s.handleBookmark)
	s.mux.HandleFunc("POST /api/follow/{type}/{slug}", s.handleFollow)
	s.mux.HandleFunc("POST /api/feedback/{id}", s.handleFeedbackPOST)
	s.mux.HandleFunc("POST /api/anket/{id}/oy", s.handlePollVote)
	s.mux.HandleFunc("GET /asistan", s.handleAsistanGET)
	s.mux.HandleFunc("POST /asistan", s.handleAsistanPOST)
	s.mux.HandleFunc("GET /api/ad-tikla/{id}", s.handleAdClick)
	s.mux.HandleFunc("GET /api/ad-gorselleme", s.handleAdView)
	s.mux.HandleFunc("POST /api/ad-gorselleme", s.handleAdView)
	s.mux.HandleFunc("GET /api/namaz/{slug}", s.handleAPINamaz)
	s.mux.HandleFunc("GET /api/eczane/{slug}", s.handleAPIEczane)
	s.mux.HandleFunc("GET /api/borsa", s.handleAPIBorsa)
	s.mux.HandleFunc("GET /api/piyasa/canli", s.handleFinansCanli)
	s.mux.HandleFunc("GET /api/fikstur", s.handleAPIFikstur)
	s.mux.HandleFunc("GET /api/astroloji", s.handleAPIAstroloji)
	s.mux.HandleFunc("GET /api/push/vapid", s.handlePushVAPID)
	s.mux.HandleFunc("POST /api/scroll/{id}", s.handleScroll)
	s.mux.HandleFunc("POST /api/push/abone", s.handlePushSubscribe)
	s.mux.HandleFunc("POST /api/push/iptal", s.handlePushUnsubscribe)
	s.mux.HandleFunc("GET /sw.js", s.handleSW)
	s.mux.HandleFunc("GET /indexnow/{file}", s.handleIndexNowKeyFile)
	s.mux.HandleFunc("GET /amp/{rest}", s.handleAMP)
	s.mux.HandleFunc("POST /yorum/", s.handleAddComment)
	// İletişim formu (Faz 145): /iletisim sayfasındaki formu alır.
	s.mux.HandleFunc("POST /iletisim", s.handleContactSubmit)
	s.mux.HandleFunc("POST /yorum/{id}/begen", s.handleCommentLike)
	s.mux.HandleFunc("POST /api/kart-tepki", s.handleCardReaction)
	s.mux.HandleFunc("GET /akis", s.handleAkisPage)
	s.mux.HandleFunc("GET /api/akis", s.handleAkisAPI)

	// ---- Gazete (e-gazete): arşiv, flip görüntüleyici, abonelik ----
	s.mux.HandleFunc("GET /gazete", s.handleGazeteArsiv)
	s.mux.HandleFunc("POST /gazete/abone", s.handleGazeteAboneOl)
	s.mux.HandleFunc("GET /gazete/abonelik/iptal/{token}", s.handleGazeteAbonelikIptal)
	// ---- Public üye girihi: giriş, kayıt, şifre sıfırlama ----
	s.uyeRoute("GET /giris", s.handleUyeGiris)
	s.uyeRoute("POST /giris", s.handleUyeGirisPost)
	s.uyeRoute("GET /kayit", s.handleUyeKayit)
	s.uyeRoute("POST /kayit", s.handleUyeKayitPost)
	s.uyeRoute("GET /sifremi-unuttum", s.handleUyeSifremiUnuttum)
	s.uyeRoute("POST /sifremi-unuttum", s.handleUyeSifremiUnuttumPost)
	s.uyeRoute("GET /sifre-sifirla", s.handleUyeSifreSifirla)
	s.uyeRoute("POST /sifre-sifirla", s.handleUyeSifreSifirlaPost)
	s.uyeRoute("POST /cikis", s.handleUyeCikis)
	s.uyeRoute("GET /e-posta-onay", s.handleUyeOnay)
	// ---- Abonelik katmanları (fiyat sayfası + iyzico) ----
	s.uyeRoute("GET /abonelik", s.handleAbonelikPage)
	s.uyeRoute("POST /abonelik/odeme", s.handleAbonelikOdeme)
	s.uyeRoute("POST /abonelik/geri-donus", s.handleAbonelikGeriDonus)
	s.uyeRoute("GET /abonelik/geri-donus", s.handleAbonelikGeriDonus)
	s.uyeRoute("GET /hesabim", s.handleHesap)
	s.uyeRoute("GET /hesabim/profil", s.handleHesapProfil)
	s.uyeRoute("POST /hesabim/profil", s.handleHesapProfilPost)
	s.uyeRoute("POST /hesabim/avatar", s.handleHesapAvatar)
	s.uyeRoute("POST /hesabim/sifre", s.handleHesapSifre)
	s.uyeRoute("POST /hesabim/e-posta-gonder", s.handleHesapEpostaGonder)
	s.uyeRoute("GET /hesabim/abonelik", s.handleHesapAbonelik)
	s.uyeRoute("POST /hesabim/abonelik/iptal", s.handleHesapAbonelikIptal)
	s.uyeRoute("GET /hesabim/yorumlar", s.handleHesapYorumlar)
	s.uyeRoute("GET /hesabim/takip", s.handleHesapTakip)
	s.uyeRoute("POST /hesabim/sil", s.handleHesapSil)
	s.uyeRoute("GET /hesabim/kaydedilenler", s.handleMyBookmarks)
	s.uyeRoute("GET /hesabim/bildirimler", s.handleMyNotifications)
	s.uyeRoute("GET /hesabim/rozetler", s.handleAchievements)
	// Stripe üyelik (paywall): checkout, dönüş sayfaları ve webhook.
	s.uyeRoute("GET /giris/google", s.handleGirisGoogle)
	s.uyeRoute("GET /giris/google/callback", s.handleGirisGoogleCallback)
	s.uyeRoute("POST /uyelik/odeme", s.handleUyelikOdeme)
	s.uyeRoute("GET /uyelik/basarili", s.handleUyelikBasarili)
	s.uyeRoute("GET /uyelik/iptal", s.handleUyelikIptal)
	s.mux.HandleFunc("POST /webhooks/stripe", s.handleStripeWebhook)
	s.mux.HandleFunc("GET /gazete/{tarih}", s.handleGazeteOkuyucu)
	s.mux.HandleFunc("GET /gazete/{tarih}/pdf", s.handleGazetePDFIndir)

	// ---- Yönetim paneli ----
	s.mux.HandleFunc("GET /admin/giris", s.handleLoginGet)
	s.mux.HandleFunc("POST /admin/giris", s.handleLoginPost)
	s.mux.HandleFunc("GET /admin/giris/2fa", s.handle2FALoginGet)
	s.mux.HandleFunc("POST /admin/giris/2fa", s.handle2FALoginPost)
	// Çıkış: GET (adres çubuğu / yer imi) ve POST (form) birlikte desteklenir.
	// GET, requireAuth ile SARILMAZ: oturumu zaten bitmiş biri adresi
	// yenilediğinde "giriş yap → yine çıkış" döngüsü oluşurdu.
	s.mux.HandleFunc("GET /admin/cikis", s.handleLogout)
	s.mux.Handle("POST /admin/cikis", s.requireAuth(http.HandlerFunc(s.handleLogout)))

	// Admin rotaları: literal önek + açık metot (POST'lar için de çalışır).
	type route struct {
		method, path string
		h            http.HandlerFunc
	}
	adminRoutes := []route{
		// Panelin giriş sayfası Pano 2.0 (Pacely düzeni). Önceden burada
		// eski pano vardı ve yeni tasarım yalnız /admin/pano adresinden
		// görülebiliyordu — yani yapılmış ama kullanıcıya ulaşmamıştı.
		{"GET", "/admin", s.handlePano},
		{"GET", "/admin/icerikler", s.handleContentsList},
		{"GET", "/admin/icerikler/yeni", s.handleContentFormNew},
		{"POST", "/admin/icerikler", s.handleContentSave},
		{"GET", "/admin/icerikler/{id}", s.handleContentFormEdit},
		{"POST", "/admin/icerikler/{id}", s.handleContentSave},
		{"POST", "/admin/icerikler/{id}/seslendir", s.handleContentTTS},
		{"POST", "/admin/icerikler/{id}/ses", s.handleAudioUpload},
		{"POST", "/admin/icerikler/{id}/kalite", s.handleContentQuality},
		{"GET", "/admin/icerikler/{id}/ses/durum", s.handleAudioStatus},
		{"POST", "/admin/icerikler/{id}/ses/{media}/yeniden", s.handleAudioRetry},
		{"POST", "/admin/icerikler/{id}/ses/{media}/sil", s.handleAudioDelete},
		{"POST", "/admin/icerikler/{id}/ses/{media}/taslak", s.handleAudioDraft},
		{"GET", "/admin/icerikler/{id}/ses/{media}/altyazi/{bicim}", s.handleAudioSubtitle},
		{"GET", "/admin/ses/{id}/stream", s.handlePrivateAudioStream},
		{"GET", "/admin/ses/ara", s.handleTranscriptSearch},
		{"GET", "/admin/ses-ayarlari", s.handleASRSettingsGet},
		{"POST", "/admin/ses-ayarlari", s.handleASRSettingsPost},
		{"POST", "/admin/icerikler/{id}/yayinla", s.handleQuickStatus},
		{"POST", "/admin/icerikler/{id}/taslak", s.handleQuickStatus},
		{"POST", "/admin/icerikler/{id}/arsiv", s.handleQuickStatus},
		{"POST", "/admin/icerikler/{id}/sil", s.handleContentDelete},
		{"GET", "/admin/icerikler/{id}/onizleme", s.handleContentPreview},
		{"POST", "/admin/icerikler/toplu", s.handleContentsBulk},
		{"POST", "/admin/icerikler/gorunum-kaydet", s.handleListViewSave},
		{"POST", "/admin/icerikler/gorunum-sil", s.handleListViewDel},
		{"POST", "/admin/icerikler/{id}/otomatik-kaydet", s.handleContentAutosave},
		{"POST", "/admin/icerikler/{id}/rozet/{code}", s.handleToggleRozet},
		{"POST", "/admin/icerikler/{id}/cevir/{lang}", s.handleCreateTranslation},
		// Faz 190: AI çeviri kuyruğa yazılır; panel bu adresi yoklar.
		{"GET", "/admin/icerikler/{id}/cevir/{lang}/durum", s.handleTranslationStatus},
		{"GET", "/admin/icerikler/ic-link-ara", s.handleInternalLinkSearch},
		{"GET", "/admin/ara", s.handleAdminAra},
		{"POST", "/admin/tercih", s.handleTercihKaydet},
		{"GET", "/admin/icerikler/bookmark", s.handleBookmarkPreview},
		{"GET", "/admin/icerikler/{id}/collab-token", s.handleCollabToken},
		{"GET", "/admin/icerikler/{id}/duzenleme", s.handleEditorLiveSSE},
		{"GET", "/admin/icerikler/{id}/yorumlar", s.handleEditorNotesList},
		{"POST", "/admin/icerikler/{id}/yorumlar", s.handleEditorNoteCreate},
		{"POST", "/admin/icerikler/{id}/yorumlar/{note}/cozum", s.handleEditorNoteResolve},
		{"POST", "/admin/icerikler/{id}/yorumlar/{note}/sil", s.handleEditorNoteDelete},
		{"POST", "/admin/icerikler/ai", s.handleAIAction},
		{"POST", "/admin/icerikler/{id}/ai/eylem", s.handleAIEditorAction},
		{"GET", "/admin/icerikler/{id}/ai/gecmis", s.handleAIEditorHistory},
		{"GET", "/admin/icerikler/{id}/ai/surum/{rid}/fark", s.handleContentRevisionDiff},
		{"GET", "/admin/icerikler/{id}/ai/is/{job}", s.handleAIEditorJob},
		{"GET", "/admin/icerikler/{id}/ai/is/{job}/durum", s.handleAIEditorStatus},
		{"POST", "/admin/icerikler/{id}/ai/is/{job}/iptal", s.handleAIEditorCancel},
		{"POST", "/admin/icerikler/{id}/ai/yeni-taslak", s.handleAIEditorNewDraft},
		{"POST", "/admin/icerikler/{id}/ai/seo-doldur", s.handleSeoDoldur},
		{"POST", "/admin/icerikler/{id}/ai/geri-al/{rid}", s.handleAIEditorRestore},
		{"POST", "/admin/icerikler/{id}/ai-skor", s.handleAIScore},
		{"POST", "/admin/icerikler/{id}/kaynak-ekle", s.handleSourceEkle},
		{"POST", "/admin/icerikler/{id}/kaynak-kontrol", s.handleSourceKontrol},
		{"POST", "/admin/icerikler/kaynaklar/sil/{sid}", s.handleSourceSil},
		{"GET", "/admin/kategoriler", s.handleCatsGet},
		{"POST", "/admin/kategoriler", s.handleCatsCreate},
		{"POST", "/admin/kategoriler/{id}", s.handleCatsUpdate},
		{"POST", "/admin/kategoriler/{id}/sil", s.handleCatsDelete},
		// Site Yönetim Merkezi kaldırıldı (kullanılmıyordu): iframe kabuğu
		// yerine doğrudan panoya yönlendiriyoruz ki eski yer imleri ve
		// tarayıcı geçmişi 404 görmesin.
		{"GET", "/admin/merkez", s.handleMerkezKaldirildi},
		{"GET", "/admin/pano", s.handlePano},
		{"POST", "/admin/pano/tasi", s.handlePanoTasi},
		{"GET", "/admin/ikonlar", s.handleIkonlar},
		{"GET", "/admin/menu", s.handleMenuGet},
		{"POST", "/admin/menu/kaydet", s.handleMenuSave},
		{"POST", "/admin/menu/tasarim", s.handleMenuDesignSave},
		{"POST", "/admin/menu/tasarim/arka-plan", s.handleMenuBackgroundSave},
		{"POST", "/admin/menu/sifirla", s.handleMenuReset},
		{"GET", "/admin/tab-kategorileri", s.handleTabCatsGet},
		{"POST", "/admin/tab-kategorileri/kaydet", s.handleTabCatsSave},
		{"POST", "/admin/tab-kategorileri/sifirla", s.handleTabCatsReset},
		{"GET", "/admin/header", s.handleHeaderGet},
		{"POST", "/admin/header/kaydet", s.handleHeaderSave},
		{"GET", "/admin/footer", s.handleFooterGet},
		{"POST", "/admin/footer/kaydet", s.handleFooterSave},
		{"GET", "/admin/marka", s.handleBrandGet},
		{"POST", "/admin/marka/kaydet", s.handleBrandSave},
		// YouTube video & canlı yayın yönetimi (kanal, aralık, öne çıkan,
		// canlı işareti). Sayfalar: /videos · /videos/shorts · /canli-yayin.
		{"GET", "/admin/video-yayin", s.handleVideoYayinGet},
		{"POST", "/admin/video-yayin/ayar", s.handleVideoYayinAyar},
		{"POST", "/admin/video-yayin/cek", s.handleVideoYayinCek},
		{"POST", "/admin/video-yayin/{id}/{islem}", s.handleVideoYayinIslem},
		{"GET", "/admin/geri-bildirim", s.handleFeedbackQueue},
		{"POST", "/admin/geri-bildirim/{id}", s.handleFeedbackResolve},
		{"GET", "/admin/mahrecler", s.handleMahretsGet},
		{"POST", "/admin/mahrecler", s.handleMahretsCreate},
		{"POST", "/admin/mahrecler/{id}/sil", s.handleMahretsDelete},
		{"GET", "/admin/etiketler", s.handleTagsGet},
		{"POST", "/admin/etiketler/{id}/adlandir", s.handleTagRename},
		{"POST", "/admin/etiketler/birlestir", s.handleTagMerge},
		{"POST", "/admin/etiketler/{id}/sil", s.handleTagDelete},
		{"GET", "/admin/ceviriler", s.handleCevirilerGet},
		{"POST", "/admin/ceviriler/kaydet", s.handleCevirilerSave},
		{"POST", "/admin/ceviriler/anahtar-ekle", s.handleCevirilerAddKey},
		{"POST", "/admin/ceviriler/anahtar-sil", s.handleCevirilerDelKey},
		{"GET", "/admin/reklamlar", s.handleAdsGet},
		{"POST", "/admin/reklamlar/yeni", s.handleAdCreate},
		{"POST", "/admin/reklamlar/{id}", s.handleAdUpdate},
		{"POST", "/admin/reklamlar/{id}/sil", s.handleAdDelete},
		{"POST", "/admin/reklam-yerleri/yeni", s.handleAdSlotCreate},
		{"POST", "/admin/reklam-yerleri/{id}", s.handleAdSlotUpdate},
		{"POST", "/admin/reklam-yerleri/{id}/sil", s.handleAdSlotDelete},
		{"GET", "/admin/tasarim", s.handleBuilderGet},
		{"POST", "/admin/tasarim/kaydet", s.handleBuilderSave},
		{"POST", "/admin/tasarim/onizleme", s.handleBuilderPreview},
		{"POST", "/admin/tasarim/sifirla", s.handleBuilderReset},
		{"POST", "/admin/tasarim/kopyala", s.handleBuilderCopy},
		{"GET", "/admin/tasarim/gecmis", s.handleBuilderHistory},
		{"POST", "/admin/tasarim/surum/{id}/geri", s.handleBuilderRestore},
		{"GET", "/admin/tasarim/ab", s.handleBuilderABGet},
		{"POST", "/admin/tasarim/ab/kaydet", s.handleBuilderABSave},
		{"GET", "/admin/tasarim/kategori-gorunum", s.handleBuilderCategoryViewGet},
		{"POST", "/admin/tasarim/kategori-gorunum", s.handleBuilderCategoryViewSave},
		{"POST", "/admin/tasarim/sayfa-metinleri", s.handleBuilderPageTextSave},
		{"GET", "/admin/iletisim", s.handleContactAdminGet},
		{"POST", "/admin/iletisim/kaydet", s.handleContactAdminSave},
		{"POST", "/admin/iletisim/degistir", s.handleContactReplace},
		{"POST", "/admin/iletisim/mesaj/{id}", s.handleContactMessageStatus},
		{"POST", "/admin/tasarim/yerlesim", s.handleBuilderPageLayoutSave},
		{"GET", "/admin/tasarim/yerlesim", s.handleBuilderPageLayoutGet},
		{"POST", "/admin/arka-plan/kaydet", s.handleBackgroundSave},
		{"POST", "/admin/olcu/kaydet", s.handleLayoutSave},
		{"POST", "/admin/olcu/parca", s.handleLayoutPartial},
		{"POST", "/admin/kategori-panelleri/kaydet", s.handlePanelsSave},
		{"GET", "/admin/tasima", s.handleTasimaGet},
		{"POST", "/admin/tasima/yonlendirme", s.handleRedirectSave},
		{"POST", "/admin/tasima/yonlendirme-sil", s.handleRedirectDelete},
		{"POST", "/admin/tasima/yonlendirme-ice-aktar", s.handleRedirectImport},
		{"POST", "/admin/tasima/icerik-ice-aktar", s.handleContentImport},
		{"GET", "/admin/tasima/ornek-csv", s.handleTasimaSampleCSV},
		{"GET", "/admin/seo", s.handleSEOPage},
		{"POST", "/admin/seo/guncelle", s.handleSEOTouch},
		{"GET", "/admin/etkinlikler", s.handleEventsGet},
		{"POST", "/admin/etkinlikler/yeni", s.handleEventCreate},
		{"POST", "/admin/etkinlikler/{id}", s.handleEventUpdate},
		{"POST", "/admin/etkinlikler/{id}/sil", s.handleEventDelete},
		{"POST", "/admin/etkinlikler/{id}/haber", s.handleEventDraft},
		{"GET", "/admin/anketler", s.handlePollsAdmin},
		{"POST", "/admin/anketler/yeni", s.handlePollCreate},
		{"POST", "/admin/anketler/{id}", s.handlePollSave},
		{"POST", "/admin/anketler/{id}/sil", s.handlePollDelete},
		{"GET", "/admin/bulten", s.handleBultenAdmin},
		{"POST", "/admin/bulten/ayar", s.handleBultenSave},
		{"POST", "/admin/bulten/test", s.handleBultenTest},
		{"POST", "/admin/bulten/gonder", s.handleBultenSend},
		{"GET", "/admin/gazeteler", s.handleNewspapersAdmin},
		{"POST", "/admin/gazeteler/yeni", s.handleNewspaperSave},
		{"POST", "/admin/gazeteler/{id}/sil", s.handleNewspaperDelete},
		{"POST", "/admin/gazeteler/cek", s.handleNewspaperFetch},
		// ---- Gazete modülü (e-gazete) ----
		{"GET", "/admin/gazete", s.handleGazeteAdmin},
		{"POST", "/admin/gazete/lisans", s.handleGazeteLisans},
		{"POST", "/admin/gazete/ayar", s.handleGazeteAyar},
		{"POST", "/admin/gazete/depolama", s.handleGazeteDepolama},
		{"POST", "/admin/gazete/depolama/test", s.handleGazeteDepolamaTest},
		{"POST", "/admin/gazete/ekip", s.handleGazeteEkip},
		{"POST", "/admin/gazete/ekip/{user_id}/sil", s.handleGazeteEkipSil},
		{"POST", "/admin/gazete/sablonlar", s.handleGazeteSablonKaydet},
		{"POST", "/admin/gazete/sablonlar/tohum", s.handleGazeteSablonTohum},
		{"POST", "/admin/gazete/sablonlar/{id}/sil", s.handleGazeteSablonSil},
		{"GET", "/admin/gazete/sablonlar", s.handleGazeteSablonlar},
		{"GET", "/admin/gazete/sablonlar/{id}", s.handleGazeteSablonJSON},
		{"POST", "/admin/gazete/sablonlar/{id}/varsayilan", s.handleGazeteSablonVarsayilan},
		{"POST", "/admin/gazete/sayilar/{id}/sablona-kaydet", s.handleGazeteSayiSablonaKaydet},
		{"POST", "/admin/gazete/sayilar", s.handleGazeteSayiYeni},
		{"GET", "/admin/gazete/sayilar/{id}/duzenle", s.handleGazeteDuzen},
		{"POST", "/admin/gazete/sayilar/{id}/durum", s.handleGazeteSayiDurum},
		{"POST", "/admin/gazete/sayilar/{id}/atama", s.handleGazeteSayiAtama},
		{"POST", "/admin/gazete/sayilar/{id}/sil", s.handleGazeteSayiSil},
		{"POST", "/admin/gazete/sayilar/{id}/doldur", s.handleGazeteSayiDoldur},
		{"POST", "/admin/gazete/sayilar/{id}/sayfa", s.handleGazeteSayfaKaydet},
		{"POST", "/admin/gazete/sayilar/{id}/blok-html", s.handleGazeteBlokHTML},
		{"POST", "/admin/gazete/sayilar/{id}/dogrula", s.handleGazeteDogrula},
		{"POST", "/admin/gazete/sayilar/{id}/sayfalar", s.handleGazeteSayfalarKaydet},
		{"POST", "/admin/gazete/sayilar/{id}/matbaa", s.handleGazeteMatbaa},
		{"POST", "/admin/gazete/sayilar/{id}/pdf", s.handleGazetePDF},
		{"POST", "/admin/gazete/sayilar/{id}/gonder", s.handleGazeteGonder},
		{"POST", "/admin/gazete/aboneler", s.handleGazeteAboneEkle},
		{"POST", "/admin/gazete/aboneler/{id}/sil", s.handleGazeteAboneSil},
		{"POST", "/admin/gazete/aboneler/{id}/durum", s.handleGazeteAboneDurum},
		// ---- Aboneler merkezi (bülten + gazete birleşik) ----
		{"GET", "/admin/aboneler", s.handleAbonelerGet},
		{"POST", "/admin/aboneler/bulten", s.handleAbonelerBultenEkle},
		{"POST", "/admin/aboneler/bulten/{id}/durum", s.handleAbonelerBultenDurum},
		{"POST", "/admin/aboneler/bulten/{id}/sil", s.handleAbonelerBultenSil},
		{"POST", "/admin/aboneler/gazete", s.handleAbonelerGazeteEkle},
		{"POST", "/admin/aboneler/gazete/{id}/durum", s.handleAbonelerGazeteDurum},
		{"POST", "/admin/aboneler/gazete/{id}/sil", s.handleAbonelerGazeteSil},
		{"GET", "/admin/sinemalar", s.handleFilmsAdmin},
		{"POST", "/admin/sinemalar/yeni", s.handleFilmSave},
		{"POST", "/admin/sinemalar/{id}/sil", s.handleFilmDelete},
		{"GET", "/admin/resmi-ilanlar", s.handleNoticesAdmin},
		{"POST", "/admin/resmi-ilanlar/yeni", s.handleNoticeSave},
		{"POST", "/admin/resmi-ilanlar/{id}/sil", s.handleNoticeDelete},
		{"GET", "/admin/adsense", s.handleAdsenseGet},
		{"POST", "/admin/adsense/ayar", s.handleAdsenseSave},
		{"GET", "/admin/adsense/baglan", s.handleAdsenseConnect},
		{"GET", "/admin/adsense/callback", s.handleAdsenseCallback},
		{"POST", "/admin/adsense/senkron", s.handleAdsenseSync},
		{"POST", "/admin/adsense/kes", s.handleAdsenseDisconnect},
		{"GET", "/admin/backlink", s.handleBacklinkAdmin},
		{"GET", "/admin/paylasim", s.handleShareAdmin},
		{"POST", "/admin/paylasim/ayar", s.handleShareSave},
		{"POST", "/admin/paylasim/gonder", s.handleShareNow},
		{"POST", "/admin/paylasim/test", s.handleShareTest},
		{"GET", "/admin/indexleme", s.handleIndexingAdmin},
		{"POST", "/admin/indexleme/ayar", s.handleIndexingSave},
		{"POST", "/admin/indexleme/indexnow", s.handleIndexNowSubmit},
		{"POST", "/admin/indexleme/google", s.handleGoogleIndexingSubmit},
		{"POST", "/admin/indexleme/news-sitemap", s.handleNewsSitemapCheck},
		{"POST", "/admin/indexleme/blok-kaldir", s.handleBlockUnblock},
		{"GET", "/admin/bildirim", s.handlePushAdmin},
		{"POST", "/admin/bildirim/gonder", s.handlePushSend},
		{"POST", "/admin/bildirim/sub-sil", s.handlePushSubDelete},
		{"GET", "/admin/dosyalar", s.handleMediaGet},
		{"GET", "/admin/dosyalar/disa-aktar.csv", s.handleMediaExport},
		{"GET", "/admin/medya/unsplash", s.handleUnsplashSearch},
		{"POST", "/admin/dosyalar", s.handleMediaUpload},
		{"POST", "/admin/dosyalar/url", s.handleMediaURLUpload},
		{"POST", "/admin/dosyalar/{id}/meta", s.handleMediaMeta},
		{"POST", "/admin/dosyalar/{id}/duzenle", s.handleMediaTransform},
		{"POST", "/admin/dosyalar/{id}/sil", s.handleMediaDelete},
		{"POST", "/admin/dosyalar/toplu", s.handleMediaBulk},
		{"GET", "/admin/raporlar", s.handleReports},
		{"GET", "/admin/raporlar/ai-export", s.handleReportsAIExport},
		{"GET", "/admin/yorumlar", s.handleCommentsGet},
		{"POST", "/admin/yorumlar/{id}/onayla", s.handleCommentApprove},
		{"POST", "/admin/yorumlar/{id}/sil", s.handleCommentDelete},
		{"POST", "/admin/yorumlar/toplu", s.handleCommentsBulk},
		{"GET", "/admin/ai-sohbet", s.handleAIChatGet},
		{"POST", "/admin/ai-sohbet/yeni", s.handleAIChatNew},
		{"POST", "/admin/ai-sohbet/{id}/mesaj", s.handleAIChatMsg},
		{"POST", "/admin/ai-sohbet/{id}/sil", s.handleAIChatDelete},
		{"GET", "/admin/ayarlar", s.handleSettingsGet},
		{"POST", "/admin/ayarlar", s.handleSettingsSave},
		{"POST", "/admin/ayarlar/loglar-temizle", s.handleSettingsLogsClear},
		{"GET", "/admin/gorunum", s.handleGorunumAdmin},
		{"POST", "/admin/gorunum", s.handleGorunumSave},
		{"GET", "/admin/bilesenler", s.handleBilesenler},
		{"GET", "/admin/yardim", s.handleYardim},
		{"GET", "/admin/gelen-kutusu", s.handleGelenKutusu},
		{"GET", "/admin/manset-editor", s.handleMansetGet},
		{"GET", "/admin/manset-editor/template/{id}", s.handleMansetGetTemplate},
		{"POST", "/admin/manset-editor/kaydet", s.handleMansetSave},
		{"POST", "/admin/manset-editor/sil", s.handleMansetDelete},
		{"POST", "/admin/manset-editor/export", s.handleMansetExport},
		{"GET", "/admin/manset-editor/media/{id}", s.handleMansetMediaURL},
		// Faz 204 — toplu AI ile yeniden yazım. Öneri üretir, yayına YAZMAZ;
		// yazma yalnız onayla ve sürüm korumalı.
		{"GET", "/admin/yeniden-yazim", s.handleYenidenYazim},
		{"POST", "/admin/yeniden-yazim/pilot", s.handleYenidenYazimPilot},
		{"GET", "/admin/yeniden-yazim/{id}", s.handleYenidenYazimDetay},
		{"POST", "/admin/yeniden-yazim/{id}/onayla", s.handleYenidenYazimOnayla},
		{"POST", "/admin/yeniden-yazim/{id}/reddet", s.handleYenidenYazimReddet},
		{"POST", "/admin/yeniden-yazim/{id}/tekrar", s.handleYenidenYazimTekrar},
		{"GET", "/admin/bot", s.handleBotGet},
		{"POST", "/admin/bot/ekle", s.handleBotEkle},
		{"POST", "/admin/bot/{id}/sil", s.handleBotSil},
		{"POST", "/admin/bot/{id}/cek", s.handleBotCek},
		{"GET", "/admin/ai-kaynak", s.handleAISourceGet},
		{"POST", "/admin/ai-kaynak/tara", s.handleAISourceScan},
		{"POST", "/admin/ai-kaynak/zenginlestir", s.handleAISourceEnrich},
		{"POST", "/admin/ai-kaynak/plan/{id}", s.handleAISourcePlanSave},
		{"POST", "/admin/ai-kaynak/plan/{id}/calistir", s.handleAISourcePlanRun},
		{"POST", "/admin/ai-kaynak/kume/{id}/taslak", s.handleAISourceDraft},
		{"POST", "/admin/bot/hepsi-cek", s.handleBotCek},
		{"POST", "/admin/bot/kelime", s.handleBotKelime},
		{"POST", "/admin/trend/yenile", s.handleTrendYenile},
		{"POST", "/admin/trend/kaynak", s.handleTrendKaynak},
		{"POST", "/admin/trend/{id}/taslak", s.handleTrendTaslak},
		{"POST", "/admin/trend/cek", s.handleTrendCenterCek},
		{"POST", "/admin/trend/toplu-taslak", s.handleTrendBulkDraft},
		{"GET", "/admin/trend/rapor.csv", s.handleTrendReportCSV},
		{"POST", "/admin/trend/kaynak-ekle", s.handleTrendSourceEkle},
		{"POST", "/admin/trend/kaynak/{id}/cek", s.handleTrendSourceCek},
		{"POST", "/admin/trend/kaynak/{id}/durum", s.handleTrendSourceDurum},
		{"POST", "/admin/trend/kaynak/{id}/sil", s.handleTrendSourceSil},
		{"POST", "/admin/trend/terim/{id}/taslak", s.handleTrendTermTaslak},
		{"POST", "/admin/trend/terim/{id}/brifing", s.handleTrendTermBrifing},
		{"POST", "/admin/trend/terim/{id}/gizle", s.handleTrendTermGizle},
		{"POST", "/admin/trend/uyarilar/okundu", s.handleTrendAlertsSeen},
		{"POST", "/admin/trend/ayar", s.handleTrendSettings},
		{"POST", "/admin/trend/gsc/ayar", s.handleTrendGSCSetting},
		{"POST", "/admin/trend/gsc/senkron", s.handleTrendGSCSync},
		{"POST", "/admin/trend/ga4/ayar", s.handleTrendGA4Setting},
		{"POST", "/admin/trend/ga4/senkron", s.handleTrendGA4Sync},
		{"GET", "/admin/siteler", s.handleSitesGet},
		{"POST", "/admin/siteler/ekle", s.handleSitesEkle},
		{"POST", "/admin/siteler/{id}", s.handleSitesGuncelle},
		{"POST", "/admin/siteler/{id}/sil", s.handleSitesSil},
		{"GET", "/admin/site-sec", s.handleSiteSec},
		{"GET", "/admin/servisler", s.handleServicesGet},
		{"POST", "/admin/servisler/yenile", s.handleServicesRefresh},
		{"POST", "/admin/servisler/ayar", s.handleServicesSettingsSave},
		{"GET", "/admin/kumeler", s.handleKumelerGet},
		{"GET", "/admin/2fa", s.handle2FAGet},
		{"POST", "/admin/2fa/kur", s.handle2FAEnable},
		{"POST", "/admin/2fa/kapat", s.handle2FADisable},
		{"GET", "/admin/yazarlar", s.handlePeopleAdmin},
		{"GET", "/admin/yazarlar/{id}", s.handlePeopleAdmin},
		{"POST", "/admin/yazarlar/{id}", s.handlePersonAdminSave},
		{"GET", "/admin/members", s.handleMembersGet},
		{"POST", "/admin/members/{id}/{plan}", s.handleMembersSet},
		{"POST", "/admin/members/stripe", s.handleMembersStripe},
		{"POST", "/admin/members/uye", s.handleMembersUye},
		{"POST", "/admin/members/iyzico", s.handleMembersIyzico},
		{"POST", "/admin/members/kademe", s.handleMembersKademe},
		{"POST", "/admin/members/kademe/{id}/sil", s.handleMembersKademeSil},
		{"POST", "/admin/members/abonelik", s.handleMembersAbonelikAta},
		{"POST", "/admin/members/odeme/{id}/iade", s.handleMembersOdemeIade},
		{"POST", "/admin/members/kupon", s.handleMembersKupon},
		{"POST", "/admin/members/kupon/{id}/sil", s.handleMembersKuponSil},
		{"POST", "/admin/members/{id}/sil", s.handleMembersSil},
		{"POST", "/admin/members/google", s.handleMembersGoogle},
		{"POST", "/admin/kumeler/ekle", s.handleKumeEkle},
		{"POST", "/admin/kumeler/{id}/sil", s.handleKumeSil},
		{"POST", "/admin/kumeler/{id}/icerik/{cid}", s.handleKumeIcerikEkle},
	}
	for _, rt := range adminRoutes {
		s.mux.Handle(rt.method+" "+rt.path, s.requireAdmin(rt.path, rt.h))
	}
}

// requireAdmin, panel rotalarında hem geçerli oturum hem de personel rolü
// ister. Eskiden yalnız requireAuth kullanılıyordu: halka açık /kayit ile
// açılan "Üye" hesabı da tb_session aldığı için 284 /admin rotasının
// tamamına girebiliyordu (yetki yükseltme). Artık Üye rolü hiçbir panel
// rotasına giremez; ayrıca rota bazlı asgari rol (bkz. adminMinRole)
// uygulanır.
func (s *Server) requireAdmin(path string, next http.Handler) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		// Oturum kapsamı: okuyucu kapısından açılmış çerez, rol sonradan
		// yükseltilse bile panele giremez (yeniden kimlik doğrulama gerekir).
		if u.Scope != "" && u.Scope != auth.ScopePanel {
			s.forbidden(w, r)
			return
		}
		if !adminAllowed(u.Role, path) {
			s.forbidden(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// requireAuth oturum kontrolü; yoksa girişe yönlendirir.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			http.Redirect(w, r, "/admin/giris?sonra="+r.URL.Path, http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUser, &u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---- Genel site akışı ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","app":"theblock","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

// permissionsPolicy, sayfanın kullanmadığı tarayıcı yeteneklerini kapatır.
// Kullanılanlar açık bırakılır:
//   - microphone: dikte özelliği Web Speech API kullanıyor
//     (static/editor.js, static/app.js)
//   - autoplay / picture-in-picture / fullscreen / encrypted-media: kendi video
//     oynatıcımız (static/cd-player.js) ve YouTube çerçevesi
//
// Kullanılmayanlar kapatılır: konum, kamera, ödeme API'si, USB, hareket
// sensörleri. Kodda geolocation ve getUserMedia hiç geçmiyor.
//
// ---- YouTube çerçevesi (7 Kasım 2026 — GERÇEK HATA) ----
// Önceki değer `autoplay=(self)` idi: yalnız BİZİM origin'imiz otomatik
// oynatabiliyordu, çapraz kaynak çerçeve (youtube-nocookie.com) izin
// ALMIYORDU. Sonuç sahada şöyle görünüyordu: kullanıcı oynat düğmesine
// basıyor, YouTube çerçevesi otomatik oynatamıyor, kendi "play" poster'ında
// kalıyor; video ancak çerçevenin İÇİNE ikinci kez tıklanınca başlıyordu
// ("birden fazla tıklamadan oynamıyor, ortada play simgesi kalıyor").
// Aynı kök neden iki belirti daha üretiyordu: `fullscreen=(self)` yüzünden
// çerçeve tam ekran isteyemiyor, `picture-in-picture=(self)` yüzünden
// konsola sürekli "Permissions policy violation" düşüyordu.
//
// ÇÖZÜM: izin listesine YouTube origin'leri EKLENDİ (self korunuyor).
// `encrypted-media` da açıldı: YouTube bazı içerikleri EME ile sunuyor ve
// kapalıyken oynatma sessizce düşebiliyor.
//
// GÜVENLİK NOTU: bu izinler yalnız bu origin'lere verilir; rastgele bir
// üçüncü taraf çerçevesi otomatik oynatma/tam ekran yetkisi almaz.
const permissionsPolicy = "accelerometer=(), " +
	"autoplay=(self \"https://www.youtube.com\" \"https://www.youtube-nocookie.com\"), " +
	"camera=(), display-capture=(), " +
	"encrypted-media=(self \"https://www.youtube.com\" \"https://www.youtube-nocookie.com\"), " +
	"fullscreen=(self \"https://www.youtube.com\" \"https://www.youtube-nocookie.com\"), " +
	"geolocation=(), gyroscope=(), magnetometer=(), microphone=(self), midi=(), payment=(), " +
	"picture-in-picture=(self \"https://www.youtube.com\" \"https://www.youtube-nocookie.com\"), " +
	"usb=(), xr-spatial-tracking=()"

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	// CSP burada HTML için AYARLANMAZ: politikadaki satır içi script
	// hash'leri ancak render edilmiş gövdeden üretilebilir. Bu yüzden
	// varsayılan (hash'siz, en kısıtlayıcı) politika konur; HTML üreten
	// yollar (s.render, s.renderPublic, renderPartial, AMP, akış) gövdeyi
	// tamponlayıp setCSPForHTML ile başlığı kendi hash'leriyle değiştirir.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", permissionsPolicy)
		w.Header().Set("Content-Security-Policy", buildCSP(nil, nil, collabURL(), s.cspRaporURL()))
		// report-to grubu: CSP politikasındaki "report-to csp" ile eşleşir.
		w.Header().Set("Report-To", `{"group":"csp","max_age":10886400,"endpoints":[{"url":"`+s.cspRaporURL()+`"}]}`)
		// HSTS: yalnız sitenin genel adresi HTTPS ise gönderilir (yerel HTTP
		// kurulumlarında tarayıcıyı kilitlememek için). includeSubDomains
		// bilinçli olarak YOK: alt alan adlarından biri HTTP'ye bağımlıysa
		// site komple erişilemez hâle gelirdi. Bir yıl, preload listesine
		// başvurmadan geri alınabilir bir süre.
		if s.secureCookie() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// withSameOrigin, durum değiştiren isteklerde Origin/Referer başlığının kendi
// host'umuza ait olduğunu doğrular. CSRF token'ı olmayan formlarda SameSite=Lax
// tek savunmaydı; bu ara katman tarayıcıların gönderdiği Origin başlığını
// kullanarak çapraz kaynak POST'ları reddeder. Origin/Referer hiç yoksa
// (curl, sunucudan sunucuya webhook, e2e) istek geçer: bu durumlar tarayıcı
// kaynaklı CSRF değildir.
func (s *Server) withSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if !sameOriginRequest(r) {
			s.logger.Warn("csrf: çapraz kaynak istek reddedildi",
				"path", r.URL.Path, "method", r.Method,
				"origin", r.Header.Get("Origin"), "referer", r.Referer(),
				"remote", r.RemoteAddr)
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "Geçersiz istek kaynağı.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sameOriginRequest Origin varsa host eşleşmesini, yoksa Referer'ı denetler.
func sameOriginRequest(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		// "null" (sandbox/file kaynaklı) kabul edilmez.
		if origin == "null" {
			return false
		}
		u, err := neturl.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		return sameOriginHost(u.Host, r.Host)
	}
	if ref := strings.TrimSpace(r.Referer()); ref != "" {
		u, err := neturl.Parse(ref)
		if err != nil || u.Host == "" {
			return false
		}
		return sameOriginHost(u.Host, r.Host)
	}
	return true
}

// sameOriginHost host'ları karşılaştırır; baştaki "www." yok sayılır
// (site hem www hem kök alan adından servis edilebiliyor).
func sameOriginHost(a, b string) bool {
	norm := func(h string) string {
		h = strings.ToLower(strings.TrimSpace(h))
		return strings.TrimPrefix(h, "www.")
	}
	return norm(a) == norm(b)
}
