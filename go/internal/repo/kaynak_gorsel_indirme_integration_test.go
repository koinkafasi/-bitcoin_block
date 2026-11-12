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

// Önden indirme kuyruğu iki kuralı uygulamalı: deneme sınırını aşan
// kayıtlar atlanır, yeni başarısız olanlar bekleme süresi dolmadan
// yeniden denenmez. İkisi de olmazsa işçi her turda aynı ölü adresleri
// çeker ve bant genişliği harcar.
func TestIndirilecekGorsellerSiniriUygular(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var b [8]byte
	_, _ = rand.Read(b[:])
	onek := hex.EncodeToString(b[:])

	ekle := func(ad string, deneme int, sonDeneme string) string {
		h := (onek + ad + "00000000000000000000000000000000")[:32]
		_, err := pool.Exec(ctx, `
			INSERT INTO source_images (hash, url, bytes, indirme_deneme, son_deneme_at)
			VALUES ($1, $2, 0, $3, CASE WHEN $4 = '' THEN NULL ELSE now() - $4::interval END)`,
			h, "https://ornek.test/"+ad+".jpg", deneme, sonDeneme)
		require.NoError(t, err)
		return h
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM source_images WHERE hash LIKE $1`, onek+"%")
	})

	yeni := ekle("a", 0, "")             // hiç denenmemiş → gelmeli
	eski := ekle("b", 2, "10 hours")     // denenmiş ama bekleme doldu → gelmeli
	taze := ekle("c", 1, "5 minutes")    // az önce denendi → GELMEMELİ
	biten := ekle("d", 5, "10 hours")    // sınır doldu → GELMEMELİ

	liste, err := repo.IndirilecekGorseller(ctx, pool, 500)
	require.NoError(t, err)

	var gelen = map[string]bool{}
	for _, g := range liste {
		gelen[g.Hash] = true
	}
	assert.True(t, gelen[yeni], "hiç denenmemiş kayıt kuyruğa girmeli")
	assert.True(t, gelen[eski], "bekleme süresi dolan kayıt yeniden denenmeli")
	assert.False(t, gelen[taze], "az önce denenen kayıt beklemeli")
	assert.False(t, gelen[biten], "deneme sınırı dolan kayıt bırakılmalı")
}

// Başarısızlık sayacı artmalı; aksi hâlde sınır hiç dolmaz ve ölü
// adres sonsuza dek denenir.
func TestGorselIndirmeBasarisizSayaciArtirir(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var b [8]byte
	_, _ = rand.Read(b[:])
	h := (hex.EncodeToString(b[:]) + "0000000000000000")[:32]

	_, err := pool.Exec(ctx, `
		INSERT INTO source_images (hash, url, bytes) VALUES ($1, $2, 0)`,
		h, "https://ornek.test/x.jpg")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM source_images WHERE hash = $1`, h)
	})

	repo.GorselIndirmeBasarisiz(ctx, pool, h, "404 bulunamadı")
	repo.GorselIndirmeBasarisiz(ctx, pool, h, "404 bulunamadı")

	var deneme int
	var hata string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT indirme_deneme, COALESCE(son_hata,'') FROM source_images WHERE hash=$1`, h).
		Scan(&deneme, &hata))
	assert.Equal(t, 2, deneme)
	assert.Contains(t, hata, "404")
}
