package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Reklam yönetimi (spec 01 §E / 02 P2-4, 05 §3) ----

// AdSlot reklam yerleşimi (başlık bandı, kenar çubuğu, içerik içi…).
type AdSlot struct {
	ID     int64
	Key    string
	Label  string
	Width  int
	Height int
	Sticky bool
	Status string
}

type Ad struct {
	ID           int64
	Name         string
	Kind         string
	SlotKey      string
	HTML         string
	ImageMediaID *int64
	TargetURL    string
	StartsAt     *time.Time
	EndsAt       *time.Time
	Status       string
	Weight       int
	CreatedAt    time.Time
	// birleştirilmiş görünüm
	ImageURL string
	SiteID   int64
}

type AdStatsRow struct {
	AdID          int64
	Name          string
	Kind          string
	Views         int
	Clicks        int
	CTR           float64
	RevenueMicros int64
}

type AdStatsSummary struct {
	Rows          []AdStatsRow
	Views         int
	Clicks        int
	CTR           float64
	RevenueMicros int64
	EstTL         float64
}

// ListAdSlots yerleşimleri sıralı döndürür.
func ListAdSlots(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]AdSlot, error) {
	rows, err := pool.Query(ctx, `SELECT id, key, label, width, height, sticky, status
		FROM ad_slots WHERE site_id=$1 ORDER BY id`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdSlot
	for rows.Next() {
		var a AdSlot
		if err := rows.Scan(&a.ID, &a.Key, &a.Label, &a.Width, &a.Height, &a.Sticky, &a.Status); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAdSlot yerleşim ekler.
func CreateAdSlot(ctx context.Context, pool *pgxpool.Pool, siteID int64, key, label string, w, h int, sticky bool) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO ad_slots (site_id, key, label, width, height, sticky)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, siteID, key, label, w, h, sticky).Scan(&id)
	return id, err
}

// UpdateAdSlot yerleşimi günceller.
func UpdateAdSlot(ctx context.Context, pool *pgxpool.Pool, id int64, key, label string, w, h int, sticky bool, status string) error {
	_, err := pool.Exec(ctx, `UPDATE ad_slots SET key=$1, label=$2, width=$3, height=$4, sticky=$5, status=$6 WHERE id=$7`,
		key, label, w, h, sticky, status, id)
	return err
}

// DeleteAdSlot yerleşimi siler (reklam bağlıysa engellenir).
func DeleteAdSlot(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM ad_slots WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM ads WHERE slot_key=(SELECT key FROM ad_slots WHERE id=$1))`, id)
	return err
}

// ListAds reklamları döndürür (durum filtresiyle).
func ListAds(ctx context.Context, pool *pgxpool.Pool, siteID int64, status string) ([]Ad, error) {
	q := `SELECT a.id, a.name, a.kind, COALESCE(a.slot_key,''), COALESCE(a.html,''),
		       a.image_media_id, COALESCE(a.target_url,''), a.starts_at, a.ends_at,
		       a.status, a.weight, a.created_at,
		       COALESCE('/media/'||m.path,''), a.site_id
		FROM ads a LEFT JOIN media_files m ON m.id = a.image_media_id
		WHERE a.site_id=$1`
	args := []any{siteID}
	if status != "" && status != "tumu" {
		q += ` AND a.status=$2`
		args = append(args, status)
	}
	q += ` ORDER BY a.created_at DESC`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Ad
	for rows.Next() {
		var a Ad
		var img *int64
		var starts, ends *time.Time
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &a.SlotKey, &a.HTML, &img, &a.TargetURL,
			&starts, &ends, &a.Status, &a.Weight, &a.CreatedAt, &a.ImageURL, &a.SiteID); err != nil {
			return nil, err
		}
		a.ImageMediaID = img
		a.StartsAt = starts
		a.EndsAt = ends
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAd reklam ekler.
func CreateAd(ctx context.Context, pool *pgxpool.Pool, a *Ad) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO ads (site_id, name, kind, slot_key, html, image_media_id, target_url, starts_at, ends_at, status, weight)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		a.SiteID, a.Name, a.Kind, a.SlotKey, a.HTML, a.ImageMediaID, nullableText(&a.TargetURL),
		a.StartsAt, a.EndsAt, a.Status, a.Weight).Scan(&id)
	return id, err
}

// UpdateAd reklamı günceller.
func UpdateAd(ctx context.Context, pool *pgxpool.Pool, a *Ad) error {
	_, err := pool.Exec(ctx, `UPDATE ads SET name=$1, kind=$2, slot_key=$3, html=$4, image_media_id=$5,
		target_url=$6, starts_at=$7, ends_at=$8, status=$9, weight=$10, updated_at=now() WHERE id=$11`,
		a.Name, a.Kind, a.SlotKey, a.HTML, a.ImageMediaID, a.TargetURL, a.StartsAt, a.EndsAt, a.Status, a.Weight, a.ID)
	return err
}

// DeleteAd reklamı siler.
func DeleteAd(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM ads WHERE id=$1`, id)
	return err
}

