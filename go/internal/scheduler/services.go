package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// RunServices P3 servislerini periyodik çeker; her servis kendi goroutine'inde
// (yavaş dış API'ler birbirini bloke etmez), dış kapalıysa demo satırı yazar.
func RunServices(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 30 * time.Minute
	}
	logger.Info("servis poller başladı", "her", every.String())
	refreshCore(ctx, pool, logger) // ilk koşu

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("servis poller durdu")
			return
		case <-ticker.C:
			refreshCore(ctx, pool, logger)
		}
	}
}

func refreshCore(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	var wg sync.WaitGroup
	run := func(name string, f func() (int, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := f()
			if err != nil {
				logger.Warn("servis: "+name, "error", err)
			} else if n > 0 {
				logger.Info("servis güncellendi: "+name, "adet", n)
			}
		}()
	}
	run("hava", func() (int, error) { return repo.RefreshWeather(ctx, pool) })
	run("namaz", func() (int, error) { return repo.RefreshNamaz(ctx, pool) })
	run("eczane", func() (int, error) { return repo.RefreshEczane(ctx, pool) })
	run("fikstur", func() (int, error) { return 0, repo.RefreshFikstur(ctx, pool) })
	run("fikstur-h", func() (int, error) { return 0, repo.RefreshFixtures(ctx, pool) })
	run("puan", func() (int, error) { return 0, repo.RefreshStandings(ctx, pool) })
	run("astroloji", func() (int, error) { return 0, repo.RefreshAstroloji(ctx, pool) })
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = repo.RefreshImsakiye(ctx, pool, 6)
		logger.Info("servis güncellendi: imsakiye")
	}()
	wg.Wait()
}

func RunFinance(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 2 * time.Minute
	}
	logger.Info("finans poller başladı", "her", every.String())
	refreshFinance(ctx, pool, logger, true) // ilk koşu

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	tur := 0
	for {
		select {
		case <-ctx.Done():
			logger.Info("finans poller durdu")
			return
		case <-ticker.C:
			tur++
			refreshFinance(ctx, pool, logger, tur%3 == 0)
		}
	}
}

func refreshFinance(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, hisse bool) {
	var wg sync.WaitGroup
	run := func(name string, f func() (int, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := f()
			if err != nil {
				logger.Warn("finans: "+name, "error", err)
			} else if n > 0 {
				logger.Info("finans güncellendi: "+name, "adet", n)
			}
		}()
	}
	run("kur", func() (int, error) { return 0, repo.RefreshForex(ctx, pool) })
	run("borsa", func() (int, error) { return 0, repo.RefreshBorsa(ctx, pool) })
	run("piyasa", func() (int, error) { return 0, repo.RefreshPiyasa(ctx, pool) })
	run("kurlar", func() (int, error) { return 0, repo.RefreshKurlar(ctx, pool) })
	run("kripto", func() (int, error) { return 0, repo.RefreshKripto(ctx, pool) })
	if hisse {
		run("hisseler", func() (int, error) { return 0, repo.RefreshHisseler(ctx, pool) })
	}
	wg.Wait()
}
