package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HeadlineDesign görsel manşet tasarım belgesidir (JSON — 02 §8).
type HeadlineDesign struct {
	ID         int64
	Name       string
	Design     []byte
	IsDefault  bool
	CategoryID *int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateHeadlineDesign tasarım kaydeder; isDefault ise diğerlerini default'tan çıkarır.
func CreateHeadlineDesign(ctx context.Context, pool *pgxpool.Pool, name string, design []byte, isDefault bool, categoryID *int64, userID int64) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE headline_designs SET is_default = false WHERE category_id IS NOT DISTINCT FROM $1`, nullableInt64P(categoryID)); err != nil {
			return 0, err
		}
	}
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO headline_designs (name, design, is_default, category_id, created_by)
		VALUES ($1,$2::jsonb,$3,$4,$5) RETURNING id`,
		name, design, isDefault, nullableInt64P(categoryID), userID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

// ListHeadlineDesigns tüm tasarımları döndürür.
func ListHeadlineDesigns(ctx context.Context, pool *pgxpool.Pool) ([]HeadlineDesign, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, design, is_default, category_id, created_at, updated_at
		FROM headline_designs ORDER BY is_default DESC, updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HeadlineDesign
	for rows.Next() {
		var h HeadlineDesign
		if err := rows.Scan(&h.ID, &h.Name, &h.Design, &h.IsDefault, &h.CategoryID, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetHeadlineDesign id ile tasarımı döndürür.
func GetHeadlineDesign(ctx context.Context, pool *pgxpool.Pool, id int64) (HeadlineDesign, error) {
	var h HeadlineDesign
	err := pool.QueryRow(ctx, `
		SELECT id, name, design, is_default, category_id, created_at, updated_at
		FROM headline_designs WHERE id = $1`, id).Scan(&h.ID, &h.Name, &h.Design, &h.IsDefault, &h.CategoryID, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return HeadlineDesign{}, ErrNotFound
	}
	return h, err
}

// UpdateHeadlineDesign ad/değer/default günceller.
func UpdateHeadlineDesign(ctx context.Context, pool *pgxpool.Pool, id int64, name string, design []byte, isDefault bool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE headline_designs SET is_default = false`); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE headline_designs SET name=$1, design=$2::jsonb, is_default=$3 WHERE id=$4`,
		name, design, isDefault, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteHeadlineDesign tasarımı siler.
func DeleteHeadlineDesign(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM headline_designs WHERE id = $1`, id)
	return err
}

// ContentDesignCover blok başına kapak şablonu eşlemesinin önbellek kaydıdır
// (Faz 3): bir içerik için bir tasarım şablonunun ürettiği kapak görseli.
type ContentDesignCover struct {
	ContentID        int64
	DesignID         int64
	MediaID          int64
	ContentUpdatedAt time.Time
	UpdatedAt        time.Time
}

// GetContentDesignCover içerik için önbellekte kapak arar; yoksa ErrNotFound.
// İçerik, kayıttan sonra güncellendiyse önbellek geçersiz sayılır (yeniden üretim).
func GetContentDesignCover(ctx context.Context, pool *pgxpool.Pool, contentID, designID int64) (ContentDesignCover, error) {
	var c ContentDesignCover
	err := pool.QueryRow(ctx, `
		SELECT content_id, design_id, media_id, content_updated_at, updated_at
		FROM content_design_covers WHERE content_id=$1 AND design_id=$2`,
		contentID, designID).Scan(&c.ContentID, &c.DesignID, &c.MediaID, &c.ContentUpdatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentDesignCover{}, ErrNotFound
	}
	return c, err
}

// SaveContentDesignCover üretilen kapak görselinin medya kaydını önbelleğe yazar.
func SaveContentDesignCover(ctx context.Context, pool *pgxpool.Pool, contentID, designID, mediaID int64, contentUpdatedAt time.Time) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content_design_covers (content_id, design_id, media_id, content_updated_at, updated_at)
		VALUES ($1,$2,$3,$4, now())
		ON CONFLICT (content_id, design_id)
		DO UPDATE SET media_id=EXCLUDED.media_id, content_updated_at=EXCLUDED.content_updated_at, updated_at=now()`,
		contentID, designID, mediaID, contentUpdatedAt)
	return err
}

// DeleteContentDesignCovers içeriğin tüm kapak eşlemelerini siler (içerik güncellenince).
func DeleteContentDesignCovers(ctx context.Context, pool *pgxpool.Pool, contentID int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM content_design_covers WHERE content_id=$1`, contentID)
	return err
}
