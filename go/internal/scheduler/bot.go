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

// botPoller RSS/ajans feed'lerini periyodik çeker (P1 içerik akışı).
type botPoller struct {
	logger    *slog.Logger
	lastCrawl map[int64]time.Time
	fails     map[int64]int
	mu        sync.Mutex
}

// RunBot feed politeleyi başlatır; her <every> saniyede kontrol eder.
func RunBot(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 60 * time.Second
	}
	bp := &botPoller{
		logger:    logger,
		lastCrawl: map[int64]time.Time{},
		fails:     map[int64]int{},
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	logger.Info("bot poller başladı", "her", every.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info("bot poller durdu")
			return
		case <-ticker.C:
			bp.tick(ctx, pool)
		}
	}
}

func (bp *botPoller) tick(ctx context.Context, pool *pgxpool.Pool) {
	// otomatik çek kapalıysa pas
	if b, err := repo.GetSetting(ctx, pool, "bot", "otomatik"); err == nil || string(b) != "true" {
		if err != nil {
			return
		}
		if string(b) != "true" {
			return
		}
	}
	dakika := 5
	if b, err := repo.GetSetting(ctx, pool, "bot", "dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil && n > 0 {
			dakika = n
		}
	}
	author := int64(1)
	if b, err := repo.GetSetting(ctx, pool, "bot", "editorde"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil && n > 0 {
			author = int64(n)
		}
	}
	rules, _ := repo.LoadWordRules(ctx, pool)

	feeds, err := repo.ListFeeds(ctx, pool)
	if err != nil {
		return
	}
	base := time.Duration(dakika) * time.Minute
	now := time.Now()
	for _, f := range feeds {
		if !f.Aktif {
			continue
		}
		bp.mu.Lock()
		next := bp.lastCrawl[f.ID]
		fails := bp.fails[f.ID]
		bp.mu.Unlock()
		if !next.IsZero() && now.Before(next) {
			continue
		}
		n, cerr := repo.CrawlFeed(ctx, pool, &f, rules, author)
		bp.mu.Lock()
		if cerr != nil {
			fails++
			bp.fails[f.ID] = fails
			bp.lastCrawl[f.ID] = now.Add(backoffDelay(base, fails))
		} else {
			bp.fails[f.ID] = 0
			bp.lastCrawl[f.ID] = now.Add(base)
		}
		bp.mu.Unlock()
		if cerr != nil {
			bp.logger.Warn("feed çekilemedi", "feed", f.Name, "hata_sayisi", fails,
				"sonraki_deneme", backoffDelay(base, fails).String(), "error", cerr)
			continue
		}
		if n > 0 {
			bp.logger.Info("feed çekildi", "feed", f.Name, "yeni", n)
		}
	}
}

func backoffDelay(base time.Duration, fails int) time.Duration {
	const (
		minDelay = 15 * time.Minute
		maxDelay = 6 * time.Hour
	)
	if fails <= 0 {
		return base
	}
	d := base
	if d < minDelay {
		d = minDelay
	}
	for i := 1; i < fails; i++ {
		d *= 2
		if d >= maxDelay {
			return maxDelay
		}
	}
	if d > maxDelay {
		d = maxDelay
	}
	return d
}
