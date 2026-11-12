package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User veritabanı kullanıcı kaydı.
type User struct {
	ID               int64
	Name             string
	Email            string
	Role             string
	Status           string
	PasswordHash     string
	TwoFactorEnabled bool
	TwoFactorSecret  *string
}

// GetUserByEmail e-posta ile kullanıcı döndürür.
func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (User, error) {
	var u User
	err := pool.QueryRow(ctx,
		`SELECT id, name, email, role, status, password_hash, two_factor_enabled, two_factor_secret FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.PasswordHash, &u.TwoFactorEnabled, &u.TwoFactorSecret)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// GetUserByID id ile kullanıcı döndürür (parola dahil değil).
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, id int64) (User, error) {
	var u User
	err := pool.QueryRow(ctx,
		`SELECT id, name, email, role, status, password_hash, two_factor_enabled, two_factor_secret FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.PasswordHash, &u.TwoFactorEnabled, &u.TwoFactorSecret)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// UpdateUser2FA 2FA durumunu günceller.
func UpdateUser2FA(ctx context.Context, pool *pgxpool.Pool, id int64, secret *string, enabled bool) error {
	_, err := pool.Exec(ctx, `UPDATE users SET two_factor_secret = $1, two_factor_enabled = $2 WHERE id = $3`,
		secret, enabled, id)
	return err
}

// CreateUser yeni kullanıcı oluşturur ve id döndürür.
func CreateUser(ctx context.Context, pool *pgxpool.Pool, name, email, role, passwordHash string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO users (name, email, role, password_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, email, role, passwordHash,
	).Scan(&id)
	if err == nil && id > 0 {
		// Profil adresi için slug (Faz 51).
		_, _ = EnsurePersonSlug(ctx, pool, id, name)
	}
	return id, err
}

// UpdateUserPassword kullanıcı parolasını günceller.
func UpdateUserPassword(ctx context.Context, pool *pgxpool.Pool, id int64, hash string) error {
	_, err := pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, hash, id)
	return err
}

func UpdateUserRole(ctx context.Context, pool *pgxpool.Pool, id int64, role string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE users SET role = $1 WHERE id = $2`, role, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateMembership üyelik planını günceller (” = üye değil, 'premium' = üyelik).
func UpdateMembership(ctx context.Context, pool *pgxpool.Pool, id int64, plan string) error {
	_, err := pool.Exec(ctx, `UPDATE users SET membership = $1 WHERE id = $2`, plan, id)
	return err
}

// MemberRow panel üyelik satırıdır.
type MemberRow struct {
	ID         int64
	Name       string
	Email      string
	Role       string
	Membership string
	// Katman bilgisi (abonelik modeli).
	TierName  string
	TierColor string
	Until     string
	Cycle     string
}

// ListUsers kullanıcıları döndürür (üyelik yönetimi için).
func ListUsers(ctx context.Context, pool *pgxpool.Pool) ([]MemberRow, error) {
	return ListUsersAra(ctx, pool, "")
}

func ListUsersAra(ctx context.Context, pool *pgxpool.Pool, ara string) ([]MemberRow, error) {
	q := `
		SELECT u.id, u.name, u.email, u.role, u.membership,
		       COALESCE(mt.name,''), COALESCE(mt.color_hex,''),
		       to_char(u.membership_expires_at, 'YYYY-MM-DD'),
		       COALESCE(u.membership_billing_cycle,'')
		FROM users u
		LEFT JOIN membership_tiers mt ON mt.id = u.membership_tier_id`
	args := []any{}
	if ara != "" {
		args = append(args, "%"+ara+"%")
		q += ` WHERE (
			lower(public.immutable_unaccent(COALESCE(u.name,''))) LIKE lower(public.immutable_unaccent($1))
		 OR lower(public.immutable_unaccent(COALESCE(u.email,''))) LIKE lower(public.immutable_unaccent($1))
		 OR lower(public.immutable_unaccent(COALESCE(u.role,''))) LIKE lower(public.immutable_unaccent($1))
		)`
	}
	q += ` ORDER BY u.id`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberRow
	for rows.Next() {
		var m MemberRow
		var until *string
		if err := rows.Scan(&m.ID, &m.Name, &m.Email, &m.Role, &m.Membership,
			&m.TierName, &m.TierColor, &until, &m.Cycle); err != nil {
			return nil, err
		}
		if until != nil {
			m.Until = *until
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
