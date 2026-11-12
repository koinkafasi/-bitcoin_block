package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// RunOtomatikYayin skor eşiğini geçen taslakları kademeli olarak yayına
// alır.
//
// AYAR KAPALIYKEN HİÇBİR ŞEY YAPMAZ — ne sorgu ne yazma. Ayar
// `otomatik_yayin.aktif` panelden açılır; varsayılan kapalı, çünkü
// içeriğin insan onayı olmadan yayına çıkması bir politika kararıdır.
//
// Neden kademeli (tur başına bir içerik): eşiği geçen 429 taslak var.
// Hepsi bir anda çıksa akış anlamsız görünür ve arama motoruna ani kütle
// gider. Tur başına bir içerik + günlük limit, doğal bir yayın ritmi verir.
func RunOtomatikYayin(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 10 * time.Minute
	}
	logger.Info("otomatik yayın işçisi başladı", "her", every.String())

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("otomatik yayın işçisi durdu")
			return
		case <-ticker.C:
			otomatikYayinTuru(ctx, pool, logger)
		}
	}
}

func otomatikYayinTuru(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	ayar := repo.OtomatikYayinAyarOku(ctx, pool)
	if !ayar.Aktif {
		return // kapalı: sorgu bile yapılmaz
	}

	bugun, err := repo.OtomatikYayinBugunSayisi(ctx, pool)
	if err != nil {
		logger.Warn("otomatik yayın: günlük sayaç okunamadı", "error", err)
		return
	}
	if ayar.GunlukLimit > 0 && bugun >= ayar.GunlukLimit {
		return // günlük kota dolu; yarın devam
	}

	// Tur başına BİR içerik: ritmi korur ve bir hata durumunda etkiyi
	// tek içerikle sınırlar.
	adaylar, err := repo.OtomatikYayinAdaylari(ctx, pool, ayar, 1)
	if err != nil {
		logger.Warn("otomatik yayın: adaylar okunamadı", "error", err)
		return
	}
	if len(adaylar) == 0 {
		return
	}

	a := adaylar[0]
	if err := repo.OtomatikYayinla(ctx, pool, a.ID); err != nil {
		logger.Warn("otomatik yayın: yayınlanamadı", "icerik", a.ID, "error", err)
		return
	}
	logger.Info("otomatik yayın",
		"icerik", a.ID, "baslik", a.Title, "dil", a.Locale,
		"guven", a.GuvenSkor, "trend", a.TrendSkor, "ai", a.AISkor,
		"bugun", bugun+1, "limit", ayar.GunlukLimit)
}
