package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TagIcerikleri bir etiketin yayındaki içeriklerini sayfalı döndürür.
func TagIcerikleri(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale, tagSlug string, sayfa, perPage int) ([]Content, int, error) {
	if sayfa < 1 {
		sayfa = 1
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 24
	}
	rows, err := pool.Query(ctx, `
		SELECT `+contentCols+`
		FROM contents c
		JOIN content_tags ct ON ct.content_id = c.id
		JOIN tags t ON t.id = ct.tag_id
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status = 'yayinda' AND c.site_id = $1 AND c.locale = $2 AND t.slug = $3
		ORDER BY c.published_at DESC NULLS LAST, c.id DESC
		LIMIT $4 OFFSET $5`,
		siteID, locale, tagSlug, perPage, (sayfa-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var toplam int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM contents c
		JOIN content_tags ct ON ct.content_id = c.id
		JOIN tags t ON t.id = ct.tag_id
		WHERE c.status = 'yayinda' AND c.site_id = $1 AND c.locale = $2 AND t.slug = $3`,
		siteID, locale, tagSlug).Scan(&toplam)
	return out, toplam, err
}
