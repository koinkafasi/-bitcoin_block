package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- D9: Site-içi backlink (iç link) ağı ----

// RelatedContents bir içerik için küme/etiket/kategori/başlık temelli iç link önerilerini
// içerik kayıtları olarak döndürür (P5-14 puanlaması yeniden kullanılır).
func RelatedContents(ctx context.Context, pool *pgxpool.Pool, siteID, contentID int64, limit int) ([]Content, error) {
	sugs, err := LinkSuggestions(ctx, pool, siteID, contentID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Content, 0, len(sugs))
	for _, s := range sugs {
		if c, err := GetContent(ctx, pool, s.ID); err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

// LinkNetworkStats backlink ağının sağlık özetini döndürür.
type LinkNetworkStats struct {
	Published   int64
	WithCluster int64
	Orphan      int64
}

// NetworkStats yayındaki içerik, küme bağlı içerik ve öksüz (kategorisiz+etiketiz+kümesiz) sayıları.
func NetworkStats(ctx context.Context, pool *pgxpool.Pool, siteID int64) (LinkNetworkStats, error) {
	var s LinkNetworkStats
	err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM contents WHERE site_id=$1 AND status='yayinda'),
		  (SELECT count(*) FROM contents c WHERE c.site_id=$1 AND c.status='yayinda'
		     AND EXISTS (SELECT 1 FROM content_clusters cc WHERE cc.content_id=c.id)),
		  (SELECT count(*) FROM contents c WHERE c.site_id=$1 AND c.status='yayinda'
		     AND c.category_id IS NULL
		     AND NOT EXISTS (SELECT 1 FROM content_clusters cc WHERE cc.content_id=c.id)
		     AND NOT EXISTS (SELECT 1 FROM content_tags ct WHERE ct.content_id=c.id))`,
		siteID).Scan(&s.Published, &s.WithCluster, &s.Orphan)
	return s, err
}

// OrphanContents yayındaki ama kategori/etiket/küme bağı olmayan (iç link ağına girmeyen) içerikler.
func OrphanContents(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id=$1 AND c.status='yayinda'
		  AND c.category_id IS NULL
		  AND NOT EXISTS (SELECT 1 FROM content_clusters cc WHERE cc.content_id=c.id)
		  AND NOT EXISTS (SELECT 1 FROM content_tags ct WHERE ct.content_id=c.id)
		ORDER BY c.updated_at DESC LIMIT $2`
	rows, err := pool.Query(ctx, q, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
