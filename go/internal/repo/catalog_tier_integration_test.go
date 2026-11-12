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

func TestCategoryTierWeightCRUD(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var b [6]byte
	_, _ = rand.Read(b[:])
	slug := "kategori-katman-" + hex.EncodeToString(b[:])

	cat := &repo.Category{
		Name:          "Katman Test",
		Slug:          slug,
		Status:        "yayinda",
		MinTierWeight: 3,
	}
	id, err := repo.CreateCategory(ctx, pool, cat)
	require.NoError(t, err)
	require.NotZero(t, id)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id=$1`, id)
		var kalan int
		_ = pool.QueryRow(context.Background(),
			`SELECT count(*) FROM categories WHERE slug=$1`, slug).Scan(&kalan)
		assert.Zero(t, kalan, "test kategorisi silinmeli")
	})

	// Oluşturmada yazılan değer okunmalı.
	n, err := repo.CategoryMinTierWeight(ctx, pool, id)
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	// Listeleme de değeri taşımalı (kategori paneli/formu bundan beslenir).
	cats, err := repo.ListCategories(ctx, pool, 1)
	require.NoError(t, err)
	var bulunan *repo.Category
	for i := range cats {
		if cats[i].ID == id {
			bulunan = &cats[i]
			break
		}
	}
	require.NotNil(t, bulunan, "yeni kategori listede olmalı")
	assert.Equal(t, 3, bulunan.MinTierWeight)

	// Güncelleme değeri değiştirmeli.
	cat.ID = id
	cat.MinTierWeight = 5
	require.NoError(t, repo.UpdateCategory(ctx, pool, cat))
	n, err = repo.CategoryMinTierWeight(ctx, pool, id)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// Aralık dışı değer 5'e sıkıştırılmalı (DB check kısıtı 0..5).
	cat.MinTierWeight = 9
	require.NoError(t, repo.UpdateCategory(ctx, pool, cat))
	n, err = repo.CategoryMinTierWeight(ctx, pool, id)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// 0 = kategori kuralı yok.
	cat.MinTierWeight = 0
	require.NoError(t, repo.UpdateCategory(ctx, pool, cat))
	n, err = repo.CategoryMinTierWeight(ctx, pool, id)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Var olmayan kategori güvenli biçimde 0 döndürmeli.
	n, err = repo.CategoryMinTierWeight(ctx, pool, id+999999)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
