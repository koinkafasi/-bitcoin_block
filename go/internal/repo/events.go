package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Etkinlikler (spec 01 §B2, 02 §207) ----

// Event takvim etkinliği (hatırlatma şeridi için).
type Event struct {
	ID           int64
	SiteID       int64
	Title        string
	Description  string
	Location     string
	URL          string
	CategoryID   int64
	CategoryName *string
	StartAt      time.Time
	EndAt        *time.Time
	RemindDays   []int
	Status       string
	CreatedByID  *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const eventCols = `e.id, e.site_id, e.title, COALESCE(e.description,''), COALESCE(e.location,''),
	COALESCE(e.url,''), COALESCE(e.category_id,0), e.start_at, e.end_at, COALESCE(e.remind_days,'{}'), e.status,
	e.created_at, e.updated_at, cat.name`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.SiteID, &e.Title, &e.Description, &e.Location, &e.URL,
		&e.CategoryID, &e.StartAt, &e.EndAt, &e.RemindDays, &e.Status, &e.CreatedAt, &e.UpdatedAt, &e.CategoryName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	return e, err
}

const eventSelect = `SELECT ` + eventCols + ` FROM events e
	LEFT JOIN categories cat ON cat.id = e.category_id`

// ListEvents siteye ait etkinlikleri döndürür (status boşsa hepsi).
func ListEvents(ctx context.Context, pool *pgxpool.Pool, siteID int64, status string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := pool.Query(ctx, eventSelect+`
		WHERE e.site_id=$1 AND ($2='' OR e.status=$2)
		ORDER BY e.start_at DESC LIMIT $3`, siteID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpcomingEvents bugünden itibaren yaklaşan aktif etkinlikler (start_at artan).
func UpcomingEvents(ctx context.Context, pool *pgxpool.Pool, siteID int64, days, limit int) ([]Event, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := pool.Query(ctx, eventSelect+`
		WHERE e.site_id=$1 AND e.status='aktif'
		  AND e.start_at >= now() - interval '1 day'
		  AND e.start_at <= now() + make_interval(days => $2)
		ORDER BY e.start_at ASC LIMIT $3`, siteID, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEvent tek etkinlik.
func GetEvent(ctx context.Context, pool *pgxpool.Pool, id int64) (Event, error) {
	return scanEvent(pool.QueryRow(ctx, eventSelect+` WHERE e.id=$1`, id))
}

// CreateEvent yeni etkinlik ekler.
func CreateEvent(ctx context.Context, pool *pgxpool.Pool, e *Event) (int64, error) {
	if e.Status == "" {
		e.Status = "aktif"
	}
	if e.RemindDays == nil {
		e.RemindDays = []int{1, 3, 7}
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO events (site_id, title, description, location, url, category_id, start_at, end_at, remind_days, status, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		e.SiteID, e.Title, e.Description, e.Location, e.URL, idOrNil(e.CategoryID), e.StartAt, e.EndAt, e.RemindDays, e.Status, e.CreatedByID).Scan(&id)
	return id, err
}

// UpdateEvent etkinliği günceller.
func UpdateEvent(ctx context.Context, pool *pgxpool.Pool, e *Event) error {
	if e.RemindDays == nil {
		e.RemindDays = []int{}
	}
	_, err := pool.Exec(ctx, `
		UPDATE events SET title=$1, description=$2, location=$3, url=$4, category_id=$5,
		       start_at=$6, end_at=$7, remind_days=$8, status=$9 WHERE id=$10`,
		e.Title, e.Description, e.Location, e.URL, idOrNil(e.CategoryID), e.StartAt, e.EndAt, e.RemindDays, e.Status, e.ID)
	return err
}

// DeleteEvent etkinliği siler.
func DeleteEvent(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM events WHERE id=$1`, id)
	return err
}

// idOrNil 0 ise NULL döndürür (nullable FK alanları için).
func idOrNil(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}
