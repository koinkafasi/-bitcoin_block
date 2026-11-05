package httpapi

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// audioWords: boş metinli kelimeler atılmalı, hiç kelime kalmazsa nil
// dönmeli (şablon o zaman segmenti düz metin basar).
func TestAudioWordsEsleme(t *testing.T) {
	assert.Nil(t, audioWords(nil))
	assert.Nil(t, audioWords([]repo.AudioWord{}))
	assert.Nil(t, audioWords([]repo.AudioWord{{StartMS: 0, EndMS: 10, Text: "   "}}),
		"yalnız boşluktan oluşan kelime kalınca nil dönmeli")

	got := audioWords([]repo.AudioWord{
		{StartMS: 0, EndMS: 400, Text: " Merhaba "},
		{StartMS: 400, EndMS: 900, Text: "dünya"},
		{StartMS: 900, EndMS: 950, Text: ""},
	})
	require.Len(t, got, 2)
	assert.Equal(t, "Merhaba", got[0].Text, "kenar boşlukları kırpılmalı")
	assert.Equal(t, int64(0), got[0].StartMS)
	assert.Equal(t, int64(400), got[0].EndMS)
	assert.Equal(t, "dünya", got[1].Text)
	assert.Equal(t, int64(900), got[1].EndMS)
}

// Kelime zamanları uçtan uca şablona taşınmalı: repo → view → HTML.
func TestIcerikAudioViewsKelimeleriTasir(t *testing.T) {
	s := &Server{}
	got := s.icerikAudioViews(42, []repo.ContentAudio{{
		Media: repo.Media{ID: 7, Filename: "roportaj.mp3",
			CreatedAt: time.Date(2026, 10, 28, 4, 0, 0, 0, time.UTC)},
		Job: repo.TranscriptionJob{ID: 3, Status: "completed", Progress: 100},
		Transcript: &repo.AudioTranscript{
			ID: 9, Text: "Merhaba dünya.", DurationMS: 4_000,
			Segments: []repo.AudioSegment{{
				StartMS: 1_000, EndMS: 4_000, Speaker: "A", Text: "Merhaba dünya.",
				Words: []repo.AudioWord{
					{StartMS: 1_000, EndMS: 1_400, Text: "Merhaba"},
					{StartMS: 1_400, EndMS: 1_900, Text: "dünya."},
				},
			}},
		},
	}})
	require.Len(t, got, 1)
	require.Len(t, got[0].Segments, 1)
	require.Len(t, got[0].Segments[0].Words, 2)
	assert.Equal(t, "Merhaba", got[0].Segments[0].Words[0].Text)
	assert.Equal(t, int64(1_400), got[0].Segments[0].Words[0].EndMS)
}

// renderSesPanelBody ses panelini templ ile işler.
func renderSesPanelBody(t *testing.T, v templkit.IcerikFormView) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, templkit.SesPanelBody(v).Render(context.Background(), &buf))
	return buf.String()
}

func audioViewWithWords(words []templkit.IcerikAudioWord) templkit.IcerikFormView {
	return templkit.IcerikFormView{
		SesUploadURL: "/admin/icerikler/42/ses",
		Audios: []templkit.IcerikAudio{{
			MediaID: "7", Filename: "roportaj.mp3", StreamURL: "/admin/ses/7/stream",
			Status: "completed", StatusLabel: "Tamamlandı", Progress: 100,
			Duration: "0:04", TranscriptText: "Merhaba dünya.",
			Segments: []templkit.IcerikAudioSegment{{
				StartMS: 1_000, StartLabel: "0:01", Speaker: "A",
				Text: "Merhaba dünya.", Words: words,
			}},
		}},
	}
}

// Kelime varsa her kelime kendi data-ms/data-end-ms'i ile basılmalı ve
// aralarında gerçek boşluk olmalı — kopyalanan metin birleşik çıkmasın.
func TestSesPanelBodyKelimeleriBasar(t *testing.T) {
	out := renderSesPanelBody(t, audioViewWithWords([]templkit.IcerikAudioWord{
		{StartMS: 1_000, EndMS: 1_400, Text: "Merhaba"},
		{StartMS: 1_400, EndMS: 1_900, Text: "dünya."},
	}))

	assert.Contains(t, out, `data-ses-word`, "kelime işaretleyicisi basılmalı")
	assert.Contains(t, out, `<span class="ses-w" data-ses-word data-ms="1000" data-end-ms="1400">Merhaba</span>`)
	assert.Contains(t, out, `data-ms="1400" data-end-ms="1900">dünya.</span>`)
	assert.Contains(t, out, `>Merhaba</span> <span class="ses-w"`,
		"iki kelime arasında tek gerçek boşluk olmalı")

	// Oynatıcı Media Session için dosya adını başlık olarak kullanır.
	assert.Contains(t, out, `data-title="roportaj.mp3"`)
	// Segment satırı eski davranışını korur: başına sarma.
	assert.Contains(t, out, `data-audio-seek="7"`)
	assert.Contains(t, out, `data-start-ms="1000"`)
}

// Kelime yoksa segment düz metin basılmalı: sağlayıcı kelime döndürmeyen
// eski kayıtlarda oynatıcı ve transkript eskisi gibi çalışır.
func TestSesPanelBodyKelimeYokDuzMetin(t *testing.T) {
	out := renderSesPanelBody(t, audioViewWithWords(nil))

	assert.NotContains(t, out, "data-ses-word")
	// "ses-w" dizesi ses-wave sınıflarının içinde de geçer; kelime sınıfını
	// kapanış tırnağıyla aramak gerekir.
	assert.NotContains(t, out, `class="ses-w"`)
	assert.Contains(t, out, `<span class="ses-seg-text">Merhaba dünya.</span>`)
}

// Kelime metni HTML olarak kaçırılmalı: transkript sağlayıcıdan gelen
// denetlenmemiş metindir.
func TestSesPanelBodyKelimeMetniKacirilir(t *testing.T) {
	out := renderSesPanelBody(t, audioViewWithWords([]templkit.IcerikAudioWord{
		{StartMS: 0, EndMS: 500, Text: `<script>alert(1)</script>`},
	}))

	assert.NotContains(t, out, "<script>alert(1)</script>")
	assert.True(t, strings.Contains(out, "&lt;script&gt;"),
		"kelime metni kaçırılmış olmalı")
}
