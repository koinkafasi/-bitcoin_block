package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TranscriptionJob struct {
	ID               int64
	SiteID           int64
	MediaID          int64
	ContentID        *int64
	UserID           *int64
	Status           string
	Provider         string
	Model            string
	Language         string
	ProviderJobID    string
	Progress         int
	DurationMS       int64
	SpeakersExpected int
	Attempts         int
	Error            string
	ConsentAt        *time.Time
	LockedAt         *time.Time
	CreatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
}

// AudioSegment transkriptin zaman damgalı tek konuşma bölümüdür.
type AudioSegment struct {
	StartMS    int64       `json:"start_ms"`
	EndMS      int64       `json:"end_ms"`
	Speaker    string      `json:"speaker,omitempty"`
	Text       string      `json:"text"`
	Confidence float64     `json:"confidence,omitempty"`
	Words      []AudioWord `json:"words,omitempty"`
}

type AudioWord struct {
	StartMS    int64   `json:"start_ms"`
	EndMS      int64   `json:"end_ms"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// AudioTranscript bir ses kaydının transkriptidir. Segments JSONB olarak
// saklanır; metin araması için search tsvector trigger ile üretilir.
type AudioTranscript struct {
	ID         int64
	SiteID     int64
	MediaID    int64
	ContentID  *int64
	JobID      *int64
	Language   string
	Provider   string
	Model      string
	DurationMS int64
	Text       string
	Segments   []AudioSegment
	Speakers   []string
	// Waveform dalga formu tepe noktaları (0..1, ~400 kova). Boşsa oynatıcı
	// düz ilerleme çubuğu gösterir.
	Waveform  []float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ContentAudio içerik formundaki bir ses kaydını işi ve transkriptiyle taşır.
type ContentAudio struct {
	Media      Media
	Job        TranscriptionJob
	Transcript *AudioTranscript
}

const transcriptionJobCols = `id, site_id, media_id, content_id, user_id, status, provider,
	model, language, provider_job_id, progress, duration_ms, speakers_expected, attempts, error,
	consent_at, locked_at, created_at, started_at, finished_at`

const transcriptionJobColsJ = `j.id, j.site_id, j.media_id, j.content_id, j.user_id, j.status,
	j.provider, j.model, j.language, j.provider_job_id, j.progress, j.duration_ms,
	j.speakers_expected, j.attempts, j.error, j.consent_at, j.locked_at, j.created_at,
	j.started_at, j.finished_at`

func scanTranscriptionJob(row pgx.Row) (TranscriptionJob, error) {
	var j TranscriptionJob
	err := row.Scan(&j.ID, &j.SiteID, &j.MediaID, &j.ContentID, &j.UserID, &j.Status,
		&j.Provider, &j.Model, &j.Language, &j.ProviderJobID, &j.Progress, &j.DurationMS,
		&j.SpeakersExpected, &j.Attempts, &j.Error, &j.ConsentAt, &j.LockedAt, &j.CreatedAt,
		&j.StartedAt, &j.FinishedAt)
	return j, err
}

// CreateTranscriptionJob yeni ASR işi oluşturur. Aynı medya için aktif iş
// varsa çağıran taraf önce GetActiveTranscriptionJobForMedia ile kontrol eder.
func CreateTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, j *TranscriptionJob) (int64, error) {
	status := j.Status
	if status == "" {
		status = "queued"
	}
	provider := j.Provider
	if provider == "" {
		provider = "assemblyai"
	}
	language := j.Language
	if language == "" {
		language = "tr"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO transcription_jobs
			(site_id, media_id, content_id, user_id, status, provider, model, language,
			 speakers_expected, consent_at, started_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			CASE WHEN $5='running' THEN now() ELSE NULL END)
		RETURNING id`,
		siteIDOr1(j.SiteID), j.MediaID, nullableInt64P(j.ContentID), nullableInt64P(j.UserID),
		status, provider, j.Model, language, j.SpeakersExpected, j.ConsentAt,
	).Scan(&id)
	return id, err
}

