package repo

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AIEnrichResult bir zenginleştirme turunun sonucudur.
type AIEnrichResult struct {
	Enriched      int
	Failed        int
	Pending       int
	Recategorized int
}

type aiEnrichCandidate struct {
	ID                int64
	Title             string
	URL               string
	Summary           string
	PublishedAt       *time.Time
	CategoryID        *int64
	NeedsCategoryOnly bool
}

// EnrichAISourceItems, HTML/sitemap kaynaklarından alınmış öğeler için
// JSON-LD/meta özeti, yayın tarihi ve makale bölümüne göre kanonik kategori
// çeker. Tam metin saklanmaz; yalnızca kısa açıklama ve künye alanları
// güncellenir.
//
// Daha önce zenginleştirilmiş ama kategori kontrolü yapılmamış öğeler de
// (category_checked_at IS NULL) kategori düzeltmesi için yeniden işlenir.
func EnrichAISourceItems(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) (AIEnrichResult, error) {
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 {
		limit = 60
	}
	if limit > 120 {
		limit = 120
	}
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.title, i.url, i.summary, i.published_at, i.category_id,
		       (i.enriched_at IS NOT NULL) AS needs_category_only
		FROM ai_source_items i
		JOIN feeds f ON f.id=i.feed_id
		WHERE i.site_id=$1 AND f.kind IN ('html','sitemap')
		  AND (i.enriched_at IS NULL OR i.category_checked_at IS NULL)
		ORDER BY (i.enriched_at IS NULL) DESC, i.fetched_at DESC, i.id DESC
		LIMIT $2`, siteID, limit)
	if err != nil {
		return AIEnrichResult{}, err
	}
	var candidates []aiEnrichCandidate
	for rows.Next() {
		var c aiEnrichCandidate
		if err := rows.Scan(&c.ID, &c.Title, &c.URL, &c.Summary, &c.PublishedAt,
			&c.CategoryID, &c.NeedsCategoryOnly); err != nil {
			rows.Close()
			return AIEnrichResult{}, err
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return AIEnrichResult{}, err
	}

	categories, _ := ListCategories(ctx, pool, siteID)
	catBySlug := map[string]int64{}
	for _, c := range categories {
		catBySlug[c.Slug] = c.ID
	}

	var enriched, failed, recategorized atomic.Int64
	jobs := make(chan aiEnrichCandidate)
	workers := 4
	if len(candidates) < workers {
		workers = len(candidates)
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				timeout := 10 * time.Second
				if c.NeedsCategoryOnly {
					// Kategori geri kontrolü eski ilanlarda yavaş/ölü sayfalara
					// denk gelebilir; kısa tutulur.
					timeout = 4 * time.Second
				}
				fetchCtx, cancel := context.WithTimeout(ctx, timeout)
				meta, err := FetchHTMLArticle(fetchCtx, c.URL)
				cancel()
				if err != nil {
					failed.Add(1)
					if c.NeedsCategoryOnly {
						// Bir kez denenip ulaşılamayan eski öğe tekrar tekrar
						// kuyruğa dönmesin; mevcut kategori korunur.
						_, _ = pool.Exec(ctx, `UPDATE ai_source_items SET category_checked_at=now() WHERE id=$1`, c.ID)
					}
					continue
				}
				catID, catChanged := enrichedCategoryID(meta.Section, c.CategoryID, catBySlug)
				section := truncateRunes(strings.TrimSpace(meta.Section), 120)

				// Daha önce zenginleştirilmiş öğede başlık/özet/tarih korunur;
				// yalnızca bölüm ve kategori düzeltilir.
				if c.NeedsCategoryOnly {
					if _, err := pool.Exec(ctx, `
						UPDATE ai_source_items
						SET section=$2, category_id=$3, category_checked_at=now()
						WHERE id=$1`, c.ID, section, nullableInt(catID)); err != nil {
						failed.Add(1)
						continue
					}
					if catChanged {
						recategorized.Add(1)
						refreshAIItemClusterCategories(ctx, pool, c.ID)
					}
					continue
				}

				title := c.Title
				if shouldUpgradeHTMLTitle(c.Title, meta.Title) {
					title = meta.Title
				}
				summary := c.Summary
				if shouldUseEnrichedSummary(c.Summary, c.Title, meta.Description) {
					summary = meta.Description
				}
				summary = truncateRunes(summary, 600)
				title = truncateRunes(title, 300)
				var published any
				if c.PublishedAt == nil && !meta.Published.IsZero() {
					published = meta.Published
				}
				if _, err := pool.Exec(ctx, `
					UPDATE ai_source_items
					SET title=$2, summary=$3, published_at=COALESCE(published_at,$4),
					    section=$5, category_id=$6, enriched_at=now(), category_checked_at=now()
					WHERE id=$1`, c.ID, title, summary, published, section, nullableInt(catID)); err != nil {
					failed.Add(1)
					continue
				}
				enriched.Add(1)
				if catChanged {
					recategorized.Add(1)
					refreshAIItemClusterCategories(ctx, pool, c.ID)
				}
				if summary != "" && normalizeNewsTitle(summary) != normalizeNewsTitle(c.Summary) {
					_, _ = pool.Exec(ctx, `
						UPDATE ai_story_clusters c
						SET summary=$2, updated_at=now()
						WHERE c.id IN (SELECT cluster_id FROM ai_story_cluster_items WHERE item_id=$1)
						  AND (c.summary='' OR c.summary=$3 OR c.summary=$4 OR length(c.summary) < 40)`,
						c.ID, summary, c.Summary, c.Title)
				}
			}
		}()
	}
	for _, c := range candidates {
		jobs <- c
	}
	close(jobs)
	wg.Wait()

	pending, _ := CountPendingAIEnrichment(ctx, pool, siteID)
	return AIEnrichResult{
		Enriched:      int(enriched.Load()),
		Failed:        int(failed.Load()),
		Pending:       pending,
		Recategorized: int(recategorized.Load()),
	}, nil
}

// enrichedCategoryID, makale bölümünü kanonik kategoriye çevirir. Bölüm
// bilinmiyorsa veya kategori değişmiyorsa mevcut kategori korunur.
func enrichedCategoryID(section string, current *int64, catBySlug map[string]int64) (*int64, bool) {
	slug := sectionCategorySlug(section)
	if slug == "" {
		return current, false
	}
	id, ok := catBySlug[slug]
	if !ok || id == 0 {
		return current, false
	}
	if current != nil && *current == id {
		return current, false
	}
	cid := id
	return &cid, true
}

// refreshAIItemClusterCategories, kategori düzeltmesinden sonra öğenin
// bağlı olduğu kümelerin baskın kategorisini yeniden hesaplar.
func refreshAIItemClusterCategories(ctx context.Context, pool *pgxpool.Pool, itemID int64) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT cluster_id FROM ai_story_cluster_items WHERE item_id=$1`, itemID)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		_ = refreshAIClusterCategory(ctx, pool, id)
	}
}

