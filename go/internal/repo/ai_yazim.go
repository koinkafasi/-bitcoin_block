package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Otomatik AI ile yazım — veri katmanı (Faz 194).
//
// Akış:
//  1. `YazimBekleyenTaslaklar` bot kaynaklı, henüz AI'dan geçmemiş taslakları
//     döndürür.
//  2. İşçi her biri için `news` işini kuyruğa yazar (ai_jobs).
//  3. Var olan AI işçisi işi yürütür; sonuç `ai_jobs.result`ta bekler.
//  4. `YazimUygulanacakIsler` tamamlanmış ama UYGULANMAMIŞ işleri döndürür;
//     işçi sonucu içeriğe yazar ve `applied_at` işaretler.

// yazimEylem AI haber yazımı eyleminin kimliğidir.
const yazimEylem = "news"

// yazimAzamiHata aynı içerik için kaç başarısız denemeden sonra vazgeçilir.
//
// Neden: başarısız içerik her turda yeniden denenirse sürekli AI maliyeti
// doğar. İki denemeden sonra içerik elle ele alınmalı.
const yazimAzamiHata = 2

// YazimBekleyenTaslaklar bot kaynaklı, AI ile henüz yazılmamış taslakları
// döndürür (en yeni önce).
//
// Ölçütler:
//   - taslak, tr, çeviri değil, haber türü
//   - `source_url` dolu (bot/ajans kaynağı; elle yazılan taslaklara dokunma)
//   - başlığı ve gövdesi var (boş kayda AI çağrısı harcamak anlamsız)
//   - bu içerik için TAMAMLANMIŞ `news` işi yok
//   - başarısız `news` denemesi sınırı aşmamış
//   - gövdesi AI kaynak iskeleti DEĞİLSE de dahil: iskeleti değiştirmek
//     zaten bu işin amacı
func YazimBekleyenTaslaklar(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]Content, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id
		  FROM contents c
		 WHERE c.site_id = $1
		   AND c.status = 'taslak'
		   AND c.locale = 'tr'
		   AND c.translation_of IS NULL
		   AND c.type_key = 'haber'
		   AND c.source_url IS NOT NULL
		   AND COALESCE(c.title,'') <> ''
		   AND c.detail <> '[]'::jsonb
		   AND NOT EXISTS (
				SELECT 1 FROM ai_jobs j
				 WHERE j.content_id = c.id
				   AND j.action = $2
				   AND j.status = 'completed')
		   AND (SELECT count(*) FROM ai_jobs j
				 WHERE j.content_id = c.id
				   AND j.action = $2
				   AND j.status = 'failed') < $3
		   -- Şu anda kuyrukta/çalışan iş varsa yeniden yazma.
		   AND NOT EXISTS (
				SELECT 1 FROM ai_jobs j
				 WHERE j.content_id = c.id
				   AND j.action = $2
				   AND j.status IN ('queued','running'))
		 ORDER BY c.id DESC
		 LIMIT $4`, siteID, yazimEylem, yazimAzamiHata, limit)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Content, 0, len(ids))
	for _, id := range ids {
		c, err := GetContent(ctx, pool, id)
		if err != nil {
			continue // arada silinmiş olabilir
		}
		out = append(out, c)
	}
	return out, nil
}

// YazimIsi tamamlanmış ama içeriğe yazılmamış bir AI işidir.
type YazimIsi struct {
	JobID     int64
	ContentID int64
	Result    []byte
}

// YazimUygulanacakIsler tamamlanmış, henüz uygulanmamış `news` işlerini
// döndürür (en eski önce: sıra korunur).
func YazimUygulanacakIsler(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]YazimIsi, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT j.id, j.content_id, j.result
		  FROM ai_jobs j
		  JOIN contents c ON c.id = j.content_id
		 WHERE j.site_id = $1
		   AND j.action = $2
		   AND j.status = 'completed'
		   AND j.applied_at IS NULL
		   AND j.result IS NOT NULL
		   AND c.status = 'taslak'
		 ORDER BY j.id
		 LIMIT $3`, siteID, yazimEylem, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []YazimIsi
	for rows.Next() {
		var y YazimIsi
		if err := rows.Scan(&y.JobID, &y.ContentID, &y.Result); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

// YazimSonucu içeriğe yazılacak AI çıktısıdır.
type YazimSonucu struct {
	Title string
	Spot  string
	// Ozet `contents.ai_summary` alanıdır: makale sayfasındaki özet kutusu
	// (spec §10.2.2) yalnız bu alan DOLUYSA basılır. Boş bırakılırsa var
	// olan özet korunur — AI'ın özet üretmemesi elle yazılmış özeti silmez.
	Ozet     string
	Paragraf []string
	SeoTitle string
	SeoDesc  string
}

// SetContentYazim AI çıktısını içeriğe yazar.
//
// DAR kapsam: yalnız başlık, spot, gövde, SEO alanları. Kategori, görsel,
// yazar, kaynak ve durum DEĞİŞMEZ — AI bunları bilmez, ezmemeli.
//
// `beklenenSurum` > 0 ise yalnız o sürümde yazar (editör bu arada içeriği
// kaydettiyse AI sonucu onun değişikliğini ezmez). 0 = sürüm denetimi yok.
func SetContentYazim(ctx context.Context, pool *pgxpool.Pool, id int64, surum int64, y YazimSonucu) error {
	detay, err := yazimDetay(y.Paragraf)
	if err != nil {
		return err
	}
	komut, err := pool.Exec(ctx, `
		UPDATE contents SET
			title = $2,
			spot = $3,
			detail = $4::jsonb,
			seo_title = NULLIF($5,''),
			seo_description = NULLIF($6,''),
			ai_summary = COALESCE(NULLIF($8,''), ai_summary),
			version = version + 1,
			updated_at = now()
		 WHERE id = $1 AND ($7 = 0 OR version = $7)`,
		id, y.Title, y.Spot, detay, y.SeoTitle, y.SeoDesc, surum, y.Ozet)
	if err != nil {
		return err
	}
	if komut.RowsAffected() == 0 {
		// Sürüm çakışması ya da içerik yok.
		var varMi bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contents WHERE id=$1)`, id).Scan(&varMi)
		if !varMi {
			return ErrNotFound
		}
		return ErrConflict
	}
	return nil
}

// yazimDetay paragrafları blok dizisine çevirir.
//
// Boş paragraflar atılır; hiç paragraf kalmazsa hata döner (boş gövde
// yazmak içeriği bozardı).
func yazimDetay(paragraflar []string) ([]byte, error) {
	bloklar := make([]map[string]any, 0, len(paragraflar))
	for _, p := range paragraflar {
		if p = strings.TrimSpace(p); p != "" {
			bloklar = append(bloklar, map[string]any{"type": "p", "text": p})
		}
	}
	if len(bloklar) == 0 {
		return nil, errors.New("yazım sonucu boş: hiç paragraf yok")
	}
	return json.Marshal(bloklar)
}

// MarkAIJobApplied işi "uygulandı" olarak işaretler.
//
// İşaretlemeden sonra işçi aynı sonucu yeniden yazmaz; bu olmadan her turda
// içerik sürümü artar ve editör düzeltmeleri ezilir.
func MarkAIJobApplied(ctx context.Context, pool *pgxpool.Pool, jobID int64) error {
	_, err := pool.Exec(ctx,
		`UPDATE ai_jobs SET applied_at = now() WHERE id = $1 AND applied_at IS NULL`, jobID)
	return err
}

// YazimSayaciSonDakika son 1 dakikada kuyruğa eklenen `news` işi sayısıdır.
// Hız ayarı (dakikada N) bununla uygulanır.
func YazimSayaciSonDakika(ctx context.Context, pool *pgxpool.Pool, siteID int64) int {
	var n int
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM ai_jobs
		 WHERE site_id = $1 AND action = $2 AND created_at > now() - interval '1 minute'`,
		siteID, yazimEylem).Scan(&n)
	return n
}

