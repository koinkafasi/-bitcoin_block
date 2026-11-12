package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
)

func TestHiddenAIActionsNotTriggerableFromSelectionForm(t *testing.T) {
	audio, ok := aieditor.ActionByID("audio_draft")
	require.True(t, ok, "audio_draft eylemi tanımlı olmalı")
	assert.True(t, audio.Hidden, "audio_draft seçili metin menüsünde gizli olmalı")

	news, ok := aieditor.ActionByID("news")
	require.True(t, ok, "news eylemi tanımlı olmalı")
	assert.False(t, news.Hidden, "news gizli olmamalı")
}

func TestAudioUploadAllowed(t *testing.T) {
	allowed := []string{
		"roportaj.mp3", "kayit.M4A", "ses.aac", "toplanti.wav", "not.ogg",
		"konusma.oga", "kayit.opus", "tarayici.weba", "ekran.webm",
		"video.mp4", "klip.mov", "telefon.m4v",
	}
	for _, name := range allowed {
		assert.Truef(t, audioUploadAllowed(name), "%s kabul edilmeliydi", name)
	}

	rejected := []string{
		"zararli.exe", "belge.pdf", "resim.png", "arsiv.zip", "ses", "ses.mp3.exe",
	}
	for _, name := range rejected {
		assert.Falsef(t, audioUploadAllowed(name), "%s reddedilmeliydi", name)
	}
}

func TestAudioMime(t *testing.T) {
	cases := map[string]string{
		"a.mp3":        "audio/mpeg",
		"a.m4a":        "audio/mp4",
		"a.aac":        "audio/aac",
		"a.wav":        "audio/wav",
		"a.ogg":        "audio/ogg",
		"a.opus":       "audio/ogg",
		"a.weba":       "audio/webm",
		"a.webm":       "audio/webm",
		"a.flac":       "audio/flac",
		"a.mp4":        "video/mp4",
		"a.mov":        "video/quicktime",
		"a.bilinmeyen": "application/octet-stream",
	}
	for name, want := range cases {
		assert.Equalf(t, want, audioMime(name), "%s için MIME", name)
	}
}

func TestAudioTimeLabel(t *testing.T) {
	cases := map[int64]string{
		0:         "",
		-500:      "",
		1000:      "0:01",
		65000:     "1:05",
		3600000:   "1:00:00",
		3723000:   "1:02:03",
		59_000:    "0:59",
		3_599_000: "59:59",
	}
	for ms, want := range cases {
		assert.Equalf(t, want, audioTimeLabel(ms), "%d ms", ms)
	}
}

func TestParseSpeakerCount(t *testing.T) {
	cases := map[string]int{
		"":    0,
		"0":   0,
		"1":   1,
		"2":   2,
		"10":  10,
		"11":  10,
		"99":  10,
		"-1":  0,
		"abc": 0,
		" 3 ": 3,
	}
	for in, want := range cases {
		assert.Equalf(t, want, parseSpeakerCount(in), "girdi %q", in)
	}
}

func TestReadPartBoolAndString(t *testing.T) {
	for _, in := range []string{"1", "on", "true", "evet", "YES", " On "} {
		assert.Truef(t, readPartBool(strings.NewReader(in)), "%q doğru olmalıydı", in)
	}
	for _, in := range []string{"", "0", "off", "false", "hayir", "belki"} {
		assert.Falsef(t, readPartBool(strings.NewReader(in)), "%q yanlış olmalıydı", in)
	}
	assert.Equal(t, "tr", readPartString(strings.NewReader("  tr \n")))
}

func TestAudioTranscriptForDraft(t *testing.T) {
	tr := &repo.AudioTranscript{
		Text: "Merhaba dünya. Nasılsın?",
		Segments: []repo.AudioSegment{
			{StartMS: 0, EndMS: 4000, Speaker: "A", Text: " Merhaba dünya. "},
			{StartMS: 4000, EndMS: 9000, Speaker: "B", Text: "Nasılsın?"},
		},
	}
	assert.Equal(t, "Konuşmacı A: Merhaba dünya.\nKonuşmacı B: Nasılsın?", audioTranscriptForDraft(tr))

	// Segment yoksa düz transkript metni kullanılır.
	assert.Equal(t, "Merhaba dünya.", audioTranscriptForDraft(&repo.AudioTranscript{Text: "  Merhaba dünya.  "}))
	assert.Equal(t, "", audioTranscriptForDraft(nil))
}

