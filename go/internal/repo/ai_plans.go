package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AICategoryPlan bir kategorinin günlük AI taslak üretim planıdır.
type AICategoryPlan struct {
	ID               int64
	SiteID           int64
	CategoryID       int64
	CategoryName     string
	CategorySlug     string
	DailyTarget      int
	MinSources       int
	TrendWeight      int
	MaxAgeHours      int
	Aktif            bool
	LastRunAt        *time.Time
	DraftedToday     int
	EligibleClusters int
}

const aiPlanSelect = `
	SELECT p.id, p.site_id, p.category_id, c.name, c.slug,
	       p.daily_target, p.min_sources, p.trend_weight, p.max_age_hours, p.aktif, p.last_run_at,
	       (SELECT count(*) FROM ai_story_clusters s
	        WHERE s.site_id=p.site_id AND s.category_id=p.category_id
	          AND s.content_id IS NOT NULL AND s.updated_at >= date_trunc('day', now())) AS drafted_today,
	       (SELECT count(*) FROM ai_story_clusters s
	        WHERE s.site_id=p.site_id AND s.category_id=p.category_id
	          AND s.status='yeni' AND s.source_count >= GREATEST(p.min_sources,1)
	          AND s.created_at > now() - make_interval(hours => GREATEST(p.max_age_hours,1))) AS eligible
	FROM ai_category_plans p
	JOIN categories c ON c.id=p.category_id`

func ListAICategoryPlans(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]AICategoryPlan, error) {
	rows, err := pool.Query(ctx, aiPlanSelect+`
		WHERE p.site_id=$1
		ORDER BY p.daily_target DESC, c.name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AICategoryPlan
	for rows.Next() {
		p, err := scanAIPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func GetAICategoryPlan(ctx context.Context, pool *pgxpool.Pool, siteID, id int64) (*AICategoryPlan, error) {
	p, err := scanAIPlan(pool.QueryRow(ctx, aiPlanSelect+` WHERE p.site_id=$1 AND p.id=$2`, siteID, id))
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanAIPlan(row interface{ Scan(dest ...any) error }) (AICategoryPlan, error) {
	var p AICategoryPlan
	err := row.Scan(&p.ID, &p.SiteID, &p.CategoryID, &p.CategoryName, &p.CategorySlug,
		&p.DailyTarget, &p.MinSources, &p.TrendWeight, &p.MaxAgeHours, &p.Aktif, &p.LastRunAt,
		&p.DraftedToday, &p.EligibleClusters)
	return p, err
}

func UpdateAICategoryPlan(ctx context.Context, pool *pgxpool.Pool, siteID, id int64, target, minSources, trendWeight, maxAge int, aktif bool) error {
	_, err := pool.Exec(ctx, `
		UPDATE ai_category_plans
		SET daily_target=$3, min_sources=$4, trend_weight=$5, max_age_hours=$6, aktif=$7, updated_at=now()
		WHERE site_id=$1 AND id=$2`,
		siteID, id, target, minSources, trendWeight, maxAge, aktif)
	return err
}

func TouchAICategoryPlan(ctx context.Context, pool *pgxpool.Pool, siteID, id int64) {
	_, _ = pool.Exec(ctx, `UPDATE ai_category_plans SET last_run_at=now(), updated_at=now() WHERE site_id=$1 AND id=$2`, siteID, id)
}

// ListAIClustersForPlan plan için uygun, henüz taslağa dönüşmemiş kümeleri
// trend ağırlığına göre sıralar.
func ListAIClustersForPlan(ctx context.Context, pool *pgxpool.Pool, siteID, categoryID int64, minSources, trendWeight, maxAgeHours, limit int) ([]AIStoryCluster, error) {
	if minSources < 1 {
		minSources = 1
	}
	if maxAgeHours < 1 {
		maxAgeHours = 24
	}
	if limit < 1 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, title, summary, source_count, trend_score, score, created_at
		FROM ai_story_clusters
		WHERE site_id=$1 AND category_id=$2 AND status='yeni'
		  AND source_count >= $3
		  AND created_at > now() - make_interval(hours => $4)
		ORDER BY score + (trend_score * $5 / 100) DESC, created_at DESC
		LIMIT $6`, siteID, categoryID, minSources, maxAgeHours, trendWeight, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIStoryCluster
	for rows.Next() {
		var c AIStoryCluster
		if err := rows.Scan(&c.ID, &c.Title, &c.Summary, &c.SourceCount, &c.TrendScore, &c.Score, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
