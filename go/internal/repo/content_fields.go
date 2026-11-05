package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Önyüz/alan spec (07): içerik ek alanları ----

// ContentFields kantan-hizalı ek içerik alanları.
type ContentFields struct {
	ImageCredit   string
	SocialSummary string
	AISummary     string
	AudioURL      string
	AudioSeconds  float64
	AudioStatus   string
	ScrollCount   int64
	// VideoSeconds video içeriğinin süresi (saniye); 0 = bilinmiyor (Faz 17).
	VideoSeconds int
}

// GetContentFields içeriğin ek alanlarını okur.
func GetContentFields(ctx context.Context, pool *pgxpool.Pool, id int64) ContentFields {
	var f ContentFields
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(image_credit,''), COALESCE(social_summary,''), COALESCE(ai_summary,''),
		       COALESCE(audio_url,''), COALESCE(audio_seconds,0), audio_status,
		       COALESCE(video_seconds,0),
		       COALESCE((SELECT scroll_count FROM content_metrics WHERE content_id=$1),0)
		FROM contents WHERE id=$1`, id).
		Scan(&f.ImageCredit, &f.SocialSummary, &f.AISummary, &f.AudioURL, &f.AudioSeconds, &f.AudioStatus, &f.VideoSeconds, &f.ScrollCount)
	return f
}

// SetContentVideoDuration video içeriğinin süresini yazar (Faz 17).
// seconds <= 0 ise NULL yapılır — "bilinmiyor".
func SetContentVideoDuration(ctx context.Context, pool *pgxpool.Pool, id int64, seconds int) error {
	if seconds < 0 {
		seconds = 0
	}
	_, err := pool.Exec(ctx, `UPDATE contents SET video_seconds=$2 WHERE id=$1`, id, seconds)
	return err
}

// VideoDurations verilen içerik kimliklerinin video sürelerini tek sorguda
// döndürür (yoksa 0). Video seçici listesi için — N+1 sorgu yapmamak adına.
func VideoDurations(ctx context.Context, pool *pgxpool.Pool, ids []int64) map[int64]int {
	out := map[int64]int{}
	if len(ids) == 0 {
		return out
	}
	rows, err := pool.Query(ctx, `SELECT id, COALESCE(video_seconds,0) FROM contents WHERE id = ANY($1)`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var sec int
		if err := rows.Scan(&id, &sec); err == nil {
			out[id] = sec
		}
	}
	return out
}

// SetContentEditorial görsel kredisi / sosyal özet / AI özet alanlarını yazar.
func SetContentEditorial(ctx context.Context, pool *pgxpool.Pool, id int64, imageCredit, socialSummary, aiSummary string) error {
	_, err := pool.Exec(ctx, `
		UPDATE contents SET image_credit=nullif($2,''), social_summary=nullif($3,''), ai_summary=nullif($4,'')
		WHERE id=$1`, id, imageCredit, socialSummary, aiSummary)
	return err
}

// SetContentAudio makale sesi (TTS) bilgisini yazar.
func SetContentAudio(ctx context.Context, pool *pgxpool.Pool, id int64, url string, seconds float64, status string) error {
	if status == "" {
		status = "yok"
	}
	_, err := pool.Exec(ctx, `
		UPDATE contents SET audio_url=nullif($2,''), audio_seconds=$3, audio_status=$4 WHERE id=$1`,
		id, url, seconds, status)
	return err
}

// IncrementScroll içeriğin kaydırma derinliği sayacını artırır (contents.updated_at'i etkilemez).
func IncrementScroll(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content_metrics (content_id, scroll_count) VALUES ($1,1)
		ON CONFLICT (content_id) DO UPDATE SET scroll_count = content_metrics.scroll_count + 1, updated_at=now()`, id)
	return err
}
