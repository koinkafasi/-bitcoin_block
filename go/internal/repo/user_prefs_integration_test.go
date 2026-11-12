//go:build integration

package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestKullaniciTercihleriIntegration(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var uid int64
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash)
		VALUES ('Tercih Test', 'tercih-test-' || md5(random()::text) || '@t.local', 'x')
		RETURNING id`).Scan(&uid)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
	})

	// 1) Kayıt yok → boş harita.
	m, err := repo.KullaniciTercihleri(ctx, pool, uid)
	require.NoError(t, err)
	assert.Empty(t, m)

	// 2) Yaz ve oku.
	require.NoError(t, repo.KullaniciTercihiKaydet(ctx, pool, uid, "tb_form_grup:Künye", true))
	m, err = repo.KullaniciTercihleri(ctx, pool, uid)
	require.NoError(t, err)
	assert.Equal(t, true, m["tb_form_grup:Künye"])

	// 3) Birleştirme: önceki anahtar korunmalı.
	require.NoError(t, repo.KullaniciTercihiKaydet(ctx, pool, uid, "pn:rail", false))
	m, err = repo.KullaniciTercihleri(ctx, pool, uid)
	require.NoError(t, err)
	assert.Equal(t, true, m["tb_form_grup:Künye"], "önceki anahtar silinmemeli")
	assert.Equal(t, false, m["pn:rail"])

	// 4) Güncelleme: son değer kazanır.
	require.NoError(t, repo.KullaniciTercihiKaydet(ctx, pool, uid, "pn:rail", true))
	m, err = repo.KullaniciTercihleri(ctx, pool, uid)
	require.NoError(t, err)
	assert.Equal(t, true, m["pn:rail"])

	// 5) CASCADE.
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
	require.NoError(t, err)
	m, err = repo.KullaniciTercihleri(ctx, pool, uid)
	require.NoError(t, err)
	assert.Empty(t, m, "kullanıcı gitti, tercih kalmamalı")
}
