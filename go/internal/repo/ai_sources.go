package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
)

// ---- AI haber kaynak merkezi ----

type AISourceItem struct {
	ID          int64
	FeedID      *int64
	SourceName  string
	CategoryID  *int64
	Title       string
	URL         string
	Summary     string
	PublishedAt *time.Time
	FetchedAt   time.Time
}

type AIStoryCluster struct {
	ID            int64
	CategoryID    *int64
	CategoryName  string
	CategorySlug  string
	Title         string
	Summary       string
	Keywords      string
	ItemCount     int
	SourceCount   int
	TrendTerm     string
	TrendScore    int
	Score         int
	Status        string
	CreatedAt     time.Time
	Sources       string
	ContentID     *int64
	CrossCategory bool
}

// ScanAndClusterAISources tarama ve kümelemeyi tek kritik bölümde çalıştırır;
// panel ile zamanlanmış işçinin aynı anda küme üretmesini engeller.
func ScanAndClusterAISources(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, int, int, error) {
	aiSourceMu.Lock()
	defer aiSourceMu.Unlock()
	items, err := ScanAISources(ctx, pool, siteID)
	if err != nil {
		return 0, 0, 0, err
	}
	clusters, err := BuildAIStoryClusters(ctx, pool, siteID)
	if err != nil {
		return items, 0, 0, err
	}
	merged, _ := MergeCrossCategoryAIStoryClusters(ctx, pool, siteID, 30)
	return items, clusters, merged, nil
}

func chooseRSSSection(categories []string, catBySlug map[string]int64, feedCatID *int64) string {
	gundemID := catBySlug["gundem"]
	gundemRaw := ""
	firstSpecific := ""
	feedSpecific := ""
	for _, raw := range categories {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		slug := sectionCategorySlug(raw)
		if slug == "" {
			continue
		}
		id := catBySlug[slug]
		if id == 0 {
			continue
		}
		if slug == "gundem" {
			if gundemRaw == "" {
				gundemRaw = raw
			}
			continue
		}
		if firstSpecific == "" {
			firstSpecific = raw
		}
		if feedCatID != nil && id == *feedCatID {
			feedSpecific = raw
		}
	}
	if feedSpecific != "" {
		return feedSpecific
	}
	if firstSpecific != "" {
		return firstSpecific
	}
	if gundemRaw != "" && (feedCatID == nil || (gundemID != 0 && *feedCatID == gundemID)) {
		return gundemRaw
	}
	return ""
}

var aiSourceMu sync.Mutex

