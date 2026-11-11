// theblock — migration CLI (Go, şema sahibi).
// Kullanım:
//
//	go run ./cmd/migrate              # up (bekleyenleri uygula)
//	go run ./cmd/migrate status       # durum
//	go run ./cmd/migrate down         # son migration'ı geri al
//	go run ./cmd/migrate down --all   # tümünü geri al
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/jackc/pgx/v5"

	"theblock/site/internal/config"
	"theblock/site/internal/db"
	"theblock/site/internal/platform"
)

func main() {
	ctx := context.Background()
	logger := platform.NewLogger(slog.LevelInfo)
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL boş — deploy/.env yükleyip çalıştırın")
		os.Exit(1)
	}

	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("veritabanına bağlanılamadı", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close(ctx) }()

	if err := conn.Ping(ctx); err != nil {
		logger.Error("veritabanı ping başarısız", "error", err)
		os.Exit(1)
	}

	cmd := "up"
	args := os.Args[1:]
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "up":
		applied, err := db.Up(ctx, conn)
		if err != nil {
			logger.Error("up başarısız", "error", err)
			os.Exit(1)
		}
		if len(applied) == 0 {
			fmt.Println("schema güncel; uygulanacak migration yok")
		} else {
			for _, m := range applied {
				fmt.Printf("UYGULANDI  %04d_%s\n", m.Version, m.Name)
			}
		}

	case "status":
		status(ctx, conn, logger)

	case "down":
		all := len(args) > 1 && args[1] == "--all"
		rolled, err := db.Down(ctx, conn, all)
		if err != nil {
			logger.Error("down başarısız", "error", err)
			os.Exit(1)
		}
		if len(rolled) == 0 {
			fmt.Println("geri alınacak migration yok")
		} else {
			for _, m := range rolled {
				fmt.Printf("GERİ ALINDI  %04d_%s\n", m.Version, m.Name)
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "bilinmeyen komut: %q\n(up | status | down [--all])\n", cmd)
		os.Exit(2)
	}

	if cmd != "status" {
		_ = status(ctx, conn, logger)
	}
}

// status uygulanan ve bekleyen migration'ları listeler.
func status(ctx context.Context, conn *pgx.Conn, logger *slog.Logger) int {
	ls, err := db.LoadMigrations()
	if err != nil {
		logger.Error("migration listesi okunamadı", "error", err)
		os.Exit(1)
	}
	applied, err := db.Applied(ctx, conn)
	if err != nil {
		logger.Error("uygulanan listesi okunamadı", "error", err)
		os.Exit(1)
	}
	for _, m := range ls {
		state := "bekliyor  "
		if _, ok := applied[m.Version]; ok {
			state = "uygulandi "
		}
		fmt.Printf("%s  %04d_%s\n", state, m.Version, m.Name)
	}

	// bekleyen sürümleri de göster
	var pending []int
	for _, m := range ls {
		if _, ok := applied[m.Version]; !ok {
			pending = append(pending, m.Version)
		}
	}
	sort.Ints(pending)
	if len(pending) > 0 {
		fmt.Printf("bekleyen: %d\n", len(pending))
	}
	return len(pending)
}
