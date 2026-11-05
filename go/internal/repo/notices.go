package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Resmi ilanlar / BİK (spec 02 §217) ----

// OfficialNotice resmi ilan kaydı.
type OfficialNotice struct {
	ID          int64
	SiteID      int64
	Title       string
	Institution string
	City        string
	NoticeDate  string
	URL         string
	Body        string
	Sort        int
	// UpdatedAt site haritasındaki lastmod ve detay sayfasındaki
	// dateModified için gerekli (BİK tebliği site haritası maddesi).
	UpdatedAt time.Time
}

// UpsertNotice resmi ilan ekler/günceller (site+başlık+tarih benzersiz).
func UpsertNotice(ctx context.Context, pool *pgxpool.Pool, n *OfficialNotice) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO official_notices (site_id, title, institution, city, notice_date, url, body, sort)
		VALUES ($1,$2,nullif($3,''),nullif($4,''),$5::date,nullif($6,''),nullif($7,''),$8)
		ON CONFLICT (site_id, title, notice_date) DO UPDATE
		  SET institution=EXCLUDED.institution, city=EXCLUDED.city, url=EXCLUDED.url, body=EXCLUDED.body, sort=EXCLUDED.sort, updated_at=now()`,
		n.SiteID, n.Title, n.Institution, n.City, n.NoticeDate, n.URL, n.Body, n.Sort)
	return err
}

// ListNotices verilen tarih (boşsa en son tarih) ilanlarını döndürür.
func ListNotices(ctx context.Context, pool *pgxpool.Pool, siteID int64, date string, limit int) ([]OfficialNotice, string, error) {
	if limit <= 0 {
		limit = 300
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, title, COALESCE(institution,''), COALESCE(city,''), notice_date::text, COALESCE(url,''), COALESCE(body,''), sort
		FROM official_notices
		WHERE site_id=$1 AND notice_date = COALESCE(NULLIF($2,'')::date,
		       (SELECT max(notice_date) FROM official_notices WHERE site_id=$1))
		ORDER BY sort, institution, title LIMIT $3`, siteID, date, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []OfficialNotice
	last := ""
	for rows.Next() {
		var n OfficialNotice
		if err := rows.Scan(&n.ID, &n.SiteID, &n.Title, &n.Institution, &n.City, &n.NoticeDate, &n.URL, &n.Body, &n.Sort); err != nil {
			return nil, "", err
		}
		last = n.NoticeDate
		out = append(out, n)
	}
	return out, last, rows.Err()
}

// NoticeDates mevcut ilan tarihlerini (en yeni önce) döndürür.
func NoticeDates(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT notice_date::text FROM official_notices WHERE site_id=$1 ORDER BY notice_date DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteNotice ilanı siler.
func DeleteNotice(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM official_notices WHERE id=$1`, id)
	return err
}

// CountNotices toplam ilan sayısı.
func CountNotices(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM official_notices WHERE site_id=$1`, siteID).Scan(&n)
	return n, err
}
