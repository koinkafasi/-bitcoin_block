package main

// Görsel geri doldurma: mutlak http(s) `image_url` → `/kg/{hash}`.
//
// NEDEN: kaynak görselleri KENDİ origin'imizden servis etme kararı
// (migration 0116) yalnız YENİ çekilen içeriklere uygulanıyordu
// (repo/bot.go:748). Ölçüm (10 Kasım 2026): yayındaki 3376 içeriğin
// **3176'sı** hâlâ `cdn.bitcosar.com/wp-content/...` adresini taşıyordu;
// ana sayfada tek bir kart bile yerel `/kg/` yolundan gelmiyordu.
//
// Bu iş tek seferliktir ve İDEMPOTENT'tir: zaten `/kg/` olan satıra
// dokunmaz, `-dry` ile hiçbir şey yazmaz.
//
// NEDEN YIKICI DEĞİL: `source_images` upsert edilir (adres korunur),
// ardından içeriğin image_url'i proxy yoluna çevrilir. Görselin kendisi
// gösterim anında indirilir (repo.KaynakGorselURL); orijinal adres
// tabloda durduğu için geri alınabilir.

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

// gorselProbe görsel doldurma için gereken alanlar.
type gorselProbe struct {
	ID       int64
	ImageURL string
}

// backfillGorseller mutlak HTTP(S) görsel adreslerini /kg/ proxy yoluna çevirir.
func backfillGorseller(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	if batch <= 0 {
		batch = 500
	}

	var (
		sonID    int64
		toplam   int
		cevrilen int
		atlanan  int
		hatalar  int
	)

	for {
		rows, err := pool.Query(ctx, `
			SELECT id, COALESCE(image_url,'')
			  FROM contents
			 WHERE id > $1
			   AND site_id = 1
			   AND image_url IS NOT NULL
			   AND image_url <> ''
			 ORDER BY id
			 LIMIT $2`, sonID, batch)
		if err != nil {
			return err
		}
		sayfa := make([]gorselProbe, 0, batch)
		for rows.Next() {
			var p gorselProbe
			if err := rows.Scan(&p.ID, &p.ImageURL); err != nil {
				rows.Close()
				return err
			}
			sayfa = append(sayfa, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(sayfa) == 0 {
			break
		}

		for _, p := range sayfa {
			sonID = p.ID
			toplam++

			l := strings.ToLower(strings.TrimSpace(p.ImageURL))
			// Zaten proxy/yerel yolda: DOKUNMA (idempotent).
			if strings.HasPrefix(l, "/kg/") || strings.HasPrefix(l, "/media/") {
				atlanan++
				continue
			}
			if !strings.HasPrefix(l, "http://") && !strings.HasPrefix(l, "https://") {
				atlanan++
				continue
			}
			if dry {
				cevrilen++
				continue
			}
			h := repo.KaynakGorselKaydet(ctx, pool, p.ImageURL)
			if h == "" {
				hatalar++
				continue
			}
			yol := repo.KaynakGorselYol(h)
			if yol == "" {
				hatalar++
				continue
			}
			if _, err := pool.Exec(ctx,
				`UPDATE contents SET image_url = $1 WHERE id = $2`, yol, p.ID); err != nil {
				hatalar++
				continue
			}
			cevrilen++
		}

		fmt.Printf("  … %d satır tarandı (son id %d)\n", toplam, sonID)
	}

	fmt.Printf("görsel doldurma: %d satır, %d çevrildi, %d atlandı, %d hata (dry=%v)\n",
		toplam, cevrilen, atlanan, hatalar, dry)
	return nil
}