func TestAudioStatusTR(t *testing.T) {
	cases := map[string]string{
		"queued":     "Sırada",
		"running":    "Metne çevriliyor",
		"completed":  "Tamamlandı",
		"failed":     "Başarısız",
		"cancelled":  "İptal edildi",
		"bilinmeyen": "bilinmeyen",
	}
	for in, want := range cases {
		assert.Equal(t, want, audioStatusTR(in))
	}
}

func TestHandleMediaRejectsPrivateAudioPrefix(t *testing.T) {
	s := &Server{}
	for _, path := range []string{
		"/media/audio/202601/kayit.mp3",
		"/media/audio/202601/kayit.mp3?x=1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		s.handleMedia(rr, req)
		assert.Equalf(t, http.StatusNotFound, rr.Code, "%s public /media/ üzerinden servis edilmemeli", path)
	}
}

func TestIcerikAudioViewsCompleted(t *testing.T) {
	size := int64(12 * 1024 * 1024)
	s := &Server{}
	got := s.icerikAudioViews(42, []repo.ContentAudio{{
		Media: repo.Media{
			ID: 7, Filename: "roportaj.mp3", SizeBytes: &size,
			CreatedAt: time.Date(2026, 10, 27, 9, 30, 0, 0, time.UTC),
		},
		Job: repo.TranscriptionJob{ID: 3, Status: "completed", Progress: 100},
		Transcript: &repo.AudioTranscript{
			ID: 9, Text: "Merhaba dünya.", DurationMS: 65_000,
			Speakers: []string{"A", "B"},
			Segments: []repo.AudioSegment{
				{StartMS: 1_000, EndMS: 4_000, Speaker: "A", Text: " Merhaba dünya. "},
				{StartMS: 4_000, EndMS: 9_000, Speaker: "B", Text: "Merhaba."},
			},
		},
	}})
	require.Len(t, got, 1)
	a := got[0]
	assert.Equal(t, "7", a.MediaID)
	assert.Equal(t, "roportaj.mp3", a.Filename)
	assert.Equal(t, "/admin/ses/7/stream", a.StreamURL)
	assert.Equal(t, "Tamamlandı", a.StatusLabel)
	assert.Equal(t, 100, a.Progress)
	assert.False(t, a.IsActive)
	assert.True(t, a.CanDraft)
	assert.False(t, a.CanRetry)
	assert.Equal(t, "1:05", a.Duration)
	assert.Equal(t, "A, B", a.Speakers)
	assert.Equal(t, "Merhaba dünya.", a.TranscriptText)
	assert.Equal(t, "/admin/icerikler/42/ses/7/taslak", a.DraftURL)
	require.Len(t, a.Segments, 2)
	assert.Equal(t, "0:01", a.Segments[0].StartLabel)
	assert.Equal(t, "A", a.Segments[0].Speaker)
	assert.Equal(t, "Merhaba dünya.", a.Segments[0].Text)
	assert.Equal(t, "0:04", a.Segments[1].StartLabel)
	assert.Equal(t, "B", a.Segments[1].Speaker)
}

func TestIcerikAudioViewsFailedAndRunning(t *testing.T) {
	s := &Server{}
	got := s.icerikAudioViews(42, []repo.ContentAudio{
		{
			Media: repo.Media{ID: 8, Filename: "bozuk.m4a"},
			Job:   repo.TranscriptionJob{Status: "failed", Error: "sağlayıcı hatası"},
		},
		{
			Media: repo.Media{ID: 9, Filename: "uzun.mp3"},
			Job:   repo.TranscriptionJob{Status: "running", Progress: 55},
		},
	})
	require.Len(t, got, 2)
	assert.True(t, got[0].CanRetry)
	assert.False(t, got[0].CanDraft)
	assert.Equal(t, "Başarısız", got[0].StatusLabel)
	assert.Equal(t, "sağlayıcı hatası", got[0].Error)
	assert.Equal(t, "/admin/icerikler/42/ses/8/yeniden", got[0].RetryURL)
	assert.Equal(t, "/admin/icerikler/42/ses/8/sil", got[0].DeleteURL)

	assert.True(t, got[1].IsActive)
	assert.False(t, got[1].CanRetry)
	assert.False(t, got[1].CanDraft)
	assert.Equal(t, 55, got[1].Progress)
	assert.Equal(t, "Metne çevriliyor", got[1].StatusLabel)
}