// ActiveAds bugünkü tarih aralığında aktif olan reklamları slot anahtarına göre toplar.
func ActiveAds(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string][]Ad, error) {
	q := `SELECT a.id, a.name, a.kind, COALESCE(a.slot_key,''), COALESCE(a.html,''),
		       a.image_media_id, COALESCE(a.target_url,''), a.starts_at, a.ends_at,
		       a.status, a.weight, COALESCE('/media/'||m.path,''), a.site_id
		FROM ads a LEFT JOIN media_files m ON m.id = a.image_media_id
		WHERE a.site_id=$1 AND a.status='aktif'
		  AND (a.starts_at IS NULL OR a.starts_at <= now())
		  AND (a.ends_at IS NULL OR a.ends_at >= now())
		ORDER BY a.weight DESC, a.id`
	rows, err := pool.Query(ctx, q, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Ad{}
	for rows.Next() {
		var a Ad
		var img *int64
		var starts, ends *time.Time
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &a.SlotKey, &a.HTML, &img, &a.TargetURL,
			&starts, &ends, &a.Status, &a.Weight, &a.ImageURL, &a.SiteID); err != nil {
			return nil, err
		}
		a.ImageMediaID = img
		a.StartsAt = starts
		a.EndsAt = ends
		out[a.SlotKey] = append(out[a.SlotKey], a)
	}
	return out, rows.Err()
}

// RecordAdView reklam gösterimini artırır (US bump: views + $2).
func RecordAdView(ctx context.Context, pool *pgxpool.Pool, siteID, adID int64, n int) error {
	if n <= 0 {
		n = 1
	}
	_, err := pool.Exec(ctx, `INSERT INTO ad_stats (site_id, ad_id, stat_date, views)
		VALUES ($1,$2,CURRENT_DATE,$3)
		ON CONFLICT (ad_id, stat_date) DO UPDATE SET views = ad_stats.views + EXCLUDED.views`,
		siteID, adID, n)
	return err
}

// RecordAdClick reklam tıklamasını artırır.
func RecordAdClick(ctx context.Context, pool *pgxpool.Pool, siteID, adID int64) error {
	_, err := pool.Exec(ctx, `INSERT INTO ad_stats (site_id, ad_id, stat_date, clicks)
		VALUES ($1,$2,CURRENT_DATE,1)
		ON CONFLICT (ad_id, stat_date) DO UPDATE SET clicks = ad_stats.clicks + 1`,
		siteID, adID)
	return err
}

// AdTarget reklamın DB'de kayıtlı hedef URL'ini döndürür (yoksa boş dize).
// Tıklama yönlendirmesinde kullanıcı kontrollü parametre yerine bu güvenilir
// değer tercih edilir (açık yönlendirme/open-redirect koruması).
func AdTarget(ctx context.Context, pool *pgxpool.Pool, siteID, adID int64) (string, error) {
	var target string
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(target_url,'') FROM ads WHERE id=$1 AND site_id=$2`,
		adID, siteID).Scan(&target)
	if err != nil {
		return "", err
	}
	return target, nil
}

// AdStats birleşik performans özeti (son günler).
func AdStats(ctx context.Context, pool *pgxpool.Pool, siteID int64, days int) (*AdStatsSummary, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := pool.Query(ctx, `SELECT a.id, a.name, a.kind,
		       COALESCE(SUM(s.views),0)::bigint AS views, COALESCE(SUM(s.clicks),0)::bigint AS clicks,
		       COALESCE(SUM(s.revenue_micros),0)::bigint AS rev
		FROM ads a LEFT JOIN ad_stats s ON s.ad_id = a.id AND s.site_id=a.site_id
		    AND s.stat_date >= CURRENT_DATE - ($1::int - 1)
		WHERE a.site_id=$2
		GROUP BY a.id, a.name, a.kind ORDER BY views DESC`, days, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sum AdStatsSummary
	for rows.Next() {
		var r AdStatsRow
		var views, clicks, rev int64
		if err := rows.Scan(&r.AdID, &r.Name, &r.Kind, &views, &clicks, &rev); err != nil {
			return nil, err
		}
		r.Views = int(views)
		r.Clicks = int(clicks)
		r.RevenueMicros = rev
		if r.Views > 0 {
			r.CTR = float64(r.Clicks) * 100 / float64(r.Views)
		}
		sum.Rows = append(sum.Rows, r)
		sum.Views += r.Views
		sum.Clicks += r.Clicks
		sum.RevenueMicros += rev
	}
	if sum.Views > 0 {
		sum.CTR = float64(sum.Clicks) * 100 / float64(sum.Views)
	}
	// kural tabanlı gelir tahmini: %1 CTR × 10 TL bin gösterim
	sum.EstTL = float64(sum.Views) * 0.01 * 10 / 1000
	return &sum, rows.Err()
}