// KuyruktakiYazimSayisi bekleyen/çalışan `news` işi sayısı (panel göstergesi).
func KuyruktakiYazimSayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64) int {
	var n int
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM ai_jobs
		 WHERE site_id = $1 AND action = $2 AND status IN ('queued','running')`,
		siteID, yazimEylem).Scan(&n)
	return n
}

// YazimDurumOzeti panel için tek turda okunan sayaçlardır.
type YazimDurumOzeti struct {
	Bekleyen    int // AI'dan geçmemiş bot taslağı
	Kuyrukta    int // queued/running news işi
	Uygulanacak int // completed ama yazılmamış
	SonUygulama *time.Time
}

// YazimDurumOzetiOku panel göstergesini doldurur.
func YazimDurumOzetiOku(ctx context.Context, pool *pgxpool.Pool, siteID int64) YazimDurumOzeti {
	var o YazimDurumOzeti
	if siteID <= 0 {
		siteID = 1
	}
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM contents c
		 WHERE c.site_id=$1 AND c.status='taslak' AND c.locale='tr'
		   AND c.translation_of IS NULL AND c.type_key='haber'
		   AND c.source_url IS NOT NULL AND COALESCE(c.title,'')<>''
		   AND c.detail <> '[]'::jsonb
		   AND NOT EXISTS (SELECT 1 FROM ai_jobs j WHERE j.content_id=c.id
		                    AND j.action=$2 AND j.status IN ('completed','queued','running'))`,
		siteID, yazimEylem).Scan(&o.Bekleyen)
	o.Kuyrukta = KuyruktakiYazimSayisi(ctx, pool, siteID)
	_ = pool.QueryRow(ctx, `
		SELECT count(*) FROM ai_jobs j JOIN contents c ON c.id=j.content_id
		 WHERE j.site_id=$1 AND j.action=$2 AND j.status='completed'
		   AND j.applied_at IS NULL AND c.status='taslak'`,
		siteID, yazimEylem).Scan(&o.Uygulanacak)
	var son pgtype.Timestamptz
	if err := pool.QueryRow(ctx, `
		SELECT max(applied_at) FROM ai_jobs WHERE site_id=$1 AND action=$2`,
		siteID, yazimEylem).Scan(&son); err == nil && son.Valid {
		t := son.Time
		o.SonUygulama = &t
	}
	return o
}
