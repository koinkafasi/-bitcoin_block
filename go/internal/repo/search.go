package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchPublic genel sitede yayındaki içerikte arar (ILIKE + pg_trgm).
func SearchPublic(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale, q string, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 30
	}
	q = strings.ReplaceAll(strings.ReplaceAll(q, "%", ""), "_", "")
	like := "%" + strings.TrimSpace(q) + "%"
	rows, err := pool.Query(ctx, `
		SELECT `+contentCols+` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status='yayinda' AND c.site_id=$1 AND c.locale=$2
		  AND (c.title ILIKE $3 OR c.spot ILIKE $3 OR c.detail::text ILIKE $3)
		ORDER BY c.published_at DESC LIMIT $4`,
		siteID, locale, like, limit)
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
