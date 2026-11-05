package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Bot tuzağı (honeypot) — spec 06 K-I ----

// BotTrapHit tuzağa düşen bot isabet kaydı.
type BotTrapHit struct {
	ID        int64
	IP        string
	UserAgent string
	Path      string
	Referer   string
	CreatedAt string
}

// LogBotTrap tuzağa düşen isteği kaydeder.
func LogBotTrap(ctx context.Context, pool *pgxpool.Pool, ip, ua, path, referer string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO bot_trap_hits (ip, user_agent, path, referer)
		VALUES (nullif($1,''),nullif($2,''),$3,nullif($4,''))`, ip, ua, path, referer)
	return err
}

// ListBotTrapHits son isabetleri döndürür.
func ListBotTrapHits(ctx context.Context, pool *pgxpool.Pool, limit int) ([]BotTrapHit, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(ip,''), COALESCE(user_agent,''), path, COALESCE(referer,''), created_at::text
		FROM bot_trap_hits ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BotTrapHit
	for rows.Next() {
		var h BotTrapHit
		if err := rows.Scan(&h.ID, &h.IP, &h.UserAgent, &h.Path, &h.Referer, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// CountBotTrapHits toplam isabet sayısı.
func CountBotTrapHits(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM bot_trap_hits`).Scan(&n)
	return n, err
}

// BotTrap24h son 24 saatteki tuzak isabeti (Faz 37 — rapor).
func BotTrap24h(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM bot_trap_hits WHERE created_at >= now() - interval '24 hours'`).Scan(&n)
	return n, err
}
