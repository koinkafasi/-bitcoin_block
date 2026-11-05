package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Bookmarks (spec 06 K-G / 01 §L7) ----

// ToggleBookmark kaydı açar/kapatır; yeni durumı ve toplamı döndürür.
func ToggleBookmark(ctx context.Context, pool *pgxpool.Pool, userID, contentID int64) (saved bool, total int, err error) {
	res, err := pool.Exec(ctx, `DELETE FROM bookmarks WHERE user_id=$1 AND content_id=$2`, userID, contentID)
	if err != nil {
		return false, 0, err
	}
	if res.RowsAffected() > 0 {
		err = pool.QueryRow(ctx, `SELECT count(*) FROM bookmarks WHERE user_id=$1`, userID).Scan(&total)
		return false, total, err
	}
	if _, err = pool.Exec(ctx, `INSERT INTO bookmarks (user_id, content_id) VALUES ($1,$2)
		ON CONFLICT DO NOTHING`, userID, contentID); err != nil {
		return false, 0, err
	}
	err = pool.QueryRow(ctx, `SELECT count(*) FROM bookmarks WHERE user_id=$1`, userID).Scan(&total)
	return true, total, err
}

// ListBookmarks kullanıcının kaydettiği içerikleri döndürür.
func ListBookmarks(ctx context.Context, pool *pgxpool.Pool, userID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT ` + contentCols + ` FROM bookmarks b
		JOIN contents c ON c.id = b.content_id
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE b.user_id = $1 AND c.status='yayinda'
		ORDER BY b.created_at DESC LIMIT $2`
	rows, err := pool.Query(ctx, q, userID, limit)
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

// IsBookmarked içeriğin kullanıcı için kayıtlı olup olmadığını döndürür.
func IsBookmarked(ctx context.Context, pool *pgxpool.Pool, userID, contentID int64) bool {
	var n int
	if pool.QueryRow(ctx, `SELECT count(*) FROM bookmarks WHERE user_id=$1 AND content_id=$2`, userID, contentID).Scan(&n) != nil {
		return false
	}
	return n > 0
}

// ---- Follow (kategori/etiket/yazar) ----

// Follow takip girdisidir.
type Follow struct {
	ID        int64
	Type      string
	Slug      string
	TargetID  *int64
	CreatedAt time.Time
}

// ToggleFollow takibi açar/kapatır; yeni durumu döndürür.
func ToggleFollow(ctx context.Context, pool *pgxpool.Pool, userID int64, targetType, slug string, targetID *int64) (bool, error) {
	var cat any
	if targetID != nil {
		cat = *targetID
	}
	res, err := pool.Exec(ctx, `DELETE FROM follows WHERE user_id=$1 AND target_type=$2 AND target_slug=$3`,
		userID, targetType, slug)
	if err != nil {
		return false, err
	}
	if res.RowsAffected() > 0 {
		return false, nil
	}
	_, err = pool.Exec(ctx, `INSERT INTO follows (user_id, target_type, target_id, target_slug) VALUES ($1,$2,$3,$4)
		ON CONFLICT DO NOTHING`, userID, targetType, cat, slug)
	return err == nil, err
}

// ListFollows kullanıcının takip listesini döndürür.
func ListFollows(ctx context.Context, pool *pgxpool.Pool, userID int64) ([]Follow, error) {
	rows, err := pool.Query(ctx, `SELECT id, target_type, target_slug, target_id, created_at
		FROM follows WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Follow
	for rows.Next() {
		var f Follow
		if err := rows.Scan(&f.ID, &f.Type, &f.Slug, &f.TargetID, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// IsFollowing takip durumunu döndürür.
func IsFollowing(ctx context.Context, pool *pgxpool.Pool, userID int64, targetType, slug string) bool {
	var n int
	if pool.QueryRow(ctx, `SELECT count(*) FROM follows WHERE user_id=$1 AND target_type=$2 AND target_slug=$3`,
		userID, targetType, slug).Scan(&n) != nil {
		return false
	}
	return n > 0
}

// ---- Bildirimler ----

// Notification bildirim kaydıdır.
type Notification struct {
	ID        int64
	Kind      string
	Title     string
	Body      string
	URL       string
	IsRead    bool
	CreatedAt time.Time
}

// AddNotification bildirim ekler.
func AddNotification(ctx context.Context, pool *pgxpool.Pool, userID int64, kind, title, body, url string) error {
	_, err := pool.Exec(ctx, `INSERT INTO notifications (user_id, kind, title, body, url) VALUES ($1,$2,$3,$4,$5)`,
		userID, kind, title, body, url)
	return err
}

// ListNotifications kullanıcının bildirimlerini döndürür.
func ListNotifications(ctx context.Context, pool *pgxpool.Pool, userID int64, limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `SELECT id, kind, title, body, url, is_read, created_at
		FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.URL, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// UnreadNotifications okunmamış bildirim sayısı.
func UnreadNotifications(ctx context.Context, pool *pgxpool.Pool, userID int64) int {
	var n int
	if pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id=$1 AND is_read=FALSE`, userID).Scan(&n) != nil {
		return 0
	}
	return n
}

// MarkNotificationsRead tüm bildirimleri okunur işaretler.
func MarkNotificationsRead(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	_, err := pool.Exec(ctx, `UPDATE notifications SET is_read=TRUE WHERE user_id=$1 AND is_read=FALSE`, userID)
	return err
}

// HasNotificationToday aynı türde bugün bildirim gönderilmiş mi?
// (günlük hatırlatma/düşme bildirimlerinin tekrarını önler)
func HasNotificationToday(ctx context.Context, pool *pgxpool.Pool, userID int64, kind string) bool {
	var n int
	if pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE user_id=$1 AND kind=$2 AND created_at::date = CURRENT_DATE`, userID, kind).Scan(&n) != nil {
		return false
	}
	return n > 0
}

// ---- Gamification (puan/seri/rozet) ----

// UserPoints kullanıcı puan durumudur.
type UserPoints struct {
	UserID       int64
	Points       int
	Streak       int
	LastActive   *time.Time
	Achievements []string
}

// Achievement puan eşiğine bağlı rozet tanımı.
type Achievement struct {
	Key    string
	Ad     string
	Min    int
	Streak int
}

var achievementTable = []Achievement{
	{"okuyucu", "Okuyucu", 10, 0},
	{"okumak", "Okumak", 50, 0},
	{"deneyimli", "Deneyimli Okur", 150, 0},
	{"sadik", "Sadık Okur", 400, 0},
	{"seri3", "3 Günlük Seri", 0, 3},
	{"seri7", "7 Günlük Seri", 0, 7},
	{"seri30", "30 Günlük Seri", 0, 30},
}

// Achievements rozet tanım tablosu.
func Achievements() []Achievement { return achievementTable }

// TouchActivity kullanıcı etkinliğini işler: seri güncelle, puan ekle, yeni rozetleri döndürür.
func TouchActivity(ctx context.Context, pool *pgxpool.Pool, userID int64, addPoints int) (*UserPoints, []string, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var up UserPoints
	err := pool.QueryRow(ctx, `INSERT INTO user_points (user_id, points, streak, last_active)
		VALUES ($1, $2, 1, CURRENT_DATE)
		ON CONFLICT (user_id) DO UPDATE SET
			points = user_points.points + $2,
			streak = CASE
				WHEN user_points.last_active = CURRENT_DATE THEN user_points.streak
				WHEN user_points.last_active = CURRENT_DATE - 1 THEN user_points.streak + 1
				ELSE 1 END,
			last_active = CURRENT_DATE,
			updated_at = now()
		RETURNING user_id, points, streak, last_active, achievements`,
		userID, addPoints).Scan(&up.UserID, &up.Points, &up.Streak, &up.LastActive, &up.Achievements)
	if err != nil {
		return nil, nil, err
	}
	_ = today
	have := map[string]bool{}
	for _, a := range up.Achievements {
		have[a] = true
	}
	var gained []string
	for _, a := range achievementTable {
		if have[a.Key] {
			continue
		}
		if (a.Min > 0 && up.Points >= a.Min) || (a.Streak > 0 && up.Streak >= a.Streak) {
			up.Achievements = append(up.Achievements, a.Key)
			gained = append(gained, a.Ad)
		}
	}
	if len(gained) > 0 {
		if _, err := pool.Exec(ctx, `UPDATE user_points SET achievements=$1, updated_at=now() WHERE user_id=$2`,
			up.Achievements, userID); err != nil {
			return &up, gained, err
		}
		for _, g := range gained {
			_ = AddNotification(ctx, pool, userID, "rozet", "Yeni rozet: "+g, "Başarımlar sayfanı", "/hesabim/rozetler")
		}
	}
	return &up, gained, nil
}

// Leaderboard en çok puanlı kullanıcıları döndürür.
func Leaderboard(ctx context.Context, pool *pgxpool.Pool, limit int) ([]UserPoints, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `SELECT user_id, points, streak, last_active, achievements
		FROM user_points ORDER BY points DESC, streak DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserPoints
	for rows.Next() {
		var u UserPoints
		if err := rows.Scan(&u.UserID, &u.Points, &u.Streak, &u.LastActive, &u.Achievements); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
