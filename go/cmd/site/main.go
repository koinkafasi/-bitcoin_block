// theblock — Go web/API giriş noktası (ana üretim dili katmanı).
// Yükler: config + slog + PostgreSQL havuzu + otomatik migration (idempotent)
// + i18n (24 dil) + AI istemcisi + /healthz + SSR site + yönetim paneli.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/aieditor"
	"theblock/site/internal/ceviri"
	"theblock/site/internal/config"
	"theblock/site/internal/db"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
	"theblock/site/internal/platform"
	"theblock/site/internal/scheduler"
	"theblock/site/internal/transcribe"
)

func main() {
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := platform.NewLogger(level)

	// Sinyal alınca çalışan işler (worker'lar, migration) durdurulabilsin.
	// Shutdown ayrı bir zaman aşımıyla yönetilir (aşağıda).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL boş — ortamı kontrol edin")
		os.Exit(1)
	}

	// Migration (idempotent; ayrıca cmd/migrate ile de çalışır)
	runMigrations(ctx, logger, cfg.DatabaseURL)

	// Havuz yapılandırması: bozuk DSN burada temiz bir hatayla yakalanır.
	// (Doğrudan NewWithConfig çağrılsaydı bozuk DSN'de PANIC olurdu:
	//  "config must be created by ParseConfig". Ölçüldü.)
	poolCfg, err := cfg.PoolConfig()
	if err != nil {
		logger.Error("veritabanı yapılandırması geçersiz", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Error("havuz kurulamadı", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("veritabanına erişilemiyor", "error", err)
		os.Exit(1)
	}

	bundle, err := i18n.Load(ctx, pool)
	if err != nil {
		logger.Error("i18n yüklenemedi", "error", err)
		os.Exit(1)
	}
	logger.Info("i18n", "diller", len(bundle.Enabled()))

	aiclient := ai.NewClient(logger)
	logger.Info("ai", "enabled", aiclient.Enabled(), "providers", aiclient.Providers())

	// Görsel hattını Rust mikroservisine devret (varsa). Adres yoksa hiçbir
	// şey değişmez: tüm görsel işlemleri yerel Go yolunda kalır.
	platform.SetImageSvcURL(os.Getenv("IMAGE_SVC_URL"))
	if u := platform.ImageSvcURL(); u != "" {
		logger.Info("image-svc devrede", "url", u)
	}
	// Sıkıştırma seviyesi: boşsa paket varsayılanı (4) kalır.
	if cfg.GzipLevel > 0 {
		httpapi.SetGzipLevel(cfg.GzipLevel)
	}
	logger.Info("gzip", "seviye", httpapi.GzipLevel())

	// Tam sayfa önbelleği süreleri: boşsa 10 sn taze + 30 sn bayat.
	if cfg.SayfaOnbellekTTL > 0 {
		httpapi.SetSayfaOnbellekSuresi(cfg.SayfaOnbellekTTL, cfg.SayfaOnbellekBayat)
	}
	ttl, bayat := httpapi.SayfaOnbellekSuresi()
	logger.Info("sayfa önbelleği", "taze", ttl.String(), "bayat", bayat.String())

	srv, app := httpapi.NewServerApp(cfg, logger, pool, bundle, aiclient)
	logger.Info("theblock site dinliyor", "addr", cfg.ListenAddr, "env", cfg.Env)

	// Profil dinleyicisi (opsiyonel). Ana mux'a DEĞİL, ayrı bir adrese bağlanır:
	// pprof çıktısı heap/goroutine/dosya yolu sızdırır ve site tünelden
	// erişilebilir durumdadır. PPROF_ADDR boşsa hiç açılmaz — üretimde
	// varsayılan kapalı, açmak için 127.0.0.1:6060 gibi bir adres verilir.
	pprofSrv := startPprof(cfg.PprofAddr, logger)

	// Periyodik işler: planlı yayın, çöp temizliği, oturum temizliği.
	workerCtx, workerCancel := context.WithCancel(ctx)
	// Okunma sayacı: istek yolundan çıkarıldı; biriken sayılar ~2 sn'de bir
	// tek turda yazılır (iki UPSERT/görüntüleme yerine).
	httpapi.OkunmaSayaciBaslat(workerCtx, pool, logger, 0)
	go scheduler.Run(workerCtx, pool, logger, 30*time.Second)
	go scheduler.RunBot(workerCtx, pool, logger, 60*time.Second)
	go scheduler.RunGundem(workerCtx, pool, logger, 60*time.Second)
	go scheduler.RunServices(workerCtx, pool, logger, 30*time.Minute)
	go scheduler.RunFinance(workerCtx, pool, logger, 2*time.Minute)
	// CANLI PİYASA MERKEZİ (11 Kasım 2026): Binance fiyatlarını 5 sn'de bir
	// tazeler ve DEĞİŞTİĞİ ANDA bağlı tüm tarayıcılara iter (SSE).
	// Sayfa isteklerinden bağımsız: kimse girmezse de önbellek sıcak kalır.
	app.CanliPiyasaBaslat(workerCtx)
	// CANLI PİYASA MERKEZİ (11 Kasım 2026): Binance fiyatlarını 5 sn'de bir
	// tazeler ve DEĞİŞTİĞİ ANDA bağlı tüm tarayıcılara iter (SSE).
	// Sayfa isteklerinden bağımsız: kimse girmezse de önbellek sıcak kalır.
	app.CanliPiyasaBaslat(workerCtx)
	go scheduler.RunTrends(workerCtx, pool, logger, 6*time.Hour)
	go scheduler.RunTrendFetch(workerCtx, pool, logger, 30*time.Second)
	go scheduler.RunTrendScore(workerCtx, pool, logger, 15*time.Minute)
	go scheduler.RunIcerikSkor(workerCtx, pool, logger, 15*time.Minute)
	go scheduler.RunAISource(workerCtx, pool, logger, 15*time.Minute)
	// KAYNAK GÖRSELLERİNİ ÖNDEN İNDİR (12 Kasım 2026): `/kg/` tembel
	// çalışıyordu — görsel ancak biri sayfayı açınca iniyordu ve 4.918
	// kayıttan yalnız 136'sı diskteydi (%3). Kaynak sunucu görseli
	// taşıyınca/silince içerik kalıcı görselsiz kalıyor; eski WordPress
	// görsellerinde yaşandı (cdn.bitcosar.com artık 403). Bu işçi
	// arşivi küçük gruplar hâlinde kendi diskimize alır.
	app.GorselOnIndiriciBaslat(workerCtx)
	// YouTube video arşivi: kanal RSS'i + /streams + /live periyodik çekilir
	// (aralık panelden: settings/youtube.dakika). /videos, /videos/shorts,
	// /canli-yayin ve ana sayfa modülleri bu tablodan beslenir.
	go scheduler.RunYoutube(workerCtx, pool, logger, time.Minute)
	// Faz 189: İngilizce içerik. Türkçe yayımlanmış haberler AI ile
	// çevrilip ayrı `contents` satırı olarak yazılır (locale='en',
	// translation_of=kaynak). /en bölümleri gerçek İngilizce içerikle dolar.
	go ceviri.Run(workerCtx, pool, aiclient, logger, 1)
	// Faz 188: otomatik haberlerin SEO başlığı/meta açıklaması boş kalıyordu
	// (RSS botu yalnız başlık/spot/gövde yazıyor). Bu iş boş alanları
	// içeriğe göre doldurur; editörün yazdığı alanlara dokunmaz.
	go aieditor.RunSeoDoldurma(workerCtx, pool, aiclient, logger, 2*time.Minute)
	// Faz 194 — otomatik AI ile yazım: açıksa bot taslakları AI'dan geçer.
	// Ayar kapalıyken döngü hiçbir şey yapmaz (AI çağrısı yok).
	go aieditor.RunYazimDoldurma(workerCtx, pool, aiclient, logger, time.Minute)
	// Faz 204 — toplu AI ile yeniden yazım: yalnız editörün başlattığı pilot
	// önerilerini üretir (yayına yazmaz; onay paneldedir).
	go aieditor.RunYenidenYazim(workerCtx, pool, aiclient, logger, 10*time.Second)
	go scheduler.RunGSC(workerCtx, pool, logger, 6*time.Hour)
	go scheduler.RunPodcast(workerCtx, pool, logger)
	go app.RunGazeteJobs(workerCtx, time.Minute)
	go aieditor.StartWorker(workerCtx, pool, aiclient, logger, time.Second)
	asrStore := transcribe.NewSettingsStore(pool, asrDefaults(cfg))
	go transcribe.StartWorker(workerCtx, pool, transcribe.NewDynamic(asrStore),
		transcribe.NewFFmpeg(cfg.FFmpegBin, cfg.FFprobeBin), logger, 2*time.Second)
	{
		cur := asrStore.Load(ctx)
		if cur.Empty() {
			logger.Warn("transkripsiyon: ASR anahtarı yok — panelden (/admin/ses-ayarlari) ekleyin",
				"ad", cur.Provider)
		} else {
			logger.Info("transkripsiyon sağlayıcısı", "ad", cur.Provider, "model", cur.Model)
		}
	}
	defer workerCancel()

	// ---- Kapanış ----
	//
	// ÖNCE: `srv.ListenAndServe()` + `os.Exit(1)` idi. SIGTERM'de süreç
	// ANINDA ölüyordu: uçuştaki HTTP istekleri yanıtsız kopuyor, yarım
	// kalan AI/transkripsiyon işleri veritabanında "işleniyor" olarak
	// kalıyordu. Deploy sırasında kullanıcı 502/boş yanıt görüyordu.
	//
	// ŞİMDİ: dinlemeyi ayrı goroutine'de başlatıp sinyali bekliyoruz.
	// Sinyal gelince:
	//   1) /readyz 503 döner (yeni trafik gelmesin),
	//   2) Shutdown uçuştaki isteklerin bitmesini bekler,
	//   3) worker'lar iptal edilir, havuz kapanır.
	sunucuHata := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sunucuHata <- err
		}
		close(sunucuHata)
	}()

	select {
	case err, ok := <-sunucuHata:
		if ok && err != nil {
			logger.Error("sunucu duramadı", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		pencere := cfg.ShutdownWindow()
		logger.Info("kapanış sinyali alındı", "pencere", pencere.String())

		// 1) Yeni trafiği kes (hazırlık sondası 503).
		httpapi.SetReady(false)

		// 2) Uçuştaki istekler bitsin. Ana ctx iptal edildiği için
		//    Shutdown'a AYRI bir ctx vermek zorunlu — yoksa anında döner
		//    ve hiçbir isteği beklemez.
		kapanisCtx, iptal := context.WithTimeout(context.Background(), pencere)
		defer iptal()
		if err := srv.Shutdown(kapanisCtx); err != nil {
			// Süre doldu: kalan bağlantılar zorla kapatılır.
			logger.Warn("uçuştaki istekler pencere içinde bitmedi", "error", err)
			_ = srv.Close()
		}

		// 3) Arka plan işleri durdur, havuzu kapat.
		workerCancel()
		if pprofSrv != nil {
			_ = pprofSrv.Close()
		}
		pool.Close()
		logger.Info("temiz kapanış tamamlandı")
	}
}

// startPprof PPROF_ADDR doluysa yalnız /debug/pprof/* sunan ayrı bir HTTP
// sunucusu başlatır ve döndürür; boşsa nil.
//
// Ayrı dinleyici bilinçli bir güvenlik kararı: profil çıktısı heap içeriği,
// goroutine yığınları ve dosya yolları taşır. Genel mux'a eklenseydi
// demo.bitcosar.com tünelinden herkese açık olurdu. Varsayılan davranış
// (PPROF_ADDR boş) hiç açmamaktır.
func startPprof(addr string, logger *slog.Logger) *http.Server {
	if strings.TrimSpace(addr) == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("pprof dinliyor", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Profil sunucusu çökerse site çalışmaya devam eder: ölçüm
			// aracı, ürünün tek hata noktası olmamalı.
			logger.Warn("pprof sunucusu duramadı", "error", err)
		}
	}()
	return srv
}

// asrDefaults .env'den gelen başlangıç ASR yapılandırmasını kurar.
// Panel değerleri bunları geçersiz kılar (bkz. transcribe.SettingsStore).
func asrDefaults(cfg config.Config) transcribe.Config {
	return transcribe.DefaultsFromValues(
		cfg.TranscribeProvider,
		cfg.AssemblyAIKey, cfg.AssemblyAIBase, cfg.AssemblyAIModel,
		cfg.DeepgramKey, cfg.DeepgramBase, cfg.DeepgramModel,
	)
}

// runMigrations şemayı açar; yalnız bekleyenleri uygular.
func runMigrations(ctx context.Context, logger *slog.Logger, databaseURL string) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		logger.Error("migration bağlantısı kurulamadı", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close(ctx) }()
	applied, err := db.Up(ctx, conn)
	if err != nil {
		logger.Error("migration başarısız", "error", err)
		os.Exit(1)
	}
	for _, m := range applied {
		logger.Info("migration uygulandı", "version", m.Version, "name", m.Name)
	}
}
