//go:build integration

package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// RevenueTrend: son N ay, eksik aylar 0 ile doldurulur, iade dahil,
// başarısız/pending hariç. Test kendi ödeme kayıtlarını oluşturup siler.
func TestRevenueTrendAylikToplam(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()
	uid := scratchUser(t, pool)

	conv1 := "trend-int-1-" + time.Now().Format("150405.000000000")
	conv2 := "trend-int-2-" + time.Now().Format("150405.000000000")
	conv3 := "trend-int-3-" + time.Now().Format("150405.000000000")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM subscription_payments WHERE conversation_id = ANY($1)`,
			[]string{conv1, conv2, conv3})
	})

	for _, c := range []struct {
		conv   string
		amount float64
	}{
		{conv1, 100}, {conv2, 250}, {conv3, 500},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO subscription_payments (user_id, provider, conversation_id, amount,
				currency, billing_cycle, status, created_at)
			VALUES ($1,'manual',$2,$3,'TRY','monthly','success', now())`, uid, c.conv, c.amount)
		require.NoError(t, err)
	}
	// Başarısız ödeme toplama girmemeli.
	_, err := pool.Exec(ctx, `
		INSERT INTO subscription_payments (user_id, provider, conversation_id, amount,
			currency, billing_cycle, status, created_at)
		VALUES ($1,'manual',$2,900,'TRY','monthly','failed', now())`,
		uid, conv3+"-fail")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM subscription_payments WHERE conversation_id=$1`, conv3+"-fail")
	})

	trend, err := repo.RevenueTrend(ctx, pool, 12)
	require.NoError(t, err)
	require.Len(t, trend, 12, "her zaman 12 nokta (boş aylar 0)")
	assert.Equal(t, time.Now().Format("2006-01"), trend[11].Label, "son nokta bu ay olmalı")
	assert.InDelta(t, 850.0, trend[11].Amount, 0.01, "bu ayın toplamı 100+250+500")
	assert.Equal(t, 3, trend[11].Count)
	assert.NotEmpty(t, trend[11].Short)

	// Grafik sırası eskiden yeniye.
	for i := 1; i < len(trend); i++ {
		assert.Less(t, trend[i-1].Label, trend[i].Label, "aylar artan sırada olmalı")
	}

	// Ay sınırı: 24 isteği 24'e sınırlanır, 0/negatif 12'ye düşer.
	if _, err := repo.RevenueTrend(ctx, pool, 0); err != nil {
		t.Fatalf("0 ay hatası: %v", err)
	}
	kucuk, err := repo.RevenueTrend(ctx, pool, 3)
	require.NoError(t, err)
	assert.Len(t, kucuk, 3)
}
