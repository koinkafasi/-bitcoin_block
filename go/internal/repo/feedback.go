package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Okuyucu geri bildirimi / içerik güvenliği (spec 06 L12) ----

// Feedback okuyucu bildirim kaydıdır.
type Feedback struct {
	ID         int64
	ContentID  int64
	SiteID     int64
	Kind       string
	Detail     string
	Reporter   string
	Status     string
	Resolution string
	HandledBy  *int64
	HandledAt  *time.Time
	CreatedAt  time.Time
	Title      string // birleştirilmiş görünüm
	Slug       string
}

// AddFeedback okuyucu bildirimi ekler.
func AddFeedback(ctx context.Context, pool *pgxpool.Pool, contentID, siteID int64, kind, detail, reporter string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO content_feedback (content_id, site_id, kind, detail, reporter)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		contentID, siteID, kind, detail, reporter).Scan(&id)
	return id, err
}

// ListFeedback duruma göre bildirim listesi (en yeni önce).
func ListFeedback(ctx context.Context, pool *pgxpool.Pool, siteID int64, status string, limit int) ([]Feedback, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `SELECT f.id, f.content_id, f.site_id, f.kind, COALESCE(f.detail,''), COALESCE(f.reporter,''),
		         f.status, COALESCE(f.resolution,''), f.handled_by, f.handled_at, f.created_at,
		         c.title, COALESCE(c.slug,'')
		FROM content_feedback f JOIN contents c ON c.id = f.content_id
		WHERE f.site_id = $1`
	args := []any{siteID}
	if status != "" && status != "tumu" {
		q += ` AND f.status = $2`
		args = append(args, status)
	}
	q += ` ORDER BY f.created_at DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Feedback
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(&f.ID, &f.ContentID, &f.SiteID, &f.Kind, &f.Detail, &f.Reporter, &f.Status,
			&f.Resolution, &f.HandledBy, &f.HandledAt, &f.CreatedAt, &f.Title, &f.Slug); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ResolveFeedback bildirimi sonuçlandırır.
func ResolveFeedback(ctx context.Context, pool *pgxpool.Pool, id, userID int64, status, resolution string) error {
	_, err := pool.Exec(ctx, `UPDATE content_feedback
		SET status=$1, resolution=$2, handled_by=$3, handled_at=now() WHERE id=$4`,
		status, resolution, userID, id)
	return err
}

// CountFeedback durum sayaçları.
func CountFeedback(ctx context.Context, pool *pgxpool.Pool, siteID int64) map[string]int {
	out := map[string]int{}
	rows, err := pool.Query(ctx, `SELECT status, count(*) FROM content_feedback WHERE site_id=$1 GROUP BY status`, siteID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var n int
		if rows.Scan(&s, &n) == nil {
			out[s] = n
		}
	}
	return out
}

// ---- AI asistan (spec 06 L12) ----

// AssistantLog asistan sorgu kaydıdır.
type AssistantLog struct {
	ID        int64
	SiteID    int64
	ContentID *int64
	Question  string
	Answer    string
	Provider  string
	UserID    *int64
	Locale    string
	CreatedAt time.Time
}

// AddAssistantLog asistan yanıtını loglar (gizlilik için soru/yanıt saklanır).
func AddAssistantLog(ctx context.Context, pool *pgxpool.Pool, l AssistantLog) error {
	_, err := pool.Exec(ctx, `INSERT INTO assistant_logs (site_id, content_id, question, answer, provider, user_id, locale)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		l.SiteID, l.ContentID, l.Question, l.Answer, l.Provider, l.UserID, l.Locale)
	return err
}

// ListAssistantLogs son asistan sorgularını döndürür (panel istatistiği).
func ListAssistantLogs(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]AssistantLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `SELECT id, site_id, content_id, question, COALESCE(answer,''),
		COALESCE(provider,''), user_id, locale, created_at
		FROM assistant_logs WHERE site_id=$1 ORDER BY created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssistantLog
	for rows.Next() {
		var l AssistantLog
		if err := rows.Scan(&l.ID, &l.SiteID, &l.ContentID, &l.Question, &l.Answer, &l.Provider,
			&l.UserID, &l.Locale, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CountAssistantLogs asistan sorgu sayısı.
func CountAssistantLogs(ctx context.Context, pool *pgxpool.Pool, siteID int64) int {
	var n int
	if pool.QueryRow(ctx, `SELECT count(*) FROM assistant_logs WHERE site_id=$1`, siteID).Scan(&n) != nil {
		return 0
	}
	return n
}
