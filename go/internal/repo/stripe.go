package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Stripe üyelik işlemleri ----

// FindUserByEmailID e-posta ile kullanıcı arar (varsa).
func FindUserByEmailID(ctx context.Context, pool *pgxpool.Pool, email string) (int64, bool, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email::text) = lower($1)`, email).Scan(&id)
	if err != nil {
		return 0, false, nil //nolint:nilerr // bulunamama hata değildir
	}
	return id, true, nil
}

// CreateReaderUser Stripe ile üye olan okuyucu için şifresiz kullanıcı açar.
// (Giriş büyülü bağlantı ile yapılır; şifre alanı rastgele doldurulur.)
func CreateReaderUser(ctx context.Context, pool *pgxpool.Pool, email, name string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, status, membership)
		VALUES ($1, lower($2), '!', 'Üye', 'aktif', '')
		ON CONFLICT (email) DO UPDATE SET updated_at = now()
		RETURNING id`, name, email).Scan(&id)
	return id, err
}

// SetUserStripeIDs Stripe müşteri/abonelik kimliklerini yazar.
func SetUserStripeIDs(ctx context.Context, pool *pgxpool.Pool, userID int64, customerID, subscriptionID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET
			stripe_customer_id = CASE WHEN $2 <> '' THEN $2 ELSE stripe_customer_id END,
			stripe_subscription_id = CASE WHEN $3 <> '' THEN $3 ELSE stripe_subscription_id END,
			updated_at = now()
		WHERE id = $1`, userID, customerID, subscriptionID)
	return err
}

// SetUserMembershipByStripe üyelik durumunu Stripe sonucuna göre günceller.
// premium: 'premium' veya ” (bitiş tarihi ile birlikte).
func SetUserMembershipByStripe(ctx context.Context, pool *pgxpool.Pool, userID int64, plan string, until *time.Time) error {
	// Abonelik modeliyle uyum: premium açılırken en yüksek ağırlıklı aktif
	// katman atanır; içerik bazlı katman kilidi Stripe abonesini dışarıda
	// bırakmaz. Üyelik kaldırılınca katman/süre alanları temizlenir.
	var tierID *int64
	if plan == "premium" {
		var id int64
		if pool.QueryRow(ctx, `SELECT id FROM membership_tiers
			WHERE is_active = TRUE ORDER BY weight DESC, id DESC LIMIT 1`).Scan(&id) == nil {
			tierID = &id
		}
	}
	_, err := pool.Exec(ctx, `
		UPDATE users SET
			membership = $2::text,
			membership_until = $3::timestamptz,
			membership_tier_id =
				CASE WHEN $2::text = 'premium' THEN COALESCE($4::bigint, membership_tier_id) ELSE NULL END,
			membership_expires_at =
				CASE WHEN $2::text = 'premium' THEN $3::timestamptz ELSE NULL END,
			membership_billing_cycle =
				CASE WHEN $2::text = 'premium' THEN COALESCE(membership_billing_cycle, 'monthly') ELSE NULL END,
			updated_at = now()
		WHERE id = $1`, userID, plan, until, tierID)
	return err
}

// FindUserByStripeCustomer Stripe müşteri kimliğiyle kullanıcı arar.
func FindUserByStripeCustomer(ctx context.Context, pool *pgxpool.Pool, customerID string) (int64, bool, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&id)
	if err != nil {
		return 0, false, nil //nolint:nilerr
	}
	return id, true, nil
}

// MarkStripeEventSeen webhook olayını idempotent kaydeder.
// false dönerse olay daha önce işlenmiştir.
func MarkStripeEventSeen(ctx context.Context, pool *pgxpool.Pool, eventID string) (bool, error) {
	tag, err := pool.Exec(ctx, `
		INSERT INTO stripe_events (id) VALUES ($1)
		ON CONFLICT (id) DO NOTHING`, eventID)
	if err != nil {
		return false, err
	}
	// Saklama: 30 günden eski olay kayıtları (yeniden deneme penceresi
	// dışında) temizlenir; tablo sınırsız büyümesin.
	_, _ = pool.Exec(ctx, `DELETE FROM stripe_events WHERE received_at < now() - interval '30 days'`)
	return tag.RowsAffected() > 0, nil
}

// DeleteUser kullanıcıyı siler (KVKK). İçeriğe bağlıysa veritabanı hatası
// döner; çağıran bu durumu kullanıcıya bildirmelidir.
func DeleteUser(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
