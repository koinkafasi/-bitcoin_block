package repo

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Modüler sayfa düzeni (spec 08) ----

// GetLayout sayfa düzeni bloklarını (JSON) döndürür; yoksa boş dizi.
func GetLayout(ctx context.Context, pool DB, siteID int64, pageKey string) ([]byte, error) {
	var b []byte
	err := pool.QueryRow(ctx, `
		SELECT blocks FROM page_layouts WHERE site_id=$1 AND page_key=$2`, siteID, pageKey).Scan(&b)
	if err != nil {
		//nolint:nilerr // sayfa düzeni kaydı yoksa boş dizi döner; şablon varsayılan sırayı basar
		return []byte("[]"), nil
	}
	return b, nil
}

// SaveLayout sayfa düzenini (blok dizisi JSON) yazar/günceller.
func SaveLayout(ctx context.Context, pool DB, siteID int64, pageKey string, blocks []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO page_layouts (site_id, page_key, blocks, updated_at) VALUES ($1,$2,$3::jsonb, now())
		ON CONFLICT (site_id, page_key) DO UPDATE SET blocks=EXCLUDED.blocks, updated_at=now()`,
		siteID, pageKey, blocks)
	return err
}

// DeleteLayout düzeni sıfırlar (varsayılana dönüş).
func DeleteLayout(ctx context.Context, pool DB, siteID int64, pageKey string) error {
	_, err := pool.Exec(ctx, `DELETE FROM page_layouts WHERE site_id=$1 AND page_key=$2`, siteID, pageKey)
	return err
}

// ---- Sayfa düzeni sürüm geçmişi (Faz 4) ----

// LayoutVersion sayfa düzeninin kaydedilmiş bir sürümüdür.
type LayoutVersion struct {
	ID        int64
	SiteID    int64
	PageKey   string
	Blocks    []byte
	Note      string
	CreatedBy *int64
	CreatedAt time.Time
}

// ListLayoutKeys verilen önekle başlayan sayfa anahtarlarını döndürür.
// Kategoriye özel düzenlerin varlığını tek sorguda görmek için kullanılır.
func ListLayoutKeys(ctx context.Context, pool DB, siteID int64, prefix string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT page_key FROM page_layouts WHERE site_id=$1 AND page_key LIKE $2`, siteID, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// CreateLayoutVersion düzenin mevcut bloklarını yeni bir sürüm kaydı olarak
// saklar. SaveLayout HER kayıtta önce bu fonksiyonla önceki durumu arşivler
// (böylece her kayıt bir sürüm olur; sürüm geçmişi düzenin evrimini tutar).
func CreateLayoutVersion(ctx context.Context, pool DB, siteID int64, pageKey string, blocks []byte, note string, createdBy int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO page_layout_versions (site_id, page_key, blocks, note, created_by)
		VALUES ($1,$2,$3::jsonb,$4,$5) RETURNING id`,
		siteID, pageKey, blocks, note, int64OrNil(createdBy)).Scan(&id)
	return id, err
}

// ListLayoutVersions düzenin sürümlerini, en yenisinden başlayarak döndürür.
// Varsayılan/açık düzeni hariç tutmak için "yoksa" davranışı yoktur: kayıt
// yoksa boş liste döner.
func ListLayoutVersions(ctx context.Context, pool DB, siteID int64, pageKey string, limit int) ([]LayoutVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, page_key, blocks, note, created_by, created_at
		FROM page_layout_versions WHERE site_id=$1 AND page_key=$2
		ORDER BY id DESC LIMIT $3`, siteID, pageKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LayoutVersion
	for rows.Next() {
		var v LayoutVersion
		if err := rows.Scan(&v.ID, &v.SiteID, &v.PageKey, &v.Blocks, &v.Note, &v.CreatedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetLayoutVersion belirli bir sürümün bloklarını döndürür.
func GetLayoutVersion(ctx context.Context, pool DB, id int64) (LayoutVersion, error) {
	var v LayoutVersion
	err := pool.QueryRow(ctx, `
		SELECT id, site_id, page_key, blocks, note, created_by, created_at
		FROM page_layout_versions WHERE id=$1`, id).Scan(&v.ID, &v.SiteID, &v.PageKey, &v.Blocks, &v.Note, &v.CreatedBy, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LayoutVersion{}, ErrNotFound
	}
	return v, err
}

// RestoreLayoutVersion istenen sürümün bloklarını aktif düzene yazar ve bu
// geri almayı da geçmişe yeni bir sürüm olarak kaydeder (geri alma geri de
// alınabilir).
func RestoreLayoutVersion(ctx context.Context, pool DB, siteID int64, pageKey string, versionID int64, note string, createdBy int64) error {
	v, err := GetLayoutVersion(ctx, pool, versionID)
	if err != nil {
		return err
	}
	if err := SaveLayout(ctx, pool, siteID, pageKey, v.Blocks); err != nil {
		return err
	}
	if note == "" {
		note = "Sürüm #" + itoa64(versionID) + " geri alındı"
	}
	_, err = CreateLayoutVersion(ctx, pool, siteID, pageKey, v.Blocks, note, createdBy)
	return err
}

// int64OrNil 0 ise nil (yabancı anahtar) döndürür.
func int64OrNil(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

// itoa64 int64'ü ondalık dizeye çevirir.
func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// ---- Sayfa düzeni A/B testi (Faz 7) ----

// LayoutAB sayfa düzeninin A/B testi ayarıdır. B varyantı tıpkı normal blok
// dizisi gibi JSON'dur; serving sırasında b_blocks NormalizeDesign ile normalize
// edilir. Varyant A her zaman aktif düzendir (page_layouts.blocks).
type LayoutAB struct {
	SiteID    int64
	PageKey   string
	Enabled   bool
	WeightB   int
	BBlocks   []byte
	UpdatedAt time.Time
}

// GetLayoutAB A/B ayarını döndürür; kayıt yoksa devre dışı varsayılan.
func GetLayoutAB(ctx context.Context, pool DB, siteID int64, pageKey string) (LayoutAB, error) {
	var ab LayoutAB
	err := pool.QueryRow(ctx, `
		SELECT site_id, page_key, enabled, weight_b, b_blocks, updated_at
		FROM layout_ab WHERE site_id=$1 AND page_key=$2`, siteID, pageKey).
		Scan(&ab.SiteID, &ab.PageKey, &ab.Enabled, &ab.WeightB, &ab.BBlocks, &ab.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LayoutAB{SiteID: siteID, PageKey: pageKey, WeightB: 50, BBlocks: []byte("[]")}, nil
	}
	return ab, err
}

// SaveLayoutAB A/B ayarını yazar (upsert).
func SaveLayoutAB(ctx context.Context, pool DB, siteID int64, pageKey string, enabled bool, weight int, bBlocks []byte) error {
	if weight < 0 {
		weight = 0
	}
	if weight > 100 {
		weight = 100
	}
	if len(bBlocks) == 0 {
		bBlocks = []byte("[]")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO layout_ab (site_id, page_key, enabled, weight_b, b_blocks, updated_at)
		VALUES ($1,$2,$3,$4,$5::jsonb, now())
		ON CONFLICT (site_id, page_key)
		DO UPDATE SET enabled=EXCLUDED.enabled, weight_b=EXCLUDED.weight_b,
		              b_blocks=EXCLUDED.b_blocks, updated_at=now()`,
		siteID, pageKey, enabled, weight, bBlocks)
	return err
}

// RecordABHit günlük A/B görüntüleme sayacını artırır.
func RecordABHit(ctx context.Context, pool DB, siteID int64, pageKey, variant string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO layout_ab_hits (site_id, page_key, day, variant, hits)
		VALUES ($1,$2,CURRENT_DATE,$3,1)
		ON CONFLICT (site_id, page_key, day, variant)
		DO UPDATE SET hits = layout_ab_hits.hits + 1`,
		siteID, pageKey, variant)
	return err
}

// LayoutABHits son günlük A/B görüntüleme sayılarını döndürür (yönetim paneli).
type LayoutABHits struct {
	Day     string
	Variant string
	Hits    int64
}

// ListLayoutABHits son N günün A/B sayılarını döndürür.
func ListLayoutABHits(ctx context.Context, pool DB, siteID int64, pageKey string, days int) ([]LayoutABHits, error) {
	if days <= 0 || days > 31 {
		days = 14
	}
	rows, err := pool.Query(ctx, `
		SELECT to_char(day,'YYYY-MM-DD') AS day, variant, hits
		FROM layout_ab_hits
		WHERE site_id=$1 AND page_key=$2 AND day >= CURRENT_DATE - ($3::int - 1)
		ORDER BY day ASC, variant ASC`, siteID, pageKey, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LayoutABHits
	for rows.Next() {
		var h LayoutABHits
		if err := rows.Scan(&h.Day, &h.Variant, &h.Hits); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
