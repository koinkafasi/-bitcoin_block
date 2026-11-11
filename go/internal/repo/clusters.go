package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cluster konu kümesidir (pillar/cluster SEO yönetimi).
type Cluster struct {
	ID           int64
	SiteID       int64
	Name         string
	Intent       string
	PillarID     *int64
	ContentCount int
}

// CreateCluster küme ekler.
func CreateCluster(ctx context.Context, pool *pgxpool.Pool, siteID int64, name, intent string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO keyword_clusters (site_id, name, intent) VALUES ($1,$2,$3) RETURNING id`,
		siteID, name, intent).Scan(&id)
	return id, err
}

// ListClusters site kümelerini içerik sayısıyla döndürür.
func ListClusters(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]Cluster, error) {
	rows, err := pool.Query(ctx, `
		SELECT k.id, k.site_id, k.name, k.intent, k.pillar_id,
		       (SELECT count(*) FROM content_clusters cc WHERE cc.cluster_id = k.id)
		FROM keyword_clusters k WHERE k.site_id = $1 ORDER BY k.name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cluster
	for rows.Next() {
		var c Cluster
		if err := rows.Scan(&c.ID, &c.SiteID, &c.Name, &c.Intent, &c.PillarID, &c.ContentCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteCluster küme siler.
func DeleteCluster(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM keyword_clusters WHERE id = $1`, id)
	return err
}

// SetContentCluster içeriği tek kümeye bağlar (diğer bağlar silinir).
func SetContentCluster(ctx context.Context, pool *pgxpool.Pool, contentID, clusterID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM content_clusters WHERE content_id = $1`, contentID); err != nil {
		return err
	}
	if clusterID > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO content_clusters (content_id, cluster_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, contentID, clusterID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ContentClusterID içeriğin bağlı küme id'sini döndürür (0 = yok).
func ContentClusterID(ctx context.Context, pool *pgxpool.Pool, contentID int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT cluster_id FROM content_clusters WHERE content_id = $1 LIMIT 1`, contentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// ClusterContents kümeye bağlı içerikleri döndürür.
func ClusterContents(ctx context.Context, pool *pgxpool.Pool, clusterID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		JOIN content_clusters cc ON cc.content_id = c.id
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE cc.cluster_id = $1 ORDER BY c.updated_at DESC LIMIT $2`
	rows, err := pool.Query(ctx, q, clusterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ClusterContentsPool küme ve site için "eklenecek içerik" havuzu.
func ClusterContentsPool(ctx context.Context, pool *pgxpool.Pool, siteID, clusterID int64, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 80
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id = $1 AND c.id NOT IN (SELECT content_id FROM content_clusters WHERE cluster_id = $2)
		ORDER BY c.updated_at DESC LIMIT $3`
	rows, err := pool.Query(ctx, q, siteID, clusterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CannPair cannibalization (içerik çakışması) çiftidir.
type CannPair struct {
	A, B    int64
	TitleA  string
	TitleB  string
	Overlap int
}

// Cannibalization küme içindeki benzer başlık çiftlerini bulur.
func Cannibalization(ctx context.Context, pool *pgxpool.Pool, clusterID int64) ([]CannPair, error) {
	items, err := ClusterContents(ctx, pool, clusterID, 60)
	if err != nil {
		return nil, err
	}
	var out []CannPair
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			ov := titleOverlap(items[i].Title, items[j].Title)
			if ov >= 2 {
				out = append(out, CannPair{A: items[i].ID, TitleA: items[i].Title, B: items[j].ID, TitleB: items[j].Title, Overlap: ov})
			}
		}
	}
	return out, nil
}

// titleOverlap iki başlıktaki ortak anlamlı kelime sayısı.
func titleOverlap(a, b string) int {
	ta := tokenSet(a)
	tb := tokenSet(b)
	n := 0
	for w := range ta {
		if tb[w] {
			n++
		}
	}
	return n
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= '\u00e0') }) {
		if len(w) > 2 {
			out[w] = true
		}
	}
	return out
}
