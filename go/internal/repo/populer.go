package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func PopulerIcerikler(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, adet int) ([]Content, error) {
	if adet <= 0 {
		adet = 6
	}
	rows, err := pool.Query(ctx, `
		SELECT `+contentCols+`
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		JOIN LATERAL (
			SELECT COALESCE(SUM(h.hits), 0) AS gunluk
			FROM hit_daily h
			WHERE h.content_id = c.id AND h.day >= CURRENT_DATE - INTERVAL '7 days'
		) hd ON TRUE
		LEFT JOIN counters_hit ch ON ch.content_id = c.id
		WHERE c.status = 'yayinda' AND c.site_id = $1 AND c.locale = $2
		ORDER BY hd.gunluk DESC, COALESCE(ch.total, 0) DESC, c.published_at DESC NULLS LAST
		LIMIT $3`, siteID, locale, adet)
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

func BolumIcerikleri(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale, slug string, adet int) ([]Content, error) {
	if adet <= 0 {
		adet = 6
	}
	rows, err := pool.Query(ctx, `
		SELECT `+contentCols+`
		FROM contents c
		JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status = 'yayinda' AND c.site_id = $1 AND c.locale = $2 AND cat.slug = $3
		ORDER BY c.published_at DESC NULLS LAST, c.id DESC
		LIMIT $4`, siteID, locale, slug, adet)
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
