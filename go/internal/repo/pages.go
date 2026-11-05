package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Page kurumsal/genel sayfa kaydıdır.
type Page struct {
	ID        int64
	Kind      string
	Title     string
	Slug      string
	CustomURL *string
	Spot      *string
	Detail    []byte
	Status    string
	InMenu    bool
	InFooter  bool
	IsLocked  bool
	CreatedAt time.Time
}

// GetPageBySlug sayfayı döndürür (yalnız yayındakiler genel sitede gösterilir).
func GetPageBySlug(ctx context.Context, pool *pgxpool.Pool, slug string, onlyPublished bool) (Page, error) {
	q := `SELECT id, kind, title, slug, custom_url, spot, detail, status, in_menu, in_footer, is_locked, created_at
	      FROM pages WHERE slug = $1`
	if onlyPublished {
		q += ` AND status = 'yayinda'`
	}
	var p Page
	err := pool.QueryRow(ctx, q, slug).Scan(&p.ID, &p.Kind, &p.Title, &p.Slug, &p.CustomURL,
		&p.Spot, &p.Detail, &p.Status, &p.InMenu, &p.InFooter, &p.IsLocked, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Page{}, ErrNotFound
	}
	return p, err
}

// ListPages menü/footer için yayındaki sayfaları döndürür.
func ListPages(ctx context.Context, pool *pgxpool.Pool) ([]Page, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, kind, title, slug, custom_url, spot, detail, status, in_menu, in_footer, is_locked, created_at
		FROM pages WHERE status = 'yayinda' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.Kind, &p.Title, &p.Slug, &p.CustomURL,
			&p.Spot, &p.Detail, &p.Status, &p.InMenu, &p.InFooter, &p.IsLocked, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
