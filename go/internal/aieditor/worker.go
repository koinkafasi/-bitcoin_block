package aieditor

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
)

// StartWorker DB kuyruğundaki AI işlerini arka planda yürütür. Birden fazla
// süreç çalıştırılabilir; ClaimAIJob SKIP LOCKED ile aynı işi iki kez almaz.
func StartWorker(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	svc := NewService(pool, aiclient, logger)
	if n, err := repo.RecoverStaleAIJobs(ctx, pool, 15*time.Minute); err != nil {
		logger.Warn("ai worker stale temizliği", "error", err)
	} else if n > 0 {
		logger.Info("ai worker stale işleri kuyruğa aldı", "adet", n)
	}
	concurrency := envInt("AI_WORKER_CONCURRENCY", 2)
	if concurrency <= 0 {
		concurrency = 1
	}
	logger.Info("ai worker başladı", "concurrency", concurrency, "interval", interval)
	for i := 0; i < concurrency; i++ {
		go svc.workerLoop(ctx, interval)
	}
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := repo.RecoverStaleAIJobs(ctx, pool, 15*time.Minute); err == nil && n > 0 {
					logger.Info("ai worker stale işleri kuyruğa aldı", "adet", n)
				}
			}
		}
	}()
}

func (s *Service) workerLoop(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for {
				ok, err := s.ProcessNext(ctx)
				if err != nil {
					s.logger.Warn("ai worker iş hatası", "error", err)
				}
				if !ok {
					break
				}
			}
		}
	}
}

// ProcessNext kuyruktaki bir işi sahiplenip yürütür. Testlerde tek adım
// çalıştırmak için de kullanılır.
func (s *Service) ProcessNext(ctx context.Context) (bool, error) {
	job, ok, err := repo.ClaimAIJob(ctx, s.pool)
	if err != nil || !ok {
		return false, err
	}
	_, err = s.ProcessJob(ctx, job.ID)
	if err != nil && err != ErrJobCancelled {
		return true, err
	}
	return true, nil
}
