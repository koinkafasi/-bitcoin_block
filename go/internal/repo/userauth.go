package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Public üye girihi: kayıt, şifre sıfırlama, e-posta onayı ----

// ErrTokenGecersiz sıfırlama/onay token'ı geçersiz veya kullanılmış.
var ErrTokenGecersiz = errors.New("bağlantı geçersiz veya süresi dolmuş")

// CreateReader kayıtlı okuyucu oluşturur (parola hash'i hazır gelir).
// E-posta benzersizdir; çakışmada ErrEmailKayitli döner.
func CreateReader(ctx context.Context, pool *pgxpool.Pool, name, email, passwordHash string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, status, terms_accepted_at)
		VALUES ($1, lower($2), $3, 'Üye', 'aktif', now())
		RETURNING id`, name, email, passwordHash).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailKayitli
		}
		return 0, err
	}
	return id, nil
}

// ErrEmailKayitli aynı e-posta ile ikinci hesap açılamaz.
var ErrEmailKayitli = errors.New("bu e-posta zaten kayıtlı")

// FindUserForLogin giriş için kullanıcı arar (e-posta).
func FindUserForLogin(ctx context.Context, pool *pgxpool.Pool, login string) (id int64, hash, role, status string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT id, password_hash, role, status FROM users
		WHERE email = lower($1)`, login).
		Scan(&id, &hash, &role, &status)
	return
}

// TouchLastLogin son giriş zamanını günceller.
func TouchLastLogin(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	_, err := pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID)
	return err
}

// SetPassword parola özeti değiştirir.
func SetPassword(ctx context.Context, pool *pgxpool.Pool, userID int64, hash string) error {
	_, err := pool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, hash)
	return err
}

// NewPasswordResetToken sıfırlama token'ı üretir (ham token döner; yalnız
// özeti saklanır).ömür: 30 dakika.
func NewPasswordResetToken(ctx context.Context, pool *pgxpool.Pool, userID int64) (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b[:])
	sum := sha256.Sum256([]byte("pw:" + token))
	_, err := pool.Exec(ctx, `
		INSERT INTO password_resets (user_id, token_hash, expires_at)
		VALUES ($1, $2, now() + interval '30 minutes')`, userID, hex.EncodeToString(sum[:]))
	if err != nil {
		return "", err
	}
	// Eski/tek kullanılmış kayıtları temizle.
	_, _ = pool.Exec(ctx, `DELETE FROM password_resets WHERE expires_at < now() - interval '1 day'`)
	return token, nil
}

// ConsumePasswordResetToken token'ı doğrular, tüketir ve kullanıcıyı döndürür.
func ConsumePasswordResetToken(ctx context.Context, pool *pgxpool.Pool, token string) (int64, error) {
	sum := sha256.Sum256([]byte("pw:" + token))
	var id int64
	err := pool.QueryRow(ctx, `
		UPDATE password_resets SET used_at = now()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, hex.EncodeToString(sum[:])).Scan(&id)
	if err != nil {
		return 0, ErrTokenGecersiz
	}
	return id, nil
}

// PasswordResetTokenGecerli token'ı tüketmeden doğrular. GET /sifre-sifirla
// bu sayede süresi dolmuş bağlantıda boş bir form göstermiyor.
func PasswordResetTokenGecerli(ctx context.Context, pool *pgxpool.Pool, token string) bool {
	if token == "" {
		return false
	}
	sum := sha256.Sum256([]byte("pw:" + token))
	var one int
	err := pool.QueryRow(ctx, `
		SELECT 1 FROM password_resets
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		LIMIT 1`, hex.EncodeToString(sum[:])).Scan(&one)
	return err == nil && one == 1
}

// SetEmailVerifyToken e-posta onay token'ı üretip saklar (ham token döner).
func SetEmailVerifyToken(ctx context.Context, pool *pgxpool.Pool, userID int64) (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b[:])
	sum := sha256.Sum256([]byte("ev:" + token))
	_, err := pool.Exec(ctx, `UPDATE users SET email_verify_token = $2 WHERE id = $1`,
		userID, hex.EncodeToString(sum[:]))
	return token, err
}

// VerifyEmail token ile e-postayı doğrular.
func VerifyEmail(ctx context.Context, pool *pgxpool.Pool, token string) (int64, error) {
	sum := sha256.Sum256([]byte("ev:" + token))
	var id int64
	err := pool.QueryRow(ctx, `
		UPDATE users SET email_verified_at = coalesce(email_verified_at, now()), email_verify_token = NULL
		WHERE email_verify_token = $1
		RETURNING id`, hex.EncodeToString(sum[:])).Scan(&id)
	if err != nil {
		return 0, ErrTokenGecersiz
	}
	return id, nil
}

func LoginRateLimited(ctx context.Context, pool *pgxpool.Pool, email, ip string, max int) (bool, error) {
	var n int
	var err error
	if ip == "" {
		err = pool.QueryRow(ctx, `
			SELECT count(*) FROM login_attempts
			WHERE email = lower($1) AND attempted_at > now() - interval '10 minutes'`,
			email).Scan(&n)
	} else {
		err = pool.QueryRow(ctx, `
			SELECT count(*) FROM login_attempts
			WHERE email = lower($1) AND ip = $2
			  AND attempted_at > now() - interval '10 minutes'`,
			email, ip).Scan(&n)
	}
	if err != nil {
		return false, err
	}
	return n >= max, nil
}

// RecordLoginAttempt denemeyi kaydeder (IP ile birlikte).
func RecordLoginAttempt(ctx context.Context, pool *pgxpool.Pool, email, ip string, ok bool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO login_attempts (email, ip, success) VALUES (lower($1), $2, $3)`,
		email, ip, ok)
	if err == nil && ok {
		// başarılı girişte bu hesabın deneme kayıtlarını temizle
		_, _ = pool.Exec(ctx, `DELETE FROM login_attempts WHERE email = lower($1)`, email)
	}
	return err
}

// PurgeLoginAttempts eski deneme kayıtlarını temizler (saatlik iş).
func PurgeLoginAttempts(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DELETE FROM login_attempts WHERE attempted_at < now() - interval '1 day'`)
	return err
}
