package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func HitsPerDay(ctx context.Context, pool *pgxpool.Pool, siteID int64, days int) []DayCount {
	if days <= 0 {
		days = 14
	}
	rows, err := pool.Query(ctx, `
		SELECT g::date::text AS day, COALESCE(sum(h.hits), 0)::int
		FROM generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day') g
		LEFT JOIN hit_daily h ON h.day = g::date
		LEFT JOIN contents c ON c.id = h.content_id AND c.site_id = $1
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

type CatSlice struct {
	Name  string
	Count int
}

func CategoryShare(ctx context.Context, pool *pgxpool.Pool, siteID int64, top int) []CatSlice {
	if top <= 0 {
		top = 6
	}
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(cat.name, 'Kategorisiz') AS ad, count(*)::int
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		WHERE c.site_id = $1 AND c.status = 'yayinda'
		GROUP BY 1 ORDER BY 2 DESC`, siteID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []CatSlice
	for rows.Next() {
		var s CatSlice
		if err := rows.Scan(&s.Name, &s.Count); err != nil {
			return out
		}
		out = append(out, s)
	}
	if len(out) <= top {
		return out
	}
	// İlk (top-1) dilim korunur, gerisi "Diğer" olur: halkada okunabilir
	// sayıda dilim kalsın.
	rest := 0
	for _, s := range out[top-1:] {
		rest += s.Count
	}
	head := append([]CatSlice{}, out[:top-1]...)
	return append(head, CatSlice{Name: "Diğer", Count: rest})
}

// PublishPerMonth son N ayın yayın sayısı (eskiden yeniye).
func PublishPerMonth(ctx context.Context, pool *pgxpool.Pool, siteID int64, months int) []DayCount {
	if months <= 0 {
		months = 6
	}
	rows, err := pool.Query(ctx, `
		SELECT to_char(g, 'YYYY-MM') AS ay, COALESCE(count(c.id), 0)::int
		FROM generate_series(date_trunc('month', CURRENT_DATE) - (($2::int - 1) || ' month')::interval,
		                     date_trunc('month', CURRENT_DATE), '1 month') g
		LEFT JOIN contents c ON date_trunc('month', c.published_at) = g
		                    AND c.site_id = $1 AND c.status = 'yayinda'
		GROUP BY g ORDER BY g`, siteID, months)
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

// PlanRow yayın şeridindeki (sürüklenebilir) tek içerik.
type PlanRow struct {
	ID       int64
	Title    string
	When     time.Time
	Status   string
	Category string
	TypeKey  string
}

func PlannedUpcoming(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) []PlanRow {
	if limit <= 0 || limit > 60 {
		limit = 24
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, c.published_at, c.status, COALESCE(cat.name, ''), COALESCE(c.type_key, '')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		WHERE c.site_id = $1
		  AND c.published_at IS NOT NULL
		  AND c.published_at >= now() - interval '6 hours'
		  AND c.published_at <  now() + interval '36 hours'
		  AND c.status IN ('yayinda', 'planlanmis', 'taslak')
		ORDER BY c.published_at LIMIT $2`, siteID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []PlanRow
	for rows.Next() {
		var p PlanRow
		if err := rows.Scan(&p.ID, &p.Title, &p.When, &p.Status, &p.Category, &p.TypeKey); err != nil {
			return out
		}
		out = append(out, p)
	}
	return out
}

func RescheduleContent(ctx context.Context, pool *pgxpool.Pool, siteID, id int64, when time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE contents SET published_at = $3, updated_at = now()
		WHERE id = $1 AND site_id = $2`, id, siteID, when)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// HitTotals son N gün ile ondan önceki N günün toplamını verir.
// KPI döşemelerindeki "↑ %x" rozetini besler; yüzde hesabı çağıranda.
func HitTotals(ctx context.Context, pool *pgxpool.Pool, siteID int64, days int) (son, onceki int64) {
	if days <= 0 {
		days = 7
	}
	_ = pool.QueryRow(ctx, `
		SELECT
		  COALESCE(sum(hits) FILTER (WHERE day > CURRENT_DATE - $2::int), 0),
		  COALESCE(sum(hits) FILTER (WHERE day <= CURRENT_DATE - $2::int), 0)
		FROM hit_daily h JOIN contents c ON c.id = h.content_id AND c.site_id = $1
		WHERE day > CURRENT_DATE - (2 * $2::int)`, siteID, days).Scan(&son, &onceki)
	return son, onceki
}

// PublishTotals bu ay ile geçen ayın yayın sayısı.
func PublishTotals(ctx context.Context, pool *pgxpool.Pool, siteID int64) (bu, gecen int64) {
	_ = pool.QueryRow(ctx, `
		SELECT
		  COALESCE(count(*) FILTER (WHERE date_trunc('month', published_at) = date_trunc('month', CURRENT_DATE)), 0),
		  COALESCE(count(*) FILTER (WHERE date_trunc('month', published_at) = date_trunc('month', CURRENT_DATE) - interval '1 month'), 0)
		FROM contents
		WHERE site_id = $1 AND status = 'yayinda'`, siteID).Scan(&bu, &gecen)
	return bu, gecen
}
