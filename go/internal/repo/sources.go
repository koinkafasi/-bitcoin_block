package repo

import (
	"context"
	"io"
	neturl "net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
)

// ContentSource içerik kanıt/source kaydıdır.
type ContentSource struct {
	ID          int64
	ContentID   int64
	URL         string
	Title       *string
	BodySnippet string
	Accessible  bool
	StatusCode  *int
	CheckedAt   *time.Time
}

// AddContentSource kaynak ekler (title/body boşsa bağlanıp doldurur).
func AddContentSource(ctx context.Context, pool *pgxpool.Pool, contentID int64, url string) (int64, error) {
	snippet, st, accessible := probeSource(url)
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO content_sources (content_id, url, body_snippet, accessible, status_code, checked_at)
		VALUES ($1,$2,$3,$4,$5, now()) RETURNING id`,
		contentID, url, snippet, accessible, st).Scan(&id)
	return id, err
}

// ListContentSources kaynakları döndürür.
func ListContentSources(ctx context.Context, pool *pgxpool.Pool, contentID int64) ([]ContentSource, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, content_id, url, title, body_snippet, accessible, status_code, checked_at
		FROM content_sources WHERE content_id = $1 ORDER BY id`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentSource
	for rows.Next() {
		var c ContentSource
		if err := rows.Scan(&c.ID, &c.ContentID, &c.URL, &c.Title, &c.BodySnippet, &c.Accessible, &c.StatusCode, &c.CheckedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteContentSource kaynağı siler.
func DeleteContentSource(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM content_sources WHERE id = $1`, id)
	return err
}

// CheckSources tüm kaynakları yeniden yoklar (erişilebilirlik + snippet).
func CheckSources(ctx context.Context, pool *pgxpool.Pool, contentID int64) (int, error) {
	srcs, err := ListContentSources(ctx, pool, contentID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, src := range srcs {
		snippet, st, ok := probeSource(src.URL)
		_, _ = pool.Exec(ctx, `
			UPDATE content_sources SET body_snippet=$1, accessible=$2, status_code=$3, checked_at=now() WHERE id=$4`,
			snippet, ok, st, src.ID)
		if ok {
			n++
		}
	}
	return n, nil
}

func probeSource(url string) (snippet string, statusCode *int, accessible bool) {
	u, err := neturl.Parse(url)
	if err != nil || platform.ValidateOutboundURL(u) != nil {
		return "", nil, false
	}
	client := platform.SafeFetchClient(6 * time.Second)
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, false
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code < 200 || code >= 400 {
		return "", &code, false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return truncateRunes(string(body), 1600), &code, true
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// OriginalityIndex içerik metniyle kaynaklar arası token benzerliğinden özgünlük indeksi (0-100).
func OriginalityIndex(ctx context.Context, pool *pgxpool.Pool, contentID int64, text string) (int, int, error) {
	srcs, err := ListContentSources(ctx, pool, contentID)
	if err != nil {
		return 100, 0, err
	}
	if len(srcs) == 0 {
		return 100, 0, nil
	}
	tt := tokenSet(strings.ToLower(text))
	maxOverlap := 0.0
	for _, src := range srcs {
		if src.BodySnippet == "" {
			continue
		}
		st := tokenSet(strings.ToLower(src.BodySnippet))
		inter := 0
		for w := range tt {
			if st[w] {
				inter++
			}
		}
		if len(tt) == 0 {
			continue
		}
		ov := float64(inter) / float64(len(tt)) * 100
		if ov > maxOverlap {
			maxOverlap = ov
		}
	}
	index := 100 - int(maxOverlap+0.5)
	if index < 0 {
		index = 0
	}
	return index, len(srcs), nil
}
