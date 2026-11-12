package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/scoring"
)

// IcerikEksen bir içeriğin yayın önceliği eksenleridir (hepsi 0–100).
type IcerikEksen struct {
	Tamlik  int
	Trend   int
	Guven   int
	Tazelik int
	Oncelik int
}

func IcerikEksenleri(ctx context.Context, pool *pgxpool.Pool, ids []int64) (map[int64]IcerikEksen, error) {
	out := make(map[int64]IcerikEksen, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	q := fmt.Sprintf(`
		SELECT c.id,
		       COALESCE(c.ai_score,0),
		       COALESCE(c.score_trend,0),
		       COALESCE(c.score_guven,0),
		       ROUND((%s)::numeric)::int,
		       ROUND((%s)::numeric)::int
		  FROM contents c
		 WHERE c.id = ANY($1)`, scoring.TazelikSQL("c"), scoring.OncelikSQL("c"))
	rows, err := pool.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var e IcerikEksen
		if err := rows.Scan(&id, &e.Tamlik, &e.Trend, &e.Guven, &e.Tazelik, &e.Oncelik); err != nil {
			return nil, err
		}
		out[id] = e
	}
	return out, rows.Err()
}
