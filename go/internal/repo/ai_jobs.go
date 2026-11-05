package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIJob AI editör işinin denetim kaydıdır. Sonuç, kullanıcı onayından önce
// yalnızca burada tutulur; contents tablosuna otomatik yazılmaz.
type AIJob struct {
	ID               int64
	SiteID           int64
	ContentID        *int64
	UserID           *int64
	Action           string
	Status           string
	SelectedText     string
	Instruction      string
	ArticleVersion   int
	Context          []byte
	Result           []byte
	Error            string
	Provider         string
	Model            string
	PromptVersion    string
	PromptTokens     int
	CompletionTokens int
	CostEst          float64
	CreatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
}

// AIJobSource araştırma sırasında bulunan kaynaktır.
type AIJobSource struct {
	ID          int64
	JobID       int64
	URL         string
	Title       string
	Publisher   string
	PublishedAt *time.Time
	Snippet     string
}

// ContentRevision kaydetmeden önceki içerik anlık görüntüsüdür.
type ContentRevision struct {
	ID        int64
	ContentID int64
	Version   int
	Title     string
	Spot      string
	Detail    []byte
	Snapshot  []byte
	UserID    *int64
	Source    string
	AIJobID   *int64
	CreatedAt time.Time
}

// CreateAIJob işi oluşturur. Senkron işler doğrudan 'running' başlar; böylece
// worker aynı işi kapıp ikinci kez çalıştırmaz.
func CreateAIJob(ctx context.Context, pool *pgxpool.Pool, j *AIJob) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO ai_jobs
			(site_id, content_id, user_id, action, status, selected_text, instruction,
			 article_version, context, started_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,
			CASE WHEN $5='running' THEN now() ELSE NULL END)
		RETURNING id`,
		j.SiteID, nullableInt64P(j.ContentID), nullableInt64P(j.UserID), j.Action,
		j.Status, j.SelectedText, j.Instruction, j.ArticleVersion, orEmptyJSON(j.Context),
	).Scan(&id)
	return id, err
}

// StartAIJob queued işi running'e alır. İş bu arada iptal edildiyse false döner.
func StartAIJob(ctx context.Context, pool *pgxpool.Pool, id int64) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE ai_jobs SET status='running', started_at=now(), error=''
		WHERE id=$1 AND status='queued'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FinishAIJob işi terminal duruma getirir. Yalnız çalışan işi günceller; bu
// arada iptal edilmiş işin sonucu iptal durumunun üzerine yazılmaz.
func FinishAIJob(ctx context.Context, pool *pgxpool.Pool, id int64, status string, result []byte, errMsg, provider, model, promptVersion string, promptTokens, completionTokens int, cost float64) error {
	var res any
	if len(result) > 0 {
		res = string(result)
	}
	_, err := pool.Exec(ctx, `
		UPDATE ai_jobs SET status=$2, result=$3::jsonb, error=$4, provider=$5, model=$6,
			prompt_version=$7, prompt_tokens=$8, completion_tokens=$9, cost_est=$10,
			finished_at=now()
		WHERE id=$1 AND status='running'`,
		id, status, res, errMsg, provider, model, promptVersion, promptTokens, completionTokens, cost)
	return err
}

// ClaimAIJob kuyruktaki en eski işi atomik olarak sahiplenir. Birden fazla site
// süreci aynı anda çalışsa bile FOR UPDATE SKIP LOCKED aynı işi iki kez almaz.
func ClaimAIJob(ctx context.Context, pool *pgxpool.Pool) (AIJob, bool, error) {
	j, err := scanAIJob(pool.QueryRow(ctx, `
		WITH next AS (
			SELECT id FROM ai_jobs WHERE status='queued'
			ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE ai_jobs j SET status='running', started_at=now(), error=''
		FROM next WHERE j.id=next.id
		RETURNING `+aiJobColsJ))
	if errors.Is(err, pgx.ErrNoRows) {
		return AIJob{}, false, nil
	}
	if err != nil {
		return AIJob{}, false, err
	}
	return j, true, nil
}

// CancelAIJob kuyruktaki/çalışan işi iptal eder.
func CancelAIJob(ctx context.Context, pool *pgxpool.Pool, id int64) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE ai_jobs SET status='cancelled', error='kullanıcı iptal etti', finished_at=now()
		WHERE id=$1 AND status IN ('queued','running')`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// RecoverStaleAIJobs süreç çöktüğü için running kalmış işleri kuyruğa alır.
func RecoverStaleAIJobs(ctx context.Context, pool *pgxpool.Pool, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		olderThan = 15 * time.Minute
	}
	tag, err := pool.Exec(ctx, `
		UPDATE ai_jobs SET status='queued', started_at=NULL, error=''
		WHERE status='running' AND started_at < now() - ($1::int * interval '1 second')`,
		int64(olderThan.Seconds()))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const aiJobCols = `id, site_id, content_id, user_id, action, status, selected_text,
	instruction, article_version, context, result, error, provider, model, prompt_version,
	prompt_tokens, completion_tokens, cost_est, created_at, started_at, finished_at`

const aiJobColsJ = `j.id, j.site_id, j.content_id, j.user_id, j.action, j.status,
	j.selected_text, j.instruction, j.article_version, j.context, j.result, j.error,
	j.provider, j.model, j.prompt_version, j.prompt_tokens, j.completion_tokens,
	j.cost_est, j.created_at, j.started_at, j.finished_at`

func scanAIJob(row pgx.Row) (AIJob, error) {
	var j AIJob
	err := row.Scan(&j.ID, &j.SiteID, &j.ContentID, &j.UserID, &j.Action, &j.Status,
		&j.SelectedText, &j.Instruction, &j.ArticleVersion, &j.Context, &j.Result,
		&j.Error, &j.Provider, &j.Model, &j.PromptVersion, &j.PromptTokens,
		&j.CompletionTokens, &j.CostEst, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	return j, err
}

// GetAIJob tek işi döndürür.
func GetAIJob(ctx context.Context, pool *pgxpool.Pool, id int64) (AIJob, error) {
	j, err := scanAIJob(pool.QueryRow(ctx, `SELECT `+aiJobCols+` FROM ai_jobs WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return AIJob{}, ErrNotFound
	}
	return j, err
}

// ListAIJobs içeriğin son AI işlerini döndürür.
func ListAIJobs(ctx context.Context, pool *pgxpool.Pool, contentID int64, limit int) ([]AIJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT `+aiJobCols+` FROM ai_jobs WHERE content_id=$1 ORDER BY id DESC LIMIT $2`,
		contentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIJob
	for rows.Next() {
		j, err := scanAIJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// AddAIJobSource araştırma kaynağını işe bağlar.
func AddAIJobSource(ctx context.Context, pool *pgxpool.Pool, s *AIJobSource) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO ai_job_sources (job_id, url, title, publisher, published_at, snippet)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		s.JobID, s.URL, s.Title, s.Publisher, s.PublishedAt, s.Snippet)
	return err
}

// ListAIJobSources işin kaynaklarını döndürür.
func ListAIJobSources(ctx context.Context, pool *pgxpool.Pool, jobID int64) ([]AIJobSource, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, job_id, url, title, publisher, published_at, snippet
		FROM ai_job_sources WHERE job_id=$1 ORDER BY id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIJobSource
	for rows.Next() {
		var s AIJobSource
		if err := rows.Scan(&s.ID, &s.JobID, &s.URL, &s.Title, &s.Publisher, &s.PublishedAt, &s.Snippet); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SnapshotContentRevision mevcut içeriği kaydetmeden önce arşivler.
// Aynı sürüm iki kez yazılmaz (ON CONFLICT DO NOTHING).
func SnapshotContentRevision(ctx context.Context, pool *pgxpool.Pool, contentID int64, userID *int64, source string, aiJobID *int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content_revisions
			(content_id, version, title, spot, detail, snapshot, user_id, source, ai_job_id)
		SELECT c.id, c.version, c.title, COALESCE(c.spot,''), c.detail, to_jsonb(c), $2, $3, $4
		FROM contents c WHERE c.id=$1
		ON CONFLICT (content_id, version) DO NOTHING`,
		contentID, nullableInt64P(userID), source, nullableInt64P(aiJobID))
	return err
}

// ListContentRevisions içeriğin sürüm geçmişini döndürür.
func ListContentRevisions(ctx context.Context, pool *pgxpool.Pool, contentID int64, limit int) ([]ContentRevision, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT id, content_id, version, title, COALESCE(spot,''), detail, '{}'::jsonb,
			user_id, source, ai_job_id, created_at
		FROM content_revisions WHERE content_id=$1 ORDER BY version DESC LIMIT $2`,
		contentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentRevision
	for rows.Next() {
		var r ContentRevision
		if err := rows.Scan(&r.ID, &r.ContentID, &r.Version, &r.Title, &r.Spot, &r.Detail,
			&r.Snapshot, &r.UserID, &r.Source, &r.AIJobID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetContentRevision tek sürümü anlık görüntüsüyle döndürür.
func GetContentRevision(ctx context.Context, pool *pgxpool.Pool, contentID, revisionID int64) (ContentRevision, error) {
	var r ContentRevision
	err := pool.QueryRow(ctx, `
		SELECT id, content_id, version, title, COALESCE(spot,''), detail, snapshot,
			user_id, source, ai_job_id, created_at
		FROM content_revisions WHERE id=$1 AND content_id=$2`,
		revisionID, contentID,
	).Scan(&r.ID, &r.ContentID, &r.Version, &r.Title, &r.Spot, &r.Detail,
		&r.Snapshot, &r.UserID, &r.Source, &r.AIJobID, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentRevision{}, ErrNotFound
	}
	return r, err
}

// RestoreContentRevision sürümü yeni bir sürüm olarak geri yükler.
func RestoreContentRevision(ctx context.Context, pool *pgxpool.Pool, contentID, revisionID int64, userID *int64) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var snap []byte
	if err := tx.QueryRow(ctx, `
		SELECT snapshot FROM content_revisions WHERE id=$1 AND content_id=$2`,
		revisionID, contentID).Scan(&snap); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	var old struct {
		Title  string          `json:"title"`
		Spot   string          `json:"spot"`
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(snap, &old); err != nil {
		return 0, err
	}
	detail := old.Detail
	if len(detail) == 0 || string(detail) == "null" {
		detail = []byte("[]")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO content_revisions
			(content_id, version, title, spot, detail, snapshot, user_id, source, ai_job_id)
		SELECT c.id, c.version, c.title, COALESCE(c.spot,''), c.detail, to_jsonb(c), $2, 'geri-al', NULL
		FROM contents c WHERE c.id=$1
		ON CONFLICT (content_id, version) DO NOTHING`,
		contentID, nullableInt64P(userID)); err != nil {
		return 0, err
	}
	var version int
	if err := tx.QueryRow(ctx, `
		UPDATE contents SET title=$2, spot=$3, detail=$4::jsonb, version=version+1, updated_at=now()
		WHERE id=$1 RETURNING version`,
		contentID, old.Title, old.Spot, string(detail)).Scan(&version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return version, nil
}
