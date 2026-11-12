//go:build integration

package repo_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/db"
	"theblock/site/internal/repo"
)

func intPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL yok — integration testi atlandı")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	// Şema test veritabanında yoksa kur (migration'lar idempotenttir).
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, err = db.Up(ctx, conn.Conn())
	require.NoError(t, err, "test şeması kurulamadı")
	return pool
}

func scratchUser(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var b [6]byte
	_, _ = rand.Read(b[:])
	email := "tier-int-" + hex.EncodeToString(b[:]) + "@test.local"
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (name, email, password_hash, role, status)
		VALUES ('Kademe Test', $1, 'x', 'Üye', 'aktif') RETURNING id`, email).Scan(&id)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
		var kalan int
		_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email=$1`, email).Scan(&kalan)
		assert.Zero(t, kalan, "test kullanıcısı silinmeli")
	})
	return id
}

func tierID(t *testing.T, pool *pgxpool.Pool, slug string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id FROM membership_tiers WHERE slug=$1`, slug).Scan(&id))
	return id
}

func TestActivateSubscriptionSureEkler(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()
	uid := scratchUser(t, pool)
	tid := tierID(t, pool, "bronz")

	ilk, err := repo.ActivateSubscription(ctx, pool, uid, tid, "monthly")
	require.NoError(t, err)
	require.NotNil(t, ilk)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 30), *ilk, 2*time.Minute)

	// Aynı kademede yenileme: mevcut bitişin ÜZERİNE 30 gün eklenir.
	ikinci, err := repo.ActivateSubscription(ctx, pool, uid, tid, "monthly")
	require.NoError(t, err)
	assert.WithinDuration(t, ilk.AddDate(0, 0, 30), *ikinci, 2*time.Minute)

	// Yıllık: 365 gün (mevcut sürenin üzerine).
	yillik, err := repo.ActivateSubscription(ctx, pool, uid, tid, "annual")
	require.NoError(t, err)
	assert.WithinDuration(t, ikinci.AddDate(0, 0, 365), *yillik, 2*time.Minute)

	var membership, cycle string
	var tier *int64
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT membership, COALESCE(membership_billing_cycle,''), membership_tier_id FROM users WHERE id=$1`, uid).
		Scan(&membership, &cycle, &tier))
	assert.Equal(t, "premium", membership)
	assert.Equal(t, "annual", cycle)
	require.NotNil(t, tier)
	assert.Equal(t, tid, *tier)

	// Aktif katman ağırlığı doğru okunur.
	assert.Equal(t, 2, repo.UserTierWeight(ctx, pool, uid))
}

