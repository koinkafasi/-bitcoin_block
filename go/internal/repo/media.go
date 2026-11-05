package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Media veritabanı medya kaydı.
type Media struct {
	ID             int64
	SiteID         int64
	UserID         int64
	Kind           string
	Filename       string
	Path           string
	Mime           string
	SizeBytes      *int64
	Width          *int
	Height         *int
	Alt            *string
	Caption        *string
	Title          string
	Description    string
	SEOTitle       string
	SEODescription string
	Category       string
	ThumbPath      *string
	IsStarred      bool
	Source         string
	Visibility     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MediaFilter medya kütüphanesi arama/filtre parametreleri.
type MediaFilter struct {
	Query    string
	Category string
	Year     int
	Month    int
	Limit    int
	Offset   int
}

// MediaFacet kategori filtresi için değer ve adet.
type MediaFacet struct {
	Value string
	Count int
}

// MediaPeriod yıl/ay klasörlemesi için dönem ve adet.
type MediaPeriod struct {
	Year  int
	Month int
	Count int
}

const mediaColumns = `id, site_id, user_id, kind, filename, path, mime, size_bytes, width, height,
	alt, caption, COALESCE(title,''), COALESCE(description,''), COALESCE(seo_title,''),
	COALESCE(seo_description,''), COALESCE(category,'genel'), thumb_path, is_starred, source,
	COALESCE(visibility,'public'), created_at, updated_at`

type mediaScanner interface {
	Scan(dest ...any) error
}

func scanMedia(row mediaScanner) (Media, error) {
	var m Media
	err := row.Scan(&m.ID, &m.SiteID, &m.UserID, &m.Kind, &m.Filename, &m.Path, &m.Mime,
		&m.SizeBytes, &m.Width, &m.Height, &m.Alt, &m.Caption, &m.Title, &m.Description,
		&m.SEOTitle, &m.SEODescription, &m.Category, &m.ThumbPath, &m.IsStarred, &m.Source,
		&m.Visibility, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

// CreateMedia yeni medya kaydı ekler (dosya depolamaya önceden yazılmış olmalı).
func CreateMedia(ctx context.Context, pool *pgxpool.Pool, m *Media) (int64, error) {
	title := strings.TrimSpace(m.Title)
	if title == "" {
		title = m.Filename
	}
	category := strings.TrimSpace(m.Category)
	if category == "" {
		category = "genel"
	}
	visibility := strings.TrimSpace(m.Visibility)
	if visibility != "private" {
		visibility = "public"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO media_files
			(site_id, user_id, kind, filename, path, mime, size_bytes, width, height, alt, caption,
			 title, description, seo_title, seo_description, category, thumb_path, is_starred, source, visibility)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) RETURNING id`,
		siteIDOr1(m.SiteID), m.UserID, m.Kind, m.Filename, m.Path, m.Mime, nullableInt64Val(m.SizeBytes),
		nullableIntVal(m.Width), nullableIntVal(m.Height), nullableText(m.Alt), nullableText(m.Caption),
		title, m.Description, m.SEOTitle, m.SEODescription, category, nullableText(m.ThumbPath),
		m.IsStarred, m.Source, visibility,
	).Scan(&id)
	return id, err
}

// GetMedia id ile medya döndürür.
func GetMedia(ctx context.Context, pool *pgxpool.Pool, id int64) (Media, error) {
	row := pool.QueryRow(ctx, `SELECT `+mediaColumns+` FROM media_files WHERE id = $1`, id)
	m, err := scanMedia(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, ErrNotFound
	}
	return m, err
}

// CountMedia siteye ait toplam medya sayısı (dashboard için).
func CountMedia(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM media_files
		WHERE ($1 <= 0 OR site_id = $1) AND COALESCE(visibility,'public')='public'`, siteID).Scan(&n)
	return n, err
}

// ListMedia içerik seçicileri için site medyasını listeler.
func ListMedia(ctx context.Context, pool *pgxpool.Pool, siteID, userID int64, limit int) ([]Media, error) {
	if limit <= 0 {
		limit = 60
	}
	rows, err := pool.Query(ctx, `SELECT `+mediaColumns+` FROM media_files
		WHERE site_id = $1 AND COALESCE(visibility,'public')='public'
		ORDER BY created_at DESC, id DESC LIMIT $2`, siteIDOr1(siteID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListMediaLibrary medya yöneticisi için arama ve dönem/kategori filtreli listeler.
func ListMediaLibrary(ctx context.Context, pool *pgxpool.Pool, siteID int64, f MediaFilter) ([]Media, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 120
	}
	if limit > 5000 {
		limit = 5000
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	where, args := mediaLibraryWhere(siteID, f)
	args = append(args, limit, offset)
	sql := `SELECT ` + mediaColumns + ` FROM media_files ` + where +
		fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountMediaLibrary medya filtresine uyan toplam kayıt sayısını döndürür.
func CountMediaLibrary(ctx context.Context, pool *pgxpool.Pool, siteID int64, f MediaFilter) (int, error) {
	where, args := mediaLibraryWhere(siteID, f)
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM media_files `+where, args...).Scan(&n)
	return n, err
}

func mediaLibraryWhere(siteID int64, f MediaFilter) (string, []any) {
	args := []any{siteIDOr1(siteID)}
	where := []string{"site_id = $1", "COALESCE(visibility,'public')='public'"}
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		p := add("%" + q + "%")
		where = append(where, fmt.Sprintf(`(filename ILIKE %s OR COALESCE(title,'') ILIKE %s OR
			COALESCE(alt,'') ILIKE %s OR COALESCE(caption,'') ILIKE %s OR COALESCE(description,'') ILIKE %s)`,
			p, p, p, p, p))
	}
	if cat := strings.TrimSpace(f.Category); cat != "" {
		where = append(where, "category = "+add(cat))
	}
	if f.Year > 0 {
		where = append(where, "EXTRACT(YEAR FROM created_at)::int = "+add(f.Year))
	}
	if f.Month > 0 {
		where = append(where, "EXTRACT(MONTH FROM created_at)::int = "+add(f.Month))
	}
	return "WHERE " + strings.Join(where, " AND "), args
}

// SetMediaCategory seçili medyaların kategorisini topluca değiştirir.
func SetMediaCategory(ctx context.Context, pool *pgxpool.Pool, siteID int64, ids []int64, category string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := pool.Exec(ctx, `UPDATE media_files SET category=$3, updated_at=now()
		WHERE site_id=$1 AND id = ANY($2)`, siteIDOr1(siteID), ids, category)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// MediaStorageBytes site medyasının toplam boyutunu döndürür.
func MediaStorageBytes(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COALESCE(sum(size_bytes),0) FROM media_files
		WHERE site_id = $1 AND COALESCE(visibility,'public')='public'`, siteIDOr1(siteID)).Scan(&n)
	return n, err
}

// ListMediaCategories site medyasındaki kategori dağılımını döndürür.
func ListMediaCategories(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]MediaFacet, error) {
	rows, err := pool.Query(ctx, `SELECT COALESCE(category,'genel') AS category, count(*)
		FROM media_files WHERE site_id = $1 AND COALESCE(visibility,'public')='public'
		GROUP BY category ORDER BY category`, siteIDOr1(siteID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaFacet
	for rows.Next() {
		var f MediaFacet
		if err := rows.Scan(&f.Value, &f.Count); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// MediaCategoryCounts chip sayaçları için kategori dağılımını döndürür.
//
// Fark: ListMediaCategories tüm kütüphaneyi sayar; burada aktif arama ve
// tarih filtresi uygulanır, KATEGORİ filtresi uygulanmaz. Böylece chip'te
// yazan sayı, chip'e tıklanınca gerçekten gelecek sonuç sayısıdır.
func MediaCategoryCounts(ctx context.Context, pool *pgxpool.Pool, siteID int64, f MediaFilter) ([]MediaFacet, error) {
	// Kategori filtresi kaldırılır; aksi hâlde yalnız seçili kategori sayılırdı.
	f.Category = ""
	where, args := mediaLibraryWhere(siteID, f)
	rows, err := pool.Query(ctx, `SELECT COALESCE(category,'genel') AS category, count(*)
		FROM media_files `+where+`
		GROUP BY category ORDER BY count(*) DESC, category`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaFacet
	for rows.Next() {
		var c MediaFacet
		if err := rows.Scan(&c.Value, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListMediaPeriods yıl/ay klasör dağılımını döndürür.
func ListMediaPeriods(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]MediaPeriod, error) {
	rows, err := pool.Query(ctx, `SELECT EXTRACT(YEAR FROM created_at)::int, EXTRACT(MONTH FROM created_at)::int, count(*)
		FROM media_files WHERE site_id = $1 AND COALESCE(visibility,'public')='public'
		GROUP BY 1,2 ORDER BY 1 DESC, 2 DESC`, siteIDOr1(siteID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaPeriod
	for rows.Next() {
		var p MediaPeriod
		if err := rows.Scan(&p.Year, &p.Month, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateMediaMetadata medya başlığı, alt metni, SEO alanları ve kategorisini günceller.
func UpdateMediaMetadata(ctx context.Context, pool *pgxpool.Pool, siteID, id int64, m *Media) error {
	tag, err := pool.Exec(ctx, `UPDATE media_files
		SET filename=$3, title=$4, alt=$5, caption=$6, description=$7,
		    seo_title=$8, seo_description=$9, category=$10, updated_at=now()
		WHERE id=$1 AND site_id=$2`,
		id, siteIDOr1(siteID), m.Filename, m.Title, nullableText(m.Alt), nullableText(m.Caption),
		m.Description, m.SEOTitle, m.SEODescription, m.Category)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateMediaImageMeta yeniden düzenlenen görselin boyutlarını ve önizleme yolunu günceller.
func UpdateMediaImageMeta(ctx context.Context, pool *pgxpool.Pool, id int64, width, height int, thumbPath *string) error {
	tag, err := pool.Exec(ctx, `UPDATE media_files
		SET width=$2, height=$3, thumb_path=$4, updated_at=now() WHERE id=$1`,
		id, width, height, nullableText(thumbPath))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateMediaThumb üretilen önizleme yolunu kaydeder.
func UpdateMediaThumb(ctx context.Context, pool *pgxpool.Pool, id int64, path string) error {
	tag, err := pool.Exec(ctx, `UPDATE media_files SET thumb_path=$2, updated_at=now() WHERE id=$1`, id, nullableText(&path))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteMedia kaydı siler (dosya silinmez; depolama katmanının işi ayrı).
// İçerikte kullanılıyorsa (kapak/manşet/sosyal) ErrMediaInUse döner.
func DeleteMedia(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM media_files WHERE id = $1 AND NOT EXISTS (
		SELECT 1 FROM contents WHERE cover_media_id = $1 OR manset_media_id = $1 OR social_media_id = $1
	)`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMediaInUse
	}
	return nil
}

// RenameMedia medyanın görünen adını (filename) değiştirir.
func RenameMedia(ctx context.Context, pool *pgxpool.Pool, id int64, newFilename string) error {
	tag, err := pool.Exec(ctx, `UPDATE media_files SET filename=$2, updated_at=now() WHERE id=$1`, id, newFilename)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullableIntVal(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableInt64Val(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
