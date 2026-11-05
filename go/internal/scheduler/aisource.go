package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// RunAISource, AI Kaynak Merkezi için RSS kaynaklarını periyodik tarar ve
// benzer haber kümelerini oluşturur. Taslak üretmez; üretim editör onayına
// bağlıdır.
func RunAISource(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 15 * time.Minute
	}
	logger.Info("AI kaynak tarayıcı başladı", "her", every.String())
	runAISourceOnce(ctx, pool, logger)

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("AI kaynak tarayıcı durdu")
			return
		case <-ticker.C:
			runAISourceOnce(ctx, pool, logger)
		}
	}
}

func runAISourceOnce(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	items, clusters, merged, err := repo.ScanAndClusterAISources(ctx, pool, 1)
	if err != nil {
		logger.Warn("AI kaynak tarama/kümeleme hatası", "error", err)
		return
	}
	if items > 0 || clusters > 0 || merged > 0 {
		logger.Info("AI kaynak taraması tamamlandı",
			"yeni_öğe", items, "yeni_küme", clusters, "birleştirilen", merged)
	}
	// HTML kaynaklarında yalnızca başlıkla kalan öğeler için kontrollü
	// zenginleştirme. Her turda en fazla 60 makale sayfası okunur.
	enrich, err := repo.EnrichAISourceItems(ctx, pool, 1, 60)
	if err != nil {
		logger.Warn("AI kaynak zenginleştirme hatası", "error", err)
		return
	}
	if enrich.Enriched > 0 || enrich.Failed > 0 || enrich.Recategorized > 0 {
		logger.Info("AI kaynak zenginleştirmesi tamamlandı",
			"zenginleştirilen", enrich.Enriched, "kategori_düzeltilen", enrich.Recategorized,
			"hata", enrich.Failed, "bekleyen", enrich.Pending)
	}
}
