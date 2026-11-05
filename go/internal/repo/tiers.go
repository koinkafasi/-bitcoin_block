package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Abonelik katmanları (membership_tiers) ----

// Tier satılan abonelik katmanı.
type Tier struct {
	ID           int64
	Slug         string
	Name         string
	Description  string
	ColorHex     string
	Weight       int
	MonthlyPrice float64
	AnnualPrice  float64
	Perks        []string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ListTiers katmanları ağırlık sırasıyla döndürür.
func ListTiers(ctx context.Context, pool *pgxpool.Pool, activeOnly bool) ([]Tier, error) {
	q := `SELECT id, slug, name, description, color_hex, weight, monthly_price,
	             annual_price, perks, is_active, created_at, updated_at
	      FROM membership_tiers`
	if activeOnly {
		q += ` WHERE is_active = TRUE`
	}
	q += ` ORDER BY weight ASC, id ASC`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tier
	for rows.Next() {
		t, err := scanTier(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTier(row interface{ Scan(...any) error }) (Tier, error) {
	var t Tier
	var perks []byte
	err := row.Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &t.ColorHex, &t.Weight,
		&t.MonthlyPrice, &t.AnnualPrice, &perks, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if len(perks) > 0 {
		_ = json.Unmarshal(perks, &t.Perks)
	}
	return t, nil
}

// GetTier kimlikle katman döndürür.
func GetTier(ctx context.Context, pool *pgxpool.Pool, id int64) (Tier, error) {
	row := pool.QueryRow(ctx, `SELECT id, slug, name, description, color_hex, weight,
		monthly_price, annual_price, perks, is_active, created_at, updated_at
		FROM membership_tiers WHERE id = $1`, id)
	t, err := scanTier(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tier{}, ErrNotFound
	}
	return t, err
}

// GetTierBySlug slug ile katman döndürür.
func GetTierBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (Tier, error) {
	row := pool.QueryRow(ctx, `SELECT id, slug, name, description, color_hex, weight,
		monthly_price, annual_price, perks, is_active, created_at, updated_at
		FROM membership_tiers WHERE slug = $1`, slug)
	t, err := scanTier(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tier{}, ErrNotFound
	}
	return t, err
}

// SaveTier katmanı ekler/günceller (ID=0 ise ekler).
func SaveTier(ctx context.Context, pool *pgxpool.Pool, t Tier) (int64, error) {
	perks, err := json.Marshal(t.Perks)
	if err != nil {
		return 0, err
	}
	if t.ID > 0 {
		_, err := pool.Exec(ctx, `
			UPDATE membership_tiers SET slug=$2, name=$3, description=$4, color_hex=$5,
			       weight=$6, monthly_price=$7, annual_price=$8, perks=$9, is_active=$10,
			       updated_at=now()
			WHERE id=$1`,
			t.ID, t.Slug, t.Name, t.Description, t.ColorHex, t.Weight,
			t.MonthlyPrice, t.AnnualPrice, perks, t.IsActive)
		return t.ID, err
	}
	var id int64
	err = pool.QueryRow(ctx, `
		INSERT INTO membership_tiers (slug, name, description, color_hex, weight,
		       monthly_price, annual_price, perks, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		t.Slug, t.Name, t.Description, t.ColorHex, t.Weight,
		t.MonthlyPrice, t.AnnualPrice, perks, t.IsActive).Scan(&id)
	return id, err
}

// DeleteTier katmanı siler (abonelerin kademesi NULL'a düşer).
func DeleteTier(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM membership_tiers WHERE id = $1`, id)
	return err
}

// UserTier kullanıcının AKTİF katmanını döndürür (yoksa ok=false).
func UserTier(ctx context.Context, pool *pgxpool.Pool, userID int64) (Tier, bool) {
	row := pool.QueryRow(ctx, `
		SELECT mt.id, mt.slug, mt.name, mt.description, mt.color_hex, mt.weight,
		       mt.monthly_price, mt.annual_price, mt.perks, mt.is_active,
		       mt.created_at, mt.updated_at
		FROM users u
		JOIN membership_tiers mt ON mt.id = u.membership_tier_id
		WHERE u.id = $1 AND mt.is_active = TRUE
		  AND (u.membership_expires_at IS NULL OR u.membership_expires_at > now())`, userID)
	t, err := scanTier(row)
	if err != nil {
		return Tier{}, false
	}
	return t, true
}

// UserTierWeight kullanıcının erişim ağırlığı (0 = ücretsiz).
func UserTierWeight(ctx context.Context, pool *pgxpool.Pool, userID int64) int {
	t, ok := UserTier(ctx, pool, userID)
	if !ok {
		return 0
	}
	return t.Weight
}

// SetUserTier kullanıcıya katman atar (nil = ücretsize düşür).
func SetUserTier(ctx context.Context, pool *pgxpool.Pool, userID int64, tierID *int64,
	cycle string, expiresAt *time.Time) error {
	var membership string
	if tierID != nil {
		membership = "premium"
	}
	_, err := pool.Exec(ctx, `
		UPDATE users SET membership_tier_id=$2, membership_billing_cycle=NULLIF($3,''),
		       membership_expires_at=$4, membership=$5, updated_at=now()
		WHERE id=$1`, userID, tierID, cycle, expiresAt, membership)
	return err
}

// ActivateSubscription ödeme sonrası kademeyi atar. Aynı kademede yenileme
// mevcut bitişin ÜZERİNE eklenir (edirnegercek davranışı).
func ActivateSubscription(ctx context.Context, pool *pgxpool.Pool, userID, tierID int64,
	cycle string) (*time.Time, error) {
	tier, err := GetTier(ctx, pool, tierID)
	if err != nil {
		return nil, err
	}
	days := 30
	if cycle == "annual" {
		days = 365
	}
	var base time.Time
	err = pool.QueryRow(ctx, `
		UPDATE users SET
			membership_tier_id = $2,
			membership_billing_cycle = $3,
			membership = 'premium',
			membership_expires_at =
				CASE WHEN membership_tier_id = $2 AND membership_expires_at > now()
					THEN membership_expires_at + make_interval(days => $4)
					ELSE now() + make_interval(days => $4)
				END,
			updated_at = now()
		WHERE id = $1
		RETURNING membership_expires_at`,
		userID, tier.ID, cycle, days).Scan(&base)
	if err != nil {
		return nil, err
	}
	return &base, nil
}

// ExtendSubscription süreyi elle uzatır (admin).
func ExtendSubscription(ctx context.Context, pool *pgxpool.Pool, userID int64, days int) (*time.Time, error) {
	var until time.Time
	err := pool.QueryRow(ctx, `
		UPDATE users SET
			membership_expires_at = GREATEST(COALESCE(membership_expires_at, now()), now()) + make_interval(days => $2),
			updated_at = now()
		WHERE id = $1 AND membership_tier_id IS NOT NULL
		RETURNING membership_expires_at`, userID, days).Scan(&until)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &until, err
}

// ExpireSubscriptions süresi dolan aboneleri ücretsize düşürür ve düşen
// kullanıcıların listesini döndürür (bildirim için).
func ExpireSubscriptions(ctx context.Context, pool *pgxpool.Pool) ([]int64, error) {
	rows, err := pool.Query(ctx, `
		UPDATE users SET membership_tier_id = NULL, membership_billing_cycle = NULL,
		       membership_cancel_at = NULL, membership = '', updated_at = now()
		WHERE membership_tier_id IS NOT NULL
		  AND membership_expires_at IS NOT NULL AND membership_expires_at < now()
		RETURNING id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// ExpiringSoon bitişi belirtilen gün içinde olan aboneleri döndürür.
func ExpiringSoon(ctx context.Context, pool *pgxpool.Pool, days int) ([]int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT id FROM users
		WHERE membership_tier_id IS NOT NULL
		  AND membership_cancel_at IS NULL
		  AND membership_expires_at IS NOT NULL
		  AND membership_expires_at::date = (now() + make_interval(days => $1))::date`,
		days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// ---- Ödeme kayıtları (subscription_payments) ----

// Payment ödeme kaydı.
type Payment struct {
	ID              int64
	UserID          int64
	UserName        string
	TierID          *int64
	TierName        string
	Provider        string
	ConversationID  string
	ProviderPayment string
	Amount          float64
	Currency        string
	BillingCycle    string
	Status          string
	Discount        float64
	CouponCode      string
	CreatedAt       time.Time
}

// CreatePayment bekleyen ödeme kaydı açar (indirim/kupon opsiyonel).
func CreatePayment(ctx context.Context, pool *pgxpool.Pool, userID int64, tierID *int64,
	provider, conversationID string, amount, discount float64, currency, cycle, coupon string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO subscription_payments
			(user_id, tier_id, provider, conversation_id, amount, discount, currency,
			 billing_cycle, coupon_code, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),'pending') RETURNING id`,
		userID, tierID, provider, conversationID, amount, discount, currency, cycle, coupon).Scan(&id)
	return id, err
}

// CompletePayment ödemeyi başarılı işaretler.
func CompletePayment(ctx context.Context, pool *pgxpool.Pool, conversationID, providerPaymentID string, raw []byte) error {
	_, err := pool.Exec(ctx, `
		UPDATE subscription_payments
		SET status='success', provider_payment_id=$2, raw_response=$3, updated_at=now()
		WHERE conversation_id=$1`, conversationID, providerPaymentID, raw)
	return err
}

// SetPaymentToken iyzico checkout token'ını kayda yazar.
func SetPaymentToken(ctx context.Context, pool *pgxpool.Pool, conversationID, token string) error {
	_, err := pool.Exec(ctx,
		`UPDATE subscription_payments SET token=$2, updated_at=now() WHERE conversation_id=$1`,
		conversationID, token)
	return err
}

// GetPaymentByToken iyzico callback token'ı ile ödeme bulur.
func GetPaymentByToken(ctx context.Context, pool *pgxpool.Pool, token string) (Payment, error) {
	var p Payment
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, COALESCE(tier_id,0), COALESCE(provider,''), COALESCE(conversation_id,''),
		       COALESCE(provider_payment_id,''), amount, discount, COALESCE(coupon_code,''),
		       currency, billing_cycle, status, created_at
		FROM subscription_payments WHERE token=$1`, token).
		Scan(&p.ID, &p.UserID, &p.TierID, &p.Provider, &p.ConversationID,
			&p.ProviderPayment, &p.Amount, &p.Discount, &p.CouponCode,
			&p.Currency, &p.BillingCycle, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	return p, err
}

// FailPayment ödemeyi başarısız işaretler.
func FailPayment(ctx context.Context, pool *pgxpool.Pool, conversationID string, raw []byte) error {
	_, err := pool.Exec(ctx, `
		UPDATE subscription_payments SET status='failed', raw_response=$2, updated_at=now()
		WHERE conversation_id=$1`, conversationID, raw)
	return err
}

// GetPaymentConversation conversation_id ile ödeme bulur.
func GetPaymentConversation(ctx context.Context, pool *pgxpool.Pool, conversationID string) (Payment, error) {
	var p Payment
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, COALESCE(tier_id,0), COALESCE(provider,''), COALESCE(conversation_id,''),
		       COALESCE(provider_payment_id,''), amount, discount, COALESCE(coupon_code,''),
		       currency, billing_cycle, status, created_at
		FROM subscription_payments WHERE conversation_id=$1`, conversationID).
		Scan(&p.ID, &p.UserID, &p.TierID, &p.Provider, &p.ConversationID,
			&p.ProviderPayment, &p.Amount, &p.Discount, &p.CouponCode,
			&p.Currency, &p.BillingCycle, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	return p, err
}

// ListPayments son ödemeleri (kullanıcı/katman adlarıyla) döndürür.
func ListPayments(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Payment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.user_id, u.name, p.tier_id, COALESCE(mt.name,''),
		       p.provider, COALESCE(p.conversation_id,''), COALESCE(p.provider_payment_id,''),
		       p.amount, p.currency, p.billing_cycle, p.status, p.created_at
		FROM subscription_payments p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN membership_tiers mt ON mt.id = p.tier_id
		ORDER BY p.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		var p Payment
		var tierID *int64
		if err := rows.Scan(&p.ID, &p.UserID, &p.UserName, &tierID, &p.TierName,
			&p.Provider, &p.ConversationID, &p.ProviderPayment, &p.Amount,
			&p.Currency, &p.BillingCycle, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.TierID = tierID
		out = append(out, p)
	}
	return out, rows.Err()
}

// SubscriptionStats abonelik panosu sayıları.
type SubscriptionStats struct {
	ActiveSubs  int
	TierCounts  map[string]int
	Revenue30   float64
	RevenueYear float64
	Failed30    int
}

// SubscriptionSummary ödeme panosu için özet üretir.
func SubscriptionSummary(ctx context.Context, pool *pgxpool.Pool) (SubscriptionStats, error) {
	st := SubscriptionStats{TierCounts: map[string]int{}}
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users WHERE membership_tier_id IS NOT NULL
			   AND (membership_expires_at IS NULL OR membership_expires_at > now())),
			(SELECT COALESCE(sum(amount),0) FROM subscription_payments
			   WHERE status='success' AND created_at > now() - interval '30 days'),
			(SELECT COALESCE(sum(amount),0) FROM subscription_payments
			   WHERE status='success' AND created_at > now() - interval '365 days'),
			(SELECT count(*) FROM subscription_payments
			   WHERE status='failed' AND created_at > now() - interval '30 days')`).
		Scan(&st.ActiveSubs, &st.Revenue30, &st.RevenueYear, &st.Failed30)
	if err != nil {
		return st, err
	}
	rows, err := pool.Query(ctx, `
		SELECT mt.name, count(*) FROM users u
		JOIN membership_tiers mt ON mt.id = u.membership_tier_id
		WHERE u.membership_expires_at IS NULL OR u.membership_expires_at > now()
		GROUP BY mt.name, mt.weight ORDER BY mt.weight`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err == nil {
			st.TierCounts[name] = n
		}
	}
	return st, rows.Err()
}
