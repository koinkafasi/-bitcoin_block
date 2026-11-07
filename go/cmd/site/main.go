// theblock — Go web/API giriş noktası (ana üretim dili katmanı).
// Yükler: config + slog + PostgreSQL havuzu + otomatik migration (idempotent)
// + i18n (24 dil) + AI istemcisi + /healthz + SSR site + yönetim paneli.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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

	srv, app := httpapi.NewServerApp(cfg, logger, pool, bundle, aiclient)
	logger.Info("theblock site dinliyor", "addr", cfg.ListenAddr, "env", cfg.Env)

	// Periyodik işler: planlı yayın, çöp temizliği, oturum temizliği.
	workerCtx, workerCancel := context.WithCancel(ctx)
	go scheduler.Run(workerCtx, pool, logger, 30*time.Second)
	go scheduler.RunBot(workerCtx, pool, logger, 60*time.Second)
	go scheduler.RunGundem(workerCtx, pool, logger, 60*time.Second)
	go scheduler.RunServices(workerCtx, pool, logger, 30*time.Minute)
	go scheduler.RunFinance(workerCtx, pool, logger, 2*time.Minute)
	go scheduler.RunTrends(workerCtx, pool, logger, 6*time.Hour)
	go scheduler.RunTrendFetch(workerCtx, pool, logger, 30*time.Second)
	go scheduler.RunTrendScore(workerCtx, pool, logger, 15*time.Minute)
	go scheduler.RunIcerikSkor(workerCtx, pool, logger, 15*time.Minute)
	go scheduler.RunAISource(workerCtx, pool, logger, 15*time.Minute)
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
		pool.Close()
		logger.Info("temiz kapanış tamamlandı")
	}
}

// asrDefaults .env'den gelen başlangıç ASR yapılandırmasını kurar. Panel
// değerleri bunları geçersiz kılar (bkz. transcribe.SettingsStore).
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
	defer conn.Close(ctx)
	applied, err := db.Up(ctx, conn)
	if err != nil {
		logger.Error("migration başarısız", "error", err)
		os.Exit(1)
	}
	for _, m := range applied {
		logger.Info("migration uygulandı", "version", m.Version, "name", m.Name)
	}
}
