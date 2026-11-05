package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- P5-G: Çoklu site pilot dashboard (site × içerik/trafik/gelir/maliyet) ----

// SiteStat bir sitenin karşılaştırma metrikleri.
type SiteStat struct {
	ID        int64
	Name      string
	Domain    string
	Status    string
	Content   int64
	Published int64
	Hits      int64
	Revenue   float64 // hit/1000 × RPM
	AICost    float64
	Profit    float64
}

// SiteStats tüm sitelerin gelir/trafik/maliyet özetini döndürür (RPM kuralı ile).
func SiteStats(ctx context.Context, pool *pgxpool.Pool, rpm float64) ([]SiteStat, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.name, s.domain, s.status,
		  (SELECT count(*) FROM contents c WHERE c.site_id=s.id) AS icerik,
		  (SELECT count(*) FROM contents c WHERE c.site_id=s.id AND c.status='yayinda') AS yayinda,
		  COALESCE((SELECT sum(ch.total) FROM counters_hit ch JOIN contents c ON c.id=ch.content_id WHERE c.site_id=s.id),0) AS hit,
		  COALESCE((SELECT sum(a.cost_est) FROM ai_usage_logs a JOIN contents c ON c.id=a.content_id WHERE c.site_id=s.id),0) AS ai
		FROM sites s ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SiteStat
	for rows.Next() {
		var st SiteStat
		var ai float64
		if err := rows.Scan(&st.ID, &st.Name, &st.Domain, &st.Status, &st.Content, &st.Published, &st.Hits, &ai); err != nil {
			return nil, err
		}
		st.Revenue = float64(st.Hits) / 1000.0 * rpm
		st.AICost = ai
		st.Profit = st.Revenue - ai
		out = append(out, st)
	}
	return out, rows.Err()
}
