//go:build integration

package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestRecentPublishedForCompare(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var authorID int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&authorID))

	var publishedID, draftID int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, spot, detail, slug, author_id, site_id, locale, published_at)
		VALUES ('haber','yayinda','Benzerlik testi yayinda','Spot',
		        '[{"type":"p","text":"govde"}]'::jsonb, 'benzerlik-testi-yayinda-int', $1, 1, 'tr', now())
		RETURNING id`, authorID).Scan(&publishedID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, spot, detail, slug, author_id, site_id, locale)
		VALUES ('haber','taslak','Benzerlik testi taslak','Spot',
		        '[{"type":"p","text":"govde"}]'::jsonb, 'benzerlik-testi-taslak-int', $1, 1, 'tr')
		RETURNING id`, authorID).Scan(&draftID))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM contents WHERE id = ANY($1)`, []int64{publishedID, draftID})
	})

	got, err := repo.RecentPublishedForCompare(ctx, pool, 1, 0, 200)
	require.NoError(t, err)

	var sawPublished, sawDraft bool
	for _, c := range got {
		if c.ID == publishedID {
			sawPublished = true
			assert.NotEmpty(t, c.Title)
			assert.NotEmpty(t, c.Detail, "gövde karşılaştırma için gerekli")
		}
		if c.ID == draftID {
			sawDraft = true
		}
	}
	assert.True(t, sawPublished, "yayındaki içerik karşılaştırma kümesinde olmalı")
	assert.False(t, sawDraft, "taslak içerik karşılaştırma kümesinde olmamalı")

	// Kendisi hariç tutulmalı.
	got2, err := repo.RecentPublishedForCompare(ctx, pool, 1, publishedID, 200)
	require.NoError(t, err)
	for _, c := range got2 {
		assert.NotEqual(t, publishedID, c.ID, "dışlanan içerik dönmemeli")
	}
}
