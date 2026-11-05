package aieditor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAudioDraft(t *testing.T) {
	act, ok := ActionByID("audio_draft")
	require.True(t, ok)
	assert.True(t, act.Hidden, "ses taslağı seçili metin menüsünde görünmemeli")

	res, err := normalizeResult(act, map[string]any{
		"title":            "Röportaj: Yeni yatırım planı",
		"spot":             "Bakan, yatırımın 2027'de başlayacağını söyledi.",
		"body":             "Bakan görüşmede yatırım planını anlattı.",
		"subheadings":      []any{"Planın ayrıntıları"},
		"quotes":           []any{"\"Yatırım 2027'de başlayacak.\""},
		"claims_to_verify": []any{"Yatırım tutarı"},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, "Röportaj: Yeni yatırım planı", res.Title)
	assert.Equal(t, "Röportaj: Yeni yatırım planı", res.Fields["title"])
	assert.Equal(t, "Bakan, yatırımın 2027'de başlayacağını söyledi.", res.Fields["spot"])
	assert.Equal(t, "Bakan görüşmede yatırım planını anlattı.", res.Fields["body"])

	titles := make([]string, 0, len(res.Sections))
	for _, sec := range res.Sections {
		titles = append(titles, sec.Title)
	}
	assert.Contains(t, titles, "Spot")
	assert.Contains(t, titles, "Gövde")
	assert.Contains(t, titles, "Alt başlıklar")
	assert.Contains(t, titles, "Alıntılar")
	assert.Contains(t, titles, "Doğrulanması gereken iddialar")
}

func TestNormalizeAudioDraftRequiresTitleAndBody(t *testing.T) {
	act, ok := ActionByID("audio_draft")
	require.True(t, ok)

	_, err := normalizeResult(act, map[string]any{"title": "Yalnızca başlık"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title ve body zorunlu")

	_, err = normalizeResult(act, map[string]any{"body": "Yalnızca gövde"}, nil)
	require.Error(t, err)
}

func TestSelectionLimitAudioDraft(t *testing.T) {
	audio, ok := ActionByID("audio_draft")
	require.True(t, ok)
	assert.Equal(t, 60_000, selectionLimit(audio))

	news, ok := ActionByID("news")
	require.True(t, ok)
	assert.Equal(t, 5_000, selectionLimit(news))
}

func TestSelectionExceedsLimit(t *testing.T) {
	audio, ok := ActionByID("audio_draft")
	require.True(t, ok)
	news, ok := ActionByID("news")
	require.True(t, ok)

	assert.False(t, selectionExceedsLimit(audio, strings.Repeat("a", 60_000)))
	assert.True(t, selectionExceedsLimit(audio, strings.Repeat("a", 60_001)))
	assert.False(t, selectionExceedsLimit(news, "kısa metin"))
	assert.True(t, selectionExceedsLimit(news, strings.Repeat("a", 5_001)))
	// Baştaki/sondaki boşluk sınıra dahil edilmez.
	assert.False(t, selectionExceedsLimit(news, "  "+strings.Repeat("a", 5_000)+"  "))
}

func TestAudioDraftPromptMarksTranscriptUntrusted(t *testing.T) {
	prompt := audioDraftPrompt(articleContext{
		ContentID: 12,
		Title:     "Mevcut başlık",
		Extra: map[string]any{
			"transcript_id": 99,
			"speakers":      []string{"A", "B"},
		},
	}, "Konuşmacı A: Yatırım 2027'de başlayacak.", "Mevcut gövde", "Daha kısa yaz")

	assert.Contains(t, prompt, "GÜVENİLMEZ VERİ")
	assert.Contains(t, prompt, "Konuşmacı A: Yatırım 2027'de başlayacak.")
	assert.Contains(t, prompt, "Mevcut gövde")
	assert.Contains(t, prompt, "Daha kısa yaz")
	assert.Contains(t, prompt, `"transcript_id": 99`)
	assert.Contains(t, prompt, `"speakers"`)
	assert.True(t, strings.HasSuffix(strings.TrimSpace(prompt), "Yalnızca geçerli JSON döndür."))
}
