package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Gazete manşetleri (spec 02 §217) ----

// NewspaperPage bir gazetenin günlük ön sayfa görseli.
type NewspaperPage struct {
	ID        int64
	SiteID    int64
	Name      string
	Slug      string
	PageDate  string // YYYY-MM-DD
	ImageURL  string
	ThumbURL  string
	Sort      int
	CreatedAt string
	UpdatedAt string
}

// UpsertNewspaperPage gazete manşetini ekler/günceller (site+slug+tarih benzersiz).
func UpsertNewspaperPage(ctx context.Context, pool *pgxpool.Pool, p *NewspaperPage) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO newspaper_pages (site_id, name, slug, page_date, image_url, thumb_url, sort)
		VALUES ($1,$2,$3,$4,$5,nullif($6,''),$7)
		ON CONFLICT (site_id, slug, page_date) DO UPDATE
		  SET name=EXCLUDED.name, image_url=EXCLUDED.image_url, thumb_url=EXCLUDED.thumb_url, sort=EXCLUDED.sort, updated_at=now()`,
		p.SiteID, p.Name, p.Slug, p.PageDate, p.ImageURL, p.ThumbURL, p.Sort)
	return err
}

// ListNewspaperPages verilen tarih (boşsa en son tarih) için manşetleri döndürür.
func ListNewspaperPages(ctx context.Context, pool *pgxpool.Pool, siteID int64, date string) ([]NewspaperPage, string, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, name, slug, page_date::text, image_url, COALESCE(thumb_url,''), sort,
		       created_at::text, updated_at::text
		FROM newspaper_pages
		WHERE site_id=$1 AND page_date = COALESCE(NULLIF($2,'')::date,
		       (SELECT max(page_date) FROM newspaper_pages WHERE site_id=$1))
		ORDER BY sort, name`, siteID, date)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []NewspaperPage
	last := ""
	for rows.Next() {
		var p NewspaperPage
		if err := rows.Scan(&p.ID, &p.SiteID, &p.Name, &p.Slug, &p.PageDate, &p.ImageURL, &p.ThumbURL, &p.Sort, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", err
		}
		last = p.PageDate
		out = append(out, p)
	}
	return out, last, rows.Err()
}

// NewspaperDates mevcut manşet tarihlerini (en yeni önce) döndürür.
func NewspaperDates(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT page_date::text FROM newspaper_pages WHERE site_id=$1
		ORDER BY page_date DESC LIMIT $2`, siteID, limit)
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

// DeleteNewspaperPage manşeti siler.
func DeleteNewspaperPage(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM newspaper_pages WHERE id=$1`, id)
	return err
}

// CountNewspaperPages toplam manşet sayısı.
func CountNewspaperPages(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM newspaper_pages WHERE site_id=$1`, siteID).Scan(&n)
	return n, err
}

// PruneNewspaperPages verilen tarihten eski kapakları siler. Gazete manşetleri
// arşiv değeri taşır ama sonsuza kadar büyümemeli; 8 gün yeterli.
func PruneNewspaperPages(ctx context.Context, pool *pgxpool.Pool, siteID int64, once string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM newspaper_pages WHERE site_id=$1 AND page_date < $2::date`, siteID, once)
	return err
}

// LatestNewspaperPages en yeni tarihli kapakları (en fazla limit adet) döndürür.
// Ana sayfa modülü bunu kullanır: istek başına tek sorgu, ağ yok.
func LatestNewspaperPages(ctx context.Context, pool DB, siteID int64, limit int) ([]NewspaperPage, string, error) {
	if limit <= 0 || limit > 60 {
		limit = 12
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, name, slug, page_date::text, image_url, COALESCE(thumb_url,''), sort,
		       created_at::text, updated_at::text
		FROM newspaper_pages
		WHERE site_id=$1
		  AND page_date = (SELECT max(page_date) FROM newspaper_pages WHERE site_id=$1)
		ORDER BY sort, name
		LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []NewspaperPage
	tarih := ""
	for rows.Next() {
		var p NewspaperPage
		if err := rows.Scan(&p.ID, &p.SiteID, &p.Name, &p.Slug, &p.PageDate, &p.ImageURL,
			&p.ThumbURL, &p.Sort, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", err
		}
		tarih = p.PageDate
		out = append(out, p)
	}
	return out, tarih, rows.Err()
}