// GetTranscriptionJob tek işi döndürür.
func GetTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, id int64) (TranscriptionJob, error) {
	j, err := scanTranscriptionJob(pool.QueryRow(ctx,
		`SELECT `+transcriptionJobCols+` FROM transcription_jobs WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return TranscriptionJob{}, ErrNotFound
	}
	return j, err
}

// GetActiveTranscriptionJobForMedia medyanın kuyrukta/çalışan işini döndürür.
func GetActiveTranscriptionJobForMedia(ctx context.Context, pool *pgxpool.Pool, mediaID int64) (TranscriptionJob, bool, error) {
	j, err := scanTranscriptionJob(pool.QueryRow(ctx, `
		SELECT `+transcriptionJobCols+` FROM transcription_jobs
		WHERE media_id=$1 AND status IN ('queued','running')
		ORDER BY id DESC LIMIT 1`, mediaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return TranscriptionJob{}, false, nil
	}
	if err != nil {
		return TranscriptionJob{}, false, err
	}
	return j, true, nil
}

// GetLatestTranscriptionJobForMedia medyanın en son işini döndürür.
func GetLatestTranscriptionJobForMedia(ctx context.Context, pool *pgxpool.Pool, mediaID int64) (TranscriptionJob, error) {
	j, err := scanTranscriptionJob(pool.QueryRow(ctx, `
		SELECT `+transcriptionJobCols+` FROM transcription_jobs
		WHERE media_id=$1 ORDER BY id DESC LIMIT 1`, mediaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return TranscriptionJob{}, ErrNotFound
	}
	return j, err
}

// ListTranscriptionJobsForContent içeriğin ses işlerini yeniden eskiye listeler.
func ListTranscriptionJobsForContent(ctx context.Context, pool *pgxpool.Pool, contentID int64, limit int) ([]TranscriptionJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT `+transcriptionJobCols+` FROM transcription_jobs
		WHERE content_id=$1 ORDER BY id DESC LIMIT $2`, contentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TranscriptionJob
	for rows.Next() {
		j, err := scanTranscriptionJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ClaimTranscriptionJob kuyruktaki en eski işi atomik olarak sahiplenir.
func ClaimTranscriptionJob(ctx context.Context, pool *pgxpool.Pool) (TranscriptionJob, bool, error) {
	j, err := scanTranscriptionJob(pool.QueryRow(ctx, `
		WITH next AS (
			SELECT id FROM transcription_jobs WHERE status='queued'
			ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE transcription_jobs j
		SET status='running', started_at=now(), locked_at=now(), error='', attempts=attempts+1
		FROM next WHERE j.id=next.id
		RETURNING `+transcriptionJobColsJ))
	if errors.Is(err, pgx.ErrNoRows) {
		return TranscriptionJob{}, false, nil
	}
	if err != nil {
		return TranscriptionJob{}, false, err
	}
	return j, true, nil
}

// StartTranscriptionJob queued işi running'e alır. İş iptal edildiyse false döner.
func StartTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, id int64) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET status='running', started_at=now(), locked_at=now(), error='', attempts=attempts+1
		WHERE id=$1 AND status='queued'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateTranscriptionJobProgress ilerlemeyi ve sağlayıcı iş kimliğini yazar.
func UpdateTranscriptionJobProgress(ctx context.Context, pool *pgxpool.Pool, id int64, progress int, providerJobID string, durationMS int64) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	_, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET progress=$2,
		    provider_job_id=CASE WHEN $3<>'' THEN $3 ELSE provider_job_id END,
		    duration_ms=CASE WHEN $4>0 THEN $4 ELSE duration_ms END,
		    locked_at=now()
		WHERE id=$1 AND status='running'`,
		id, progress, providerJobID, durationMS)
	return err
}

// FinishTranscriptionJob işi terminal duruma getirir. Yalnız çalışan işi
// günceller; bu arada iptal edilmiş işin durumu korunur.
func FinishTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, id int64, status, errMsg, provider, model string, durationMS int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET status=$2, error=$3, provider=$4, model=$5,
		    progress=CASE WHEN $2='completed' THEN 100 ELSE progress END,
		    duration_ms=CASE WHEN $6>0 THEN $6 ELSE duration_ms END,
		    locked_at=NULL, finished_at=now()
		WHERE id=$1 AND status='running'`,
		id, status, errMsg, provider, model, durationMS)
	return err
}

// CancelTranscriptionJob kuyruktaki/çalışan işi iptal eder.
func CancelTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, id int64) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET status='cancelled', error='kullanıcı iptal etti', locked_at=NULL, finished_at=now()
		WHERE id=$1 AND status IN ('queued','running')`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// RetryTranscriptionJob terminal işi yeniden kuyruğa alır.
func RetryTranscriptionJob(ctx context.Context, pool *pgxpool.Pool, id int64) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET status='queued', error='', progress=0, provider_job_id='', duration_ms=0,
		    attempts=0, locked_at=NULL, started_at=NULL, finished_at=NULL
		WHERE id=$1 AND status IN ('failed','cancelled','completed')`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// RecoverStaleTranscriptionJobs süreç çöktüğü için running kalmış işleri
