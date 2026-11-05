package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LastPublishedForPlan yayın şeridi boş kaldığında kullanılan yedek listeyi
// döndürür: son yayınlanan içerikler, tarih farkı gözetmeden.
//
// Neden gerekli: şerit varsayılan olarak "-6 saat … +30 saat" penceresine
// bakıyor. Site birkaç gün yayın yapmadıysa ya da planlı içerik yoksa şerit
// tamamen boş kalıyor ve panonun en görünür bileşeni kayboluyordu. Boş kalan
// yerde "son yayınlar" göstermek, boş kutu göstermekten iyidir.
func LastPublishedForPlan(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) []PlanRow {
	if limit <= 0 || limit > 40 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, c.published_at, c.status, COALESCE(cat.name, ''), COALESCE(c.type_key, '')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		WHERE c.site_id = $1 AND c.status = 'yayinda' AND c.published_at IS NOT NULL
		ORDER BY c.published_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []PlanRow
	for rows.Next() {
		var p PlanRow
		if err := rows.Scan(&p.ID, &p.Title, &p.When, &p.Status, &p.Category, &p.TypeKey); err != nil {
			return out
		}
		out = append(out, p)
	}
	// Şerit soldan sağa zaman ekseni olduğu için eskiden yeniye çevrilir.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// plan zaman penceresi hesapları şerit için tek yerde.