// CountPendingAIEnrichment, özet veya kategori kontrolü bekleyen HTML/sitemap
// öğelerini sayar.
func CountPendingAIEnrichment(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	if siteID <= 0 {
		siteID = 1
	}
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM ai_source_items i
		JOIN feeds f ON f.id=i.feed_id
		WHERE i.site_id=$1 AND f.kind IN ('html','sitemap')
		  AND (i.enriched_at IS NULL OR i.category_checked_at IS NULL)`, siteID).Scan(&n)
	return n, err
}

func shouldUpgradeHTMLTitle(oldTitle, newTitle string) bool {
	newTitle = strings.TrimSpace(newTitle)
	if len([]rune(newTitle)) < 20 {
		return false
	}
	return normalizeNewsTitle(oldTitle) != normalizeNewsTitle(newTitle)
}

func shouldUseEnrichedSummary(oldSummary, oldTitle, newSummary string) bool {
	newSummary = strings.TrimSpace(newSummary)
	if newSummary == "" {
		return false
	}
	oldSummary = strings.TrimSpace(oldSummary)
	if oldSummary == "" {
		return true
	}
	if normalizeNewsTitle(oldSummary) == normalizeNewsTitle(oldTitle) {
		return true
	}
	if normalizeNewsTitle(oldSummary) == normalizeNewsTitle(newSummary) {
		return false
	}
	return len([]rune(oldSummary)) < 80
}
