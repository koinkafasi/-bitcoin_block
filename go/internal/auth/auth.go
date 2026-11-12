// Package auth parola (bcrypt) ve oturum (DB session) yönetimi.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	// SessionLifetime çerez/oturum ömrü.
	SessionLifetime = 30 * 24 * time.Hour
	// CookieName oturum çerezi adı.
	CookieName = "tb_session"
	// ScopePanel yönetim paneli oturumu (personel).
	ScopePanel = "panel"
	// ScopeUye okuyucu/üye oturumu.
	ScopeUye = "uye"
)

var (
	ErrInvalidCreds = errors.New("e-posta veya şifre hatalı")
	ErrNoSession    = errors.New("oturum yok")
)

// HashPassword bcrypt ile parola özeti üretir.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword düz parolayı özetle karşılaştırır.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID int64, userAgent string) (string, error) {
	return CreateSessionScoped(ctx, pool, userID, userAgent, ScopePanel)
}

// CreateSessionScoped kapsamı açıkça belirtilen oturum açar.
func CreateSessionScoped(ctx context.Context, pool *pgxpool.Pool, userID int64, userAgent, scope string) (string, error) {
	if scope != ScopeUye {
		scope = ScopePanel
	}
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, expires_at, user_agent, scope)
		 VALUES ($1, now() + $2::interval, $3, $4) RETURNING id`,
		userID, SessionLifetime.String(), userAgent, scope,
	).Scan(&id)
	return id, err
}

// SessionUser oturum id'sinden kullanıcıyı döndürür.
func SessionUser(ctx context.Context, pool *pgxpool.Pool, sessionID string) (User, error) {
	var u User
	err := pool.QueryRow(ctx, `
		SELECT u.id, u.name, u.email, u.role, u.status, coalesce(u.membership, ''),
		       coalesce(s.scope, 'panel')
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > now()`,
		sessionID,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.Membership, &u.Scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNoSession
	}
	return u, err
}

func RevokeUserSessions(ctx context.Context, pool *pgxpool.Pool, userID int64, keepScope string) (int64, error) {
	var (
		tag pgconn.CommandTag
		err error
	)
	if keepScope == "" {
		tag, err = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	} else {
		tag, err = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1 AND scope <> $2`, userID, keepScope)
	}
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteSession oturumu sonlandırır (logout).
func DeleteSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

// CleanupExpired bayat oturumları temizler (scheduler'daki periyodik iş).
func CleanupExpired(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return tag.RowsAffected(), err
}

// User oturum bağlamında taşınan kullanıcı özeti.
type User struct {
	ID         int64
	Name       string
	Email      string
	Role       string
	Status     string
	Membership string // '' | 'premium' (Stripe aboneliği)
	Scope      string // 'panel' | 'uye' — oturumun açıldığı kapı
}

// StaffRoles parola + 2FA ile giriş yapan editoryal roller.
// (Okuyucu hesapları Google ile giriş yapabilir; bunlar yapamaz.)
var StaffRoles = map[string]bool{
	"Yönetici": true, "Editör": true, "Muhabir": true, "Yazar": true,
}

// Can role'un verilen asgari rolün üzerinde olduğunu kontrol eder.
// Yetki hiyerarşisi: Yönetici > Editör > Muhabir > Yazar > Üye.
func (u User) Can(minRole string) bool {
	return roleRank(u.Role) >= roleRank(minRole)
}

func roleRank(r string) int {
	switch r {
	case "Yönetici":
		return 5
	case "Editör":
		return 4
	case "Muhabir":
		return 3
	case "Yazar":
		return 2
	case "Üye":
		return 1
	default:
		return 0
	}
}
