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

// SetContentErisim iki alanı birlikte tutmalı: min_tier_weight ve
// is_exclusive. Daha önce bunlar elle ayrı ayrı işaretlendiği için
// tutarsız kalmıştı — kilitli görünen ama kademesi olmayan içerikler
// vardı. Tek fonksiyondan geçmeleri bunu engelliyor.
func TestSetContentErisim(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var b [6]byte
	_, _ = rand.Read(b[:])
	slug := "erisim-test-" + hex.EncodeToString(b[:])

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO contents (title, slug, status, locale, site_id, type_key, author_id)
		VALUES ('Erişim Testi', $1, 'taslak', 'tr',
		        (SELECT id FROM sites ORDER BY id LIMIT 1),
		        (SELECT key FROM content_types ORDER BY sort_order, key LIMIT 1),
		        (SELECT id FROM users ORDER BY id LIMIT 1))
		RETURNING id`, slug).Scan(&id)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM contents WHERE id=$1`, id)
	})

	oku := func() (int, bool) {
		var w int
		var ozel bool
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT coalesce(min_tier_weight,0), coalesce(is_exclusive,false)
			   FROM contents WHERE id=$1`, id).Scan(&w, &ozel))
		return w, ozel
	}

	// Kademeye kilitle.
	require.NoError(t, repo.SetContentErisim(ctx, pool, id, 3))
	w, ozel := oku()
	assert.Equal(t, 3, w)
	assert.True(t, ozel, "kademe verilince is_exclusive açılmalı")

	// Herkese aç: iki alan da sıfırlanmalı.
	require.NoError(t, repo.SetContentErisim(ctx, pool, id, 0))
	w, ozel = oku()
	assert.Equal(t, 0, w)
	assert.False(t, ozel, "kilit kalkınca is_exclusive da kapanmalı")

	// Sınır dışı değerler kırpılır, hata dönmez.
	require.NoError(t, repo.SetContentErisim(ctx, pool, id, 99))
	w, _ = oku()
	assert.Equal(t, 5, w, "en çok 5. kademe")

	require.NoError(t, repo.SetContentErisim(ctx, pool, id, -2))
	w, ozel = oku()
	assert.Equal(t, 0, w)
	assert.False(t, ozel)
}
