package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Panel (dashboard) metrikleri ----

// SiteHits siteye ait toplam okunma (counters_hit).
func SiteHits(ctx context.Context, pool *pgxpool.Pool, siteID int64) int64 {
	var n int64
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(sum(ch.total),0) FROM counters_hit ch
		JOIN contents c ON c.id = ch.content_id WHERE c.site_id = $1`, siteID).Scan(&n)
	return n
}

// DayCount günlük sayım.
type DayCount struct {
	Day   string
	Count int
}

// PublishPerDay son N günün yayın sayısını döndürür (eskiden yeniye).
func PublishPerDay(ctx context.Context, pool *pgxpool.Pool, siteID int64, days int) []DayCount {
	if days <= 0 {
		days = 7
	}
	rows, err := pool.Query(ctx, `
		SELECT g::date::text AS day, COALESCE(count(c.id),0)::int
		FROM generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day') g
		LEFT JOIN contents c ON c.published_at::date = g::date AND c.site_id = $1 AND c.status = 'yayinda'
		GROUP BY g ORDER BY g`, siteID, days)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []DayCount
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return out
		}
		out = append(out, d)
	}
	return out
}

// TopRead siteye ait en çok okunan yayındaki içerikler.
func TopRead(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 5
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		JOIN counters_hit ch ON ch.content_id = c.id
		WHERE c.site_id = $1 AND c.status = 'yayinda'
		ORDER BY ch.total DESC LIMIT $2`
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

// HitsOf içeriklerin okunma sayılarını (id→total) döndürür.
func HitsOf(ctx context.Context, pool *pgxpool.Pool, ids []int64) map[int64]int64 {
	m := map[int64]int64{}
	if len(ids) == 0 {
		return m
	}
	rows, err := pool.Query(ctx, `SELECT content_id, total FROM counters_hit WHERE content_id = ANY($1)`, ids)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id, n int64
		if err := rows.Scan(&id, &n); err == nil {
			m[id] = n
		}
	}
	return m
}
