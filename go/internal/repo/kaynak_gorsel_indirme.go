package repo

import (
	"context"
	"time"
)

// ---- Kaynak görsellerini önden indirme ----
//
// NEDEN: `/kg/<hash>` tembel çalışıyor — görsel ancak biri sayfayı
// açınca iniyor. Ölçüm (12 Kasım 2026): 4.918 kayıttan yalnız 136'sı
// diskte. Kalan %97 hâlâ uzak sunucuya bağlı.
//
// Bunun bedeli daha önce ödendi: eski WordPress görselleri kaynakta
// kayboldu (cdn.bitcosar.com artık 403 dönüyor) ve geri getirilemedi.
// Haber kaynakları da görsellerini taşır, siler, adres değiştirir.
// İndirilmeyen görsel er geç kaybolur.

// IndirilecekGorsel önden indirme kuyruğundaki bir kayıt.
type IndirilecekGorsel struct {
	Hash string
	URL  string
}

// GorselIndirmeDenemeSiniri: bu kadar başarısızlıktan sonra kayıt
// bırakılır.
//
// 5 seçildi: geçici kesintiler (502, zaman aşımı, hız sınırı) birkaç
// denemede düzeliyor. Ölü adres ise hiç düzelmez; sınırsız deneme her
// turda aynı 404'leri çekip bant genişliği harcardı.
const GorselIndirmeDenemeSiniri = 5

// GorselYenidenDenemeAraligi: başarısız bir adrese bu süre geçmeden
// yeniden dokunulmaz.
const GorselYenidenDenemeAraligi = 6 * time.Hour

// IndirilecekGorseller indirilmemiş görselleri döndürür.
//
// Sıra YENİDEN ESKİYE: yeni haberlerin görseli hem daha çok okunuyor
// hem de kaynağında daha büyük olasılıkla duruyor. Eski kayıtlar zaten
// kaybolmuşsa denemenin değeri düşük.
func IndirilecekGorseller(ctx context.Context, pool DB, limit int) ([]IndirilecekGorsel, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT hash, url
		  FROM source_images
		 WHERE bytes = 0
		   AND COALESCE(indirme_deneme, 0) < $1
		   AND (son_deneme_at IS NULL OR son_deneme_at < now() - $2::interval)
		 ORDER BY created_at DESC
		 LIMIT $3`,
		GorselIndirmeDenemeSiniri,
		GorselYenidenDenemeAraligi.String(),
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []IndirilecekGorsel
	for rows.Next() {
		var g IndirilecekGorsel
		if err := rows.Scan(&g.Hash, &g.URL); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GorselIndirmeBasarisiz denemeyi kaydeder.
//
// Sayaç artar ve son deneme zamanı yazılır; ikisi birlikte "bu adrese
// tekrar ne zaman bakılacağını" belirler.
func GorselIndirmeBasarisiz(ctx context.Context, pool DB, hash, hata string) {
	if len(hata) > 300 {
		hata = hata[:300]
	}
	_, _ = pool.Exec(ctx, `
		UPDATE source_images
		   SET indirme_deneme = COALESCE(indirme_deneme, 0) + 1,
		       son_deneme_at  = now(),
		       son_hata       = $2
		 WHERE hash = $1`, hash, hata)
}

// GorselIndirmeOzeti panel/günlük için tek satırlık durum.
type GorselIndirmeOzeti struct {
	Toplam     int
	Indirilmis int
	Bekleyen   int
	Vazgecilen int // deneme sınırına ulaşmış
	ToplamBayt int64
}

// GorselIndirmeDurumu indirme ilerleyişini döndürür.
func GorselIndirmeDurumu(ctx context.Context, pool DB) (GorselIndirmeOzeti, error) {
	var o GorselIndirmeOzeti
	err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE bytes > 0),
		       count(*) FILTER (WHERE bytes = 0 AND COALESCE(indirme_deneme,0) < $1),
		       count(*) FILTER (WHERE bytes = 0 AND COALESCE(indirme_deneme,0) >= $1),
		       COALESCE(sum(bytes), 0)
		  FROM source_images`, GorselIndirmeDenemeSiniri).
		Scan(&o.Toplam, &o.Indirilmis, &o.Bekleyen, &o.Vazgecilen, &o.ToplamBayt)
	return o, err
}
