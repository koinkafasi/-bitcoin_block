// Package repo — etiket yönetimi (Faz 41): admin listesi, adlandır, birleştir, sil.
package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TagWithCount etiket + kullanım sayısı (admin listesi).
type TagWithCount struct {
	ID    int64
	Name  string
	Slug  string
	Count int
}

// ListTagsWithCount etiketleri kullanım sayısıyla döndürür; q ile ad/slug arar.
func ListTagsWithCount(ctx context.Context, pool *pgxpool.Pool, q string, limit int) ([]TagWithCount, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, count(ct.content_id)::int
		FROM tags t
		LEFT JOIN content_tags ct ON ct.tag_id = t.id
		WHERE ($1 = '' OR t.name ILIKE '%' || $1 || '%' OR t.slug ILIKE '%' || $1 || '%')
		GROUP BY t.id
		ORDER BY count(ct.content_id) DESC, t.name ASC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagWithCount
	for rows.Next() {
		var t TagWithCount
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CountTags toplam etiket sayısı (admin başlığı — Faz 41).
func CountTags(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM tags`).Scan(&n)
	return n, err
}

// GetTagByName ad veya slug ile etiketi döndürür (birleştirme hedefi için).
func GetTagByName(ctx context.Context, pool *pgxpool.Pool, name string) (Tag, error) {
	var t Tag
	err := pool.QueryRow(ctx, `SELECT id, name, slug FROM tags WHERE name = $1 OR slug = $1`, name).Scan(&t.ID, &t.Name, &t.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	return t, err
}

// RenameTag etiket adını (+slug'ı) değiştirir. Aynı ad/slug başka etikette varsa
// ErrDuplicate döner (unique 23505).
func RenameTag(ctx context.Context, pool *pgxpool.Pool, id int64, name, slug string) error {
	tag, err := pool.Exec(ctx, `UPDATE tags SET name=$2, slug=$3 WHERE id=$1`, id, name, slug)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicate
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MergeTag kaynak (from) etiketini hedefe (to) birleştirir: içerik bağlarını
// taşır ve kaynağı (bağlarıyla) siler. from==to ise no-op.
// NOT: ON CONFLICT yalnız INSERT'te geçerlidir → bağ taşıma INSERT..SELECT
// ile yapılır (UPDATE..ON CONFLICT yazım hatasıydı, Faz 41'de yakalandı).
func MergeTag(ctx context.Context, pool *pgxpool.Pool, fromID, toID int64) (int64, error) {
	if fromID == toID {
		return 0, nil
	}
	tag, err := pool.Exec(ctx, `
		INSERT INTO content_tags (content_id, tag_id)
		SELECT content_id, $2 FROM content_tags WHERE tag_id = $1
		ON CONFLICT DO NOTHING`, fromID, toID)
	if err != nil {
		return 0, err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM content_tags WHERE tag_id = $1`, fromID); err != nil {
		return 0, err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, fromID); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteTag etiketi ve içerik bağlarını siler; bağlanma adedini döndürür.
func DeleteTag(ctx context.Context, pool *pgxpool.Pool, id int64) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM content_tags WHERE tag_id = $1`, id)
	if err != nil {
		return 0, err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
