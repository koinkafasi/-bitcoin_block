package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Category veritabanı kategori kaydı.
type Category struct {
	ID         int64
	SiteID     int64
	Name       string
	Slug       string
	CustomURL  *string
	OpenTarget *string
	InHeader   bool
	InMenu     bool
	InFooter   bool
	Status     string
	SortOrder  int
	// Kantan referansı (spec 06 K-A): AI alaka skor kapıları + ikon/görsel servisi
	MinScore         int
	HomepageMinScore int
	Icon             string
	ImageService     *string
	NameEN           *string
	NameES           *string
	Label            *string
	// ParentID doluysa bu kategori bir üst kategorinin alt kategorisidir.
	ParentID   *int64
	ParentSlug *string
	ParentName *string
	// ChildCount doğrudan alt kategori sayısı (listelemede dolu olabilir).
	ChildCount int
	// ListView kategoriye özel liste görünümü (klasik|liste|izgara|mozaik; boş=genel).
	ListView string
	// MinTierWeight bu kategoride YENI içerik için varsayilan abonelik
	// katmani agirligi (0 = kategori kurali yok).
	MinTierWeight int
	// LayoutPreset uygulanan hazır kategori tasarımı anahtarı.
	LayoutPreset string
	// CustomDesign bu kategori için özel page_layout kaydı var mı (listelemede).
	CustomDesign bool
	// DesignName uygulanan şablonun görünen adı (listelemede).
	DesignName string
	// ContentCount listelemede dolu olabilir
	ContentCount int
}

// UstSlug üst kategori slug'ını şablonlarda güvenle karşılaştırmak için
// düz string döndürür (ParentSlug pointer'ı nil olabilir).
func (c Category) UstSlug() string {
	if c.ParentSlug == nil {
		return ""
	}
	return *c.ParentSlug
}

// UstAd üst kategori adını şablonlar için düz string döndürür.
func (c Category) UstAd() string {
	if c.ParentName == nil {
		return ""
	}
	return *c.ParentName
}

// ListCategories siteye ait kategorileri sıralı döndürür (siteID<=0 → tümü).
func ListCategories(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]Category, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.custom_url, c.open_target, c.in_header, c.in_menu, c.in_footer, c.status, c.sort_order,
		       c.min_score, c.homepage_min_score, c.icon, c.image_service, c.name_en, c.name_es, c.label,
		       c.list_view, c.layout_preset, c.min_tier_weight,
		       c.parent_id, p.slug, p.name,
		       (SELECT count(*) FROM contents ct WHERE ct.category_id = c.id AND ct.status = 'yayinda'),
		       (SELECT count(*) FROM categories ch WHERE ch.parent_id = c.id)
		FROM categories c
		LEFT JOIN categories p ON p.id = c.parent_id
		WHERE ($1 <= 0 OR c.site_id = $1)
		ORDER BY COALESCE(p.sort_order, c.sort_order),
		         CASE WHEN c.parent_id IS NULL THEN 0 ELSE 1 END,
		         c.sort_order, c.name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.CustomURL, &c.OpenTarget, &c.InHeader, &c.InMenu, &c.InFooter, &c.Status, &c.SortOrder,
			&c.MinScore, &c.HomepageMinScore, &c.Icon, &c.ImageService, &c.NameEN, &c.NameES, &c.Label,
			&c.ListView, &c.LayoutPreset, &c.MinTierWeight,
			&c.ParentID, &c.ParentSlug, &c.ParentName, &c.ContentCount, &c.ChildCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCategory yeni kategori ekler (site_id c.SiteID; yoksa 1).
func CreateCategory(ctx context.Context, pool *pgxpool.Pool, c *Category) (int64, error) {
	var id int64
	siteID := c.SiteID
	if siteID <= 0 {
		siteID = 1
	}
	err := pool.QueryRow(ctx, `
		INSERT INTO categories (site_id, name, slug, custom_url, open_target, in_header, in_menu, in_footer, status,
			min_score, homepage_min_score, icon, image_service, name_en, name_es, label, parent_id, list_view,
			min_tier_weight)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) RETURNING id`,
		siteID, c.Name, c.Slug, nullableText(c.CustomURL), nullableText(c.OpenTarget),
		c.InHeader, c.InMenu, c.InFooter, c.Status,
		c.MinScore, c.HomepageMinScore, c.Icon, nullableText(c.ImageService),
		nullableText(c.NameEN), nullableText(c.NameES), nullableText(c.Label), nullableInt(c.ParentID), c.ListView,
		clampTierWeight(c.MinTierWeight),
	).Scan(&id)
	return id, err
}

// UpdateCategory kategoriyi günceller.
func UpdateCategory(ctx context.Context, pool *pgxpool.Pool, c *Category) error {
	_, err := pool.Exec(ctx, `
		UPDATE categories SET name=$1, slug=$2, custom_url=$3, open_target=$4,
			in_header=$5, in_menu=$6, in_footer=$7, status=$8, sort_order=$9,
			min_score=$10, homepage_min_score=$11, icon=$12, image_service=$13, name_en=$14, name_es=$15, label=$16,
			parent_id=$17, list_view=$18, min_tier_weight=$19
		WHERE id=$20`,
		c.Name, c.Slug, nullableText(c.CustomURL), nullableText(c.OpenTarget),
		c.InHeader, c.InMenu, c.InFooter, c.Status, c.SortOrder,
		c.MinScore, c.HomepageMinScore, c.Icon, nullableText(c.ImageService),
		nullableText(c.NameEN), nullableText(c.NameES), nullableText(c.Label), nullableInt(c.ParentID), c.ListView,
		clampTierWeight(c.MinTierWeight), c.ID,
	)
	return err
}

