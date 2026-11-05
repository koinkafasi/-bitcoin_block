package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Önerilen Video (Faz 12b) ----

// RelatedVideoID makaleye elle bağlanan video içerik id'sini döndürür (0 = yok).
func RelatedVideoID(ctx context.Context, pool *pgxpool.Pool, contentID int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT related_video_id FROM contents WHERE id=$1`, contentID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// SetRelatedVideoID makalenin elle seçilen "Önerilen Video" içeriğini bağlar.
func SetRelatedVideoID(ctx context.Context, pool *pgxpool.Pool, contentID, videoID int64) error {
	var v any
	if videoID <= 0 {
		v = nil
	} else {
		v = videoID
	}
	_, err := pool.Exec(ctx, `UPDATE contents SET related_video_id=$1 WHERE id=$2`, v, contentID)
	return err
}
