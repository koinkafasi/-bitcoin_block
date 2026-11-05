package repo

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RevenueRow içerik bazında tahmini gelir/kâr satırıdır.
type RevenueRow struct {
	ID      int64
	Title   string
	Hits    int64
	Revenue float64 // hits * rpm / 1000
	Cost    float64 // AI üretim maliyeti (ai_usage_logs.cost_est toplamı)
	Profit  float64
}

// GetDefaultRPM ayarlardan varsayılan RPM döndürür (yoksa 3.0).
func GetDefaultRPM(ctx context.Context, pool *pgxpool.Pool) float64 {
	if b, err := GetSetting(ctx, pool, "gelir", "varsayilan_rpm"); err == nil {
		if f, e := strconv.ParseFloat(strings.Trim(string(b), `"`), 64); e == nil && f > 0 {
			return f
		}
	}
	return 3.0
}

// SetDefaultRPM varsayılan RPM'i kaydeder.
func SetDefaultRPM(ctx context.Context, pool *pgxpool.Pool, rpm float64) error {
	return SetSetting(ctx, pool, "gelir", "varsayilan_rpm", []byte(strconv.FormatFloat(rpm, 'f', 2, 64)))
}

// ADSupported gerçek AdSense API bağlı mı (şimdilik false; OAuth sonra).
func ADSupported() bool { return false }

// ComputeContentRevenue: tahmini gelir = hit × RPM/1000, kâr = gelir − AI maliyeti.
func ComputeContentRevenue(ctx context.Context, pool *pgxpool.Pool, siteID int64, rpm float64, limit int) ([]RevenueRow, error) {
	if limit <= 0 {
		limit = 15
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, COALESCE(h.total,0),
		       COALESCE((SELECT sum(l.cost_est) FROM ai_usage_logs l WHERE l.content_id = c.id),0)
		FROM contents c
		LEFT JOIN counters_hit h ON h.content_id = c.id
		WHERE c.site_id = $1 AND c.status = 'yayinda'
		ORDER BY COALESCE(h.total,0) DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RevenueRow
	for rows.Next() {
		var r RevenueRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Hits, &r.Cost); err != nil {
			return nil, err
		}
		r.Revenue = float64(r.Hits) * rpm / 1000.0
		r.Profit = r.Revenue - r.Cost
		out = append(out, r)
	}
	return out, rows.Err()
}
