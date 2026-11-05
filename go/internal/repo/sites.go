package repo

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Site yayın sitesi (tenant).
type Site struct {
	ID            int64
	Name          string
	Domain        string
	Brand         *string
	DefaultLocale string
	Status        string
	ContentCount  int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ListSites siteleri içerik sayısıyla döndürür.
func ListSites(ctx context.Context, pool *pgxpool.Pool) ([]Site, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.name, s.domain, s.brand, s.default_locale, s.status, s.created_at, s.updated_at,
		       (SELECT count(*) FROM contents c WHERE c.site_id = s.id) AS cc
		FROM sites s ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.Name, &s.Domain, &s.Brand, &s.DefaultLocale, &s.Status, &s.CreatedAt, &s.UpdatedAt, &s.ContentCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSiteByDomain host'tan site çözer; yoksa ErrNotFound.
func GetSiteByDomain(ctx context.Context, pool *pgxpool.Pool, domain string) (Site, error) {
	return siteQuery(ctx, pool, `WHERE s.domain = $1 AND s.status = 'aktif'`, domain)
}

// GetSite id ile site döndürür.
func GetSite(ctx context.Context, pool *pgxpool.Pool, id int64) (Site, error) {
	return siteQuery(ctx, pool, `WHERE s.id = $1`, id)
}

func siteQuery(ctx context.Context, pool *pgxpool.Pool, whereClause string, arg any) (Site, error) {
	var s Site
	err := pool.QueryRow(ctx, `
		SELECT s.id, s.name, s.domain, s.brand, s.default_locale, s.status, s.created_at, s.updated_at,
		       (SELECT count(*) FROM contents c WHERE c.site_id = s.id)
		FROM sites s `+whereClause, arg).
		Scan(&s.ID, &s.Name, &s.Domain, &s.Brand, &s.DefaultLocale, &s.Status, &s.CreatedAt, &s.UpdatedAt, &s.ContentCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	return s, err
}

// CreateSite yeni site ekler.
func CreateSite(ctx context.Context, pool *pgxpool.Pool, name, domain, brand, locale string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO sites (name, domain, brand, default_locale) VALUES ($1,$2,$3,$4) RETURNING id`,
		name, domain, strPtrOrNil(brand), defLocale(locale)).Scan(&id)
	return id, err
}

// UpdateSite site günceller.
func UpdateSite(ctx context.Context, pool *pgxpool.Pool, s *Site) error {
	_, err := pool.Exec(ctx, `
		UPDATE sites SET name=$1, domain=$2, brand=$3, default_locale=$4, status=$5 WHERE id=$6`,
		s.Name, s.Domain, s.Brand, defLocale(s.DefaultLocale), s.Status, s.ID)
	return err
}

// DeleteSite site siler (site 1 ve içerikli siteler engellenir).
func DeleteSite(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	tag, err := pool.Exec(ctx, `
		DELETE FROM sites WHERE id = $1 AND id <> 1 AND NOT EXISTS (SELECT 1 FROM contents WHERE site_id = $1)`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func defLocale(l string) string {
	if l == "" || utf8.RuneCountInString(l) > 8 {
		return "tr"
	}
	return l
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
