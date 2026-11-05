package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- P4-7 sosyal paylaşım + P4-6 indeksleme kayıtları ----

// ShareLog bir paylaşım denemesi.
type ShareLog struct {
	ID        int64
	ContentID *int64
	Platform  string
	Status    string
	Detail    string
	CreatedAt string
}

// LogShare paylaşım denemesini kaydeder.
func LogShare(ctx context.Context, pool *pgxpool.Pool, siteID int64, contentID *int64, platform, status, detail string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO share_log (site_id, content_id, platform, status, detail)
		VALUES ($1,$2,$3,$4,nullif($5,''))`, siteID, contentID, platform, status, detail)
	return err
}

// ListShares son paylaşımları döndürür.
func ListShares(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]ShareLog, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT id, content_id, platform, status, COALESCE(detail,''), created_at::text
		FROM share_log WHERE site_id=$1 ORDER BY created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ShareLog
	for rows.Next() {
		var s ShareLog
		if err := rows.Scan(&s.ID, &s.ContentID, &s.Platform, &s.Status, &s.Detail, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// IndexLog bir indeksleme bildirimi.
type IndexLog struct {
	ID        int64
	Engine    string
	URL       string
	Status    string
	Detail    string
	CreatedAt string
}

// LogIndexing indeksleme bildirimini kaydeder.
func LogIndexing(ctx context.Context, pool *pgxpool.Pool, siteID int64, engine, url, status, detail string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO indexing_log (site_id, engine, url, status, detail)
		VALUES ($1,$2,$3,$4,nullif($5,''))`, siteID, engine, url, status, detail)
	return err
}

// ListIndexing son indeksleme bildirimlerini döndürür.
func ListIndexing(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]IndexLog, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT id, engine, url, status, COALESCE(detail,''), created_at::text
		FROM indexing_log WHERE site_id=$1 ORDER BY created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndexLog
	for rows.Next() {
		var l IndexLog
		if err := rows.Scan(&l.ID, &l.Engine, &l.URL, &l.Status, &l.Detail, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
