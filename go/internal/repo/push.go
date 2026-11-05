package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- P1-8: Web push abonelikleri ----

// PushSub tarayıcı push aboneliği.
type PushSub struct {
	ID       int64
	Endpoint string
	P256dh   string
	Auth     string
	// Faz 43 — yönetim: tarayıcı + son görülme.
	UserAgent string
	LastSeen  time.Time
}

// SavePushSub aboneliği ekler/günceller (endpoint benzersiz).
func SavePushSub(ctx context.Context, pool *pgxpool.Pool, siteID int64, endpoint, p256dh, auth, ua string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO push_subscriptions (site_id, endpoint, p256dh, auth, user_agent)
		VALUES ($1,$2,$3,$4,nullif($5,''))
		ON CONFLICT (endpoint) DO UPDATE
		  SET p256dh=EXCLUDED.p256dh, auth=EXCLUDED.auth, last_seen=now(), site_id=EXCLUDED.site_id`,
		siteID, endpoint, p256dh, auth, ua)
	return err
}

// DeletePushSub endpoint ile aboneliği siler.
func DeletePushSub(ctx context.Context, pool *pgxpool.Pool, endpoint string) error {
	_, err := pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint=$1`, endpoint)
	return err
}

// ListPushSubs siteye ait abonelikleri döndürür.
func ListPushSubs(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]PushSub, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := pool.Query(ctx, `
		SELECT id, endpoint, p256dh, auth, COALESCE(user_agent,''), COALESCE(last_seen, created_at)
		FROM push_subscriptions
		WHERE site_id=$1 ORDER BY created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushSub
	for rows.Next() {
		var s PushSub
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth, &s.UserAgent, &s.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountPushSubs abonelik sayısı.
func CountPushSubs(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM push_subscriptions WHERE site_id=$1`, siteID).Scan(&n)
	return n, err
}