func ScanAISources(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	if siteID <= 0 {
		siteID = 1
	}
	feeds, err := ListFeeds(ctx, pool)
	if err != nil {
		return 0, err
	}
	var targets []Feed
	for _, f := range feeds {
		if !f.Aktif || f.SiteID != siteID || strings.TrimSpace(f.URL) == "" {
			continue
		}
		kind := strings.TrimSpace(f.Kind)
		if kind != "" && kind != "rss" && kind != "html" && kind != "sitemap" && kind != "bingnews" {
			continue
		}
		if kind == "" {
			f.Kind = "rss"
		}
		targets = append(targets, f)
	}
	if len(targets) == 0 {
		return 0, nil
	}

	// HTML kaynaklarındaki JSON-LD bölüm adlarını site kategorilerine eşlemek
	// için kategorileri bir kez yükle.
	categories, _ := ListCategories(ctx, pool, siteID)
	catBySlug := map[string]int64{}
	for _, c := range categories {
		catBySlug[c.Slug] = c.ID
	}

	type scanResult struct {
		feed  Feed
		items []RSSItem
		err   error
	}
	jobs := make(chan Feed)
	results := make(chan scanResult)
	workers := 8
	if len(targets) < workers {
		workers = len(targets)
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				var items []RSSItem
				var err error
				switch f.Kind {
				case "html":
					items, err = ScanHTMLFeed(ctx, pool, siteID, f)
				case "sitemap":
					items, err = ScanSitemapFeed(ctx, pool, siteID, f)
				case "bingnews":
					items, err = FetchBingNews(ctx, f.URL)
				default:
					items, err = FetchRSS(f.URL)
				}
				results <- scanResult{feed: f, items: items, err: err}
			}
		}()
	}
	go func() {
		for _, f := range targets {
			jobs <- f
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	inserted := 0
	for res := range results {
		f := res.feed
		if res.err != nil {
			_, _ = pool.Exec(ctx, `UPDATE feeds SET last_checked_at=now(), last_status='hata', last_error=$2, last_item_count=0 WHERE id=$1`,
				f.ID, truncateRunes(res.err.Error(), 300))
			continue
		}
		items := res.items
		if len(items) > 40 {
			items = items[:40]
		}
		for _, it := range items {
			it.Link = strings.TrimSpace(it.Link)
			it.Title = strings.TrimSpace(it.Title)
			if it.Link == "" || it.Title == "" {
				continue
			}
			h := sha1.Sum([]byte(it.Link))
			hash := hex.EncodeToString(h[:])
			var pub any
			if !it.PubDate.IsZero() {
				pub = it.PubDate
			}
			var enriched, catChecked any
			if it.Enriched {
				now := time.Now()
				enriched = now
				catChecked = now
			}
			section := it.Section
			if section == "" && len(it.Categories) > 0 {
				section = chooseRSSSection(it.Categories, catBySlug, f.CategoryID)
			}
			catID := f.CategoryID
			if slug := sectionCategorySlug(section); slug != "" {
				if id := catBySlug[slug]; id != 0 {
					cid := id
					catID = &cid
				}
			}
			tag, err := pool.Exec(ctx, `
				INSERT INTO ai_source_items
					(site_id, feed_id, source_name, category_id, title, url, url_hash, summary,
					 published_at, section, enriched_at, category_checked_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
				ON CONFLICT (url_hash) DO NOTHING`,
				siteID, f.ID, f.Name, nullableInt(catID), it.Title, it.Link, hash,
				truncateRunes(it.Description, 600), pub, truncateRunes(section, 120), enriched, catChecked)
			if err == nil && tag.RowsAffected() > 0 {
				inserted++
			}
		}
		_, _ = pool.Exec(ctx, `UPDATE feeds SET last_checked_at=now(), last_status='ok', last_error='', last_item_count=$2 WHERE id=$1`,
			f.ID, len(items))
	}
	return inserted, nil
}

// BuildAIStoryClusters henüz kümelenmemiş öğeleri kategori içinde benzerlikle
// gruplar; Google Trends terimleriyle eşleştirir.
func BuildAIStoryClusters(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	if siteID <= 0 {
		siteID = 1
	}
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.feed_id, i.source_name, i.category_id, i.title, i.url, i.summary,
		       i.published_at, i.fetched_at
		FROM ai_source_items i
		WHERE i.site_id=$1
		  AND NOT EXISTS (SELECT 1 FROM ai_story_cluster_items ci WHERE ci.item_id=i.id)
		  AND i.fetched_at > now() - interval '10 days'
		ORDER BY i.fetched_at DESC, i.id DESC LIMIT 600`, siteID)
	if err != nil {
		return 0, err
	}
	var items []AISourceItem
	for rows.Next() {
		var it AISourceItem
		if err := rows.Scan(&it.ID, &it.FeedID, &it.SourceName, &it.CategoryID, &it.Title, &it.URL,
			&it.Summary, &it.PublishedAt, &it.FetchedAt); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if len(items) == 0 {
		return 0, nil
	}

	trends, _ := TrendTermsForScoring(ctx, pool, siteID, 200)
	type trendTok struct {
		term  string
		toks  map[string]bool
		score int
	}
	var tt []trendTok
	for _, t := range trends {
		tk := aiTitleTokens(t.Term)
		if len(tk) > 0 {
			tt = append(tt, trendTok{term: t.Term, toks: tk, score: t.Score})
		}
	}

	// Son yedi günün taslağa dönüşmemiş kümeleri, yeni gelen takip
	// haberlerinin ayrı küme açmasını engellemek için aday olarak yüklenir.
	existing, err := loadAIExistingClusterCandidates(ctx, pool, siteID)
	if err != nil {
		existing = nil
	}

	existingAffected := map[int64]*aiExistingCluster{}
	var groups []*aiNewGroup
	for _, it := range items {
		tk := aiTitleTokens(it.Title)

		var bestExisting *aiExistingCluster
		bestExistingScore := 0.0
		for _, c := range existing {
			sc, ok := aiMatchTitle(tk, c.Toks, sameAICategory(it.CategoryID, c.CategoryID), true)
			if ok && sc > bestExistingScore {
				bestExistingScore, bestExisting = sc, c
			}
		}
		if bestExisting != nil {
			bestExisting.Added = append(bestExisting.Added, it)
			existingAffected[bestExisting.ID] = bestExisting
			continue
		}

		var bestGroup *aiNewGroup
		bestGroupScore := 0.0
		for _, g := range groups {
			sc, ok := aiMatchTitle(tk, g.toks, g.hasCategory(it.CategoryID), false)
			if ok && sc > bestGroupScore {
				bestGroupScore, bestGroup = sc, g
			}
		}
		if bestGroup != nil {
			bestGroup.add(it, tk)
			continue
		}
		groups = append(groups, newAINewGroup(it, tk))
	}

	for _, c := range existingAffected {
		for _, it := range c.Added {
			_, _ = pool.Exec(ctx, `INSERT INTO ai_story_cluster_items (cluster_id,item_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, c.ID, it.ID)
		}
		_ = refreshAICluster(ctx, pool, c.ID)
	}

	created := 0
	for _, g := range groups {
		if len(g.items) == 0 {
			continue
		}
		title := g.items[0].Title
		summary := ""
		sourceSet := map[string]bool{}
		for _, it := range g.items {
			sourceSet[it.SourceName] = true
			if summary == "" && it.Summary != "" {
				summary = it.Summary
			}
		}
		bestTerm, bestTrendScore := "", 0
		for _, t := range tt {
			sc := int(overlapScore(g.toks, t.toks) * 100)
			if sc > bestTrendScore {
				bestTrendScore, bestTerm = sc, t.term
			}
		}
		if bestTrendScore < 20 {
			bestTerm, bestTrendScore = "", 0
		}
		keywords := strings.Join(topTokens(g.toks, 8), ", ")
		score := len(sourceSet)*20 + len(g.items)*5 + bestTrendScore
		var id int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO ai_story_clusters
				(site_id, category_id, title, summary, keywords, item_count, source_count,
				 trend_term, trend_score, score, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'yeni') RETURNING id`,
			siteID, nullableInt(g.dominantCategory()), truncateRunes(title, 300), truncateRunes(summary, 600),
			keywords, len(g.items), len(sourceSet), bestTerm, bestTrendScore, score).Scan(&id); err != nil {
			continue
		}
		for _, it := range g.items {
			_, _ = pool.Exec(ctx, `INSERT INTO ai_story_cluster_items (cluster_id,item_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, it.ID)
		}
		created++
	}
	return created, nil
}

