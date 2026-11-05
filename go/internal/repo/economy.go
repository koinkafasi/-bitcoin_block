package repo

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Abonelik ekonomisi (MRR/ARR/churn/LTV) ----

// Economy abonelik gelir özeti.
type Economy struct {
	MRR        float64
	ARR        float64
	ARPU       float64
	Churn30    float64
	LTV        float64
	ActiveSubs int
	NewSubs30  int
	Canceled30 int
	Renewal30  float64
	Refunded30 float64
}

// MonthRevenue aylık gelir noktası (grafik + tablo).
type MonthRevenue struct {
	Label  string // "2026-10"
	Short  string // "Eki"
	Amount float64
	Count  int
}

// RevenueTrend son N ayın başarılı ödeme toplamını (eskiden yeniye) döndürür.
// Ödeme olmayan aylar 0 ile doldurulur; grafik boşluk bırakmaz.
func RevenueTrend(ctx context.Context, pool *pgxpool.Pool, months int) ([]MonthRevenue, error) {
	if months <= 0 || months > 24 {
		months = 12
	}
	rows, err := pool.Query(ctx, `
		SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') AS ay,
		       COALESCE(sum(amount), 0), count(*)
		FROM subscription_payments
		WHERE status IN ('success', 'refunded')
		  AND created_at >= date_trunc('month', now()) - make_interval(months => $1 - 1)
		GROUP BY 1`, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bulunan := map[string]MonthRevenue{}
	for rows.Next() {
		var m MonthRevenue
		if err := rows.Scan(&m.Label, &m.Amount, &m.Count); err != nil {
			return nil, err
		}
		bulunan[m.Label] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Eksik ayları 0 ile tamamla (eskiden yeniye).
	out := make([]MonthRevenue, 0, months)
	simdi := time.Now()
	for i := months - 1; i >= 0; i-- {
		t := simdi.AddDate(0, -i, 0)
		label := t.Format("2006-01")
		m, ok := bulunan[label]
		if !ok {
			m = MonthRevenue{Label: label, Amount: 0, Count: 0}
		}
		m.Short = trMonthShort(t.Month())
		out = append(out, m)
	}
	return out, nil
}

// trMonthShort Türkçe kısa ay adı.
func trMonthShort(month time.Month) string {
	adlar := []string{"Oca", "Şub", "Mar", "Nis", "May", "Haz",
		"Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}
	if month < 1 || int(month) > len(adlar) {
		return ""
	}
	return adlar[month-1]
}

// SubscriptionEconomy panel için ekonomi metriklerini hesaplar.
func SubscriptionEconomy(ctx context.Context, pool *pgxpool.Pool) (Economy, error) {
	var e Economy
	var mrr float64
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(sum(
			CASE WHEN u.membership_billing_cycle = 'annual'
				THEN mt.annual_price / 12.0 ELSE mt.monthly_price END), 0),
		       count(*)
		FROM users u
		JOIN membership_tiers mt ON mt.id = u.membership_tier_id
		WHERE u.membership_expires_at IS NULL OR u.membership_expires_at > now()`).
		Scan(&mrr, &e.ActiveSubs)
	if err != nil {
		return e, err
	}
	e.MRR = round2(mrr)
	e.ARR = round2(mrr * 12)
	if e.ActiveSubs > 0 {
		e.ARPU = round2(mrr / float64(e.ActiveSubs))
	}
	// Son 30 günde ilk kez ödeme yapanlar (yeni abone).
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT user_id, min(created_at) AS ilk
			FROM subscription_payments WHERE status = 'success'
			GROUP BY user_id
		) t WHERE t.ilk > now() - interval '30 days'`).Scan(&e.NewSubs30)
	// Son 30 günde aboneliği düşenler (ödeme geçmişi olup kademesi boşalanlar).
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM users u
		WHERE u.membership_tier_id IS NULL AND u.updated_at > now() - interval '30 days'
		  AND EXISTS (SELECT 1 FROM subscription_payments p WHERE p.user_id = u.id AND p.status = 'success')`).
		Scan(&e.Canceled30)
	base := e.ActiveSubs + e.Canceled30
	if base > 0 {
		e.Churn30 = round2(float64(e.Canceled30) / float64(base) * 100)
	}
	// LTV = ARPU / aylık churn oranı (churn yoksa 12 aylık varsayım).
	if e.Churn30 > 0 {
		e.LTV = round2(e.ARPU / (e.Churn30 / 100))
	} else {
		e.LTV = round2(e.ARPU * 12)
	}
	// Son 30 günde yenileme: önceki ödemesi olan başarılı ödemeler / tümü.
	var renew, total int
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM subscription_payments q
			WHERE q.user_id = p.user_id AND q.status='success' AND q.created_at < p.created_at)),
		       count(*)
		FROM subscription_payments p
		WHERE p.status='success' AND p.created_at > now() - interval '30 days'`).Scan(&renew, &total)
	if total > 0 {
		e.Renewal30 = round2(float64(renew) / float64(total) * 100)
	}
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount),0) FROM subscription_payments
		WHERE status='refunded' AND refunded_at > now() - interval '30 days'`).Scan(&e.Refunded30)
	return e, nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ---- Dönem sonu iptal / iade ----

// CancelSubscriptionAtPeriodEnd aboneliği dönem sonunda iptal eder.
func CancelSubscriptionAtPeriodEnd(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	res, err := pool.Exec(ctx, `
		UPDATE users SET membership_cancel_at = now(), updated_at = now()
		WHERE id = $1 AND membership_tier_id IS NOT NULL`, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UndoCancellation iptal talebini geri alır.
func UndoCancellation(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	_, err := pool.Exec(ctx,
		`UPDATE users SET membership_cancel_at = NULL, updated_at = now() WHERE id = $1`, userID)
	return err
}

// RefundPayment ödemeyi iade edildi olarak işaretler.
func RefundPayment(ctx context.Context, pool *pgxpool.Pool, paymentID int64, note string) error {
	res, err := pool.Exec(ctx, `
		UPDATE subscription_payments
		SET status='refunded', refunded_at=now(), note=$2, updated_at=now()
		WHERE id=$1 AND status='success'`, paymentID, note)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReconcilePending 24 saatten eski "pending" ödemeleri başarısız işaretler.
// (sağlayıcı yanıtı kaybolduysa kuyruğu temiz tutar)
func ReconcilePending(ctx context.Context, pool *pgxpool.Pool, olderThanHours int) (int64, error) {
	if olderThanHours <= 0 {
		olderThanHours = 24
	}
	res, err := pool.Exec(ctx, `
		UPDATE subscription_payments
		SET status='failed', note='mutabakat: yanıt alınamadı', updated_at=now()
		WHERE status='pending' AND created_at < now() - ($1 || ' hours')::interval`,
		olderThanHours)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

// ---- Promosyon kodları ----

// Promo indirim kodu.
type Promo struct {
	ID          int64
	Code        string
	Description string
	Percent     int
	TierSlug    string
	MaxUses     int
	UsedCount   int
	ExpiresAt   *time.Time
	IsActive    bool
	CreatedAt   time.Time
}

// ErrPromoGecersiz kod geçersiz/süresi dolmuş/kullanımı bitmiş.
var ErrPromoGecersiz = errors.New("indirim kodu geçersiz veya süresi dolmuş")

// ListPromos tüm kodları döndürür.
func ListPromos(ctx context.Context, pool *pgxpool.Pool) ([]Promo, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, description, percent, COALESCE(tier_slug,''), max_uses,
		       used_count, expires_at, is_active, created_at
		FROM promo_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Promo
	for rows.Next() {
		var p Promo
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Percent, &p.TierSlug,
			&p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ValidPromo kodu doğrular (kullanım hakkı/süre/kademe).
func ValidPromo(ctx context.Context, pool *pgxpool.Pool, code, tierSlug string) (Promo, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return Promo{}, ErrPromoGecersiz
	}
	var p Promo
	err := pool.QueryRow(ctx, `
		SELECT id, code, description, percent, COALESCE(tier_slug,''), max_uses,
		       used_count, expires_at, is_active, created_at
		FROM promo_codes WHERE code = $1`, code).
		Scan(&p.ID, &p.Code, &p.Description, &p.Percent, &p.TierSlug, &p.MaxUses,
			&p.UsedCount, &p.ExpiresAt, &p.IsActive, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Promo{}, ErrPromoGecersiz
	}
	if err != nil {
		return Promo{}, err
	}
	if !p.IsActive || p.Percent <= 0 {
		return Promo{}, ErrPromoGecersiz
	}
	if p.ExpiresAt != nil && p.ExpiresAt.Before(time.Now()) {
		return Promo{}, ErrPromoGecersiz
	}
	if p.MaxUses > 0 && p.UsedCount >= p.MaxUses {
		return Promo{}, ErrPromoGecersiz
	}
	if p.TierSlug != "" && p.TierSlug != tierSlug {
		return Promo{}, ErrPromoGecersiz
	}
	return p, nil
}

// UsePromo kullanım sayacını artırır.
func UsePromo(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx,
		`UPDATE promo_codes SET used_count = used_count + 1 WHERE id = $1`, id)
	return err
}

// SavePromo kod ekler/günceller.
func SavePromo(ctx context.Context, pool *pgxpool.Pool, p Promo) (int64, error) {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	if p.ID > 0 {
		_, err := pool.Exec(ctx, `
			UPDATE promo_codes SET code=$2, description=$3, percent=$4, tier_slug=NULLIF($5,''),
			       max_uses=$6, expires_at=$7, is_active=$8 WHERE id=$1`,
			p.ID, p.Code, p.Description, p.Percent, p.TierSlug, p.MaxUses, p.ExpiresAt, p.IsActive)
		return p.ID, err
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO promo_codes (code, description, percent, tier_slug, max_uses, expires_at, is_active)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7) RETURNING id`,
		p.Code, p.Description, p.Percent, p.TierSlug, p.MaxUses, p.ExpiresAt, p.IsActive).Scan(&id)
	return id, err
}

// DeletePromo kod siler.
func DeletePromo(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1`, id)
	return err
}
