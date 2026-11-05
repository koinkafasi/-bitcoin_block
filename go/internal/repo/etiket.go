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

// Etiket bir içerik etiketi.
type Etiket struct {
	Ad   string
	Slug string
}
