package scheduler

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/trends"
)

var trendHTTPClient = func() *http.Client {
	return platform.SafeFetchClient(20 * time.Second)
}

func RunTrendFetch(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 30 * time.Second
	}
	client := trendHTTPClient()
	logger.Info("trend kaynak çekici başladı", "her", every.String())
	fetchDueTrendSources(ctx, pool, logger, client)

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("trend kaynak çekici durdu")
			return
		case <-ticker.C:
			fetchDueTrendSources(ctx, pool, logger, client)
		}
	}
}

func fetchDueTrendSources(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, client *http.Client) {
	srcs, err := repo.ListActiveTrendSources(ctx, pool)
	if err != nil {
		logger.Warn("trend kaynakları okunamadı", "error", err)
		return
	}
	now := time.Now()
	for _, row := range srcs {
		if row.NextRetryAt != nil && now.Before(*row.NextRetryAt) {
			continue
		}
		interval := time.Duration(row.IntervalSeconds) * time.Second
		if interval <= 0 {
			interval = 15 * time.Minute
		}
		if row.LastFetchAt != nil && now.Sub(*row.LastFetchAt) < interval {
			continue
		}
		_, _, _ = fetchTrendSource(ctx, pool, logger, client, row)
	}
}

// FetchTrendSourceNow tek kaynağı beklemeden çeker (panel "Şimdi çek").
func FetchTrendSourceNow(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, sourceID int64) (int, int, error) {
	row, err := repo.GetTrendSource(ctx, pool, sourceID)
	if err != nil {
		return 0, 0, err
	}
	client := trendHTTPClient()
	return fetchTrendSource(ctx, pool, logger, client, row)
}

// FetchAllTrendSourcesNow tüm aktif kaynakları beklemeden çeker.
func FetchAllTrendSourcesNow(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) (int, int, error) {
	srcs, err := repo.ListActiveTrendSources(ctx, pool)
	if err != nil {
		return 0, 0, err
	}
	client := trendHTTPClient()
	added, updated := 0, 0
	var lastErr error
	for _, row := range srcs {
		a, u, ferr := fetchTrendSource(ctx, pool, logger, client, row)
		added += a
		updated += u
		if ferr != nil {
			lastErr = ferr
		}
	}
	return added, updated, lastErr
}

func fetchTrendSource(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, client *http.Client, row repo.TrendSourceRow) (int, int, error) {
	runID, _ := repo.TrendRunStart(ctx, pool, row.ID)
	provider := trends.ProviderFor(row.Kind, client)
	res, err := provider.Fetch(ctx, row.ToTrendSource())
	if err != nil {
		if runID > 0 {
			_ = repo.TrendRunFinish(ctx, pool, runID, "hata", 0, 0, 0, err.Error())
		}
		_ = repo.TrendSourceFetched(ctx, pool, row.ID, row.ETag, row.LastModified, 0, err.Error())
		logger.Warn("trend kaynağı çekilemedi", "kaynak", row.Name, "error", err)
		return 0, 0, err
	}
	if res.NotModified {
		if runID > 0 {
			_ = repo.TrendRunFinish(ctx, pool, runID, "304", 304, 0, 0, "")
		}
		_ = repo.TrendSourceFetched(ctx, pool, row.ID, res.ETag, res.LastModified, 304, "")
		return 0, 0, nil
	}

	added, updated := 0, 0
	for rank, term := range res.Terms {
		if term.Term == "" {
			continue
		}
		if !trends.HasLatin(term.Term) {
			continue
		}
		id, inserted, perr := repo.PublishTrendTerm(ctx, pool, row.ID, row.SiteID, term.Term, row.Geo, row.HL)
		if perr != nil {
			logger.Warn("trend terimi yazılamadı", "terim", term.Term, "error", perr)
			continue
		}
		if inserted {
			added++
		} else {
			updated++
		}
		_ = repo.UpdateTrendTermMeta(ctx, pool, id, term.ApproxTraffic, term.Picture, term.PictureSource)
		_ = repo.AddTrendPoint(ctx, pool, id, rank+1, term.ApproxTraffic, len(term.News))
		for _, n := range term.News {
			if n.Published.IsZero() {
				n.Published = term.PublishedAt
			}
			_ = repo.UpsertTrendNews(ctx, pool, id, n)
		}
	}
	if runID > 0 {
		_ = repo.TrendRunFinish(ctx, pool, runID, "ok", 200, added, updated, "")
	}
	_ = repo.TrendSourceFetched(ctx, pool, row.ID, res.ETag, res.LastModified, 200, "")
	logger.Info("trend kaynağı çekildi", "kaynak", row.Name, "yeni", added, "güncel", updated)
	return added, updated, nil
}