type aiNewGroup struct {
	items          []AISourceItem
	toks           map[string]bool
	categoryCounts map[int64]int
	categoryOrder  []int64
}

func newAINewGroup(it AISourceItem, tk map[string]bool) *aiNewGroup {
	g := &aiNewGroup{toks: map[string]bool{}, categoryCounts: map[int64]int{}}
	g.add(it, tk)
	return g
}

func (g *aiNewGroup) add(it AISourceItem, tk map[string]bool) {
	key := aiCategoryKey(it.CategoryID)
	if _, ok := g.categoryCounts[key]; !ok {
		g.categoryOrder = append(g.categoryOrder, key)
	}
	g.categoryCounts[key]++
	for k := range tk {
		g.toks[k] = true
	}
	g.items = append(g.items, it)
}

func (g *aiNewGroup) hasCategory(cat *int64) bool {
	_, ok := g.categoryCounts[aiCategoryKey(cat)]
	return ok
}

func (g *aiNewGroup) dominantCategory() *int64 {
	if len(g.categoryOrder) == 0 {
		return nil
	}
	bestKey := g.categoryOrder[0]
	bestCount := g.categoryCounts[bestKey]
	for _, key := range g.categoryOrder[1:] {
		if g.categoryCounts[key] > bestCount {
			bestKey, bestCount = key, g.categoryCounts[key]
		}
	}
	return aiCategoryPtr(bestKey)
}

