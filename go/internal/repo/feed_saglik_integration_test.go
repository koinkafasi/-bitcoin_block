//go:build integration

package repo_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Ardışık hata sayacı eşiğe gelince kaynağı kapatmalı; başarılı çekim
// sayacı sıfırlamalı. Eskiden yalnız son durum tutuluyordu: bozulan
// kaynak kapalı listede sessizce duruyor, "neden kapalı?" sorusunun
// cevabı kaybolu yordu.
func TestFeedSaglikSayaciVeOtoKapatma(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var b [6]byte
	_, _ = rand.Read(b[:])
	url := "https://ornek.test/feed-" + hex.EncodeToString(b[:])

	var id int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO feeds (site_id, name, url, aktif, kind, trust)
		VALUES (1, 'Saglik Testi', $1, true, 'rss', 50) RETURNING id`, url).Scan(&id))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM feeds WHERE id=$1`, id)
	})

	durum := func() (sayac int, aktif, oto bool) {
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT ardisik_hata, aktif, otomatik_kapandi FROM feeds WHERE id=$1`, id).
			Scan(&sayac, &aktif, &oto))
		return
	}

	// Eşiğin bir altına kadar: hâlâ açık kalmalı. Geçici kesinti
	// (429, bakım) yüzünden sağlam kaynak kapatılmamalı.
	for i := 1; i < repo.OtoKapatmaEsigi; i++ {
		kapandi, err := repo.FeedBasarisiz(ctx, pool, id, "deneme hatasi")
		require.NoError(t, err)
		assert.False(t, kapandi, "%d. hatada kapanmamalı", i)
		sayac, aktif, _ := durum()
		assert.Equal(t, i, sayac)
		assert.True(t, aktif)
	}

	// Eşik: kapanmalı ve bunun OTOMATİK olduğu işaretlenmeli.
	kapandi, err := repo.FeedBasarisiz(ctx, pool, id, "son hata")
	require.NoError(t, err)
	assert.True(t, kapandi, "eşikte kapanmalı")
	sayac, aktif, oto := durum()
	assert.Equal(t, repo.OtoKapatmaEsigi, sayac)
	assert.False(t, aktif)
	assert.True(t, oto, "elle değil bot kapattı")

	// Elle açınca sayaç sıfırlanmalı — yoksa ilk hatada yine kapanır
	// ve kullanıcı "açtım ama kapanıyor" derdi.
	require.NoError(t, repo.FeedYenidenAc(ctx, pool, id))
	sayac, aktif, oto = durum()
	assert.Equal(t, 0, sayac)
	assert.True(t, aktif)
	assert.False(t, oto)

	// Başarılı çekim: kalite ölçümü yazılır, sayaç sıfır kalır.
	require.NoError(t, repo.FeedBasarili(ctx, pool, id, 20, 85))
	var oge, oran int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT last_item_count, gorsel_oran FROM feeds WHERE id=$1`, id).Scan(&oge, &oran))
	assert.Equal(t, 20, oge)
	assert.Equal(t, 85, oran)

	// Sınır dışı oran kırpılır, hata dönmez.
	require.NoError(t, repo.FeedBasarili(ctx, pool, id, 5, 140))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT gorsel_oran FROM feeds WHERE id=$1`, id).Scan(&oran))
	assert.Equal(t, 100, oran)

	// NEGATİF oran mevcut ölçümü KORUMALI. Bot'un içerik üretim
	// döngüsü görsel bilgisi taşımıyor; sıfır yazsaydı her çekimde
	// gerçek ölçüm silinirdi.
	require.NoError(t, repo.FeedBasarili(ctx, pool, id, 7, -1))
	var oran2, oge2 int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT gorsel_oran, last_item_count FROM feeds WHERE id=$1`, id).Scan(&oran2, &oge2))
	assert.Equal(t, 100, oran2, "negatif oran mevcut değeri ezmemeli")
	assert.Equal(t, 7, oge2, "öğe sayısı yine de güncellenmeli")
}
