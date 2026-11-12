package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalizKart akan şeritteki tek kart.
type AnalizKart struct {
	Baslik   string
	Spot     string
	Kategori string
	Href     string
	Tarih    time.Time
}

func AnalizAkisi(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, limit int) ([]AnalizKart, error) {
	if limit <= 0 {
		limit = 12
	}
	if strings.TrimSpace(locale) == "" {
		locale = "tr"
	}
	rows, err := pool.Query(ctx, `
		SELECT c.title, c.spot, k.name, c.slug, c.id, c.published_at, k.slug
		FROM contents c
		JOIN categories k ON k.id = c.category_id
		WHERE c.status = 'yayinda'
		  AND c.locale = $1
		  AND c.site_id = $2
		  AND COALESCE(c.spot, '') <> ''
		  AND c.published_at IS NOT NULL
		  AND k.name IN ('Haber Analizi', 'Araştırma', 'Görüş', 'Piyasalar')
		ORDER BY c.published_at DESC
		LIMIT $3`, locale, siteIDOr1(siteID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AnalizKart
	for rows.Next() {
		var k AnalizKart
		var slug, katSlug string
		var id int64
		var yayin *time.Time
		if err := rows.Scan(&k.Baslik, &k.Spot, &k.Kategori, &slug, &id, &yayin, &katSlug); err != nil {
			return nil, err
		}
		if yayin != nil {
			k.Tarih = *yayin
		}
		k.Href = analizKartURL(katSlug, slug, yayin)
		out = append(out, k)
	}
	return out, rows.Err()
}

// analizKartURL kart bağlantısı: /{bolum}/{yyyy}/{mm}/{dd}/{slug}.
// Site bu biçimi kanonik sayıyor; tarih yanlışsa zaten 301'liyor.
func analizKartURL(bolum, slug string, yayin *time.Time) string {
	if slug == "" {
		return ""
	}
	if bolum == "" {
		bolum = "haberler"
	}
	if yayin == nil {
		return "/" + bolum + "/" + slug
	}
	return "/" + bolum + "/" + yayin.Format("2006/01/02") + "/" + slug
}

// AnalizSayisi son N günde yayımlanan analiz içeriği sayısı (modül rozeti).
func AnalizSayisi(ctx context.Context, pool *pgxpool.Pool, siteID int64, locale string, gun int) int {
	if gun <= 0 {
		gun = 7
	}
	if strings.TrimSpace(locale) == "" {
		locale = "tr"
	}
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM contents c
		JOIN categories k ON k.id = c.category_id
		WHERE c.status = 'yayinda'
		  AND c.locale = $1
		  AND c.site_id = $2
		  AND c.published_at >= now() - make_interval(days => $3)
		  AND k.name IN ('Haber Analizi', 'Araştırma', 'Görüş', 'Piyasalar')`,
		locale, siteIDOr1(siteID), gun).Scan(&n)
	if err != nil {
		return 0
	}
	return n
}
