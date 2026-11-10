package main

// Ölü görsel adreslerini kaldırma: `-is=gorsel-kaldir`.
//
// DURUM (10 Kasım 2026): `cdn.bitcosar.com` üzerindeki WordPress görselleri
// KAYBOLDU; sunucu her isteğe HTTP 403 dönüyor ve dosyaların kopyası yok.
// Ölçüm: yayındaki 3496 içeriğin 3176'sı bu adresi taşıyordu → ana sayfa
// dahil her yerde kırık görsel.
//
// KARAR (kullanıcı): görsel kaldırılsın, kırık görsel gösterilmesin; AI
// görsel motoru eklenince yeniden üretilecek.
//
// DAVRANIŞ:
//   - Adres `image_url_yedek`e TAŞINIR (silinmez) → yeniden üretimde
//     "orijinal görsel neydi" bilgisi durur.
//   - `image_url` boşaltılır → şablonlar görsel basmaz.
//   - Kendi medyası olan içerikler (cover/manset) ETKİLENMEZ.
//   - İDEMPOTENT: yedek doluysa tekrar yazmaz.

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// oluGorselHost ölü olduğu ÖLÇÜLMÜŞ host'lar. Yeni bir host ölürse buraya
// eklenir; tahminle eklenmez (403/200 diye tek tek sınandı).
var oluGorselHost = []string{
	"cdn.bitcosar.com",
}

// backfillGorselKaldir ölü host'lardan gelen görsel adreslerini kaldırır.
func backfillGorselKaldir(ctx context.Context, pool *pgxpool.Pool, dry bool, batch int) error {
	if batch <= 0 {
		batch = 500
	}

	// LIKE desenleri ve yer tutucular DİNAMİK kurulur.
	// (Sabit 4 parametreye "\x00" ile doldurmak denendi: Postgres metin
	// parametresinde NUL baytını reddediyor — SQLSTATE 22021.)
	desenler := make([]any, 0, len(oluGorselHost))
	sartlar := make([]string, 0, len(oluGorselHost))
	for i, h := range oluGorselHost {
		sartlar = append(sartlar, fmt.Sprintf("image_url LIKE $%d", i+1))
		desenler = append(desenler, "%"+h+"%")
	}
	sart := "(" + strings.Join(sartlar, " OR ") + ")"
	idP := fmt.Sprintf("$%d", len(oluGorselHost)+1)
	limitP := fmt.Sprintf("$%d", len(oluGorselHost)+2)

	q := `
		SELECT id, COALESCE(image_url,''), COALESCE(image_url_yedek,'')
		  FROM contents
		 WHERE id > ` + idP + `
		   AND site_id = 1
		   AND image_url IS NOT NULL AND image_url <> ''
		   AND ` + sart + `
		 ORDER BY id
		 LIMIT ` + limitP

	var (
		sonID    int64
		toplam   int
		kaldiril int
		atlanan  int
	)

	for {
		argumanlar := append(append([]any{}, desenler...), sonID, batch)
		rows, err := pool.Query(ctx, q, argumanlar...)
		if err != nil {
			return err
		}
		type satir struct {
			ID    int64
			URL   string
			Yedek string
		}
		sayfa := make([]satir, 0, batch)
		for rows.Next() {
			var x satir
			if err := rows.Scan(&x.ID, &x.URL, &x.Yedek); err != nil {
				rows.Close()
				return err
			}
			sayfa = append(sayfa, x)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(sayfa) == 0 {
			break
		}

		for _, x := range sayfa {
			sonID = x.ID
			toplam++
			if strings.TrimSpace(x.Yedek) != "" && strings.TrimSpace(x.URL) == "" {
				atlanan++
				continue
			}
			if dry {
				kaldiril++
				continue
			}
			if _, err := pool.Exec(ctx, `
				UPDATE contents
				   SET image_url_yedek = CASE
				         WHEN COALESCE(image_url_yedek,'') = '' THEN image_url
				         ELSE image_url_yedek END,
				       image_url = ''
				 WHERE id = $1`, x.ID); err != nil {
				continue
			}
			kaldiril++
		}
		fmt.Printf("  … %d satır tarandı (son id %d)\n", toplam, sonID)
	}

	fmt.Printf("görsel kaldırma: %d satır, %d kaldırıldı, %d atlandı (dry=%v)\n",
		toplam, kaldiril, atlanan, dry)
	return nil
}
