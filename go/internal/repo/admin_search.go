package repo

import (
	"context"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

// slugQuery sorgu parametresini URL için kaçırır. Başlıklarda Türkçe
// karakter, boşluk ve & bulunabildiği için ham birleştirme yapılamaz.
func slugQuery(v string) string { return url.QueryEscape(v) }

// AdminSearchSonuc panel komut paletinin varlık araması (Faz 162).
//
// NEDEN AYRI DOSYA: palet önceden yalnız sabit rota listesiydi (38 kayıt,
// static/app.js). Kullanıcı bir içerik başlığı yazdığında sonuç gelmiyor,
// "İçerikler" sayfası açılıyordu. Burada amaç hızlı, sınırlı, site kapsamlı
// bir arama: her gruptan en çok `limit` kayıt, tek turda.
//
// Indeks notu: `contents.title` üzerinde pg_trgm indeksi varsa ILIKE
// indeksten yararlanır; yoksa bile LIMIT sayesinde tarama sınırlı kalır
// (panelde içerik sayısı on binler mertebesinde).
type AdminAramaOge struct {
	ID     int64  `json:"id"`
	Baslik string `json:"baslik"`
	Alt    string `json:"alt"`   // ikincil satır: kategori · durum · tarih
	Href   string `json:"href"`  // panelde açılacak yol
	Rozet  string `json:"rozet"` // durum/rol kısa etiketi
	Tur    string `json:"tur"`   // "yayinda" | "taslak" … (renk için)
}

type AdminAramaGrup struct {
	Ad   string          `json:"ad"`
	Ikon string          `json:"ikon"`
	Oge  []AdminAramaOge `json:"oge"`
}

// AdminArama site kapsamlı varlık araması yapar.
func AdminArama(ctx context.Context, pool *pgxpool.Pool, siteID int64, q string, limit int) ([]AdminAramaGrup, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	// Türkçe katlama: "saglik" yazan kullanıcı "Sağlık" başlığını bulmalı.
	// unaccent ğ→g ve ı→i eşlemesini yapar (migration 0098). Sorgu da
	// katlanır, aksi hâlde desen "saglik" kalıp sütun "saglik" ile eşleşirdi
	// ama tersi (kullanıcı "Sağlık" yazarsa) çalışmazdı.
	desen := "%" + q + "%"
	var gruplar []AdminAramaGrup

	// 1) İçerik — en çok kullanılan arama. Slug da eşleşir (URL'den yapıştırma).
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, c.status, c.locale,
		       COALESCE(cat.name,'') , to_char(c.updated_at,'DD.MM.YYYY')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		WHERE c.site_id = $1
		  AND (
		        lower(public.immutable_unaccent(c.title)) LIKE lower(public.immutable_unaccent($2))
		     OR lower(public.immutable_unaccent(COALESCE(c.spot,''))) LIKE lower(public.immutable_unaccent($2))
		     OR lower(public.immutable_unaccent(c.slug)) LIKE lower(public.immutable_unaccent($2))
		  )
		ORDER BY c.updated_at DESC
		LIMIT $3`, siteID, desen, limit)
	if err != nil {
		return nil, err
	}
	var icerik []AdminAramaOge
	for rows.Next() {
		var id int64
		var baslik, durum, yerel, kategori, tarih string
		if err := rows.Scan(&id, &baslik, &durum, &yerel, &kategori, &tarih); err != nil {
			rows.Close()
			return nil, err
		}
		alt := tarih
		if kategori != "" {
			alt = kategori + " · " + tarih
		}
		icerik = append(icerik, AdminAramaOge{
			ID: id, Baslik: baslik, Alt: alt, Rozet: durum, Tur: durum,
			Href: "/admin/icerikler/" + itoa64(id),
		})
	}
	rows.Close()
	if len(icerik) > 0 {
		gruplar = append(gruplar, AdminAramaGrup{Ad: "İçerik", Ikon: "file-text", Oge: icerik})
	}

	// 2) Kategori
	rows, err = pool.Query(ctx, `
		SELECT id, name, slug FROM categories
		WHERE site_id = $1
		  AND (lower(public.immutable_unaccent(name)) LIKE lower(public.immutable_unaccent($2))
		    OR lower(public.immutable_unaccent(slug)) LIKE lower(public.immutable_unaccent($2)))
		ORDER BY name LIMIT $3`, siteID, desen, limit)
	if err != nil {
		return nil, err
	}
	var kategoriler []AdminAramaOge
	for rows.Next() {
		var id int64
		var ad, slug string
		if err := rows.Scan(&id, &ad, &slug); err != nil {
			rows.Close()
			return nil, err
		}
		kategoriler = append(kategoriler, AdminAramaOge{
			ID: id, Baslik: ad, Alt: slug,
			Href: "/admin/kategoriler?duzenle=" + itoa64(id),
		})
	}
	rows.Close()
	if len(kategoriler) > 0 {
		gruplar = append(gruplar, AdminAramaGrup{Ad: "Kategori", Ikon: "folder", Oge: kategoriler})
	}

	// 3) Etiket
	rows, err = pool.Query(ctx, `
		SELECT id, name, slug FROM tags
		WHERE lower(public.immutable_unaccent(name)) LIKE lower(public.immutable_unaccent($1))
		   OR lower(public.immutable_unaccent(slug)) LIKE lower(public.immutable_unaccent($1))
		ORDER BY name LIMIT $2`, desen, limit)
	if err != nil {
		return nil, err
	}
	var etiketler []AdminAramaOge
	for rows.Next() {
		var id int64
		var ad, slug string
		if err := rows.Scan(&id, &ad, &slug); err != nil {
			rows.Close()
			return nil, err
		}
		etiketler = append(etiketler, AdminAramaOge{
			ID: id, Baslik: ad, Alt: "#" + slug,
			Href: "/admin/etiketler?q=" + slugQuery(slug),
		})
	}
	rows.Close()
	if len(etiketler) > 0 {
		gruplar = append(gruplar, AdminAramaGrup{Ad: "Etiket", Ikon: "tag", Oge: etiketler})
	}

	// 4) Medya
	rows, err = pool.Query(ctx, `
		SELECT id, COALESCE(NULLIF(title,''), filename), COALESCE(mime,''), COALESCE(kind,'')
		FROM media_files
		WHERE site_id = $1
		  AND (lower(public.immutable_unaccent(filename)) LIKE lower(public.immutable_unaccent($2))
		    OR lower(public.immutable_unaccent(COALESCE(title,''))) LIKE lower(public.immutable_unaccent($2))
		    OR lower(public.immutable_unaccent(COALESCE(alt,''))) LIKE lower(public.immutable_unaccent($2)))
		ORDER BY created_at DESC LIMIT $3`, siteID, desen, limit)
	if err != nil {
		return nil, err
	}
	var medya []AdminAramaOge
	for rows.Next() {
		var id int64
		var ad, mime, kind string
		if err := rows.Scan(&id, &ad, &mime, &kind); err != nil {
			rows.Close()
			return nil, err
		}
		alt := kind
		if mime != "" {
			alt = mime
		}
		medya = append(medya, AdminAramaOge{
			ID: id, Baslik: ad, Alt: alt, Href: "/admin/dosyalar?q=" + slugQuery(ad),
		})
	}
	rows.Close()
	if len(medya) > 0 {
		gruplar = append(gruplar, AdminAramaGrup{Ad: "Medya", Ikon: "photo", Oge: medya})
	}

	// 5) Kullanıcı (yazar/editör)
	rows, err = pool.Query(ctx, `
		SELECT id, name, role FROM users
		WHERE lower(public.immutable_unaccent(name)) LIKE lower(public.immutable_unaccent($1))
		   OR lower(public.immutable_unaccent(email::text)) LIKE lower(public.immutable_unaccent($1))
		ORDER BY name LIMIT $2`, desen, limit)
	if err != nil {
		return nil, err
	}
	var kisiler []AdminAramaOge
	for rows.Next() {
		var id int64
		var ad, rol string
		if err := rows.Scan(&id, &ad, &rol); err != nil {
			rows.Close()
			return nil, err
		}
		kisiler = append(kisiler, AdminAramaOge{
			ID: id, Baslik: ad, Alt: rol, Rozet: rol,
			Href: "/admin/yazarlar/" + itoa64(id),
		})
	}
	rows.Close()
	if len(kisiler) > 0 {
		gruplar = append(gruplar, AdminAramaGrup{Ad: "Kişi", Ikon: "user", Oge: kisiler})
	}

	return gruplar, nil
}
