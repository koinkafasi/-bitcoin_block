package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Üye hesap merkezi (/hesabim) ----

// Member üye hesabının genel görünümü.
type Member struct {
	ID                     int64
	Name                   string
	Email                  string
	Role                   string
	Membership             string
	MembershipUntil        *time.Time
	MembershipExpiresAt    *time.Time
	MembershipBillingCycle string
	MembershipCancelAt     *time.Time
	Title                  string
	Bio                    string
	Slug                   string
	AvatarMediaID          *int64
	Socials                map[string]string
	EmailVerifiedAt        *time.Time
	CreatedAt              time.Time
	StripeSubID            string
	LastLoginAt            *time.Time
}

// MemberStats üye merkezi özet sayaçları.
type MemberStats struct {
	Bookmarks int
	Follows   int
	Comments  int
	Unread    int
	Points    int
	Streak    int
}

// MemberComment üyenin yorumu (içerik başlığıyla).
type MemberComment struct {
	ID           int64
	Body         string
	Status       string
	CreatedAt    time.Time
	Likes        int
	ContentID    int64
	ContentSlug  string
	ContentTitle string
}

// GetMember üyeyi profil alanlarıyla birlikte okur.
func GetMember(ctx context.Context, pool *pgxpool.Pool, id int64) (Member, error) {
	var m Member
	var socials []byte
	var title, bio, slug *string
	var stripeSub *string
	err := pool.QueryRow(ctx, `
		SELECT id, name, email, role, membership, membership_until,
		       membership_expires_at, COALESCE(membership_billing_cycle,''), membership_cancel_at,
		       COALESCE(title,''), COALESCE(bio,''), COALESCE(slug,''),
		       avatar_media_id, socials, email_verified_at, created_at,
		       stripe_subscription_id, last_login_at
		FROM users WHERE id = $1`, id).
		Scan(&m.ID, &m.Name, &m.Email, &m.Role, &m.Membership, &m.MembershipUntil,
			&m.MembershipExpiresAt, &m.MembershipBillingCycle, &m.MembershipCancelAt,
			&title, &bio, &slug, &m.AvatarMediaID, &socials, &m.EmailVerifiedAt,
			&m.CreatedAt, &stripeSub, &m.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	if err != nil {
		return Member{}, err
	}
	m.Title = derefStr(title)
	m.Bio = derefStr(bio)
	m.Slug = derefStr(slug)
	m.StripeSubID = derefStr(stripeSub)
	m.Socials = map[string]string{}
	if len(socials) > 0 {
		_ = json.Unmarshal(socials, &m.Socials)
	}
	return m, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// UpdateMemberProfile profil alanlarını günceller. Avatar nil ise korunur,
// avatarSet true + nil ise avatar kaldırılır.
func UpdateMemberProfile(ctx context.Context, pool *pgxpool.Pool, id int64, name, title, bio, slug string,
	socials map[string]string) error {
	raw, err := json.Marshal(socials)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE users SET name = $2, title = NULLIF($3,''), bio = NULLIF($4,''),
		       slug = NULLIF($5,''), socials = $6, updated_at = now()
		WHERE id = $1`,
		id, name, strings.TrimSpace(title), strings.TrimSpace(bio), strings.TrimSpace(slug), raw)
	return err
}

// SetMemberAvatar avatar medyasını bağlar (nil = kaldır).
func SetMemberAvatar(ctx context.Context, pool *pgxpool.Pool, id int64, mediaID *int64) error {
	_, err := pool.Exec(ctx, `UPDATE users SET avatar_media_id = $2, updated_at = now() WHERE id = $1`, id, mediaID)
	return err
}

// MemberStatsOf üyenin sayaçlarını döndürür.
func MemberStatsOf(ctx context.Context, pool *pgxpool.Pool, id int64) (MemberStats, error) {
	var s MemberStats
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM bookmarks WHERE user_id = $1),
			(SELECT count(*) FROM follows WHERE user_id = $1),
			(SELECT count(*) FROM comments WHERE user_id = $1),
			(SELECT count(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE)`, id).
		Scan(&s.Bookmarks, &s.Follows, &s.Comments, &s.Unread)
	if err != nil {
		return s, err
	}
	err = pool.QueryRow(ctx, `SELECT points, streak FROM user_points WHERE user_id = $1`, id).
		Scan(&s.Points, &s.Streak)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return s, err
}

// MemberComments üyenin son yorumlarını (yayındaki içeriklerle) döndürür.
func MemberComments(ctx context.Context, pool *pgxpool.Pool, id int64, limit int) ([]MemberComment, error) {
	if limit <= 0 {
		limit = 40
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.body, c.status, c.created_at, c.likes,
		       ct.id, COALESCE(ct.slug,''), ct.title
		FROM comments c
		JOIN contents ct ON ct.id = c.content_id
		WHERE c.user_id = $1 AND ct.status = 'yayinda'
		ORDER BY c.created_at DESC LIMIT $2`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberComment
	for rows.Next() {
		var mc MemberComment
		if err := rows.Scan(&mc.ID, &mc.Body, &mc.Status, &mc.CreatedAt, &mc.Likes,
			&mc.ContentID, &mc.ContentSlug, &mc.ContentTitle); err != nil {
			return nil, err
		}
		out = append(out, mc)
	}
	return out, rows.Err()
}

// DeleteMember hesabı ve kişisel verileri siler (KVKK). İçerik/yorum gibi
// editoryal kayıtlar anonimleştirilir, oturum ve bildirimler silinir.
func DeleteMember(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Yorumlar korunur ama kişiye bağlanmaz.
	if _, err := tx.Exec(ctx, `UPDATE comments SET user_id = NULL WHERE user_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
