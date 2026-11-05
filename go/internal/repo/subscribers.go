package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ActiveSubscribers siteye ait onaylı (aktif) bülten abonelerini döndürür.
func ActiveSubscribers(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Subscriber, error) {
	if limit <= 0 {
		limit = 2000
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, email, locale, status, token, created_at, confirmed_at
		FROM subscribers WHERE site_id=$1 AND status='aktif'
		ORDER BY confirmed_at NULLS LAST, id LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.ID, &s.SiteID, &s.Email, &s.Locale, &s.Status, &s.Token, &s.CreatedAt, &s.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListSubscribers site bülten abonelerini durum filtresiyle döndürür (boş = tümü).
func ListSubscribers(ctx context.Context, pool *pgxpool.Pool, siteID int64, status string, limit int) ([]Subscriber, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, email, locale, status, token, created_at, confirmed_at
		FROM subscribers WHERE site_id=$1 AND ($2='' OR status=$2)
		ORDER BY id DESC LIMIT $3`, siteID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(&s.ID, &s.SiteID, &s.Email, &s.Locale, &s.Status, &s.Token, &s.CreatedAt, &s.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SetSubscriberStatus abone durumunu id ile günceller (aktif/iptal).
func SetSubscriberStatus(ctx context.Context, pool *pgxpool.Pool, id int64, status string) error {
	_, err := pool.Exec(ctx, `UPDATE subscribers SET status=$1 WHERE id=$2`, status, id)
	return err
}

// DeleteSubscriber abone kaydını siler (KVKK silme talebi).
func DeleteSubscriber(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM subscribers WHERE id=$1`, id)
	return err
}
