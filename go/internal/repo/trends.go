package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Kategori AI-skor kapıları (spec 06 K-A / 01 §L1) ----

// SetContentAIScore içeriğin AI alaka puanını ayarlar (0-100).
func SetContentAIScore(ctx context.Context, pool *pgxpool.Pool, id int64, score int) error {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	_, err := pool.Exec(ctx, `UPDATE contents SET ai_score = $1 WHERE id = $2`, score, id)
	return err
}

// CategoryGate kategori için yayın kapısını döndürür (min_score <= 0 ise kapı yok).
func CategoryGate(ctx context.Context, pool *pgxpool.Pool, categoryID *int64) (minScore int, name string, err error) {
	if categoryID == nil || *categoryID == 0 {
		return 0, "", nil
	}
	err = pool.QueryRow(ctx, `SELECT min_score, name FROM categories WHERE id = $1`, *categoryID).Scan(&minScore, &name)
	return minScore, name, err
}

// ScoreGate sonucu: içerik yayın kapısından geçer mi?
type ScoreGate struct {
	Allowed  bool
	MinScore int
	HasScore int
	Category string
	Reason   string
}

// CheckScoreGate içerik puanını kategori eşiğiyle karşılaştırır.
func CheckScoreGate(ctx context.Context, pool *pgxpool.Pool, c Content) ScoreGate {
	g := ScoreGate{Allowed: true, HasScore: c.AiScore}
	min, name, err := CategoryGate(ctx, pool, c.CategoryID)
	if err != nil || min <= 0 {
		return g
	}
	g.MinScore = min
	g.Category = name
	if c.AiScore < min {
		g.Allowed = false
		g.Reason = "AI alaka puanı " + itoa(c.AiScore) + " < kategori eşiği " + itoa(min) + " (" + name + ")"
	}
	return g
}

// ---- Kategori trendleri (spec 06 K-B / 01 §L3) ----

// CatTrend günlük kategori trend etiketi.
type CatTrend struct {
	ID         int64
	SiteID     int64
	CategoryID *int64
	Category   string
	Tag        string
	PostCount  int
	TrendDate  time.Time
	Lang       string
	Source     string
}

// UpsertCatTrend günlük trend etiketini ekler/günceller.
func UpsertCatTrend(ctx context.Context, pool *pgxpool.Pool, t CatTrend) error {
	var cat any
	if t.CategoryID != nil {
		cat = *t.CategoryID
	}
	var day any
	if !t.TrendDate.IsZero() {
		day = t.TrendDate
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO trends (site_id, category_id, tag, post_count, trend_date, lang, source)
		VALUES ($1,$2,$3,$4,COALESCE($5::date, CURRENT_DATE),$6,$7)
		ON CONFLICT (site_id, category_id, tag, trend_date, lang)
		DO UPDATE SET post_count = EXCLUDED.post_count, source = EXCLUDED.source`,
		t.SiteID, cat, t.Tag, t.PostCount, day, t.Lang, t.Source)
	return err
}

// ListCatTrends son N günün kategori trendlerini döndürür.
func ListCatTrends(ctx context.Context, pool *pgxpool.Pool, siteID int64, lang string, days, limit int) ([]CatTrend, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.site_id, t.category_id, COALESCE(c.name,''), t.tag, t.post_count, t.trend_date, t.lang, t.source
		FROM trends t LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.site_id = $1 AND t.lang = $2 AND t.trend_date >= CURRENT_DATE - ($3::int)
		ORDER BY t.post_count DESC, t.tag LIMIT $4`, siteID, lang, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatTrend
	for rows.Next() {
		var t CatTrend
		if err := rows.Scan(&t.ID, &t.SiteID, &t.CategoryID, &t.Category, &t.Tag, &t.PostCount, &t.TrendDate, &t.Lang, &t.Source); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CategoryTagCounts son N gündeki içeriklerde en çok geçen etiketleri kategori bazında sayar.
func CategoryTagCounts(ctx context.Context, pool *pgxpool.Pool, siteID int64, lang string, days, limit int) ([]CatTrend, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.category_id, COALESCE(cat.name,''), t.name, count(*)::int
		FROM content_tags ct
		JOIN contents c ON c.id = ct.content_id
		LEFT JOIN categories cat ON cat.id = c.category_id
		JOIN tags t ON t.id = ct.tag_id
		WHERE c.site_id = $1 AND c.locale = $2 AND c.status = 'yayinda'
		  AND c.published_at >= now() - ($3::int || ' days')::interval
		GROUP BY c.category_id, cat.name, t.name
		ORDER BY count(*) DESC LIMIT $4`, siteID, lang, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatTrend
	for rows.Next() {
		var t CatTrend
		if err := rows.Scan(&t.CategoryID, &t.Category, &t.Tag, &t.PostCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