// kuyruğa alır; 3 denemeden sonra kalıcı olarak başarısız işaretler.
func RecoverStaleTranscriptionJobs(ctx context.Context, pool *pgxpool.Pool, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		olderThan = 30 * time.Minute
	}
	tag, err := pool.Exec(ctx, `
		UPDATE transcription_jobs
		SET status=CASE WHEN attempts >= 3 THEN 'failed' ELSE 'queued' END,
		    error=CASE WHEN attempts >= 3 THEN 'zaman aşımı: worker yanıt vermedi' ELSE '' END,
		    locked_at=NULL,
		    started_at=CASE WHEN attempts >= 3 THEN started_at ELSE NULL END,
		    finished_at=CASE WHEN attempts >= 3 THEN now() ELSE NULL END
		WHERE status='running'
		  AND COALESCE(locked_at, started_at, created_at) < now() - ($1::int * interval '1 second')`,
		int64(olderThan.Seconds()))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const audioTranscriptCols = `id, site_id, media_id, content_id, job_id, language, provider,
	model, duration_ms, text, segments, speakers, waveform, created_at, updated_at`

func scanAudioTranscript(row pgx.Row) (AudioTranscript, error) {
	var t AudioTranscript
	var segJSON, spkJSON, wfJSON []byte
	err := row.Scan(&t.ID, &t.SiteID, &t.MediaID, &t.ContentID, &t.JobID, &t.Language,
		&t.Provider, &t.Model, &t.DurationMS, &t.Text, &segJSON, &spkJSON, &wfJSON,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	_ = json.Unmarshal(segJSON, &t.Segments)
	_ = json.Unmarshal(spkJSON, &t.Speakers)
	_ = json.Unmarshal(wfJSON, &t.Waveform)
	return t, nil
}

// UpsertAudioTranscript medyanın transkriptini oluşturur veya günceller.
func UpsertAudioTranscript(ctx context.Context, pool *pgxpool.Pool, t *AudioTranscript) (int64, error) {
	segments := t.Segments
	if segments == nil {
		segments = []AudioSegment{}
	}
	speakers := t.Speakers
	if speakers == nil {
		speakers = []string{}
	}
	segJSON, err := json.Marshal(segments)
	if err != nil {
		return 0, err
	}
	spkJSON, err := json.Marshal(speakers)
	if err != nil {
		return 0, err
	}
	waveform := t.Waveform
	if waveform == nil {
		waveform = []float64{}
	}
	wfJSON, err := json.Marshal(waveform)
	if err != nil {
		return 0, err
	}
	var id int64
	err = pool.QueryRow(ctx, `
		INSERT INTO audio_transcripts
			(site_id, media_id, content_id, job_id, language, provider, model,
			 duration_ms, text, segments, speakers, waveform)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::jsonb,$12::jsonb)
		ON CONFLICT (media_id) DO UPDATE SET
			content_id=EXCLUDED.content_id,
			job_id=EXCLUDED.job_id,
			language=EXCLUDED.language,
			provider=EXCLUDED.provider,
			model=EXCLUDED.model,
			duration_ms=EXCLUDED.duration_ms,
			text=EXCLUDED.text,
			segments=EXCLUDED.segments,
			speakers=EXCLUDED.speakers,
			waveform=EXCLUDED.waveform,
			updated_at=now()
		RETURNING id`,
		siteIDOr1(t.SiteID), t.MediaID, nullableInt64P(t.ContentID), nullableInt64P(t.JobID),
		t.Language, t.Provider, t.Model, t.DurationMS, t.Text, string(segJSON), string(spkJSON),
		string(wfJSON),
	).Scan(&id)
	return id, err
}

// GetAudioTranscriptByMedia medyanın transkriptini döndürür.
func GetAudioTranscriptByMedia(ctx context.Context, pool *pgxpool.Pool, mediaID int64) (AudioTranscript, error) {
	t, err := scanAudioTranscript(pool.QueryRow(ctx,
		`SELECT `+audioTranscriptCols+` FROM audio_transcripts WHERE media_id=$1`, mediaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AudioTranscript{}, ErrNotFound
	}
	return t, err
}

// GetAudioTranscript tek transkripti döndürür.
func GetAudioTranscript(ctx context.Context, pool *pgxpool.Pool, id int64) (AudioTranscript, error) {
	t, err := scanAudioTranscript(pool.QueryRow(ctx,
		`SELECT `+audioTranscriptCols+` FROM audio_transcripts WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return AudioTranscript{}, ErrNotFound
	}
	return t, err
}

func ListAudioForContent(ctx context.Context, pool *pgxpool.Pool, siteID, contentID int64) ([]ContentAudio, error) {
	jobs, err := ListTranscriptionJobsForContent(ctx, pool, contentID, 100)
	if err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	out := make([]ContentAudio, 0, len(jobs))
	for _, j := range jobs {
		if seen[j.MediaID] {
			continue
		}
		seen[j.MediaID] = true
		m, err := GetMedia(ctx, pool, j.MediaID)
		if err != nil || m.SiteID != siteIDOr1(siteID) || m.Visibility != "private" {
			continue
		}
		var tr *AudioTranscript
		if t, err := GetAudioTranscriptByMedia(ctx, pool, j.MediaID); err == nil {
			tr = &t
		}
		out = append(out, ContentAudio{Media: m, Job: j, Transcript: tr})
	}
	return out, nil
}

// GetAudioForContentMedia içerik + medya çiftinin ses kaydını doğrular.
func GetAudioForContentMedia(ctx context.Context, pool *pgxpool.Pool, contentID, mediaID int64) (ContentAudio, error) {
	m, err := GetMedia(ctx, pool, mediaID)
	if err != nil {
		return ContentAudio{}, err
	}
	if m.Visibility != "private" {
		return ContentAudio{}, ErrNotFound
	}
	j, err := GetLatestTranscriptionJobForMedia(ctx, pool, mediaID)
	if err != nil {
		return ContentAudio{}, err
	}
	if j.ContentID == nil || *j.ContentID != contentID {
		return ContentAudio{}, ErrNotFound
	}
	var tr *AudioTranscript
	if t, err := GetAudioTranscriptByMedia(ctx, pool, mediaID); err == nil {
		tr = &t
	}
	return ContentAudio{Media: m, Job: j, Transcript: tr}, nil
}
