package repo

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Akış (feed) modu + kategori sekmeleri (spec 07 önyüz) ----

// MenuCategoryIDs üst menü kategori kimliklerini döndürür (in_header veya in_menu).
func MenuCategoryIDs(ctx context.Context, pool *pgxpool.Pool) ([]int64, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM categories WHERE in_header OR in_menu ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// FeedQuery akış sorgu parametreleri.
type FeedQuery struct {
	SiteID         int64
	CategoryID     int64
	Other          bool   // menü dışı kategoriler
	Day            string // YYYY-MM-DD → tam o gün
	Since          string // YYYY-MM-DD → o günden itibaren (>=)
	Page           int
	PerPage        int
	IncludeGallery bool
	// Types verilirse akış yalnızca bu içerik türlerini döndürür.
	Types []string
}

// FeedPage yayındaki içerikleri akış/ızgara için sayfalar; (içerikler, toplam, hata).
func FeedPage(ctx context.Context, pool *pgxpool.Pool, q FeedQuery) ([]Content, int, error) {
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = 12
	}
	if perPage > 50 {
		perPage = 50
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	n := 1
	where := "c.site_id=$1 AND c.status='yayinda'"
	args := []any{q.SiteID}
	// içerik türü filtresi
	if len(q.Types) > 0 {
		n++
		where += " AND c.type_key = ANY($" + strconv.Itoa(n) + ")"
		args = append(args, q.Types)
	} else if !q.IncludeGallery {
		// galeri içerikleri akışa karışmasın (kendi sayfalarında çıkar)
		where += " AND c.type_key NOT IN ('foto_galeri','video')"
	}
	if q.CategoryID > 0 {
		n++
		where += " AND c.category_id=$" + strconv.Itoa(n)
		args = append(args, q.CategoryID)
	} else if q.Other {
		where += " AND (c.category_id IS NULL OR c.category_id NOT IN (SELECT id FROM categories WHERE in_header OR in_menu))"
	}
	if q.Day != "" {
		n++
		where += " AND c.published_at::date = $" + strconv.Itoa(n) + "::date"
		args = append(args, q.Day)
	} else if q.Since != "" {
		n++
		where += " AND c.published_at::date >= $" + strconv.Itoa(n) + "::date"
		args = append(args, q.Since)
	}
	sel := `SELECT ` + contentCols + `, count(*) OVER() FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE ` + where + `
		ORDER BY c.published_at DESC NULLS LAST, c.id DESC
		LIMIT $` + strconv.Itoa(n+1) + ` OFFSET $` + strconv.Itoa(n+2)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := pool.Query(ctx, sel, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Content
	total := 0
	for rows.Next() {
		c, err := scanContentWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = c.Total
		out = append(out, c.Content)
	}
	return out, total, rows.Err()
}

func FeedByCategories(ctx context.Context, pool *pgxpool.Pool, siteID int64, catIDs []int64, perCat int) (map[int64][]Content, error) {
	out := map[int64][]Content{}
	if len(catIDs) == 0 {
		return out, nil
	}
	if perCat <= 0 {
		perCat = 5
	}
	if perCat > 20 {
		perCat = 20
	}
	sel := `WITH ranked AS (
		SELECT c.id, ROW_NUMBER() OVER (
			PARTITION BY c.category_id
			ORDER BY c.published_at DESC NULLS LAST, c.id DESC
		) AS rn
		FROM contents c
		WHERE c.site_id=$1 AND c.status='yayinda'
		  AND c.category_id = ANY($2)
		  AND c.type_key NOT IN ('foto_galeri','video')
	)
	SELECT ` + contentCols + ` FROM contents c
		JOIN ranked r ON r.id = c.id
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE r.rn <= $3
		ORDER BY c.category_id, r.rn`
	rows, err := pool.Query(ctx, sel, siteID, catIDs, perCat)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return out, err
		}
		if c.CategoryID != nil {
			out[*c.CategoryID] = append(out[*c.CategoryID], c)
		}
	}
	return out, rows.Err()
}
