package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
)

// TrendSource gündem RSS kaynağıdır.
type TrendSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// TrendItem gündem öğesidir.
type TrendItem struct {
	ID          int64
	Source      string
	Title       string
	URL         string
	Body        string
	PublishedAt *time.Time
	FetchedAt   time.Time
}

// GetTrendSources gündem/kaynaklar ayarını okur.
func GetTrendSources(ctx context.Context, pool *pgxpool.Pool) ([]TrendSource, error) {
	b, err := GetSetting(ctx, pool, "gundem", "kaynaklar")
	if err != nil {
		// varsayılan
		//nolint:nilerr // kaynak ayarı yoksa gömülü varsayılan kaynak kullanılır
		return []TrendSource{{Name: "AA Gündem", URL: "https://www.aa.com.tr/tr/rss/default?cat=gundem"}}, nil
	}
	var out []TrendSource
	_ = json.Unmarshal(b, &out)
	if len(out) == 0 {
		return []TrendSource{{Name: "AA Gündem", URL: "https://www.aa.com.tr/tr/rss/default?cat=gundem"}}, nil
	}
	return out, nil
}

// SaveTrendSources kaynakları kaydeder.
func SaveTrendSources(ctx context.Context, pool *pgxpool.Pool, srcs []TrendSource) error {
	b, _ := json.Marshal(srcs)
	return SetSetting(ctx, pool, "gundem", "kaynaklar", b)
}

// FetchTrends tüm kaynakları çeker, yeni öğeleri ekler, var olanları günceller.
func FetchTrends(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	srcs, err := GetTrendSources(ctx, pool)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, src := range srcs {
		if src.URL == "" || src.Name == "" {
			continue
		}
		items, err := FetchRSS(src.URL)
		if err != nil {
			continue
		}
		for _, it := range items {
			if it.Link == "" || it.Title == "" {
				continue
			}
			h := sha1.Sum([]byte(it.Link))
			hash := hex.EncodeToString(h[:])
			var pub *time.Time
			if !it.PubDate.IsZero() {
				pub = &it.PubDate
			}
			tag, err := pool.Exec(ctx, `
				INSERT INTO gundem_items (source, title, url, url_hash, body, published_at)
				VALUES ($1,$2,$3,$4,$5,$6)
				ON CONFLICT (url_hash) DO UPDATE SET
					title = EXCLUDED.title, body = EXCLUDED.body,
					published_at = COALESCE(gundem_items.published_at, EXCLUDED.published_at),
					fetched_at = now()`,
				src.Name, it.Title, it.Link, hash, it.Description, pub)
			if err != nil {
				continue
			}
			if tag.RowsAffected() == 1 {
				added++
			}
		}
	}
	return added, nil
}

// ListTrends son gündem öğelerini döndürür.
func ListTrends(ctx context.Context, pool *pgxpool.Pool, limit int) ([]TrendItem, error) {
	if limit <= 0 {
		limit = 40
	}
	rows, err := pool.Query(ctx, `
		SELECT id, source, title, url, body, published_at, fetched_at
		FROM gundem_items ORDER BY COALESCE(published_at, fetched_at) DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendItem
	for rows.Next() {
		var t TrendItem
		if err := rows.Scan(&t.ID, &t.Source, &t.Title, &t.URL, &t.Body, &t.PublishedAt, &t.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TrendToContent gündem öğesinden taslak içerik üretir.
func TrendToContent(ctx context.Context, pool *pgxpool.Pool, itemID int64, authorID int64) (int64, error) {
	var it TrendItem
	err := pool.QueryRow(ctx, `SELECT id, source, title, url, body FROM gundem_items WHERE id = $1`, itemID).
		Scan(&it.ID, &it.Source, &it.Title, &it.URL, &it.Body)
	if err != nil {
		return 0, err
	}
	spot := it.Body
	if len([]rune(spot)) > 240 {
		spot = string([]rune(spot)[:240])
	}
	// slug için başlıktan 4 char hash
	h := sha1.Sum([]byte(it.URL))
	slugSuf := hex.EncodeToString(h[:4])
	c := &Content{
		TypeKey:       "haber",
		Status:        "taslak",
		Title:         it.Title,
		Spot:          spot,
		Detail:        textToBlocksStmt(it.Body),
		Slug:          strPtr(platform.Slugify(it.Title) + "-" + slugSuf),
		AuthorID:      authorID,
		SourceURL:     strPtr(it.URL),
		SourceName:    strPtr(it.Source),
		CommentStatus: "acik",
		Locale:        "tr",
	}
	return CreateContent(ctx, pool, c)
}
