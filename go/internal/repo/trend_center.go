package repo

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
	"theblock/site/internal/trends"
)

// ---- Trend Merkezi: kaynaklar ----

// TrendSourceRow trend_sources satırıdır.
type TrendSourceRow struct {
	ID                int64
	SiteID            int64
	Name              string
	Kind              string
	URL               string
	Geo               string
	HL                string
	IntervalSeconds   int
	Aktif             bool
	SortOrder         int
	ETag              string
	LastModified      string
	LastFetchAt       *time.Time
	LastStatus        *int
	LastError         string
	ConsecutiveErrors int
	NextRetryAt       *time.Time
}

// ToTrendSource scheduler/adaptör katmanına dönüştürür.
func (s TrendSourceRow) ToTrendSource() trends.Source {
	return trends.Source{
		ID: s.ID, SiteID: s.SiteID, Name: s.Name, Kind: s.Kind, URL: s.URL,
		Geo: s.Geo, HL: s.HL, IntervalSeconds: s.IntervalSeconds, Aktif: s.Aktif,
		ETag: s.ETag, LastModified: s.LastModified,
	}
}

// LastFetchLabel panelde gösterilecek son çekim zamanı.
func (s TrendSourceRow) LastFetchLabel() string {
	if s.LastFetchAt == nil {
		return "—"
	}
	return s.LastFetchAt.Local().Format("02.01 15:04")
}

// StatusLabel kaynak sağlığını tek kelimeyle özetler.
func (s TrendSourceRow) StatusLabel() string {
	if !s.Aktif {
		return "pasif"
	}
	if s.LastError != "" {
		return "hata"
	}
	if s.LastStatus != nil && *s.LastStatus == 304 {
		return "değişmedi"
	}
	if s.LastFetchAt == nil {
		return "bekliyor"
	}
	return "sağlıklı"
}

// KindLabel adaptör türünü Türkçeleştirir.
func (s TrendSourceRow) KindLabel() string {
	switch s.Kind {
	case "google_trends":
		return "Google Trends"
	case "google_news":
		return "Google News"
	case "rss":
		return "RSS"
	case "dataforseo":
		return "DataForSEO (kapalı)"
	default:
		return s.Kind
	}
}

const trendSourceCols = `id, site_id, name, kind, url, geo, hl, interval_seconds, aktif,
	sort_order, etag, last_modified, last_fetch_at, last_status, last_error,
	consecutive_errors, next_retry_at`

func scanTrendSource(row interface {
	Scan(dest ...any) error
}) (TrendSourceRow, error) {
	var s TrendSourceRow
	err := row.Scan(&s.ID, &s.SiteID, &s.Name, &s.Kind, &s.URL, &s.Geo, &s.HL,
		&s.IntervalSeconds, &s.Aktif, &s.SortOrder, &s.ETag, &s.LastModified,
		&s.LastFetchAt, &s.LastStatus, &s.LastError, &s.ConsecutiveErrors, &s.NextRetryAt)
	return s, err
}

