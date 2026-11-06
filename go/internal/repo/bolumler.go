package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DoluBolumler yayında içeriği olan bölüm slug'larını döndürür.
//
// NEDEN: menü ve üst gezinme sabit tanımlı; ama kripto portalının bazı
// bölümlerinde (Focus · Videos · Podcasts · News Analysis · Press Releases)
// yayında tek içerik yok. Boş bölüme menü satırı basmak kullanıcıyı
// "içerik yok" sayfasına götürüyordu. Menü bu haritayla süzülür:
// veri geldikçe bölüm kendiliğinden görünür.
//
// Ölçüt YALNIZ `yayinda`: taslak sayısı menüyü doldurmaz — taslak
// ziyaretçiye görünmüyor.
func DoluBolumler(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]bool, error) {
	if siteID <= 0 {
		siteID = 1
	}
	rows, err := pool.Query(ctx, `
		SELECT c.slug
		  FROM categories c
		 WHERE EXISTS (
				SELECT 1 FROM contents k
				 WHERE k.category_id = c.id
				   AND k.status = 'yayinda'
				   AND k.site_id = $1
		 )`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out[slug] = true
	}
	return out, rows.Err()
}
