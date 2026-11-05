package scheduler

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// Gündem RSS agrega prolesi.
type gundemPoller struct {
	logger  *slog.Logger
	lastRun time.Time
	mu      sync.Mutex
}

// RunGundem gündem kaynaklarını periyodik çeker.
func RunGundem(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 60 * time.Second
	}
	gp := &gundemPoller{logger: logger}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	logger.Info("gündem poller başladı", "her", every.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info("gündem poller durdu")
			return
		case <-ticker.C:
			gp.tick(ctx, pool)
		}
	}
}

func (gp *gundemPoller) tick(ctx context.Context, pool *pgxpool.Pool) {
	dakika := 15
	if b, err := repo.GetSetting(ctx, pool, "gundem", "dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil && n > 0 {
			dakika = n
		}
	}
	gp.mu.Lock()
	due := time.Since(gp.lastRun) >= time.Duration(dakika)*time.Minute
	gp.mu.Unlock()
	if !due {
		return
	}
	n, err := repo.FetchTrends(ctx, pool)
	gp.mu.Lock()
	gp.lastRun = time.Now()
	gp.mu.Unlock()
	if err != nil {
		gp.logger.Warn("gündem çekilemedi", "error", err)
		return
	}
	if n > 0 {
		gp.logger.Info("gündem çekildi", "yeni", n)
	}
}
