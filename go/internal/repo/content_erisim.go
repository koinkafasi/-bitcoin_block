package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- İçerik erişim kademesi (toplu ayar) ----
//
// NEDEN TOPLU: kilitli içerikler elle tek tek işaretlenmişti ve aralarında
// rastgele kalanlar vardı (SEO için açık olması gereken evergreen rehber
// bile kilitliydi). Erişim politikası içerik içerik değil, küme küme
// verilen bir karar; panelde toplu düğme olmadan tutarlı tutulamıyor.

// SetContentErisim içeriğin erişim kademesini ayarlar.
//
// agirlik 0  → herkese açık (kilit tamamen kalkar)
// agirlik >0 → o kademe ve üstü abonelere özel
func SetContentErisim(ctx context.Context, pool *pgxpool.Pool, contentID int64, agirlik int) error {
	if agirlik < 0 {
		agirlik = 0
	}
	if agirlik > 5 {
		agirlik = 5
	}
	_, err := pool.Exec(ctx, `
		UPDATE contents
		   SET min_tier_weight = $2,
		       is_exclusive    = ($2 > 0),
		       updated_at      = now()
		 WHERE id = $1`, contentID, agirlik)
	return err
}