func ListAIStoryClusters(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]AIStoryCluster, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.category_id, COALESCE(cat.name,''), COALESCE(cat.slug,''), c.title, c.summary,
		       c.keywords, c.item_count, c.source_count, c.trend_term, c.trend_score, c.score, c.status,
		       c.created_at,
		       COALESCE((SELECT string_agg(DISTINCT i.source_name, ', ') FROM ai_story_cluster_items ci
		                 JOIN ai_source_items i ON i.id=ci.item_id WHERE ci.cluster_id=c.id), ''),
		       c.content_id,
		       EXISTS (SELECT 1 FROM ai_story_cluster_items ci2
		               JOIN ai_source_items i2 ON i2.id=ci2.item_id
		               WHERE ci2.cluster_id=c.id
		               GROUP BY ci2.cluster_id
		               HAVING count(DISTINCT COALESCE(i2.category_id,0)) > 1)
		FROM ai_story_clusters c
		LEFT JOIN categories cat ON cat.id=c.category_id
		WHERE c.site_id=$1 AND (c.item_count > 1 OR c.trend_score >= 40)
		ORDER BY c.score DESC, c.created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIStoryCluster
	for rows.Next() {
		var c AIStoryCluster
		if err := rows.Scan(&c.ID, &c.CategoryID, &c.CategoryName, &c.CategorySlug, &c.Title, &c.Summary,
			&c.Keywords, &c.ItemCount, &c.SourceCount, &c.TrendTerm, &c.TrendScore, &c.Score,
			&c.Status, &c.CreatedAt, &c.Sources, &c.ContentID, &c.CrossCategory); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func ListAISourceItems(ctx context.Context, pool *pgxpool.Pool, clusterID int64) ([]AISourceItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.feed_id, i.source_name, i.category_id, i.title, i.url, i.summary, i.published_at, i.fetched_at
		FROM ai_story_cluster_items ci JOIN ai_source_items i ON i.id=ci.item_id
		WHERE ci.cluster_id=$1 ORDER BY i.published_at DESC NULLS LAST`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AISourceItem
	for rows.Next() {
		var it AISourceItem
		if err := rows.Scan(&it.ID, &it.FeedID, &it.SourceName, &it.CategoryID, &it.Title, &it.URL,
			&it.Summary, &it.PublishedAt, &it.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CreateDraftFromCluster kümedeki kaynaklardan düzenlenebilir bir taslak üretir.
func CreateDraftFromCluster(ctx context.Context, pool *pgxpool.Pool, siteID, clusterID, authorID int64) (int64, error) {
	var c AIStoryCluster
	err := pool.QueryRow(ctx, `
		SELECT id, category_id, title, summary, item_count, source_count, trend_term, content_id
		FROM ai_story_clusters WHERE id=$1 AND site_id=$2`, clusterID, siteID).
		Scan(&c.ID, &c.CategoryID, &c.Title, &c.Summary, &c.ItemCount, &c.SourceCount, &c.TrendTerm, &c.ContentID)
	if err != nil {
		return 0, err
	}
	if c.ContentID != nil && *c.ContentID > 0 {
		return *c.ContentID, nil
	}
	items, err := ListAISourceItems(ctx, pool, clusterID)
	if err != nil {
		return 0, err
	}
	var blocks []map[string]any
	if ozet := strings.TrimSpace(c.Summary); ozet != "" {
		for _, p := range paragraflaraBol(ozet) {
			if p = strings.TrimSpace(p); p != "" {
				blocks = append(blocks, map[string]any{"type": "p", "text": p})
			}
		}
	}
	detail, _ := json.Marshal(blocks)
	slug := platform.Slugify(c.Title)
	if slug == "" {
		slug = "ai-haber"
	}
	slug = fmt.Sprintf("%s-%d", slug, clusterID)
	var sourceURL, sourceName *string
	if len(items) > 0 {
		sourceURL = &items[0].URL
		sourceName = &items[0].SourceName
	}
	content := &Content{
		SiteID: siteID, TypeKey: "haber", Status: "taslak", Title: c.Title,
		Spot: truncateRunes(c.Summary, 240), Detail: detail, Slug: &slug,
		CategoryID: c.CategoryID, AuthorID: authorID,
		SourceURL: sourceURL, SourceName: sourceName, CommentStatus: "acik", Locale: "tr",
	}
	id, err := CreateContent(ctx, pool, content)
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		// Kaynak zaten RSS taramasından geldi; tekrar dış istek atmadan bağla.
		_, _ = pool.Exec(ctx, `INSERT INTO content_sources (content_id, url, body_snippet, accessible, checked_at)
			VALUES ($1,$2,$3,TRUE,now())`, id, it.URL, truncateRunes(it.Summary, 500))
	}
	_, _ = pool.Exec(ctx, `UPDATE ai_story_clusters SET status='taslak', content_id=$2, updated_at=now() WHERE id=$1`, clusterID, id)
	return id, nil
}

func aiTitleTokens(s string) map[string]bool {
	s = strings.ToLower(strings.NewReplacer(
		"ı", "i", "İ", "i", "ş", "s", "ğ", "g", "ü", "u", "ö", "o", "ç", "c").Replace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	stop := map[string]bool{
		"bir": true, "ve": true, "ile": true, "icin": true, "bu": true, "da": true, "de": true,
		"mi": true, "ne": true, "olan": true, "sonra": true, "daha": true, "en": true, "yeni": true,
		"turkiye": true, "haber": true, "haberleri": true, "son": true, "dakika": true,
		"aciklama": true, "aciklamasi": true, "aciklamada": true, "hakkinda": true,
		"nedeniyle": true, "sonrasi": true, "once": true, "yasanan": true, "oldu": true,
		"oldugu": true, "olacak": true, "gore": true, "ilgili": true, "bugun": true,
		"yarin": true, "iddia": true, "iddiasi": true, "mac": true, "maci": true,
		"macinda": true, "macin": true, "karsilasma": true, "karsilasmasinda": true,
		"hazirlik": true, "hazirliklari": true, "hazirliklarini": true, "surdurdu": true,
		"surduruyor": true, "cumhurbaskani": true, "bakan": true, "bakanligi": true,
		"parti": true, "partisi": true, "lideri": true, "gundem": true, "dunya": true,
		"ekonomi": true, "spor": true, "teknoloji": true, "yasam": true,
	}
	out := map[string]bool{}
	for _, w := range strings.Fields(b.String()) {
		if len([]rune(w)) < 3 || stop[w] {
			continue
		}
		out[w] = true
	}
	return out
}

func overlapScore(a, b map[string]bool) float64 {
	score, _, _ := aiOverlapDetail(a, b)
	return score
}

func topTokens(t map[string]bool, n int) []string {
	out := make([]string, 0, len(t))
	for k := range t {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) > n {
		out = out[:n]
	}
	return out
}
