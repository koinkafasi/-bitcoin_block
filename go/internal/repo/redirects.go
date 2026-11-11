package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- 301 yönlendirme yönetimi (spec 01 §D7/J2, 03 P4-9) ----

// Redirect eski → yeni URL kalıcı yönlendirmesi.
type Redirect struct {
	SiteID     int64
	SourcePath string
	TargetURL  string
	StatusCode int
	Hits       int64
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ListRedirects siteye ait yönlendirmeleri (isteğe bağlı arama ile) döndürür.
func ListRedirects(ctx context.Context, pool *pgxpool.Pool, siteID int64, q string, limit int) ([]Redirect, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
		SELECT site_id, source_path, target_url, status_code, hits, COALESCE(note,''), created_at, updated_at
		FROM redirects
		WHERE site_id=$1 AND ($2='' OR source_path ILIKE '%'||$2||'%' OR target_url ILIKE '%'||$2||'%')
		ORDER BY updated_at DESC, source_path
		LIMIT $3`, siteID, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Redirect
	for rows.Next() {
		var r Redirect
		if err := rows.Scan(&r.SiteID, &r.SourcePath, &r.TargetURL, &r.StatusCode, &r.Hits, &r.Note, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountRedirects siteye ait toplam yönlendirme sayısı.
func CountRedirects(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM redirects WHERE site_id=$1`, siteID).Scan(&n)
	return n, err
}

// UpsertRedirect yönlendirmeyi ekler veya günceller (site_id+source_path üzerinde).
func UpsertRedirect(ctx context.Context, pool *pgxpool.Pool, r *Redirect) error {
	if r.StatusCode == 0 {
		r.StatusCode = 301
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO redirects (site_id, source_path, target_url, status_code, note, updated_at)
		VALUES ($1,$2,$3,$4,nullif($5,''),now())
		ON CONFLICT (site_id, source_path) DO UPDATE
		  SET target_url=EXCLUDED.target_url, status_code=EXCLUDED.status_code,
		      note=EXCLUDED.note, updated_at=now()`,
		r.SiteID, r.SourcePath, r.TargetURL, r.StatusCode, r.Note)
	return err
}

// DeleteRedirect yönlendirmeyi siler.
func DeleteRedirect(ctx context.Context, pool *pgxpool.Pool, siteID int64, sourcePath string) error {
	_, err := pool.Exec(ctx, `DELETE FROM redirects WHERE site_id=$1 AND source_path=$2`, siteID, sourcePath)
	return err
}

// BulkUpsertRedirects toplu içe aktarım; eklenen/güncellenen sayısını döndürür.
func BulkUpsertRedirects(ctx context.Context, pool *pgxpool.Pool, list []Redirect) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	n := 0
	for _, r := range list {
		if r.StatusCode == 0 {
			r.StatusCode = 301
		}
		ct, err := tx.Exec(ctx, `
			INSERT INTO redirects (site_id, source_path, target_url, status_code, note, updated_at)
			VALUES ($1,$2,$3,$4,nullif($5,''),now())
			ON CONFLICT (site_id, source_path) DO UPDATE
			  SET target_url=EXCLUDED.target_url, status_code=EXCLUDED.status_code,
			      note=EXCLUDED.note, updated_at=now()`,
			r.SiteID, r.SourcePath, r.TargetURL, r.StatusCode, r.Note)
		if err != nil {
			return n, err
		}
		n += int(ct.RowsAffected())
	}
	return n, tx.Commit(ctx)
}

// IncrementRedirectHit bir yönlendirmenin hit sayacını artırır.
func IncrementRedirectHit(ctx context.Context, pool *pgxpool.Pool, siteID int64, sourcePath string) error {
	_, err := pool.Exec(ctx, `UPDATE redirects SET hits = hits + 1 WHERE site_id=$1 AND source_path=$2`, siteID, sourcePath)
	return err
}

// LoadRedirectMap siteye ait tüm yönlendirmeleri source_path → Redirect eşlemesi olarak yükler.
func LoadRedirectMap(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]Redirect, error) {
	rows, err := pool.Query(ctx, `
		SELECT site_id, source_path, target_url, status_code, hits, COALESCE(note,''), created_at, updated_at
		FROM redirects WHERE site_id=$1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]Redirect)
	for rows.Next() {
		var r Redirect
		if err := rows.Scan(&r.SiteID, &r.SourcePath, &r.TargetURL, &r.StatusCode, &r.Hits, &r.Note, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		m[r.SourcePath] = r
	}
	return m, rows.Err()
}