func TestExtendSubscriptionVeSuresiDolanlar(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()
	uid := scratchUser(t, pool)
	tid := tierID(t, pool, "filiz")

	_, err := repo.ActivateSubscription(ctx, pool, uid, tid, "monthly")
	require.NoError(t, err)

	uzatilmis, err := repo.ExtendSubscription(ctx, pool, uid, 10)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 40), *uzatilmis, 2*time.Minute)

	// Bitişi 3 gün sonrasına çek → hatırlatma listesinde görünmeli.
	_, err = pool.Exec(ctx, `UPDATE users SET membership_expires_at = now() + interval '3 days' WHERE id=$1`, uid)
	require.NoError(t, err)
	ids, err := repo.ExpiringSoon(ctx, pool, 3)
	require.NoError(t, err)
	assert.Contains(t, ids, uid, "bitişe 3 gün kala listede olmalı")

	// İptal işaretli kullanıcı hatırlatma almaz.
	require.NoError(t, repo.CancelSubscriptionAtPeriodEnd(ctx, pool, uid))
	ids, err = repo.ExpiringSoon(ctx, pool, 3)
	require.NoError(t, err)
	assert.NotContains(t, ids, uid, "iptal talebi olan kullanıcı hatırlatılmamalı")
	require.NoError(t, repo.UndoCancellation(ctx, pool, uid))

	// Süresi dolmuş → ExpireSubscriptions düşürür.
	_, err = pool.Exec(ctx, `UPDATE users SET membership_expires_at = now() - interval '1 day' WHERE id=$1`, uid)
	require.NoError(t, err)
	expired, err := repo.ExpireSubscriptions(ctx, pool)
	require.NoError(t, err)
	assert.Contains(t, expired, uid)

	var membership string
	var tier, until *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT membership, membership_tier_id::text, membership_expires_at::text FROM users WHERE id=$1`, uid).
		Scan(&membership, &tier, &until))
	assert.Empty(t, membership)
	assert.Nil(t, tier)
}

func TestKademeKilidiIcinAgirlik(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()
	uid := scratchUser(t, pool)

	// Aboneliksiz = 0.
	assert.Equal(t, 0, repo.UserTierWeight(ctx, pool, uid))
	_, ok := repo.UserTier(ctx, pool, uid)
	assert.False(t, ok)

	_, err := repo.ActivateSubscription(ctx, pool, uid, tierID(t, pool, "gumus"), "monthly")
	require.NoError(t, err)
	assert.Equal(t, 3, repo.UserTierWeight(ctx, pool, uid))

	// Süresi geçmiş abonelik ağırlık vermez.
	_, err = pool.Exec(ctx, `UPDATE users SET membership_expires_at = now() - interval '1 hour' WHERE id=$1`, uid)
	require.NoError(t, err)
	assert.Equal(t, 0, repo.UserTierWeight(ctx, pool, uid), "süresi geçmiş abonelik erişim vermez")
}

func TestOdemeKaydiVeOzet(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()
	uid := scratchUser(t, pool)
	tid := tierID(t, pool, "altin")

	conv := "tier-int-conv-" + time.Now().Format("150405.000000000")
	pid, err := repo.CreatePayment(ctx, pool, uid, &tid, "iyzico", conv, 1000, 200, "TRY", "monthly", "TESTKOD")
	require.NoError(t, err)
	require.NotZero(t, pid)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM subscription_payments WHERE conversation_id=$1`, conv)
	})

	p, err := repo.GetPaymentConversation(ctx, pool, conv)
	require.NoError(t, err)
	assert.Equal(t, "pending", p.Status)
	assert.InDelta(t, 1000.0, p.Amount, 0.01)
	assert.InDelta(t, 200.0, p.Discount, 0.01)
	assert.Equal(t, "TESTKOD", p.CouponCode)

	require.NoError(t, repo.CompletePayment(ctx, pool, conv, "pay-int-1", []byte(`{"ok":true}`)))
	p, err = repo.GetPaymentConversation(ctx, pool, conv)
	require.NoError(t, err)
	assert.Equal(t, "success", p.Status)
	assert.Equal(t, "pay-int-1", p.ProviderPayment)

	// Token ile bulma (iyzico callback yolu).
	require.NoError(t, repo.SetPaymentToken(ctx, pool, conv, "tok-int-"+conv))
	byTok, err := repo.GetPaymentByToken(ctx, pool, "tok-int-"+conv)
	require.NoError(t, err)
	assert.Equal(t, pid, byTok.ID)

	require.NoError(t, repo.RefundPayment(ctx, pool, pid, "test iadesi"))
	p, err = repo.GetPaymentConversation(ctx, pool, conv)
	require.NoError(t, err)
	assert.Equal(t, "refunded", p.Status)

	// Özet fonksiyonları hata vermemeli.
	_, err = repo.SubscriptionSummary(ctx, pool)
	require.NoError(t, err)
	_, err = repo.SubscriptionEconomy(ctx, pool)
	require.NoError(t, err)
	_, err = repo.ListPayments(ctx, pool, 5)
	require.NoError(t, err)
}

func TestPromoKoduDogrulama(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	code := "TBI" + time.Now().Format("150405000")
	id, err := repo.SavePromo(ctx, pool, repo.Promo{
		Code: code, Percent: 25, MaxUses: 1, IsActive: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.DeletePromo(context.Background(), pool, id)
		var kalan int
		_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM promo_codes WHERE id=$1`, id).Scan(&kalan)
		assert.Zero(t, kalan, "test kuponu silinmeli")
	})

	p, err := repo.ValidPromo(ctx, pool, code, "")
	require.NoError(t, err)
	assert.Equal(t, 25, p.Percent)

	// Kademe kısıtı olmayan kod her kademede geçerli.
	_, err = repo.ValidPromo(ctx, pool, code, "gumus")
	require.NoError(t, err)

	// Kullanım hakkı bitince geçersiz.
	require.NoError(t, repo.UsePromo(ctx, pool, id))
	_, err = repo.ValidPromo(ctx, pool, code, "")
	assert.ErrorIs(t, err, repo.ErrPromoGecersiz)

	// Bilinmeyen kod.
	_, err = repo.ValidPromo(ctx, pool, "YOKBOYLEKOD", "")
	assert.ErrorIs(t, err, repo.ErrPromoGecersiz)

	// Süresi geçmiş kod.
	gecmis := time.Now().Add(-time.Hour)
	id2, err := repo.SavePromo(ctx, pool, repo.Promo{
		Code: code + "X", Percent: 10, ExpiresAt: &gecmis, IsActive: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.DeletePromo(context.Background(), pool, id2) })
	_, err = repo.ValidPromo(ctx, pool, code+"X", "")
	assert.ErrorIs(t, err, repo.ErrPromoGecersiz)
}
