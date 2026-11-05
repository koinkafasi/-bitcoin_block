package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EditorNote, yayın öncesi editör notu/satır içi yorumudur.
type EditorNote struct {
	ID        int64
	ContentID int64
	UserID    *int64
	UserName  string
	Body      string
	Quote     string
	Resolved  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func CreateEditorNote(ctx context.Context, pool *pgxpool.Pool, siteID, contentID, userID int64, body, quote string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO content_editor_notes (site_id, content_id, user_id, body, quote)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		siteID, contentID, userID, body, quote).Scan(&id)
	return id, err
}

func ListEditorNotes(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]EditorNote, error) {
	rows, err := pool.Query(ctx, `
		SELECT n.id, n.content_id, n.user_id, COALESCE(u.name,''), n.body, n.quote,
		       n.resolved, n.created_at, n.updated_at
		FROM content_editor_notes n
		LEFT JOIN users u ON u.id=n.user_id
		WHERE n.content_id=$1
		ORDER BY n.resolved ASC, n.created_at DESC
		LIMIT 200`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EditorNote
	for rows.Next() {
		var n EditorNote
		if err := rows.Scan(&n.ID, &n.ContentID, &n.UserID, &n.UserName, &n.Body, &n.Quote,
			&n.Resolved, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func ResolveEditorNote(ctx context.Context, pool *pgxpool.Pool, contentID, noteID int64, resolved bool) error {
	tag, err := pool.Exec(ctx, `
		UPDATE content_editor_notes SET resolved=$3, updated_at=now()
		WHERE id=$1 AND content_id=$2`, noteID, contentID, resolved)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteEditorNote(ctx context.Context, pool *pgxpool.Pool, contentID, noteID int64) error {
	tag, err := pool.Exec(ctx, `DELETE FROM content_editor_notes WHERE id=$1 AND content_id=$2`, noteID, contentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
