package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Pano (dashboard) zenginleştirme verileri ----

// AIJobStats AI iş kuyruğunda durum bazlı sayıları döndürür.
func AIJobStats(ctx context.Context, pool *pgxpool.Pool) (map[string]int, time.Time) {
	out := map[string]int{}
	var last time.Time
	rows, err := pool.Query(ctx, `SELECT status, count(*) FROM ai_jobs GROUP BY status`)
	if err != nil {
		return out, last
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err == nil {
			out[st] = n
		}
	}
	_ = pool.QueryRow(ctx, `SELECT COALESCE(max(finished_at), 'epoch') FROM ai_jobs`).Scan(&last)
	return out, last
}

// GazeteLastIssue site için son gazete baskısının tarih/durumunu döndürür.
func GazeteLastIssue(ctx context.Context, pool *pgxpool.Pool, siteID int64) (string, string) {
	var date, status string
	err := pool.QueryRow(ctx, `
		SELECT issue_date::text, status FROM gazete_issues
		WHERE site_id=$1 ORDER BY issue_date DESC LIMIT 1`, siteID).Scan(&date, &status)
	if err != nil {
		return "", ""
	}
	return date, status
}
