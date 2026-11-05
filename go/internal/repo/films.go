package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Sinema / film listesi (spec 02 §217) ----

// Film bir sinema filmi kaydı (hafta bazlı).
type Film struct {
	ID          int64
	SiteID      int64
	Cinema      string
	Title       string
	Genre       string
	DurationMin int
	PosterURL   string
	TrailerURL  string
	Synopsis    string
	WeekStart   string
	Sort        int
}

// UpsertFilm filmi ekler/günceller (site+cinema+title+week benzersiz olacak şekilde aranır).
func UpsertFilm(ctx context.Context, pool *pgxpool.Pool, f *Film) error {
	// aynı site+sinema+başlık+hafta varsa güncelle, yoksa ekle
	tag, err := pool.Exec(ctx, `
		UPDATE films SET genre=$1, duration_min=$2, poster_url=$3, trailer_url=$4, synopsis=$5, sort=$6
		WHERE site_id=$7 AND cinema=$8 AND title=$9 AND week_start=$10::date`,
		f.Genre, f.DurationMin, f.PosterURL, f.TrailerURL, f.Synopsis, f.Sort, f.SiteID, f.Cinema, f.Title, f.WeekStart)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO films (site_id, cinema, title, genre, duration_min, poster_url, trailer_url, synopsis, week_start, sort)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::date,$10)`,
		f.SiteID, f.Cinema, f.Title, f.Genre, f.DurationMin, f.PosterURL, f.TrailerURL, f.Synopsis, f.WeekStart, f.Sort)
	return err
}

// ListFilms verilen hafta (boşsa en son hafta) filmlerini döndürür.
func ListFilms(ctx context.Context, pool *pgxpool.Pool, siteID int64, week string) ([]Film, string, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, site_id, cinema, title, COALESCE(genre,''), COALESCE(duration_min,0),
		       COALESCE(poster_url,''), COALESCE(trailer_url,''), COALESCE(synopsis,''), week_start::text, sort
		FROM films
		WHERE site_id=$1 AND week_start = COALESCE(NULLIF($2,'')::date,
		       (SELECT max(week_start) FROM films WHERE site_id=$1))
		ORDER BY cinema, sort, title`, siteID, week)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var out []Film
	last := ""
	for rows.Next() {
		var f Film
		if err := rows.Scan(&f.ID, &f.SiteID, &f.Cinema, &f.Title, &f.Genre, &f.DurationMin, &f.PosterURL, &f.TrailerURL, &f.Synopsis, &f.WeekStart, &f.Sort); err != nil {
			return nil, "", err
		}
		last = f.WeekStart
		out = append(out, f)
	}
	return out, last, rows.Err()
}

// FilmWeeks mevcut haftaları (en yeni önce) döndürür.
func FilmWeeks(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT week_start::text FROM films WHERE site_id=$1 ORDER BY week_start DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteFilm filmi siler.
func DeleteFilm(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM films WHERE id=$1`, id)
	return err
}

// CountFilms toplam film sayısı.
func CountFilms(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT count(*) FROM films WHERE site_id=$1`, siteID).Scan(&n)
	return n, err
}
