package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// KuyrugaGeriKoy kota tükenmesi yüzünden yapılamayan bir işi yeniden
// `queued` durumuna alır.
//
// NEDEN AYRI BİR FONKSİYON: FinishAIJob işi BİTMİŞ sayar — `finished_at`
// yazar ve yalnız `running` satırı günceller. Kota hatası ise işin
// bitmediği, yalnız ERTELENDİĞİ durumdur:
//
//	24 saatlik ölçümde (13 Kasım 2026) 118 iş `failed` işaretlendi ve
//	kayboldu; hepsinin tek sebebi 429'du. O içerikler bir sonraki kota
//	sıfırlamasında üretilebilirdi.
//
// Bu yüzden burada `finished_at` ve `result` temizlenir, `started_at`
// sıfırlanır: iş kuyruğa hiç girmemiş gibi yeniden alınabilir olur.
// `error` alanı BİLGİ olarak kalır — panelde "kota bekliyor" görünsün.
func KuyrugaGeriKoy(ctx context.Context, pool *pgxpool.Pool, id int64, sebep string) error {
	_, err := pool.Exec(ctx, `
		UPDATE ai_jobs
		   SET status      = 'queued',
		       error       = $2,
		       started_at  = NULL,
		       finished_at = NULL
		 WHERE id = $1
		   AND status = 'running'`, id, sebep)
	return err
}

// KotaBekleyenIsSayisi kota yüzünden kuyrukta bekleyen iş sayısını verir.
//
// Panel/günlük için: sıfırlama sonrası bu sayının hızla düşmesi hattın
// bekleyen işi gerçekten tükettiğini gösterir.
func KotaBekleyenIsSayisi(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM ai_jobs
		 WHERE status = 'queued' AND error ILIKE '%kota%'`).Scan(&n)
	return n, err
}
