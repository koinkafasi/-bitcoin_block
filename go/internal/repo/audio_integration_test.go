//go:build integration

package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestPrivateAudioLifecycle(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var uid int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&uid))

	var contentID int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM contents ORDER BY id LIMIT 1`).Scan(&contentID))

	size := int64(1024)
	mediaID, err := repo.CreateMedia(ctx, pool, &repo.Media{
		SiteID: 1, UserID: uid, Kind: "audio", Filename: "repo-int-ses.mp3",
		Path: "audio/202610/repo-int-ses.mp3", Mime: "audio/mpeg", SizeBytes: &size,
		Title: "Repo entegrasyon ses", Category: "ses", Source: "upload", Visibility: "private",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM media_files WHERE id=$1`, mediaID)
	})

	// Private medya public listede görünmemeli.
	pub, err := repo.ListMedia(ctx, pool, 1, uid, 500)
	require.NoError(t, err)
	for _, m := range pub {
		assert.NotEqual(t, mediaID, m.ID, "private ses public medya listesinde olmamalı")
	}

	jobID, err := repo.CreateTranscriptionJob(ctx, pool, &repo.TranscriptionJob{
		SiteID: 1, MediaID: mediaID, ContentID: &contentID, UserID: &uid,
		Provider: "fake", Language: "tr", SpeakersExpected: 2,
	})
	require.NoError(t, err)
	// Aynı medya için ikinci aktif iş DB kısıtıyla reddedilmeli.
	_, err = repo.CreateTranscriptionJob(ctx, pool, &repo.TranscriptionJob{
		SiteID: 1, MediaID: mediaID, ContentID: &contentID, Provider: "fake",
	})
	require.Error(t, err, "aynı medya için ikinci aktif iş engellenmeli")

	jid := jobID
	_, err = repo.UpsertAudioTranscript(ctx, pool, &repo.AudioTranscript{
		SiteID: 1, MediaID: mediaID, ContentID: &contentID, JobID: &jid,
		Language: "tr", Provider: "fake", Model: "m", DurationMS: 5000,
		Text: "Merhaba dünya.", Segments: []repo.AudioSegment{
			{StartMS: 0, EndMS: 5000, Speaker: "A", Text: "Merhaba dünya."},
		},
		Speakers: []string{"A"},
	})
	require.NoError(t, err)

	list, err := repo.ListAudioForContent(ctx, pool, 1, contentID)
	require.NoError(t, err)
	var found bool
	for _, a := range list {
		if a.Media.ID == mediaID {
			found = true
			require.NotNil(t, a.Transcript)
			assert.Equal(t, "Merhaba dünya.", a.Transcript.Text)
		}
	}
	assert.True(t, found, "ses kaydı içerik panelinde listelenmeli")

	// DeleteMedia transkript ve işi CASCADE ile temizlemeli.
	require.NoError(t, repo.DeleteMedia(ctx, pool, mediaID))

	var jobs, trs, media int
	require.NoError(t, pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM transcription_jobs WHERE media_id=$1),
		(SELECT count(*) FROM audio_transcripts WHERE media_id=$1),
		(SELECT count(*) FROM media_files WHERE id=$1)`, mediaID).Scan(&jobs, &trs, &media))
	assert.Zero(t, jobs, "iş kaydı silinmeli")
	assert.Zero(t, trs, "transkript silinmeli")
	assert.Zero(t, media, "medya silinmeli")
}
