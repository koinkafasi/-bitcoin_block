package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Bülten aboneleri (spec 06 K-F / 01 §L6) ----

// Subscriber bülten abonesidir.
type Subscriber struct {
	ID          int64
	SiteID      int64
	Email       string
	Locale      string
	Status      string
	Token       string
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}

func subToken(email string) string {
	h := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(email)) + "|" + time.Now().Format("20060102150405")))
	return hex.EncodeToString(h[:16])
}

// CreateSubscriber bülten aboneliği açar (çift kayıt → mevcut döner).
func CreateSubscriber(ctx context.Context, pool *pgxpool.Pool, siteID int64, email, locale string) (*Subscriber, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrNotFound
	}
	if locale == "" {
		locale = "tr"
	}
	var s Subscriber
	err := pool.QueryRow(ctx, `
		INSERT INTO subscribers (site_id, email, locale, token)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (site_id, email, locale)
		DO UPDATE SET status = 'onay bekliyor'
		RETURNING id, site_id, email, locale, status, token, created_at, confirmed_at`,
		siteID, email, locale, subToken(email)).Scan(&s.ID, &s.SiteID, &s.Email, &s.Locale, &s.Status, &s.Token, &s.CreatedAt, &s.ConfirmedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ConfirmSubscriber çift kayıt (double opt-in) onayı.
func ConfirmSubscriber(ctx context.Context, pool *pgxpool.Pool, token string) error {
	_, err := pool.Exec(ctx, `UPDATE subscribers SET status='aktif', confirmed_at=now() WHERE token=$1`, token)
	return err
}

// UnsubscribeSubscriber abonelikten çıkar.
func UnsubscribeSubscriber(ctx context.Context, pool *pgxpool.Pool, token string) error {
	_, err := pool.Exec(ctx, `UPDATE subscribers SET status='iptal' WHERE token=$1`, token)
	return err
}

// SubscriberByToken token ile abone getirir.
func SubscriberByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Subscriber, error) {
	var s Subscriber
	err := pool.QueryRow(ctx, `SELECT id, site_id, email, locale, status, token, created_at, confirmed_at
		FROM subscribers WHERE token=$1`, token).Scan(&s.ID, &s.SiteID, &s.Email, &s.Locale, &s.Status, &s.Token, &s.CreatedAt, &s.ConfirmedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CountSubscribers duruma göre abone sayısı.
func CountSubscribers(ctx context.Context, pool *pgxpool.Pool, siteID int64) map[string]int {
	out := map[string]int{}
	rows, err := pool.Query(ctx, `SELECT status, count(*) FROM subscribers WHERE site_id=$1 GROUP BY status`, siteID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if rows.Scan(&st, &n) == nil {
			out[st] = n
		}
	}
	return out
}

// ---- Günlük özet / podcast (spec 06 K-E) ----

// Digest günlük özet kaydıdır.
type Digest struct {
	ID         int64
	SiteID     int64
	Date       time.Time
	Locale     string
	Kind       string
	Title      string
	Summary    string
	ContentIDs []int64
	WordCount  int
	AudioPath  *string
	Status     string
}

// BuildDailyDigest verilen gün için yayınlanan içeriklerden günlük özet üretir/kaydeder.
// AI kullanılamıyorsa ilk satır özeti üretir (mock fallback).
func BuildDailyDigest(ctx context.Context, pool *pgxpool.Pool, siteID int64, day time.Time, locale string) (*Digest, error) {
	if locale == "" {
		locale = "tr"
	}
	items, err := RecentPublishedOn(ctx, pool, siteID, locale, day)
	if err != nil {
		return nil, err
	}
	var ids []int64
	var titles []string
	words := 0
	for _, it := range items {
		ids = append(ids, it.ID)
		titles = append(titles, it.Title)
		words += len([]rune(it.Spot))
		if words > 400 {
			words = 400
		}
	}
	summary := "Bugün yayınlanan haberler."
	if len(titles) > 0 {
		summary = strings.Join(titles, " · ")
		if len([]rune(summary)) > 400 {
			summary = string([]rune(summary)[:400]) + "…"
		}
	}
	title := day.Format("2 Ocak 2006") + " Günlük Özeti"
	if locale != "tr" {
		title = day.Format("2006-01-02") + " Daily Digest"
	}
	d := &Digest{SiteID: siteID, Date: day, Locale: locale, Kind: "gunluk",
		Title: title + " (" + itoa(len(ids)) + " haber)", Summary: summary, ContentIDs: ids,
		WordCount: words, Status: "yayinda"}
	err = pool.QueryRow(ctx, `
		INSERT INTO digests (site_id, digest_date, locale, kind, title, summary, content_ids, word_count, status)
		VALUES ($1,$2::date,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (site_id, digest_date, locale, kind)
		DO UPDATE SET title=EXCLUDED.title, summary=EXCLUDED.summary,
			content_ids=EXCLUDED.content_ids, word_count=EXCLUDED.word_count
		RETURNING id`, siteID, day, locale, d.Kind, d.Title, d.Summary, d.ContentIDs, d.WordCount, d.Status).Scan(&d.ID)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// DigestByDate günlük özeti getirir.
func DigestByDate(ctx context.Context, pool *pgxpool.Pool, siteID int64, day time.Time, locale, kind string) (*Digest, error) {
	var d Digest
	err := pool.QueryRow(ctx, `
		SELECT id, site_id, digest_date, locale, kind, title, summary, content_ids, word_count, audio_path, status
		FROM digests WHERE site_id=$1 AND digest_date=$2::date AND locale=$3 AND kind=$4`,
		siteID, day, locale, kind).Scan(&d.ID, &d.SiteID, &d.Date, &d.Locale, &d.Kind, &d.Title, &d.Summary,
		&d.ContentIDs, &d.WordCount, &d.AudioPath, &d.Status)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDigests son N günün özetlerini döndürür.
func ListDigests(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale, kind string, limit int) ([]Digest, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, digest_date, locale, kind, title, summary, content_ids, word_count, audio_path, status
		FROM digests WHERE site_id=$1 AND locale=$2 AND kind=$3 ORDER BY digest_date DESC LIMIT $4`,
		siteID, locale, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Digest
	for rows.Next() {
		var d Digest
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Date, &d.Locale, &d.Kind, &d.Title, &d.Summary,
			&d.ContentIDs, &d.WordCount, &d.AudioPath, &d.Status); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetDigestAudio günlük özete ses dosyası bağlar (podcast).
// NOT: `kind` değiştirilmez — unique kısıtı (site,tarih,dil,kind) çakışması olurdu.
// Yalnız henüz ses bağlanmamış satırlar güncellenir (eşzamanlı üretim koruması).
func SetDigestAudio(ctx context.Context, pool *pgxpool.Pool, d Digest) error {
	res, err := pool.Exec(ctx, `UPDATE digests SET audio_path=$1
		WHERE id=$2 AND (audio_path IS NULL OR audio_path='')`,
		d.AudioPath, d.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return nil // başka bir üretim zaten bağlamış
	}
	return nil
}

// RecentPublishedOn belirli gün (YYYY-AA-GG) yayınlananları döndürür.
func RecentPublishedOn(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, day time.Time) ([]Content, error) {
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.status='yayinda' AND c.locale=$1 AND c.site_id=$2
		  AND c.published_at >= $3::date AND c.published_at < $3::date + interval '1 day'
		ORDER BY c.published_at ASC`
	rows, err := pool.Query(ctx, q, locale, siteID, day.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
