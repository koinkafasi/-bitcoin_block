//go:build integration

package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestUpdateContentAlakaPuaniniTazeler(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	// --- fikstür: site + yazar ---
	var siteID int64
	err := pool.QueryRow(ctx, `SELECT id FROM sites ORDER BY id LIMIT 1`).Scan(&siteID)
	if err != nil {
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO sites (domain, name) VALUES ('skor-it.test','Skor IT')
			RETURNING id`).Scan(&siteID), "site kurulamadı")
	}
	var uid int64
	err = pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&uid)
	if err != nil {
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO users (name, email, password_hash, role, status)
			VALUES ('Skor IT','skor-it@example.test','x','Yönetici','aktif')
			RETURNING id`).Scan(&uid), "kullanıcı kurulamadı")
	}

	var id int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, title, spot, author_id, site_id, ai_score)
		VALUES ('haber', 'Deprem sonrası kentsel dönüşüm takvimi açıklandı',
		        'Bakanlık riskli yapıların dönüşümü için yeni takvimi duyurdu ve başvuru sürecini açıkladı.',
		        $1, $2, 0)
		RETURNING id`, uid, siteID).Scan(&id))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM contents WHERE id = $1`, id)
	})

	c, err := repo.GetContent(ctx, pool, id)
	require.NoError(t, err)

	// --- 1) 0'a çekilmiş puan kaydetmeyle geri gelmeli ---
	_, err = pool.Exec(ctx, `UPDATE contents SET ai_score = 0 WHERE id = $1`, id)
	require.NoError(t, err)

	c.Version = 0
	require.NoError(t, repo.UpdateContent(ctx, pool, &c))

	var ilk int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COALESCE(ai_score,0) FROM contents WHERE id = $1`, id).Scan(&ilk))
	assert.Greater(t, ilk, 0, "kaydetme sonrası puan tazelenmeli (0'da kalmamalı)")

	// --- 2) İçeriği eksilt, puan düşmeli ---
	c.Title = "a"
	c.Spot = ""
	c.CoverMediaID = nil
	c.Version = 0
	require.NoError(t, repo.UpdateContent(ctx, pool, &c))

	var sonra int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COALESCE(ai_score,0) FROM contents WHERE id = $1`, id).Scan(&sonra))
	assert.Less(t, sonra, ilk,
		"eksiltilen içerik daha düşük puan almalı: %d → %d", ilk, sonra)
}
