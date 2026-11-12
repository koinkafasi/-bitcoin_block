package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Kaynak sağlığı ----
//
// NEDEN: `feeds` yalnız SON taramanın sonucunu tutuyordu. Bozulan bir
// kaynak kapalı listede sessizce duruyor, "neden kapalı?" sorusunu
// kimse yanıtlayamıyordu (12 Kasım 2026 denetimi: 34 kaynağın 22'si
// kapalı; bir kısmı sağlam, bir kısmı yıllardır ölü).

// OtoKapatmaEsigi: bu kadar ardışık hatadan sonra kaynak kendiliğinden
// devre dışı bırakılır.
//
// 5 seçildi: geçici kesintiler (CDN, 429, bakım) genelde 1-2 taramada
// düzeliyor; 5 ardışık hata gerçek bir bozulmaya işaret ediyor. Daha
// düşük eşik sağlam kaynakları gereksiz kapatırdı.
const OtoKapatmaEsigi = 5

// FeedBasarili başarılı çekimi kaydeder: sayaç sıfırlanır, son başarı
// zamanı ve kalite ölçümü güncellenir.
// gorselOran NEGATİF verilirse mevcut değer korunur: her çekim yolu
// görsel bilgisini taşımıyor (bot'un içerik üretim döngüsü yalnız öğe
// sayısını biliyor). Sıfır yazmak gerçek ölçümü ezerdi.
func FeedBasarili(ctx context.Context, pool *pgxpool.Pool, id int64, ogeSayisi, gorselOran int) error {
	if gorselOran > 100 {
		gorselOran = 100
	}
	_, err := pool.Exec(ctx, `
		UPDATE feeds
		   SET last_checked_at  = now(),
		       last_ok_at       = now(),
		       last_status      = 'ok',
		       last_error       = '',
		       last_item_count  = $2,
		       gorsel_oran      = CASE WHEN $3 >= 0 THEN $3 ELSE gorsel_oran END,
		       ardisik_hata     = 0,
		       updated_at       = now()
		 WHERE id = $1`, id, ogeSayisi, gorselOran)
	return err
}

// FeedBasarisiz hatayı kaydeder ve eşiğe ulaşılmışsa kaynağı kapatır.
//
// Dönen `kapandi` true ise kaynak bu çağrıda devre dışı bırakılmıştır.
// Elle kapatılmış kaynaklar (otomatik_kapandi=false, aktif=false)
// bundan etkilenmez: zaten kapalılar.
func FeedBasarisiz(ctx context.Context, pool *pgxpool.Pool, id int64, hata string) (kapandi bool, err error) {
	if len(hata) > 500 {
		hata = hata[:500]
	}
	err = pool.QueryRow(ctx, `
		UPDATE feeds
		   SET last_checked_at = now(),
		       last_status     = 'hata',
		       last_error      = $2,
		       ardisik_hata    = ardisik_hata + 1,
		       aktif            = CASE WHEN ardisik_hata + 1 >= $3 THEN false ELSE aktif END,
		       otomatik_kapandi = CASE WHEN ardisik_hata + 1 >= $3 AND aktif THEN true ELSE otomatik_kapandi END,
		       updated_at      = now()
		 WHERE id = $1
		 RETURNING (ardisik_hata >= $3 AND otomatik_kapandi)`,
		id, hata, OtoKapatmaEsigi).Scan(&kapandi)
	return kapandi, err
}

// FeedYenidenAc kaynağı elle açar ve hata sayacını sıfırlar.
//
// Sayaç sıfırlanmazsa kaynak ilk hatada yeniden kapanırdı — kullanıcı
// "açtım ama kapanıyor" derdi.
func FeedYenidenAc(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE feeds
		   SET aktif = true, ardisik_hata = 0, otomatik_kapandi = false,
		       updated_at = now()
		 WHERE id = $1`, id)
	return err
}

// FeedSaglikOzeti panel için tek satırlık durum.
type FeedSaglikOzeti struct {
	Toplam          int
	Acik            int
	Kapali          int
	OtomatikKapanan int
	HicCalismayan   int // last_ok_at boş
	GorselsizAcik   int // açık ama görsel oranı düşük
}

// FeedSaglik kaynakların toplu durumunu döndürür.
func FeedSaglik(ctx context.Context, pool *pgxpool.Pool) (FeedSaglikOzeti, error) {
	var o FeedSaglikOzeti
	err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE aktif),
		       count(*) FILTER (WHERE NOT aktif),
		       count(*) FILTER (WHERE otomatik_kapandi),
		       count(*) FILTER (WHERE last_ok_at IS NULL),
		       count(*) FILTER (WHERE aktif AND coalesce(gorsel_oran, 0) < 30)
		  FROM feeds`).Scan(&o.Toplam, &o.Acik, &o.Kapali,
		&o.OtomatikKapanan, &o.HicCalismayan, &o.GorselsizAcik)
	return o, err
}
