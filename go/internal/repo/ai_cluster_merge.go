package repo

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// aiExistingCluster, yeni öğelerin birleştirilebileceği son kümeleri temsil
// eder. Yalnızca taslağa dönüşmemiş "yeni" kümeler aday olur.
type aiExistingCluster struct {
	ID          int64
	CategoryID  *int64
	Title       string
	ItemCount   int
	SourceCount int
	Toks        map[string]bool
	Added       []AISourceItem
}

func aiCategoryKey(cat *int64) int64 {
	if cat == nil {
		return 0
	}
	return *cat
}

func aiCategoryPtr(key int64) *int64 {
	if key == 0 {
		return nil
	}
	id := key
	return &id
}

func sameAICategory(a, b *int64) bool {
	return aiCategoryKey(a) == aiCategoryKey(b)
}

// aiOverlapDetail, iki başlık token kümesinin min-bazlı kesişim skorunu,
// ortak token sayısını ve en az bir uzun ortak token olup olmadığını döner.
func aiOverlapDetail(a, b map[string]bool) (float64, int, bool) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0, false
	}
	common, long := 0, false
	for k := range a {
		if b[k] {
			common++
			if len([]rune(k)) >= 5 {
				long = true
			}
		}
	}
	min := len(a)
	if len(b) < min {
		min = len(b)
	}
	if min == 0 {
		return 0, common, long
	}
	return float64(common) / float64(min), common, long
}

// aiMatchTitle kategori içi eşikleri korur; kategoriler arası eşleşmede daha
// yüksek skor, en az iki ortak token ve anlamlı bir uzun token ister.
func aiMatchTitle(a, b map[string]bool, sameCategory, existing bool) (float64, bool) {
	score, common, long := aiOverlapDetail(a, b)
	if sameCategory {
		threshold := 0.48
		if existing {
			threshold = 0.52
		}
		return score, score >= threshold
	}
	threshold := 0.68
	if existing {
		threshold = 0.70
	}
	if score < threshold || common < 2 || !long {
		return 0, false
	}
	return score, true
}

func loadAIExistingClusterCandidates(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]*aiExistingCluster, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, category_id, title, item_count, source_count
		FROM ai_story_clusters
		WHERE site_id=$1 AND status='yeni' AND content_id IS NULL
		  AND updated_at > now() - interval '7 days'
		ORDER BY updated_at DESC, score DESC
		LIMIT 800`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*aiExistingCluster
	for rows.Next() {
		c := &aiExistingCluster{}
		if err := rows.Scan(&c.ID, &c.CategoryID, &c.Title, &c.ItemCount, &c.SourceCount); err != nil {
			return nil, err
		}
		c.Toks = aiTitleTokens(c.Title)
		out = append(out, c)
	}
	return out, rows.Err()
}

// refreshAIClusterCategory, kümenin öğe kategorilerine göre baskın
// kategorisini yeniden hesaplar. Taslağa dönüşmüş kümelerin kategorisi
// değiştirilmez.
func refreshAIClusterCategory(ctx context.Context, pool *pgxpool.Pool, clusterID int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE ai_story_clusters c
		SET category_id=COALESCE((
			SELECT i.category_id
			FROM ai_story_cluster_items ci
			JOIN ai_source_items i ON i.id=ci.item_id
			WHERE ci.cluster_id=$1
			GROUP BY i.category_id
			ORDER BY count(*) DESC, i.category_id NULLS LAST
			LIMIT 1
		), c.category_id),
		updated_at=now()
		WHERE c.id=$1 AND c.content_id IS NULL`, clusterID)
	return err
}

