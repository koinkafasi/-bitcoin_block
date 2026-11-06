package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Uzman ana sayfa "Görüş" modülündeki tek profil.
//
// Kaynak: `users` (rol = Yazar, durum = aktif, avatar_url dolu).
// `Makale`/`MakaleURL` profil sahibinin öne çıkan yazısıdır; şimdilik
// `socials` JSON'unda taşınıyor (son_makale, son_makale_url). Ayrı sütun
// açmak yerine mevcut yapı kullanıldı: bu alanlar profil verisinin
// parçası, içerik tablosuna yabancı anahtar gerektirmiyor.
type Uzman struct {
	Slug         string
	Ad           string
	Unvan        string
	Avatar       string
	Makale       string
	MakaleURL    string
	MakaleKaynak string
}

// Uzmanlar avatarı olan aktif yazarları döndürür.
//
// NEDEN avatar şartı: avatarı olmayan profili "Görüş" satırında basmak
// CoinDesk düzenini bozar (sağda 48 px boşluk kalır). Avatar yoksa profil
// yayına girmemiş sayılır; modül bir sonraki profille dolar.
func Uzmanlar(ctx context.Context, pool *pgxpool.Pool, adet int) ([]Uzman, error) {
	if adet <= 0 {
		adet = 4
	}
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(slug, ''), name, COALESCE(title, ''), avatar_url,
		       COALESCE(socials->>'son_makale', ''),
		       COALESCE(socials->>'son_makale_url', ''),
		       COALESCE(socials->>'son_makale_kaynak', '')
		FROM users
		WHERE role = 'Yazar' AND status = 'aktif' AND avatar_url <> ''
		ORDER BY id
		LIMIT $1`, adet)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Uzman
	for rows.Next() {
		var u Uzman
		if err := rows.Scan(&u.Slug, &u.Ad, &u.Unvan, &u.Avatar, &u.Makale, &u.MakaleURL, &u.MakaleKaynak); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
