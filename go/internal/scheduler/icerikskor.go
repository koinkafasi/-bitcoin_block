package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

func RunIcerikSkor(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 15 * time.Minute
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	logger.Info("içerik skoru başladı", "her", every.String())

	// İlk koşum hemen: servis yeniden başladığında kuyruk bayat kalmasın.
	icerikSkorTick(ctx, pool, logger)
	for {
		select {
		case <-ctx.Done():
			logger.Info("içerik skoru durdu")
			return
		case <-ticker.C:
			icerikSkorTick(ctx, pool, logger)
		}
	}
}

func icerikSkorTick(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	sites, err := repo.ListSites(ctx, pool)
	if err != nil {
		logger.Warn("içerik skoru: siteler", "error", err)
		return
	}
	for _, st := range sites {
		if n, err := repo.RefreshGuvenScores(ctx, pool, st.ID); err != nil {
			logger.Warn("içerik skoru: güven", "site", st.Domain, "error", err)
		} else if n > 0 {
			logger.Info("içerik skoru: güven", "site", st.Domain, "satir", n)
		}

		r, err := repo.RefreshTrendScores(ctx, pool, st.ID, 1000, false)
		if err != nil {
			logger.Warn("içerik skoru: trend", "site", st.Domain, "error", err)
			continue
		}
		if r.Yazilan > 0 || r.Taranan > 0 {
			logger.Info("içerik skoru: trend",
				"site", st.Domain, "terim", r.Terim, "taranan", r.Taranan,
				"eslesen", r.Eslesen, "yazilan", r.Yazilan,
				"sure", r.Sure.Round(time.Millisecond).String())
		}
	}
}
