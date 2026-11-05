package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Bloklu IP katmanı (bot tuzağı → otomatik blok) ----

// BlockedIP bloklu IP kaydı.
type BlockedIP struct {
	IP        string
	Reason    string
	Hits      int
	CreatedAt string
}

// BlockIP bir IP'yi bloklar (varsa hits artırır).
func BlockIP(ctx context.Context, pool *pgxpool.Pool, ip, reason string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO blocked_ips (ip, reason) VALUES ($1, nullif($2,''))
		ON CONFLICT (ip) DO UPDATE SET hits = blocked_ips.hits + 1, reason = COALESCE(EXCLUDED.reason, blocked_ips.reason), updated_at=now()`,
		ip, reason)
	return err
}

// UnblockIP bloğu kaldırır.
func UnblockIP(ctx context.Context, pool *pgxpool.Pool, ip string) error {
	_, err := pool.Exec(ctx, `DELETE FROM blocked_ips WHERE ip=$1`, ip)
	return err
}

// ListBlockedIPs bloklu IP'leri döndürür.
func ListBlockedIPs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]BlockedIP, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
		SELECT ip, COALESCE(reason,''), hits, created_at::text
		FROM blocked_ips ORDER BY updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BlockedIP
	for rows.Next() {
		var b BlockedIP
		if err := rows.Scan(&b.IP, &b.Reason, &b.Hits, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BlockedIPSet bloklu IP'leri hızlı arama için kümeye yükler.
func BlockedIPSet(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT ip FROM blocked_ips`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		m[ip] = true
	}
	return m, rows.Err()
}

// CountBlockedIPs bloklu IP sayısı.
func CountBlockedIPs(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM blocked_ips`).Scan(&n)
	return n, err
}
