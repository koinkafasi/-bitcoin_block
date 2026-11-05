package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSurumCakismasi ortak düzenleme kaydında iyimser eşzamanlılık
// denetiminin tutmadığını bildirir (arada panel formu yazmış).
var ErrSurumCakismasi = errors.New("sürüm çakışması")

// SaveCollabDetail, ortak düzenleme servisinden gelen gövdeyi kaydeder.
// Yalnızca detail alanı değişir; başlık/spot/künye form kaydında kalır.
//
// expected > 0 ise UPDATE yalnız o sürümde geçerlidir (compare-and-swap).
// Tutmazsa ErrSurumCakismasi döner; çağıran 409 ile yanıtlayıp istemcinin
// güncel sürümü almasını sağlar. expected = 0 eski davranıştır.
func SaveCollabDetail(ctx context.Context, pool *pgxpool.Pool, id int64, detail []byte, expected int) (int, int64, string, string, error) {
	var version int
	var siteID int64
	var status, slug string
	var err error
	if expected > 0 {
		err = pool.QueryRow(ctx, `
			UPDATE contents
			SET detail=$2::jsonb, version=version+1, updated_at=now()
			WHERE id=$1 AND version=$3
			RETURNING version, site_id, status, COALESCE(slug,'')`,
			id, string(detail), expected).Scan(&version, &siteID, &status, &slug)
		if errors.Is(err, pgx.ErrNoRows) {
			// Satır yok ile sürüm uyuşmazlığını ayır: içerik gerçekten var mı?
			var varMi bool
			if qerr := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contents WHERE id=$1)`, id).Scan(&varMi); qerr == nil && varMi {
				return 0, 0, "", "", ErrSurumCakismasi
			}
		}
		return version, siteID, status, slug, err
	}
	err = pool.QueryRow(ctx, `
		UPDATE contents
		SET detail=$2::jsonb, version=version+1, updated_at=now()
		WHERE id=$1
		RETURNING version, site_id, status, COALESCE(slug,'')`,
		id, string(detail)).Scan(&version, &siteID, &status, &slug)
	return version, siteID, status, slug, err
}

// ContentVersion içeriğin güncel sürüm numarasını döndürür (0 = bulunamadı).
func ContentVersion(ctx context.Context, pool *pgxpool.Pool, id int64) int {
	var v int
	if err := pool.QueryRow(ctx, `SELECT version FROM contents WHERE id=$1`, id).Scan(&v); err != nil {
		return 0
	}
	return v
}
