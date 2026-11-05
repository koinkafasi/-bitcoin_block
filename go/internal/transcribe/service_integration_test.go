//go:build integration

package transcribe

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

type fakeProvider struct {
	res Result
}

func (f fakeProvider) Name() string { return "fake" }

func (f fakeProvider) Transcribe(_ context.Context, _ Request, progress ProgressFunc) (Result, error) {
	if progress != nil {
		progress(50, "fake-job-1")
	}
	return f.res, nil
}

// TestServiceProcessJobIntegration gerçek şema + ffmpeg + private depolama
// üzerinden kuyruk işini uçtan uca doğrular. Varsayılan olarak atlanır.
func TestServiceProcessJobIntegration(t *testing.T) {
	if os.Getenv("TRANSCRIBE_INTEGRATION") != "1" {
		t.Skip("TRANSCRIBE_INTEGRATION=1 değil")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL yok")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg PATH'te yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	var uid int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&uid))

	dir := t.TempDir()
	src := filepath.Join(dir, "kaynak.wav")
	gen := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1.2", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("test sesi üretilemedi: %v (%s)", err, string(out))
	}
	data, err := os.ReadFile(src)
	require.NoError(t, err)

	storedPath, size, err := platform.SavePrivateAudio(bytes.NewReader(data), "entegrasyon.wav")
	require.NoError(t, err)
	defer func() { _ = platform.DeletePrivate(storedPath) }()

	mediaID, err := repo.CreateMedia(ctx, pool, &repo.Media{
		SiteID: 1, UserID: uid, Kind: "audio", Filename: "entegrasyon.wav",
		Path: storedPath, Mime: "audio/wav", SizeBytes: &size,
		Title: "Entegrasyon testi", Category: "ses", Source: "upload", Visibility: "private",
	})
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audio_transcripts WHERE media_id=$1`, mediaID)
		_, _ = pool.Exec(ctx, `DELETE FROM transcription_jobs WHERE media_id=$1`, mediaID)
		_, _ = pool.Exec(ctx, `DELETE FROM media_files WHERE id=$1`, mediaID)
	}()

	jobID, err := repo.CreateTranscriptionJob(ctx, pool, &repo.TranscriptionJob{
		SiteID: 1, MediaID: mediaID, Provider: "fake", Language: "tr", SpeakersExpected: 1,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, fakeProvider{res: Result{
		Text:       "Merhaba dünya.",
		Language:   "tr",
		DurationMS: 1200,
		Segments: []Segment{{
			StartMS: 0, EndMS: 1200, Speaker: "A", Text: "Merhaba dünya.", Confidence: 0.99,
		}},
		Speakers: []string{"A"},
		Provider: "fake",
	}}, NewFFmpeg("", ""), logger)

	require.NoError(t, svc.ProcessJob(ctx, jobID))

	job, err := repo.GetTranscriptionJob(ctx, pool, jobID)
	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status)
	assert.Equal(t, 100, job.Progress)
	assert.Equal(t, "fake", job.Provider)
	assert.NotEmpty(t, job.ProviderJobID)
	assert.Greater(t, job.DurationMS, int64(0))

	tr, err := repo.GetAudioTranscriptByMedia(ctx, pool, mediaID)
	require.NoError(t, err)
	assert.Equal(t, "Merhaba dünya.", tr.Text)
	assert.Equal(t, "tr", tr.Language)
	assert.Equal(t, "fake", tr.Provider)
	assert.Equal(t, []string{"A"}, tr.Speakers)
	require.Len(t, tr.Segments, 1)
	assert.Equal(t, "Merhaba dünya.", tr.Segments[0].Text)
	assert.Equal(t, int64(1200), tr.Segments[0].EndMS)

	m, err := repo.GetMedia(ctx, pool, mediaID)
	require.NoError(t, err)
	assert.Equal(t, "private", m.Visibility)
}
