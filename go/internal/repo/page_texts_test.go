package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPageTexts(t *testing.T) {
	db := &fakeDB{queryFn: func(sql string, args []any) ([][]any, error) {
		assert.Contains(t, sql, "FROM page_texts")
		assert.Equal(t, []any{int64(7), "kategori:ekonomi"}, args)
		return [][]any{
			{"kategori:ekonomi", "kategori.baslik", "tr", "Ekonomi Gündemi"},
			{"kategori:ekonomi", "kategori.intro", "en", "Economy news"},
		}, nil
	}}

	got, err := ListPageTexts(context.Background(), db, 7, "kategori:ekonomi")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, PageText{
		PageKey: "kategori:ekonomi", TextKey: "kategori.baslik", Locale: "tr", Value: "Ekonomi Gündemi",
	}, got[0])
	assert.Equal(t, "en", got[1].Locale)
}

func TestListPageTextsBos(t *testing.T) {
	db := &fakeDB{}
	got, err := ListPageTexts(context.Background(), db, 1, "article")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSavePageTextUpsert(t *testing.T) {
	db := &fakeDB{}
	err := SavePageText(context.Background(), db, 3, "article", "article.okuma_suresi", "tr", "  dk okuma  ", 42)
	require.NoError(t, err)

	calls := db.ExecCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].SQL, "INSERT INTO page_texts")
	assert.Contains(t, calls[0].SQL, "ON CONFLICT")
	// Kırpılmış değer ve kullanıcı kimliği yazılır.
	assert.Equal(t, "dk okuma", calls[0].Args[4])
	uid, ok := calls[0].Args[5].(*int64)
	require.True(t, ok, "updated_by *int64 olmalı")
	assert.Equal(t, int64(42), *uid)
}

func TestSavePageTextBosDegerSiler(t *testing.T) {
	db := &fakeDB{}
	err := SavePageText(context.Background(), db, 3, "article", "article.editor", "tr", "   ", 42)
	require.NoError(t, err)

	calls := db.ExecCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].SQL, "DELETE FROM page_texts")
	assert.Equal(t, []any{int64(3), "article", "article.editor", "tr"}, calls[0].Args)
}

func TestSavePageTextVarsayilanDil(t *testing.T) {
	db := &fakeDB{}
	err := SavePageText(context.Background(), db, 3, "home", "home.x", "", "değer", 0)
	require.NoError(t, err)

	calls := db.ExecCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "tr", calls[0].Args[3])
	// Kullanıcı yoksa NULL (yabancı anahtar) yazılır.
	uid, ok := calls[0].Args[5].(*int64)
	require.True(t, ok)
	assert.Nil(t, uid)
}

func TestDeletePageText(t *testing.T) {
	db := &fakeDB{}
	err := DeletePageText(context.Background(), db, 5, "kategori", "kategori.intro", "en")
	require.NoError(t, err)

	calls := db.ExecCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].SQL, "DELETE FROM page_texts")
	assert.Equal(t, []any{int64(5), "kategori", "kategori.intro", "en"}, calls[0].Args)
}
