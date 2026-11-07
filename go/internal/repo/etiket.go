package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"theblock/site/internal/platform"
)

// baglaEtiketler, kaynağın RSS <category> değerlerini `tags` tablosuna
// yazar ve içeriğe bağlar.
//
// NEDEN kaynağın kendi etiketleri: bunlar UYDURMA değil, kaynağın gerçek
// sınıflandırmasıdır (CoinDesk "Markets"/"Bitcoin", Decrypt "DeFi"…).
// Etiket sayfaları (/tag/{slug}) ve makale altındaki etiket listesi buradan
// beslenir; SEO'da konu kümesi oluşturur.
//
// Kurallar:
//   - Etiket 2-40 karakter; daha kısa/uzun değerler atılır (feed'lerde
//     bozuk/aşırı uzun değerler oluyor).
//   - Aynı içerikte tekrar eden etiket bir kez bağlanır.
//   - En çok 8 etiket: kart/başlık altında sonsuz etiket listesi olmasın.
func baglaEtiketler(ctx context.Context, pool *pgxpool.Pool, contentID int64, adlar []string) error {
	gorulen := map[string]bool{}
	var temiz []string
	for _, a := range adlar {
		a = strings.TrimSpace(a)
		if len([]rune(a)) < 2 || len([]rune(a)) > 40 {
			continue
		}
		slug := platform.Slugify(a)
		if slug == "" || gorulen[slug] {
			continue
		}
		gorulen[slug] = true
		temiz = append(temiz, a)
		if len(temiz) >= 8 {
			break
		}
	}
	if len(temiz) == 0 {
		return nil
	}

	for _, ad := range temiz {
		slug := platform.Slugify(ad)
		var tagID int64
		// ON CONFLICT ile tek sorguda upsert; RETURNING id ile kimliği al.
		err := pool.QueryRow(ctx, `
			INSERT INTO tags (name, slug) VALUES ($1, $2)
			ON CONFLICT (slug) DO UPDATE SET name = tags.name
			RETURNING id`, ad, slug).Scan(&tagID)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO content_tags (content_id, tag_id) VALUES ($1,$2)
			ON CONFLICT DO NOTHING`, contentID, tagID); err != nil {
			return err
		}
	}
	return nil
}

// Etiket bir içerik etiketidir.
type Etiket struct {
	Ad   string
	Slug string
}

// IcerikEtiketleri bir içeriğin etiketlerini (ad + slug) döndürür.
func IcerikEtiketleri(ctx context.Context, pool DB, contentID int64) ([]Etiket, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.name, t.slug FROM content_tags ct
		JOIN tags t ON t.id = ct.tag_id
		WHERE ct.content_id = $1
		ORDER BY t.name`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Etiket
	for rows.Next() {
		var e Etiket
		if err := rows.Scan(&e.Ad, &e.Slug); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EtiketlerCoklu birden çok içeriğin etiketlerini tek sorguda döndürür
// (içerik kimliği → etiketler).
//
// NEDEN: RSS beslemesi 50 öğe basıyor; öğe başına ayrı sorgu 50 gidiş-dönüş
// demekti. Bu sürüm tek `= ANY($1)` sorgusu kullanır.
func EtiketlerCoklu(ctx context.Context, pool DB, contentIDs []int64) (map[int64][]Etiket, error) {
	out := make(map[int64][]Etiket, len(contentIDs))
	if len(contentIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ct.content_id, t.name, t.slug FROM content_tags ct
		JOIN tags t ON t.id = ct.tag_id
		WHERE ct.content_id = ANY($1)
		ORDER BY ct.content_id, t.name`, contentIDs)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var e Etiket
		if err := rows.Scan(&id, &e.Ad, &e.Slug); err != nil {
			return out, err
		}
		out[id] = append(out[id], e)
	}
	return out, rows.Err()
}

// PopulerEtiketler içeriği olan etiketleri, içerik sayısına göre döndürür.
//
// NEDEN "içeriği olan": mega menüdeki etiket bağlantıları buradan beslenir.
// İçeriği olmayan bir etikete menüde yer vermek kullanıcıyı 404'e götürür
// (ölçüm: /tag/xrp 404 idi).
func PopulerEtiketler(ctx context.Context, pool DB, adet int) ([]Etiket, error) {
	if adet <= 0 {
		adet = 5
	}
	rows, err := pool.Query(ctx, `
		SELECT t.name, t.slug, count(*) AS n
		FROM content_tags ct
		JOIN tags t ON t.id = ct.tag_id
		JOIN contents c ON c.id = ct.content_id
		WHERE c.status = 'yayinda' AND c.site_id = 1
		GROUP BY t.name, t.slug
		HAVING count(*) >= 2
		ORDER BY n DESC, t.name
		LIMIT $1`, adet)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Etiket
	for rows.Next() {
		var e Etiket
		var n int
		if err := rows.Scan(&e.Ad, &e.Slug, &n); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