// clampTierWeight kategori katman agirligini 0..5 araligina sikistirir.
func clampTierWeight(n int) int {
	if n < 0 {
		return 0
	}
	if n > 5 {
		return 5
	}
	return n
}

func CategoryMinTierWeight(ctx context.Context, pool *pgxpool.Pool, categoryID int64) (int, error) {
	if categoryID <= 0 {
		return 0, nil
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT min_tier_weight FROM categories WHERE id=$1), 0)`,
		categoryID).Scan(&n); err != nil {
		return 0, err
	}
	return clampTierWeight(n), nil
}

// SetCategoryDesign kategoriye özel liste görünümü + şablon anahtarını yazar.
func SetCategoryDesign(ctx context.Context, pool *pgxpool.Pool, id int64, preset, listView string) error {
	_, err := pool.Exec(ctx, `UPDATE categories SET layout_preset=$2, list_view=$3 WHERE id=$1`, id, preset, listView)
	return err
}

// DeleteCategory kategoriyi siler (içerik varsa engellenir).
func DeleteCategory(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM categories WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM contents WHERE category_id = $1)`, id)
	return err
}

// Mahret veritabanı mahreç kaydı.
type Mahret struct {
	ID   int64
	Name string
	Slug string
}

// ListMahrets mahreçleri döndürür.
func ListMahrets(ctx context.Context, pool *pgxpool.Pool) ([]Mahret, error) {
	rows, err := pool.Query(ctx, `SELECT id, name, slug FROM mahrets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Mahret
	for rows.Next() {
		var m Mahret
		if err := rows.Scan(&m.ID, &m.Name, &m.Slug); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateMahret mahreç ekler, slug çakışması hata olur.
func CreateMahret(ctx context.Context, pool *pgxpool.Pool, name, slug string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO mahrets (name, slug) VALUES ($1,$2) RETURNING id`, name, slug).Scan(&id)
	return id, err
}

// ContentType içerik türü lookup kaydıdır.
type ContentType struct {
	Key   string
	Label string
}

// ListContentTypes türleri sıralı döndürür.
func ListContentTypes(ctx context.Context, pool *pgxpool.Pool) ([]ContentType, error) {
	rows, err := pool.Query(ctx, `SELECT key, label FROM content_types ORDER BY sort_order, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentType
	for rows.Next() {
		var t ContentType
		if err := rows.Scan(&t.Key, &t.Label); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteMahret mahreci siler (içerik bağlıysa engeller).
func DeleteMahret(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM mahrets WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM contents WHERE mahrep_id = $1)`, id)
	return err
}

// Tag veritabanı etiket kaydı..
type Tag struct {
	ID   int64
	Name string
	Slug string
}

// ListTags etiketleri döndürür.
func ListTags(ctx context.Context, pool *pgxpool.Pool) ([]Tag, error) {
	rows, err := pool.Query(ctx, `SELECT id, name, slug FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// EnsureTag var olanı döndürür, yoksa oluşturur.
func EnsureTag(ctx context.Context, pool *pgxpool.Pool, name, slug string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO tags (name, slug) VALUES ($1,$2)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, name, slug).Scan(&id)
	return id, err
}

// LinkContentTag içerik-etiket ilişkisini kurar (çiftleri engeller).
func LinkContentTag(ctx context.Context, pool *pgxpool.Pool, contentID, tagID int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content_tags (content_id, tag_id) VALUES ($1,$2)
		ON CONFLICT DO NOTHING`, contentID, tagID)
	return err
}

// ContentTagNames içeriğin etiket adlarını döndürür.
func ContentTagNames(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.name FROM content_tags ct JOIN tags t ON t.id = ct.tag_id
		WHERE ct.content_id = $1 ORDER BY t.name`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ReplaceContentTags içeriğin etiketlerini tamamen değiştirir.
// CategoryIDBySlug slug ile kategori id döndürür.
func CategoryIDBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug=$1`, slug).Scan(&id)
	return id, err
}

// TagsOf içeriğin etiket adlarını döndürür.
func TagsOf(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT t.name FROM content_tags ct JOIN tags t ON t.id = ct.tag_id
		WHERE ct.content_id = $1 ORDER BY t.name`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func ReplaceContentTags(ctx context.Context, pool *pgxpool.Pool, contentID int64, tags []string, slugify func(string) string) error {
	if _, err := pool.Exec(ctx, `DELETE FROM content_tags WHERE content_id = $1`, contentID); err != nil {
		return err
	}
	for _, t := range tags {
		if t == "" {
			continue
		}
		id, err := EnsureTag(ctx, pool, t, slugify(t))
		if err != nil {
			return err
		}
		if err := LinkContentTag(ctx, pool, contentID, id); err != nil {
			return err
		}
	}
	return nil
}