// ListTrendSources site kaynaklarını döndürür.
func ListTrendSources(ctx context.Context, pool *pgxpool.Pool, siteID int64) ([]TrendSourceRow, error) {
	rows, err := pool.Query(ctx, `SELECT `+trendSourceCols+` FROM trend_sources
		WHERE site_id = $1 ORDER BY sort_order, name`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendSourceRow
	for rows.Next() {
		s, err := scanTrendSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListActiveTrendSources tüm sitelerin aktif kaynaklarını döndürür (scheduler).
func ListActiveTrendSources(ctx context.Context, pool *pgxpool.Pool) ([]TrendSourceRow, error) {
	rows, err := pool.Query(ctx, `SELECT `+trendSourceCols+` FROM trend_sources
		WHERE aktif ORDER BY site_id, sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendSourceRow
	for rows.Next() {
		s, err := scanTrendSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetTrendSource tek kaynak.
func GetTrendSource(ctx context.Context, pool *pgxpool.Pool, id int64) (TrendSourceRow, error) {
	return scanTrendSource(pool.QueryRow(ctx, `SELECT `+trendSourceCols+` FROM trend_sources WHERE id = $1`, id))
}

// CreateTrendSource yeni kaynak ekler.
func CreateTrendSource(ctx context.Context, pool *pgxpool.Pool, s TrendSourceRow) (int64, error) {
	if s.IntervalSeconds <= 0 {
		s.IntervalSeconds = 900
	}
	if s.Geo == "" {
		s.Geo = "TR"
	}
	if s.HL == "" {
		s.HL = "tr-TR"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO trend_sources
			(site_id, name, kind, url, geo, hl, interval_seconds, aktif, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		s.SiteID, s.Name, s.Kind, s.URL, s.Geo, s.HL, s.IntervalSeconds, s.Aktif, s.SortOrder).Scan(&id)
	return id, err
}

// SetTrendSourceAktif kaynağı açar/kapatır.
func SetTrendSourceAktif(ctx context.Context, pool *pgxpool.Pool, id int64, aktif bool) error {
	_, err := pool.Exec(ctx, `UPDATE trend_sources SET aktif=$1, updated_at=now() WHERE id=$2`, aktif, id)
	return err
}

// DeleteTrendSource kaynağı siler.
func DeleteTrendSource(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM trend_sources WHERE id=$1`, id)
	return err
}

// TrendRunStart çekim koşusu başlatır.
func TrendRunStart(ctx context.Context, pool *pgxpool.Pool, sourceID int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO trend_fetch_runs (source_id, status) VALUES ($1,'calisiyor') RETURNING id`, sourceID).Scan(&id)
	return id, err
}

// TrendRunFinish çekim koşusunu kapatır.
func TrendRunFinish(ctx context.Context, pool *pgxpool.Pool, runID int64, status string, httpCode int, added, updated int, errText string) error {
	var code any
	if httpCode > 0 {
		code = httpCode
	}
	_, err := pool.Exec(ctx, `UPDATE trend_fetch_runs SET finished_at=now(), status=$1,
		http_code=$2, added=$3, updated=$4, error=$5 WHERE id=$6`,
		status, code, added, updated, errText, runID)
	return err
}

// TrendSourceFetched kaynak durumunu günceller. Başarıda backoff sıfırlanır;
// hatada ardışık hata sayısına göre sonraki deneme zamanı ileri atılır.
func TrendSourceFetched(ctx context.Context, pool *pgxpool.Pool, id int64, etag, modified string, status int, fetchErr string) error {
	if fetchErr == "" {
		_, err := pool.Exec(ctx, `UPDATE trend_sources SET etag=$1, last_modified=$2,
			last_fetch_at=now(), last_status=$3, last_error='', consecutive_errors=0,
			next_retry_at=NULL, updated_at=now() WHERE id=$4`, etag, modified, status, id)
		return err
	}
	// 5 dk → 15 → 60 → 360 dk üstel geri çekilme.
	var n int
	_ = pool.QueryRow(ctx, `SELECT consecutive_errors FROM trend_sources WHERE id=$1`, id).Scan(&n)
	minutes := 5 * (1 << minInt(n, 3))
	if n >= 3 {
		minutes = 360
	}
	_, err := pool.Exec(ctx, `UPDATE trend_sources SET last_fetch_at=now(), last_status=$1,
		last_error=$2, consecutive_errors=consecutive_errors+1,
		next_retry_at=now()+make_interval(mins => $3), updated_at=now() WHERE id=$4`,
		status, fetchErr, minutes, id)
	return err
}

// ---- Trend terimleri ----

// TrendTermRow trend_terms satırı + kaynak adıdır.
type TrendTermRow struct {
	ID            int64
	SourceID      int64
	SiteID        int64
	SourceName    string
	Term          string
	TermNorm      string
	Kind          string
	Geo           string
	HL            string
	ApproxTraffic int
	Picture       string
	PictureSource string
	FirstSeenAt   time.Time
	LastSeenAt    time.Time
	Hidden        bool
	Score         int
	CoverageCount int
}

const trendTermCols = `t.id, COALESCE(t.source_id,0), t.site_id, COALESCE(s.name,''), t.term, t.term_norm,
	t.kind, t.geo, t.hl, t.approx_traffic, t.picture, t.picture_source,
	t.first_seen_at, t.last_seen_at, t.hidden, t.score, t.coverage_count`

func scanTrendTerm(row interface {
	Scan(dest ...any) error
}) (TrendTermRow, error) {
	var t TrendTermRow
	err := row.Scan(&t.ID, &t.SourceID, &t.SiteID, &t.SourceName, &t.Term, &t.TermNorm, &t.Kind,
		&t.Geo, &t.HL, &t.ApproxTraffic, &t.Picture, &t.PictureSource,
		&t.FirstSeenAt, &t.LastSeenAt, &t.Hidden, &t.Score, &t.CoverageCount)
	return t, err
}

// PublishTrendTerm bir terimi ekler/günceller. Dönen inserted yeni mi?
func PublishTrendTerm(ctx context.Context, pool *pgxpool.Pool, sourceID, siteID int64, term string, geo, hl string) (id int64, inserted bool, err error) {
	norm := trends.NormalizeTerm(term)
	fp := trends.Fingerprint(norm, geo, hl)
	err = pool.QueryRow(ctx, `
		INSERT INTO trend_terms (source_id, site_id, term, term_norm, fingerprint, geo, hl, kind)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'top')
		ON CONFLICT (site_id, fingerprint) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			term = EXCLUDED.term,
			last_seen_at = now(),
			updated_at = now()
		RETURNING id, (xmax = 0)`,
		sourceID, siteID, term, norm, fp, geo, hl).Scan(&id, &inserted)
	return id, inserted, err
}

// UpdateTrendTermMeta trafik/görsel alanlarını günceller.
func UpdateTrendTermMeta(ctx context.Context, pool *pgxpool.Pool, id int64, traffic int, picture, pictureSource string) error {
	_, err := pool.Exec(ctx, `UPDATE trend_terms SET approx_traffic=$1, picture=$2,
		picture_source=$3, updated_at=now() WHERE id=$4`, traffic, picture, pictureSource, id)
	return err
}

// AddTrendPoint zaman serisi noktası ekler.
func AddTrendPoint(ctx context.Context, pool *pgxpool.Pool, termID int64, rank, traffic, newsCount int) error {
	_, err := pool.Exec(ctx, `INSERT INTO trend_points (term_id, rank, approx_traffic, news_count)
		VALUES ($1,$2,$3,$4)`, termID, rank, traffic, newsCount)
	return err
}

// TrendSparkPoint sparkline için kovalanmış ortalama sıralamadır.
// Rank küçüldükçe terim listede yükselir.
type TrendSparkPoint struct {
	CapturedAt time.Time
	Rank       float64
}

func LoadTrendSparklines(ctx context.Context, pool *pgxpool.Pool, termIDs []int64, since time.Time) (map[int64][]TrendSparkPoint, error) {
	out := make(map[int64][]TrendSparkPoint, len(termIDs))
	if len(termIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT term_id,
		       date_bin('6 hours', captured_at, timestamptz '2000-01-01') AS bucket,
		       avg(rank)::float8
		FROM trend_points
		WHERE term_id = ANY($1) AND captured_at >= $2 AND rank > 0
		GROUP BY term_id, bucket
		ORDER BY term_id, bucket`, termIDs, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var p TrendSparkPoint
		if err := rows.Scan(&id, &p.CapturedAt, &p.Rank); err != nil {
			return nil, err
		}
		out[id] = append(out[id], p)
	}
	return out, rows.Err()
}

// UpsertTrendNews terimin haber öğelerini yazar.
func UpsertTrendNews(ctx context.Context, pool *pgxpool.Pool, termID int64, n trends.News) error {
	key := strings.TrimSpace(n.URL)
	if key == "" {
		key = n.Title
	}
	if key == "" {
		return nil
	}
	h := sha1.Sum([]byte(key))
	var pub any
	if !n.Published.IsZero() {
		pub = n.Published
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO trend_news (term_id, title, url, url_hash, source, picture, snippet, published_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (term_id, url_hash) DO UPDATE SET
			title = EXCLUDED.title, snippet = EXCLUDED.snippet,
			picture = EXCLUDED.picture, source = EXCLUDED.source`,
		termID, n.Title, n.URL, hex.EncodeToString(h[:]), n.Source, n.Picture, n.Snippet, pub)
	return err
}

// TrendFilter panel/API filtreleridir.
type TrendFilter struct {
	SiteID        int64
	Q             string
	OnlyGap       bool
	OnlyRising    bool
	MinScore      int
	Limit         int
	IncludeHidden bool
}

// ListTrendTerms filtrelenmiş terimleri döndürür.
func ListTrendTerms(ctx context.Context, pool *pgxpool.Pool, f TrendFilter) ([]TrendTermRow, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	where := []string{"t.site_id = $1"}
	args := []any{f.SiteID}
	if !f.IncludeHidden {
		where = append(where, "t.hidden = FALSE")
	}
	if q := trends.NormalizeTerm(f.Q); q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, "t.term_norm LIKE $"+itoa(len(args)))
	}
	if f.OnlyGap {
		where = append(where, "t.coverage_count = 0")
	}
	if f.OnlyRising {
		where = append(where, "t.first_seen_at >= now() - interval '24 hours'")
	}
	if f.MinScore > 0 {
		args = append(args, f.MinScore)
		where = append(where, "t.score >= $"+itoa(len(args)))
	}
	args = append(args, f.Limit)
	q := `SELECT ` + trendTermCols + ` FROM trend_terms t
		LEFT JOIN trend_sources s ON s.id = t.source_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY t.score DESC, t.approx_traffic DESC, t.last_seen_at DESC
		LIMIT $` + itoa(len(args))
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendTermRow
	for rows.Next() {
		t, err := scanTrendTerm(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTrendTerm tek terim.
func GetTrendTerm(ctx context.Context, pool *pgxpool.Pool, id int64) (TrendTermRow, error) {
	return scanTrendTerm(pool.QueryRow(ctx, `SELECT `+trendTermCols+` FROM trend_terms t
		LEFT JOIN trend_sources s ON s.id = t.source_id WHERE t.id=$1`, id))
}

// SetTrendTermHidden terimi gizler/gösterir.
func SetTrendTermHidden(ctx context.Context, pool *pgxpool.Pool, id int64, hidden bool) error {
	_, err := pool.Exec(ctx, `UPDATE trend_terms SET hidden=$1, updated_at=now() WHERE id=$2`, hidden, id)
	return err
}

// TrendStats panel sayaçlarıdır.
type TrendStats struct {
	Total   int
	Covered int
	Weak    int
	Gap     int
	Rising  int
	Alerts  int
}

// GetTrendStats sayaçları tek sorguda üretir.
func GetTrendStats(ctx context.Context, pool *pgxpool.Pool, siteID int64) (TrendStats, error) {
	var st TrendStats
	err := pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE TRUE),
		count(*) FILTER (WHERE coverage_count >= 2),
		count(*) FILTER (WHERE coverage_count = 1),
		count(*) FILTER (WHERE coverage_count = 0),
		count(*) FILTER (WHERE first_seen_at >= now() - interval '24 hours')
		FROM trend_terms WHERE site_id=$1 AND hidden=FALSE`, siteID).
		Scan(&st.Total, &st.Covered, &st.Weak, &st.Gap, &st.Rising)
	if err != nil {
		return st, err
	}
	st.Alerts, _ = CountUnseenTrendAlerts(ctx, pool, siteID)
	return st, nil
}

// ---- Kapsama / skor ----

// TrendMatchContent eşleştirmede kullanılan yayın özetidir.
type TrendMatchContent struct {
	ID         int64
	Title      string
	Spot       string
	CategoryID *int64
	Hits       int64
	Tags       string
	// PublishedAt "taze kapsama" (haber döngüsü) ayrımı için gereklidir;
	// bkz. trends.Coverage.
	PublishedAt time.Time
}

func ListTrendMatchContents(ctx context.Context, pool *pgxpool.Pool, siteID int64, hours, limit int) ([]TrendMatchContent, error) {
	if hours <= 0 {
		hours = defaultTrendCoverageHours
	}
	if limit <= 0 || limit > maxTrendMatchContents {
		limit = maxTrendMatchContents
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, COALESCE(c.spot,''), c.category_id, COALESCE(ch.total,0),
		       COALESCE(string_agg(t.name, ' '), ''), c.published_at
		FROM contents c
		LEFT JOIN counters_hit ch ON ch.content_id = c.id
		LEFT JOIN content_tags ct ON ct.content_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.site_id = $1 AND c.status = 'yayinda'
		  AND c.published_at >= now() - ($2::int || ' hours')::interval
		GROUP BY c.id, ch.total
		ORDER BY c.published_at DESC NULLS LAST
		LIMIT $3`, siteID, hours, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendMatchContent
	for rows.Next() {
		var c TrendMatchContent
		var pub *time.Time
		if err := rows.Scan(&c.ID, &c.Title, &c.Spot, &c.CategoryID, &c.Hits, &c.Tags, &pub); err != nil {
			return nil, err
		}
		if pub != nil {
			c.PublishedAt = *pub
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TrendTaxonomy site kategori ve etiket adlarıdır (alaka skoru için).
type TrendTaxonomy struct {
	Categories map[int64]string
	Tags       []string
}

// GetTrendTaxonomy taksonomiyi yükler.
func GetTrendTaxonomy(ctx context.Context, pool *pgxpool.Pool, siteID int64) (TrendTaxonomy, error) {
	out := TrendTaxonomy{Categories: map[int64]string{}}
	cats, err := ListCategories(ctx, pool, siteID)
	if err == nil {
		for _, c := range cats {
			out.Categories[c.ID] = c.Name
		}
	}
	rows, err := pool.Query(ctx, `SELECT name FROM tags ORDER BY name LIMIT 2000`)
	if err != nil {
		return out, nil
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			out.Tags = append(out.Tags, n)
		}
	}
	return out, rows.Err()
}

// ReplaceTrendMatches terimin eşleşmelerini yeniden yazar.
func ReplaceTrendMatches(ctx context.Context, pool *pgxpool.Pool, termID int64, matches map[int64]int) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM trend_matches WHERE term_id=$1`, termID); err != nil {
		return err
	}
	for contentID, score := range matches {
		if _, err := tx.Exec(ctx, `INSERT INTO trend_matches (term_id, content_id, match_kind, score)
			VALUES ($1,$2,'baslik',$3) ON CONFLICT (term_id, content_id) DO UPDATE SET score=EXCLUDED.score`,
			termID, contentID, score); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// UpdateTrendScore skoru ve kapsama sayısını yazar.
func UpdateTrendScore(ctx context.Context, pool *pgxpool.Pool, termID int64, score, coverage int) error {
	_, err := pool.Exec(ctx, `UPDATE trend_terms SET score=$1, coverage_count=$2,
		last_scored_at=now(), updated_at=now() WHERE id=$3`, score, coverage, termID)
	return err
}

// TrendDemand site içi arama talebini 0..1 aralığına normalize eder.
func TrendDemand(ctx context.Context, pool *pgxpool.Pool, siteID int64, termNorm string) float64 {
	if strings.TrimSpace(termNorm) == "" {
		return 0
	}
	var n int64
	_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(count),0) FROM search_queries
		WHERE site_id=$1 AND query_norm LIKE '%'||$2||'%'
		  AND last_seen_at >= now() - interval '14 days'`, siteID, termNorm).Scan(&n)
	if n <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(float64(n))/math.Log1p(100))
}

// TrendTermsForScoring skorlanacak terimleri döndürür.
func TrendTermsForScoring(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]TrendTermRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 300
	}
	rows, err := pool.Query(ctx, `SELECT `+trendTermCols+` FROM trend_terms t
		LEFT JOIN trend_sources s ON s.id=t.source_id
		WHERE t.site_id=$1 AND t.hidden=FALSE
		ORDER BY t.last_seen_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendTermRow
	for rows.Next() {
		t, err := scanTrendTerm(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TrendNewsCount terimin haber sayısı.
func TrendNewsCount(ctx context.Context, pool *pgxpool.Pool, termID int64) int {
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM trend_news WHERE term_id=$1`, termID).Scan(&n)
	return n
}

// TrendNewsForTerm terimin son haberlerini döndürür (AI brifingi için).
func TrendNewsForTerm(ctx context.Context, pool *pgxpool.Pool, termID int64, limit int) ([]trends.News, error) {
	if limit <= 0 {
		limit = 8
	}
	rows, err := pool.Query(ctx, `SELECT title, url, source, picture, snippet, published_at
		FROM trend_news WHERE term_id=$1 ORDER BY created_at DESC LIMIT $2`, termID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trends.News
	for rows.Next() {
		var n trends.News
		var pub *time.Time
		if err := rows.Scan(&n.Title, &n.URL, &n.Source, &n.Picture, &n.Snippet, &pub); err != nil {
			return nil, err
		}
		if pub != nil {
			n.Published = *pub
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ---- Uyarılar ----

// InsertTrendAlert günlük tek uyarı üretir (eşik üstü yeni/yükselen trend).
func InsertTrendAlert(ctx context.Context, pool *pgxpool.Pool, termID int64, score int) error {
	_, err := pool.Exec(ctx, `INSERT INTO trend_alerts (term_id, score)
		VALUES ($1,$2) ON CONFLICT (term_id, alert_day) DO NOTHING`, termID, score)
	return err
}

// CountUnseenTrendAlerts görülmemiş uyarı sayısı.
func CountUnseenTrendAlerts(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM trend_alerts a
		JOIN trend_terms t ON t.id=a.term_id
		WHERE t.site_id=$1 AND a.seen_at IS NULL`, siteID).Scan(&n)
	return n, err
}

// TrendAlertRow panel uyarısıdır.
type TrendAlertRow struct {
	ID      int64
	TermID  int64
	Term    string
	Score   int
	Created time.Time
}

// ListTrendAlerts son uyarıları döndürür.
func ListTrendAlerts(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]TrendAlertRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `SELECT a.id, a.term_id, t.term, a.score, a.created_at
		FROM trend_alerts a JOIN trend_terms t ON t.id=a.term_id
		WHERE t.site_id=$1 ORDER BY a.created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendAlertRow
	for rows.Next() {
		var a TrendAlertRow
		if err := rows.Scan(&a.ID, &a.TermID, &a.Term, &a.Score, &a.Created); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// MarkTrendAlertsSeen tüm uyarıları okundu işaretler.
func MarkTrendAlertsSeen(ctx context.Context, pool *pgxpool.Pool, siteID int64) error {
	_, err := pool.Exec(ctx, `UPDATE trend_alerts a SET seen_at=now()
		FROM trend_terms t WHERE t.id=a.term_id AND t.site_id=$1 AND a.seen_at IS NULL`, siteID)
	return err
}

// ---- AI brifing ----

// SaveTrendBrief brifing metnini saklar.
func SaveTrendBrief(ctx context.Context, pool *pgxpool.Pool, termID int64, text, provider string, userID int64) (int64, error) {
	var uid any
	if userID > 0 {
		uid = userID
	}
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO trend_briefs (term_id, text, provider, user_id)
		VALUES ($1,$2,$3,$4) RETURNING id`, termID, text, provider, uid).Scan(&id)
	return id, err
}

// LatestTrendBrief terimin son brifingini döndürür.
func LatestTrendBrief(ctx context.Context, pool *pgxpool.Pool, termID int64) (string, error) {
	var text string
	err := pool.QueryRow(ctx, `SELECT text FROM trend_briefs WHERE term_id=$1
		ORDER BY created_at DESC LIMIT 1`, termID).Scan(&text)
	return text, err
}

// ---- Trendden içerik üretimi ----

// SuggestCategoryForTerm terime en yakın kategoriyi seçer.
func SuggestCategoryForTerm(ctx context.Context, pool *pgxpool.Pool, siteID int64, term string) *int64 {
	cats, err := ListCategories(ctx, pool, siteID)
	if err != nil {
		return nil
	}
	best := 0.0
	var bestID int64
	for _, c := range cats {
		r := trends.MatchRatio(c.Name, term)
		if r > best {
			best = r
			bestID = c.ID
		}
	}
	if bestID == 0 || best < 0.5 {
		return nil
	}
	return &bestID
}

// TrendTermToContent terimden taslak içerik üretir. title/spot boşsa terim
// ve haber özetlerinden yerel bir taslak kurulur.
func TrendTermToContent(ctx context.Context, pool *pgxpool.Pool, termID, authorID int64, title, spot string) (int64, error) {
	term, err := GetTrendTerm(ctx, pool, termID)
	if err != nil {
		return 0, err
	}
	var newsTitle, newsURL, newsSource, newsSnippet string
	_ = pool.QueryRow(ctx, `SELECT title, url, source, snippet FROM trend_news
		WHERE term_id=$1 ORDER BY created_at DESC LIMIT 1`, termID).
		Scan(&newsTitle, &newsURL, &newsSource, &newsSnippet)
	if strings.TrimSpace(title) == "" {
		title = term.Term
		if newsTitle != "" {
			title = newsTitle
		}
	}
	if strings.TrimSpace(spot) == "" {
		spot = newsSnippet
		if spot == "" {
			spot = term.Term + " gündemde en çok konuşulan konular arasında. Gelişmeler ve ayrıntılar haberimizde."
		}
	}
	if len([]rune(spot)) > 240 {
		spot = string([]rune(spot)[:240])
	}
	var sourceURL, sourceName *string
	if newsURL != "" {
		sourceURL = &newsURL
	}
	if newsSource != "" {
		sourceName = &newsSource
	} else {
		sn := "Google Trends"
		sourceName = &sn
	}
	h := sha1.Sum([]byte("trend:" + itoa64(termID)))
	slug := platform.Slugify(title) + "-" + hex.EncodeToString(h[:4])
	cat := SuggestCategoryForTerm(ctx, pool, term.SiteID, term.Term)
	c := &Content{
		TypeKey:       "haber",
		Status:        "taslak",
		Title:         title,
		Spot:          spot,
		Detail:        textToBlocksStmt(spot),
		Slug:          &slug,
		CategoryID:    cat,
		AuthorID:      authorID,
		SourceURL:     sourceURL,
		SourceName:    sourceName,
		CommentStatus: "acik",
		Locale:        "tr",
		SiteID:        term.SiteID,
	}
	nid, err := CreateContent(ctx, pool, c)
	if err != nil {
		return 0, err
	}
	// Etiketler: terim kelimeleri + başlık anahtar kelimeleri.
	for _, w := range trends.Tokens(term.Term) {
		if tagID, e := EnsureTag(ctx, pool, w, platform.Slugify(w)); e == nil && tagID > 0 {
			_ = LinkContentTag(ctx, pool, nid, tagID)
		}
	}
	if _, err := AutoTagFromTitle(ctx, pool, nid, title, 5); err != nil {
		_ = err
	}
	return nid, nil
}

// TrendSuggestionsForContent editörde gösterilecek güncel terimler.
func TrendSuggestionsForContent(ctx context.Context, pool *pgxpool.Pool, siteID, categoryID int64, limit int) ([]TrendTermRow, error) {
	if limit <= 0 {
		limit = 8
	}
	f := TrendFilter{SiteID: siteID, Limit: limit * 3, MinScore: 20}
	items, err := ListTrendTerms(ctx, pool, f)
	if err != nil {
		return nil, err
	}
	if categoryID == 0 {
		if len(items) > limit {
			items = items[:limit]
		}
		return items, nil
	}
	// Kategori adıyla örtüşen terimleri öne al; kalanlarla tamamla.
	var exact, other []TrendTermRow
	var catName string
	_ = pool.QueryRow(ctx, `SELECT name FROM categories WHERE id=$1`, categoryID).Scan(&catName)
	for _, it := range items {
		if trends.MatchRatio(catName, it.Term) > 0 {
			exact = append(exact, it)
		} else {
			other = append(other, it)
		}
	}
	out := append(exact, other...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// PublicTrendTags ana sayfa "Gündem" bloğu için dış trendleri CatTrend'e eşler.
func PublicTrendTags(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]CatTrend, error) {
	if limit <= 0 {
		limit = 8
	}
	rows, err := pool.Query(ctx, `SELECT term FROM trend_terms
		WHERE site_id=$1 AND hidden=FALSE AND score >= 20
		  AND last_seen_at >= now() - interval '24 hours'
		ORDER BY score DESC, approx_traffic DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatTrend
	for rows.Next() {
		var term string
		if err := rows.Scan(&term); err != nil {
			return nil, err
		}
		out = append(out, CatTrend{Tag: term})
	}
	return out, rows.Err()
}

// HomeTrendTags önce dış trendleri, yoksa kendi içerik etiketlerini döndürür.
func HomeTrendTags(ctx context.Context, pool *pgxpool.Pool, siteID int64, lang string, limit int) []CatTrend {
	if ext, err := PublicTrendTags(ctx, pool, siteID, limit); err == nil && len(ext) > 0 {
		return ext
	}
	rows, err := CategoryTagCounts(ctx, pool, siteID, lang, 7, limit)
	if err != nil {
		return nil
	}
	return rows
}

// ---- Site içi arama ----

// LogSearchQuery anonim site içi arama sayacını artırır (IP/kullanıcı yok).
func LogSearchQuery(ctx context.Context, pool *pgxpool.Pool, siteID int64, q string) {
	q = strings.TrimSpace(q)
	if q == "" || len([]rune(q)) > 120 {
		return
	}
	norm := trends.NormalizeTerm(q)
	if norm == "" {
		return
	}
	_, _ = pool.Exec(ctx, `INSERT INTO search_queries (site_id, query, query_norm)
		VALUES ($1,$2,$3)
		ON CONFLICT (site_id, query_norm) DO UPDATE SET
			query = EXCLUDED.query, count = search_queries.count + 1, last_seen_at = now()`,
		siteID, q, norm)
}

// SearchQueryRow top arama satırıdır.
type SearchQueryRow struct {
	Query string
	Count int64
}

// TopSearchQueries son 30 günün en çok arananları.
func TopSearchQueries(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]SearchQueryRow, error) {
	if limit <= 0 {
		limit = 15
	}
	rows, err := pool.Query(ctx, `SELECT query, count FROM search_queries
		WHERE site_id=$1 AND last_seen_at >= now() - interval '30 days'
		ORDER BY count DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchQueryRow
	for rows.Next() {
		var r SearchQueryRow
		if err := rows.Scan(&r.Query, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- Search Console ----

// GSCQueryRow Search Console sorgu satırıdır.
type GSCQueryRow struct {
	Query       string
	Page        string
	Clicks      int
	Impressions int
	CTR         float64
	Position    float64
}

// SaveGSCQueries bir günün sorgularını yazar.
func SaveGSCQueries(ctx context.Context, pool *pgxpool.Pool, siteID int64, property string, day time.Time, rows []GSCQueryRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO gsc_queries
			(site_id, property, day, query, page, clicks, impressions, ctr, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (site_id, property, day, query, page) DO UPDATE SET
				clicks=EXCLUDED.clicks, impressions=EXCLUDED.impressions,
				ctr=EXCLUDED.ctr, position=EXCLUDED.position`,
			siteID, property, day, r.Query, r.Page,
			r.Clicks, r.Impressions, r.CTR, r.Position); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// TopGSCQueries son N günün en çok gösterim alan sorguları.
func TopGSCQueries(ctx context.Context, pool *pgxpool.Pool, siteID int64, days, limit int) ([]GSCQueryRow, error) {
	if days <= 0 {
		days = 28
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `SELECT query, '' AS page, sum(clicks)::int, sum(impressions)::int,
		CASE WHEN sum(impressions)>0 THEN sum(clicks)::float8/sum(impressions) ELSE 0 END,
		CASE WHEN count(*)>0 THEN avg(position) ELSE 0 END
		FROM gsc_queries WHERE site_id=$1 AND day >= CURRENT_DATE - ($2::int)
		GROUP BY query ORDER BY sum(impressions) DESC LIMIT $3`, siteID, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GSCQueryRow
	for rows.Next() {
		var r GSCQueryRow
		if err := rows.Scan(&r.Query, &r.Page, &r.Clicks, &r.Impressions, &r.CTR, &r.Position); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- Trend ayarları ----

// GetTrendSetting trend bölümünden düz string okur.
func GetTrendSetting(ctx context.Context, pool *pgxpool.Pool, key string) string {
	b, err := GetSetting(ctx, pool, "trend", key)
	if err != nil {
		return ""
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	return strings.Trim(string(b), `"`)
}

// SetTrendSetting trend bölümüne JSON string yazar.
func SetTrendSetting(ctx context.Context, pool *pgxpool.Pool, key, val string) error {
	b, _ := json.Marshal(val)
	return SetSetting(ctx, pool, "trend", key, b)
}

// TrendWeights ayarlardan skor ağırlıklarını okur.
func TrendWeights(ctx context.Context, pool *pgxpool.Pool) trends.Weights {
	w := trends.DefaultWeights()
	b, err := GetSetting(ctx, pool, "trend", "agirliklar")
	if err == nil {
		var parsed trends.Weights
		if json.Unmarshal(b, &parsed) == nil {
			w = parsed.Normalize()
		}
	}
	return w
}

// TrendAlertThreshold uyarı eşiği.
func TrendAlertThreshold(ctx context.Context, pool *pgxpool.Pool) int {
	s := GetTrendSetting(ctx, pool, "uyari_esik")
	if s == "" {
		return 70
	}
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v > 0 && v <= 100 {
		return v
	}
	return 70
}

const (
	defaultTrendCoverageHours = 180 * 24 // 180 gün: "bu konuda içeriğimiz var mı?"
	defaultTrendRecentHours   = 72       // 3 gün: "bu haber döngüsünde işledik mi?"
	maxTrendMatchContents     = 5000     // geniş pencere sessizce kırpılmasın
)

// TrendCoverageHours arşiv kapsama penceresi (saat). Panelde ve API'de
// gösterilen gerçek içerik sayısı bu pencereden gelir.
func TrendCoverageHours(ctx context.Context, pool *pgxpool.Pool) int {
	return trendHourSetting(ctx, pool, "kapsama_saat", defaultTrendCoverageHours, 24, 5*365*24)
}

// TrendRecentHours taze (haber döngüsü) penceresi (saat). Skorlamadaki
// "içerik boşluğu" sinyali bu pencereye bakar.
func TrendRecentHours(ctx context.Context, pool *pgxpool.Pool) int {
	return trendHourSetting(ctx, pool, "kapsama_taze_saat", defaultTrendRecentHours, 1, 30*24)
}

// trendHourSetting ayarı okur; aralık dışındaysa varsayılana döner.
func trendHourSetting(ctx context.Context, pool *pgxpool.Pool, key string, def, min, max int) int {
	s := strings.TrimSpace(GetTrendSetting(ctx, pool, key))
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		return def
	}
	return v
}

// SaveTrendWeights ağırlıkları JSON olarak yazar.
func SaveTrendWeights(ctx context.Context, pool *pgxpool.Pool, w trends.Weights) error {
	b, _ := json.Marshal(w.Normalize())
	return SetSetting(ctx, pool, "trend", "agirliklar", b)
}

// ---- Saklama ----

// TrendRetention eski zaman serisi/haber/koşu kayıtlarını temizler.
func TrendRetention(ctx context.Context, pool *pgxpool.Pool) error {
	for _, q := range []string{
		`DELETE FROM trend_points WHERE captured_at < now() - interval '90 days'`,
		`DELETE FROM trend_news WHERE created_at < now() - interval '30 days'`,
		`DELETE FROM trend_alerts WHERE created_at < now() - interval '90 days'`,
		`DELETE FROM trend_briefs WHERE created_at < now() - interval '90 days'`,
		`DELETE FROM trend_fetch_runs WHERE started_at < now() - interval '30 days'`,
		`DELETE FROM gundem_items WHERE fetched_at < now() - interval '120 days'`,
		`DELETE FROM search_queries WHERE last_seen_at < now() - interval '180 days'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
