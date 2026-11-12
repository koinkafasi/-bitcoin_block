package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RecentPublishedForCompare(ctx context.Context, pool *pgxpool.Pool, siteID, excludeID int64, limit int) ([]Content, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id = $1 AND c.status = 'yayinda' AND c.id <> $2
		ORDER BY COALESCE(c.published_at, c.updated_at) DESC
		LIMIT $3`
	rows, err := pool.Query(ctx, q, siteIDOr1(siteID), excludeID, limit)
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
