package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func IncrementHit(ctx context.Context, pool *pgxpool.Pool, contentID int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO counters_hit (content_id, total) VALUES ($1, 1)
		ON CONFLICT (content_id) DO UPDATE SET total = counters_hit.total + 1, updated_at = now()`,
		contentID)
	if err != nil {
		return err
	}
	// Günlük kırılım ikincil: burada hata olsa da okunma sayılmış olur.
	_, _ = pool.Exec(ctx, `
		INSERT INTO hit_daily (content_id, day, hits) VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (content_id, day) DO UPDATE SET hits = hit_daily.hits + 1`,
		contentID)
	return nil
}

func IncrementHits(ctx context.Context, pool *pgxpool.Pool, sayilar map[int64]int64) error {
	if len(sayilar) == 0 {
		return nil
	}
	idler := make([]int64, 0, len(sayilar))
	adetler := make([]int64, 0, len(sayilar))
	for id, n := range sayilar {
		if n <= 0 {
			continue
		}
		idler = append(idler, id)
		adetler = append(adetler, n)
	}
	if len(idler) == 0 {
		return nil
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO counters_hit (content_id, total)
		SELECT id, adet FROM unnest($1::bigint[], $2::bigint[]) AS t(id, adet)
		ON CONFLICT (content_id) DO UPDATE
		SET total = counters_hit.total + EXCLUDED.total, updated_at = now()`,
		idler, adetler)
	if err != nil {
		return err
	}
	// Günlük kırılım ikincil: burada hata olsa da toplam sayaç yazılmış olur.
	_, _ = pool.Exec(ctx, `
		INSERT INTO hit_daily (content_id, day, hits)
		SELECT id, CURRENT_DATE, adet FROM unnest($1::bigint[], $2::bigint[]) AS t(id, adet)
		ON CONFLICT (content_id, day) DO UPDATE SET hits = hit_daily.hits + EXCLUDED.hits`,
		idler, adetler)
	return nil
}

// IncrementShare paylaşım sayacını artırır.
func IncrementShare(ctx context.Context, pool *pgxpool.Pool, contentID int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO counters_share (content_id, total) VALUES ($1, 1)
		ON CONFLICT (content_id) DO UPDATE SET total = counters_share.total + 1, updated_at = now()`,
		contentID)
	return err
}

// Hits content okunma sayısını döndürür.
func Hits(ctx context.Context, pool *pgxpool.Pool, contentID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COALESCE(total,0) FROM counters_hit WHERE content_id = $1`, contentID).Scan(&n)
	return n, err
}

// Shares content paylaşım sayısını döndürür.
func Shares(ctx context.Context, pool *pgxpool.Pool, contentID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COALESCE(total,0) FROM counters_share WHERE content_id = $1`, contentID).Scan(&n)
	return n, err
}

// HitRow en çok okunanlar satırıdır.
type HitRow struct {
	ID    int64
	Title string
	Hits  int64
}

// MostRead en çok okunanları döndürür.
func MostRead(ctx context.Context, pool *pgxpool.Pool, locale string, limit int) ([]HitRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, COALESCE(h.total,0)
		FROM contents c JOIN counters_hit h ON h.content_id = c.id
		WHERE c.status = 'yayinda' AND c.locale = $1
		ORDER BY h.total DESC LIMIT $2`, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HitRow
	for rows.Next() {
		var r HitRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Hits); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Comment yorum kaydıdır.
type Comment struct {
	ID         int64
	ContentID  int64
	AuthorName string
	Body       string
	Status     string
	CreatedAt  time.Time
	// Faz 36 — etkileşim: yanıt zinciri (parent_id) ve beğeni sayısı.
	ParentID int64
	Likes    int
}

// GetComment id ile yorumu döndürür (Faz 36 — yanıtın ebeveynini doğrulamak için).
func GetComment(ctx context.Context, pool *pgxpool.Pool, id int64) (Comment, error) {
	var c Comment
	var pid *int64
	var parent int64
	err := pool.QueryRow(ctx, `
		SELECT id, content_id, author_name, body, status, created_at, parent_id, likes
		FROM comments WHERE id = $1`, id).
		Scan(&c.ID, &c.ContentID, &c.AuthorName, &c.Body, &c.Status, &c.CreatedAt, &pid, &c.Likes)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, ErrNotFound
	}
	if pid != nil {
		parent = *pid
	}
	c.ParentID = parent
	return c, err
}

// ListApprovedComments onaylı yorumları döndürür (ebeveyn + beğeni ile; Faz 36).
func ListApprovedComments(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]Comment, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, content_id, author_name, body, status, created_at, COALESCE(parent_id,0), likes
		FROM comments WHERE content_id = $1 AND status = 'onayli' ORDER BY created_at ASC`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		var t time.Time
		if err := rows.Scan(&c.ID, &c.ContentID, &c.AuthorName, &c.Body, &c.Status, &t, &c.ParentID, &c.Likes); err != nil {
			return nil, err
		}
		c.CreatedAt = t
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountPendingComments onay bekleyen yorum sayısını siteye göre döndürür (siteID<=0 → tümü).
func CountPendingComments(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM comments cm JOIN contents c ON c.id = cm.content_id
		WHERE cm.status = 'onay_bekliyor' AND ($1 <= 0 OR c.site_id = $1)`, siteID).Scan(&n)
	return n, err
}

// CommentDurum yorum durum özetidir (Faz 37 — rapor paneli).
type CommentDurum struct {
	Pending  int
	Approved int
	Total    int
}

