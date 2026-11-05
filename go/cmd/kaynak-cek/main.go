package main

// kaynak-cek — RSS/ajans kaynaklarını çeker (elle tetikleme).
//
// Zamanlayıcı (`scheduler.RunBot`) aynı yolu kullanır; bu komut yalnız
// "şimdi çek" içindir (kurulum, hata ayıklama, ilk doldurma).
//
//	go run ./cmd/kaynak-cek            # tüm AKTİF kaynaklar
//	go run ./cmd/kaynak-cek --yayinla  # çekilen taslakları yayına al
//
// `--yayinla` bilinçli olarak AYRI bayrak: toplama hattı taslak üretir,
// yayına alma kararı editöründür. Demo/ilk doldurmada elle açılır.

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"theblock/site/internal/config"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	yayinla := flag.Bool("yayinla", false, "çekilen taslak içerikleri yayına al")
	limit := flag.Int("limit", 0, "kaynak başına en çok öğe (0 = sınırsız)")
	flag.Parse()

	ctx := context.Background()
	logger := platform.NewLogger(slog.LevelInfo)
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL boş")
		os.Exit(1)
	}
	pc, err := cfg.PoolConfig()
	if err != nil {
		logger.Error("havuz yapılandırması", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		logger.Error("havuz", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Toplayıcı yazar: tek sistem hesabı (migration 0113).
	var author int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM users WHERE email='toplayici@bitcosar.com'`).Scan(&author); err != nil {
		logger.Error("toplayıcı yazar bulunamadı", "error", err)
		os.Exit(1)
	}

	rules, _ := repo.LoadWordRules(ctx, pool)
	feeds, err := repo.ListFeeds(ctx, pool)
	if err != nil {
		logger.Error("kaynaklar okunamadı", "error", err)
		os.Exit(1)
	}

	var toplam int
	for _, f := range feeds {
		if !f.Aktif {
			continue
		}
		n, cerr := repo.CrawlFeed(ctx, pool, &f, rules, author)
		if cerr != nil {
			logger.Warn("kaynak çekilemedi", "kaynak", f.Name, "error", cerr)
			continue
		}
		if n > 0 {
			logger.Info("kaynak çekildi", "kaynak", f.Name, "yeni", n)
			toplam += n
		}
	}
	fmt.Printf("toplam yeni: %d\n", toplam)

	if *yayinla {
		tag, err := pool.Exec(ctx, `
			UPDATE contents SET status='yayinda', published_at=COALESCE(published_at, now()),
			       updated_at=now()
			WHERE status='taslak' AND source_url IS NOT NULL`)
		if err != nil {
			logger.Error("yayına alma", "error", err)
			os.Exit(1)
		}
		fmt.Printf("yayına alınan: %d\n", tag.RowsAffected())
	}
	_ = limit
	_ = time.Now
}