// RunTrendScore trend terimlerini site alakası, talep ve kapsamayla skorlar;
// kapsama eşleşmelerini yazar ve eşik üstü yükselen terimler için uyarı üretir.
func RunTrendScore(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 15 * time.Minute
	}
	logger.Info("trend skorlayıcı başladı", "her", every.String())
	lastRetention := time.Time{}
	run := func() {
		scoreTrends(ctx, pool, logger)
		if time.Since(lastRetention) > 24*time.Hour {
			if err := repo.TrendRetention(ctx, pool); err != nil {
				logger.Warn("trend saklama temizliği", "error", err)
			}
			lastRetention = time.Now()
		}
	}
	run()

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("trend skorlayıcı durdu")
			return
		case <-ticker.C:
			run()
		}
	}
}

func scoreTrends(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	sites, err := repo.ListSites(ctx, pool)
	if err != nil {
		logger.Warn("trend skor: site listesi", "error", err)
		return
	}
	weights := repo.TrendWeights(ctx, pool)
	threshold := repo.TrendAlertThreshold(ctx, pool)
	coverageHours := repo.TrendCoverageHours(ctx, pool)
	recentWindow := time.Duration(repo.TrendRecentHours(ctx, pool)) * time.Hour
	now := time.Now()
	total := 0
	for _, st := range sites {
		terms, err := repo.TrendTermsForScoring(ctx, pool, st.ID, 300)
		if err != nil || len(terms) == 0 {
			continue
		}
		// Geniş pencere: "bu konuda içeriğimiz var mı?" sorusunu cevaplar.
		// Taze pencere ayrımı trends.Coverage içinde yapılır.
		contents, err := repo.ListTrendMatchContents(ctx, pool, st.ID, coverageHours, 0)
		if err != nil {
			logger.Warn("trend skor: içerikler", "site", st.Domain, "error", err)
			continue
		}
		docs := make([]trends.CoverageDoc, 0, len(contents))
		for _, c := range contents {
			docs = append(docs, trends.CoverageDoc{
				ID:          c.ID,
				Text:        c.Title + " " + c.Spot + " " + c.Tags,
				PublishedAt: c.PublishedAt,
				Hits:        c.Hits,
			})
		}
		tax, _ := repo.GetTrendTaxonomy(ctx, pool, st.ID)
		taxWords := taxonomyWords(tax)
		for _, term := range terms {
			relevance := wordCoverage(trends.Tokens(term.Term), taxWords)
			cov := trends.Coverage(term.Term, docs, now, recentWindow)
			demand := repo.TrendDemand(ctx, pool, st.ID, term.TermNorm)
			if cov.HitSum > 0 {
				// okunma talebi: 1000 okunmada 1.0'a yaklaşır.
				h := logNorm(float64(cov.HitSum), 1000)
				if h > demand {
					demand = h
				}
			}
			newsCount := repo.TrendNewsCount(ctx, pool, term.ID)
			score := trends.Score(trends.ScoreInput{
				ApproxTraffic: term.ApproxTraffic,
				NewsCount:     newsCount,
				FirstSeen:     term.FirstSeenAt,
				LastSeen:      term.LastSeenAt,
				Now:           now,
				Relevance:     relevance,
				Demand:        demand,
				// Boşluk sinyali TAZE pencereye bakar: arşivde içerik olsa
				// bile bu haber döngüsünde işlenmediyse yazmaya değer.
				Coverage: cov.Recent,
				Weights:  weights,
			})
			if err := repo.ReplaceTrendMatches(ctx, pool, term.ID, cov.Matches); err != nil {
				logger.Warn("trend eşleşmeleri", "terim", term.Term, "error", err)
			}
			// coverage_count panelde/API'de gösterilen GERÇEK sayıdır:
			// arşiv dahil toplam eşleşme.
			if err := repo.UpdateTrendScore(ctx, pool, term.ID, score, cov.Total); err != nil {
				logger.Warn("trend skoru yazılamadı", "terim", term.Term, "error", err)
				continue
			}
			if score >= threshold && now.Sub(term.FirstSeenAt) <= 48*time.Hour {
				_ = repo.InsertTrendAlert(ctx, pool, term.ID, score)
			}
			total++
		}
	}
	if total > 0 {
		logger.Info("trend skorları güncellendi", "terim", total)
	}
}

// taxonomyWords kategori+etiket adlarındaki anlamlı kelimeleri tek kümede toplar.
func taxonomyWords(tax repo.TrendTaxonomy) map[string]bool {
	out := map[string]bool{}
	for _, name := range tax.Categories {
		for _, w := range trends.Tokens(name) {
			out[w] = true
		}
	}
	for _, name := range tax.Tags {
		for _, w := range trends.Tokens(name) {
			out[w] = true
		}
	}
	return out
}

// wordCoverage terim kelimelerinin taksonomide bulunma oranıdır (0..1).
func wordCoverage(termTokens []string, taxWords map[string]bool) float64 {
	if len(termTokens) == 0 {
		return 0
	}
	hit := 0
	for _, w := range termTokens {
		if taxWords[w] {
			hit++
		}
	}
	return float64(hit) / float64(len(termTokens))
}

func logNorm(v, max float64) float64 {
	if v <= 0 {
		return 0
	}
	if v > max {
		v = max
	}
	return math.Log1p(v) / math.Log1p(max)
}