// CommentSummary siteye ait yorum durum özeti (Faz 37).
func CommentSummary(ctx context.Context, pool *pgxpool.Pool, siteID int64) (CommentDurum, error) {
	var s CommentDurum
	err := pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE cm.status = 'onay_bekliyor'),
			count(*) FILTER (WHERE cm.status = 'onayli'),
			count(*)
		FROM comments cm JOIN contents c ON c.id = cm.content_id
		WHERE ($1 <= 0 OR c.site_id = $1)`, siteID).Scan(&s.Pending, &s.Approved, &s.Total)
	return s, err
}

// MostCommented en çok yorumlanan içerikleri döndürür (Faz 37 — rapor).
func MostCommented(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]HitRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, count(cm.id)::bigint
		FROM contents c JOIN comments cm ON cm.content_id = c.id
		WHERE c.site_id = $1 AND cm.status IN ('onayli','onay_bekliyor')
		GROUP BY c.id, c.title
		ORDER BY count(cm.id) DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HitRow
	for rows.Next() {
		var r HitRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Hits); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddComment yeni yorum ekler (önce incelemeye girer). IP saklanır (Faz 35 —
// spam/hız sınırı için; inet sütunu boş kalıyordu). parentID>0 ise yanıt (Faz 36).
func AddComment(ctx context.Context, pool *pgxpool.Pool, contentID int64, authorName, body, clientIP string, parentID int64) error {
	_, err := pool.Exec(ctx, `INSERT INTO comments (content_id, author_name, body, ip, parent_id)
		VALUES ($1,$2,$3, NULLIF($4,'')::inet, NULLIF($5,0))`, contentID, authorName, body, clientIP, parentID)
	return err
}

func LikeComment(ctx context.Context, pool *pgxpool.Pool, commentID int64, clientIP string) (likes int, newVote bool, err error) {
	if clientIP == "" {
		return 0, false, nil
	}
	tag, err := pool.Exec(ctx, `INSERT INTO comment_likes (comment_id, ip) VALUES ($1, $2::inet) ON CONFLICT DO NOTHING`, commentID, clientIP)
	if err != nil {
		return 0, false, err
	}
	if tag.RowsAffected() > 0 {
		err = pool.QueryRow(ctx, `UPDATE comments SET likes = likes + 1 WHERE id = $1 RETURNING likes`, commentID).Scan(&likes)
		if err != nil {
			return 0, false, err
		}
		return likes, true, nil
	}
	err = pool.QueryRow(ctx, `SELECT likes FROM comments WHERE id = $1`, commentID).Scan(&likes)
	if err != nil {
		return 0, false, err
	}
	return likes, false, nil
}

// CountCommentsSince aynı IP'den belirli sürede atılan yorum sayısı (Faz 35 —
// yorum hız sınırı; ip boşsa saymaz).
func CountCommentsSince(ctx context.Context, pool *pgxpool.Pool, clientIP string, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM comments
		WHERE ip = NULLIF($1,'')::inet AND created_at >= $2`, clientIP, since).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ListComments duruma ve siteye göre yorumları (içerik başlığıyla) döndürür.
func ListComments(ctx context.Context, pool *pgxpool.Pool, status string, siteID int64) ([]CommentRow, error) {
	return ListCommentsAra(ctx, pool, status, siteID, "")
}

func ListCommentsAra(ctx context.Context, pool *pgxpool.Pool, status string, siteID int64, ara string) ([]CommentRow, error) {
	q := `SELECT cm.id, cm.content_id, c.title, cm.author_name, cm.body, cm.status, cm.created_at
	      FROM comments cm JOIN contents c ON c.id = cm.content_id WHERE 1=1`
	args := []any{}
	if siteID > 0 {
		args = append(args, siteID)
		q += fmt.Sprintf(` AND c.site_id = $%d`, len(args))
	}
	if status != "hepsi" && status != "" {
		args = append(args, status)
		q += fmt.Sprintf(` AND cm.status = $%d`, len(args))
	}
	if ara != "" {
		args = append(args, "%"+ara+"%")
		n := len(args)
		q += fmt.Sprintf(` AND (
			lower(public.immutable_unaccent(COALESCE(cm.author_name,''))) LIKE lower(public.immutable_unaccent($%d))
		 OR lower(public.immutable_unaccent(COALESCE(cm.body,''))) LIKE lower(public.immutable_unaccent($%d))
		 OR lower(public.immutable_unaccent(COALESCE(c.title,''))) LIKE lower(public.immutable_unaccent($%d))
		)`, n, n, n)
	}
	q += ` ORDER BY cm.created_at DESC LIMIT 200`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentRow
	for rows.Next() {
		var r CommentRow
		var t time.Time
		if err := rows.Scan(&r.ID, &r.ContentID, &r.ContentTitle, &r.AuthorName, &r.Body, &r.Status, &t); err != nil {
			return nil, err
		}
		r.CreatedAt = t
		out = append(out, r)
	}
	return out, rows.Err()
}

// ApprComment yorumu onaylar.
func ApprComment(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `UPDATE comments SET status = 'onayli' WHERE id = $1`, id)
	return err
}

// ApproveComments çoklu yorumu tek sorguda onaylar; etkilenen sayı döner (Faz 33).
func ApproveComments(ctx context.Context, pool *pgxpool.Pool, ids []int64) (int64, error) {
	tag, err := pool.Exec(ctx, `UPDATE comments SET status = 'onayli' WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteComments çoklu yorumu tek sorguda siler; etkilenen sayı döner (Faz 33).
func DeleteComments(ctx context.Context, pool *pgxpool.Pool, ids []int64) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM comments WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteComment yorumu siler.
func DeleteComment(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// CommentRow yorum listesi satırıdır.
type CommentRow struct {
	ID           int64
	ContentID    int64
	ContentTitle string
	AuthorName   string
	Body         string
	Status       string
	CreatedAt    time.Time
}
