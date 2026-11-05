package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContentAfterBuSitede yayın sırasına göre verilen zamanın hemen SONRAKİ içeriğini
// döndürür (aynı site + dil, yayında, video/galeri dışı). "SIRADAKİ HABER" bandı
// için (Faz 13). categoryID > 0 ise yalnız o kategoriden arar; 0 ise tümü.
func ContentAfter(ctx context.Context, pool *pgxpool.Pool, siteID int64, after time.Time, locale string, excludeID, categoryID int64) (Content, error) {
	var c Content
	q := `
		SELECT c.id, c.type_key, c.status, c.title, COALESCE(c.spot,''), c.slug,
		       c.category_id, c.cover_media_id, c.locale, c.published_at
		FROM contents c
		WHERE c.site_id=$1 AND c.status='yayinda' AND c.id <> $4
		  AND c.type_key NOT IN ('foto_galeri','video')
		  AND (c.published_at > $2 OR (c.published_at IS NULL AND c.created_at > $2))
		  AND ($3='' OR c.locale=$3)
		  AND ($5=0 OR c.category_id=$5)
		ORDER BY COALESCE(c.published_at, c.created_at) ASC, c.id ASC
		LIMIT 1`
	err := pool.QueryRow(ctx, q, siteID, after, locale, excludeID, categoryID).
		Scan(&c.ID, &c.TypeKey, &c.Status, &c.Title, &c.Spot, &c.Slug,
			&c.CategoryID, &c.CoverMediaID, &c.Locale, &c.PublishedAt)
	if err != nil {
		return Content{}, err
	}
	return c, nil
}

// ContentBefore, ContentAfter'ın simetriği: yayın sırasına göre verilen
// zamanın hemen ÖNCEKİ içeriğini döndürür. "ÖNCEKİ HABER" bandı için (Faz 14).
// categoryID > 0 ise yalnız o kategoriden arar; 0 ise tümü.
func ContentBefore(ctx context.Context, pool *pgxpool.Pool, siteID int64, before time.Time, locale string, excludeID, categoryID int64) (Content, error) {
	var c Content
	q := `
		SELECT c.id, c.type_key, c.status, c.title, COALESCE(c.spot,''), c.slug,
		       c.category_id, c.cover_media_id, c.locale, c.published_at
		FROM contents c
		WHERE c.site_id=$1 AND c.status='yayinda' AND c.id <> $4
		  AND c.type_key NOT IN ('foto_galeri','video')
		  AND (c.published_at < $2 OR (c.published_at IS NULL AND c.created_at < $2))
		  AND ($3='' OR c.locale=$3)
		  AND ($5=0 OR c.category_id=$5)
		ORDER BY COALESCE(c.published_at, c.created_at) DESC, c.id DESC
		LIMIT 1`
	err := pool.QueryRow(ctx, q, siteID, before, locale, excludeID, categoryID).
		Scan(&c.ID, &c.TypeKey, &c.Status, &c.Title, &c.Spot, &c.Slug,
			&c.CategoryID, &c.CoverMediaID, &c.Locale, &c.PublishedAt)
	if err != nil {
		return Content{}, err
	}
	return c, nil
}
