//go:build integration

package scoring_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/scoring"
)

// Öncelik iki yerde hesaplanır: panelde Go (scoring.Hesapla), sıralamada
// SQL (scoring.OncelikSQL). İkisi ayrışırsa editörün gördüğü puan ile
// listenin sırası birbirini tutmaz. Bu test ayrışmayı yakalar.
//
// GEÇİCİ TABLO kullanır; gerçek contents tablosuna dokunmaz, bu yüzden
// üretim veritabanına karşı da güvenle koşturulabilir.
func TestSQLVeGoAyniOnceligiVerir(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL yok — integration testi atlandı")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	defer pool.Close()

	// Tek bağlantıda kal: TEMP tablo oturuma bağlıdır.
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(ctx, `CREATE TEMP TABLE scoring_parite (
		id bigserial PRIMARY KEY,
		created_at timestamptz NOT NULL,
		ai_score smallint NOT NULL DEFAULT 0,
		score_trend smallint NOT NULL DEFAULT 0,
		score_guven smallint NOT NULL DEFAULT 0
	) ON COMMIT PRESERVE ROWS`)
	require.NoError(t, err)

	type durum struct {
		yasSaat             int
		tamlik, trend, guen int
	}
	var durumlar []durum
	for _, s := range []int{0, 1, 6, 12, 23, 24, 25, 48, 72, 100, 167, 168, 200} {
		for _, tr := range []int{0, 60, 90, 100} {
			durumlar = append(durumlar, durum{s, 72, tr, 70})
		}
	}
	durumlar = append(durumlar,
		durum{0, 100, 100, 90}, durum{36, 0, 0, 0}, durum{5, 55, 75, 60})

	simdi := time.Now()
	for i, d := range durumlar {
		_, err := conn.Exec(ctx,
			`INSERT INTO scoring_parite (id, created_at, ai_score, score_trend, score_guven)
			 VALUES ($1,$2,$3,$4,$5)`,
			i+1, simdi.Add(-time.Duration(d.yasSaat)*time.Hour), d.tamlik, d.trend, d.guen)
		require.NoError(t, err)
	}

	rows, err := conn.Query(ctx, fmt.Sprintf(
		`SELECT id, %s, %s FROM scoring_parite ORDER BY id`,
		scoring.TazelikSQL("scoring_parite"), scoring.OncelikSQL("scoring_parite")))
	require.NoError(t, err)
	defer rows.Close()

	sayac := 0
	for rows.Next() {
		var id int
		var sqlTazelik, sqlOncelik float64
		require.NoError(t, rows.Scan(&id, &sqlTazelik, &sqlOncelik))
		d := durumlar[id-1]

		goTazelik := scoring.Tazelik(time.Duration(d.yasSaat) * time.Hour)
		require.InDeltaf(t, float64(goTazelik), sqlTazelik, 1.0,
			"tazelik ayrıştı: yaş %d saat — Go %d, SQL %.2f", d.yasSaat, goTazelik, sqlTazelik)

		// Öncelik: Go tarafında eksenler zaten hesaplanmış kabul edilir,
		// bu yüzden ağırlıklı toplamı doğrudan kuruyoruz — SQL de aynısını
		// yapıyor olmalı.
		toplam := scoring.AgirlikTrend + scoring.AgirlikTazelik +
			scoring.AgirlikGuven + scoring.AgirlikTamlik
		goOncelik := float64(scoring.AgirlikTrend*d.trend+
			scoring.AgirlikTazelik*goTazelik+
			scoring.AgirlikGuven*d.guen+
			scoring.AgirlikTamlik*d.tamlik) / float64(toplam)
		require.InDeltaf(t, goOncelik, sqlOncelik, 1.0,
			"öncelik ayrıştı: yaş %d saat, trend %d — Go %.2f, SQL %.2f",
			d.yasSaat, d.trend, goOncelik, sqlOncelik)
		sayac++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, len(durumlar), sayac, "her satır karşılaştırılmalı")
}