// refreshAICluster, kümenin öğe/kaynak sayılarını, skorunu, baskın
// kategorisini ve boş özetini günceller.
func refreshAICluster(ctx context.Context, pool *pgxpool.Pool, clusterID int64) error {
	if _, err := pool.Exec(ctx, `
		UPDATE ai_story_clusters c
		SET item_count=s.n,
		    source_count=s.s,
		    score=s.s*20 + s.n*5 + c.trend_score,
		    updated_at=now()
		FROM (
			SELECT count(*)::int AS n, count(DISTINCT i.source_name)::int AS s
			FROM ai_story_cluster_items ci
			JOIN ai_source_items i ON i.id=ci.item_id
			WHERE ci.cluster_id=$1
		) s
		WHERE c.id=$1`, clusterID); err != nil {
		return err
	}
	if err := refreshAIClusterCategory(ctx, pool, clusterID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		UPDATE ai_story_clusters c
		SET summary=COALESCE((
			SELECT i.summary
			FROM ai_story_cluster_items ci
			JOIN ai_source_items i ON i.id=ci.item_id
			WHERE ci.cluster_id=$1 AND i.summary <> ''
			ORDER BY i.published_at DESC NULLS LAST, i.id DESC
			LIMIT 1
		), c.summary),
		updated_at=now()
		WHERE c.id=$1 AND (c.summary='' OR c.summary=c.title)`, clusterID)
	return err
}

type aiMergeCluster struct {
	ID          int64
	CategoryID  *int64
	Title       string
	ItemCount   int
	SourceCount int
	TrendScore  int
	Toks        map[string]bool
}

type aiMergePair struct {
	A     *aiMergeCluster
	B     *aiMergeCluster
	Score float64
}

// MergeCrossCategoryAIStoryClusters, geçmişte ayrı kategorilerde oluşmuş ama
// aynı olayı anlatan kümeleri birleştirir. Yalnızca taslağa dönüşmemiş,
// yüksek benzerlikli kümeler işlenir; her turda en fazla limit kadar birleşme
// yapılır.
func MergeCrossCategoryAIStoryClusters(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) (int, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 80 {
		limit = 80
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.category_id, c.title, c.item_count, c.source_count, c.trend_score
		FROM ai_story_clusters c
		WHERE c.site_id=$1 AND c.status='yeni' AND c.content_id IS NULL
		  AND c.created_at > now() - interval '7 days'
		ORDER BY c.item_count DESC, c.created_at ASC
		LIMIT 1200`, siteID)
	if err != nil {
		return 0, err
	}
	var clusters []*aiMergeCluster
	for rows.Next() {
		c := &aiMergeCluster{}
		if err := rows.Scan(&c.ID, &c.CategoryID, &c.Title, &c.ItemCount, &c.SourceCount, &c.TrendScore); err != nil {
			rows.Close()
			return 0, err
		}
		c.Toks = aiTitleTokens(c.Title)
		clusters = append(clusters, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var pairs []aiMergePair
	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			if sameAICategory(clusters[i].CategoryID, clusters[j].CategoryID) {
				continue
			}
			score, ok := aiMatchTitle(clusters[i].Toks, clusters[j].Toks, false, true)
			if ok && score >= 0.75 {
				pairs = append(pairs, aiMergePair{A: clusters[i], B: clusters[j], Score: score})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		return pairs[i].A.ItemCount+pairs[i].B.ItemCount > pairs[j].A.ItemCount+pairs[j].B.ItemCount
	})

	active := map[int64]bool{}
	for _, c := range clusters {
		active[c.ID] = true
	}
	merged := 0
	for _, p := range pairs {
		if merged >= limit {
			break
		}
		if !active[p.A.ID] || !active[p.B.ID] {
			continue
		}
		canon, donor := p.A, p.B
		if donor.ItemCount > canon.ItemCount ||
			(donor.ItemCount == canon.ItemCount && donor.SourceCount > canon.SourceCount) {
			canon, donor = donor, canon
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ai_story_cluster_items (cluster_id, item_id)
			SELECT $1, item_id FROM ai_story_cluster_items WHERE cluster_id=$2
			ON CONFLICT DO NOTHING`, canon.ID, donor.ID); err != nil {
			continue
		}
		tag, err := pool.Exec(ctx, `
			DELETE FROM ai_story_clusters
			WHERE id=$1 AND site_id=$2 AND status='yeni' AND content_id IS NULL`, donor.ID, siteID)
		if err != nil || tag.RowsAffected() == 0 {
			continue
		}
		if err := refreshAICluster(ctx, pool, canon.ID); err != nil {
			continue
		}
		active[donor.ID] = false
		canon.ItemCount += donor.ItemCount
		canon.SourceCount += donor.SourceCount
		merged++
	}
	return merged, nil
}
