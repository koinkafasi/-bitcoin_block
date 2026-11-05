package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetContentExtra tür-özel JSON alanını döndürür (hesaplayıcı şeması vb.).
func GetContentExtra(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]byte, error) {
	var b []byte
	err := pool.QueryRow(ctx, `SELECT extra FROM contents_extra WHERE content_id = $1`, contentID).Scan(&b)
	if err != nil {
		return []byte("{}"), nil
	}
	return b, nil
}

// SetContentExtra tür-özel JSON alanını yazar/upsert eder.
func SetContentExtra(ctx context.Context, pool *pgxpool.Pool, contentID int64, extra []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO contents_extra (content_id, extra) VALUES ($1, $2::jsonb)
		ON CONFLICT (content_id) DO UPDATE SET extra = EXCLUDED.extra`, contentID, extra)
	return err
}
