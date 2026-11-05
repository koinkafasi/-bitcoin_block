package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetContentReaction bir ziyaretçinin içerik tepkisini kaydeder.
// reaction: 1 = buna benzer daha fazla, -1 = buna benzer daha az,
// 0 = tepkiyi kaldır.
func SetContentReaction(ctx context.Context, pool *pgxpool.Pool, contentID int64, visitor string, reaction int) error {
	if reaction != 1 && reaction != -1 {
		_, err := pool.Exec(ctx, `DELETE FROM content_reactions WHERE content_id=$1 AND visitor=$2`, contentID, visitor)
		return err
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO content_reactions (content_id, visitor, reaction)
		VALUES ($1,$2,$3)
		ON CONFLICT (content_id, visitor)
		DO UPDATE SET reaction=EXCLUDED.reaction, created_at=now()`,
		contentID, visitor, reaction)
	return err
}
