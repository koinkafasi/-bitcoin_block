//go:build integration

package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// TestSearchAudioTranscripts gerçek şema + tsvector trigger üzerinde arama
// yapar: kayıt eklendiğinde `search` otomatik dolar, arama bulur, alakasız
// sözcük bulmaz ve ILIKE yedeği de çalışır.
func TestSearchAudioTranscripts(t *testing.T) {
	pool := intPool(t)
	ctx := context.Background()

	var uid int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&uid))

	var contentID int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, spot, detail, slug, author_id, site_id, locale)
		VALUES ('haber','taslak','Transkript arama testi','Spot',
		        '[{"type":"p","text":"govde"}]'::jsonb, 'transkript-arama-int', $1, 1, 'tr')
		RETURNING id`, uid).Scan(&contentID))

	size := int64(100)
	mediaID, err := repo.CreateMedia(ctx, pool, &repo.Media{
		SiteID: 1, UserID: uid, Kind: "audio", Filename: "arama-int.mp3",
		Path: "audio/202610/arama-int.mp3", Mime: "audio/mpeg", SizeBytes: &size,
		Title: "Arama testi", Category: "ses", Source: "upload", Visibility: "private",
	})
	require.NoError(t, err)

	jobID, err := repo.CreateTranscriptionJob(ctx, pool, &repo.TranscriptionJob{
		SiteID: 1, MediaID: mediaID, ContentID: &contentID, UserID: &uid, Provider: "fake",
	})
	require.NoError(t, err)
	jid := jobID

	unique := "zeytinyağı"
	_, err = repo.UpsertAudioTranscript(ctx, pool, &repo.AudioTranscript{
		SiteID: 1, MediaID: mediaID, ContentID: &contentID, JobID: &jid,
		Language: "tr", Provider: "fake", DurationMS: 65_000,
		Text: "Konuşmacı A: Bu yıl " + unique + " üretimi rekor kırdı ve fiyatlar geriledi.",
		Segments: []repo.AudioSegment{
			{StartMS: 0, EndMS: 5000, Speaker: "A", Text: "Bu yıl " + unique + " üretimi rekor kırdı."},
		},
		Speakers: []string{"A"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audio_transcripts WHERE media_id=$1`, mediaID)
		_, _ = pool.Exec(ctx, `DELETE FROM transcription_jobs WHERE media_id=$1`, mediaID)
		_, _ = pool.Exec(ctx, `DELETE FROM media_files WHERE id=$1`, mediaID)
		_, _ = pool.Exec(ctx, `DELETE FROM contents WHERE id=$1`, contentID)
	})

	// Tam metin araması bulmalı.
	hits, err := repo.SearchAudioTranscripts(ctx, pool, 1, unique, 30)
	require.NoError(t, err)
	var found *repo.TranskriptAramaSonucu
	for i := range hits {
		if hits[i].MediaID == mediaID {
			found = &hits[i]
			break
		}
	}
	require.NotNil(t, found, "kayıt aramada bulunmalı")
	assert.Equal(t, contentID, *found.ContentID)
	assert.NotEmpty(t, found.Snippet, "eşleşen bölüm döndürülmeli")
	assert.Contains(t, found.Snippet, unique)
	assert.Equal(t, int64(65_000), found.DurationMS)

	// Alakasız sözcük bulmamalı.
	none, err := repo.SearchAudioTranscripts(ctx, pool, 1, "kuşkonmazzz", 30)
	require.NoError(t, err)
	for _, h := range none {
		assert.NotEqual(t, mediaID, h.MediaID, "alakasız sorgu bu kaydı getirmemeli")
	}

	// Çok kısa sorgu boş döner.
	short, err := repo.SearchAudioTranscripts(ctx, pool, 1, "a", 30)
	require.NoError(t, err)
	assert.Empty(t, short)
}
