package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContentLink, editörde iç bağlantı önerisi için hafif içerik kaydıdır.
type ContentLink struct {
	ID     int64
	Title  string
	Slug   string
	Locale string
}

// SearchPublishedContentLinks, yayındaki içeriklerde başlık araması yapar.
func SearchPublishedContentLinks(ctx context.Context, pool *pgxpool.Pool, siteID int64, query string, limit int) ([]ContentLink, error) {
	query = strings.TrimSpace(query)
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	if query == "" {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, title, COALESCE(slug,''), locale
		FROM contents
		WHERE site_id=$1 AND status='yayinda' AND title ILIKE '%' || $2 || '%'
		ORDER BY updated_at DESC, id DESC
		LIMIT $3`, siteID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentLink
	for rows.Next() {
		var c ContentLink
		if err := rows.Scan(&c.ID, &c.Title, &c.Slug, &c.Locale); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
