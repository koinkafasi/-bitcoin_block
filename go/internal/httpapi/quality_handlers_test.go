package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
)

func detailBlocks(paragraphs ...string) []byte {
	b := []byte("[")
	for i, p := range paragraphs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, []byte(`{"type":"p","text":"`+p+`"}`)...)
	}
	b = append(b, ']')
	return b
}

func TestQualityTextIncludesTitleSpotAndBody(t *testing.T) {
	c := repo.Content{
		Title:  "Başlık",
		Spot:   "Spot metni",
		Detail: detailBlocks("Birinci paragraf.", "İkinci paragraf."),
		Locale: "tr",
	}
	text := qualityText(c)
	assert.Contains(t, text, "Başlık")
	assert.Contains(t, text, "Spot metni")
	assert.Contains(t, text, "Birinci paragraf.")
	assert.Contains(t, text, "İkinci paragraf.")
}

func TestQualityViewMapsBlockersAndWarnings(t *testing.T) {
	rep := quality.Report{
		Score: 42,
		Grade: "D",
		Blockers: []quality.Finding{
			{Code: "too_short", Title: "Metin çok kısa", Detail: "80 sözcük var."},
		},
		Warnings: []quality.Finding{
			{Code: "cliche", Title: "Kalıplaşmış ifadeler", Detail: "2 kalıp bulundu."},
		},
		Imla: []quality.ImlaBulgu{
			{Code: "imla_birlesik", Message: "“herşey” bitişik yazılmış; doğrusu “her şey”."},
		},
		Metrics: quality.Metrics{
			Words: 80, Sentences: 6, AvgSentenceWords: 13.3,
			SentenceStdDev: 5.2, MATTR: 0.812, Cliques: 2, RepeatedPhrases: 1,
		},
	}

	v := qualityView(42, rep)
	assert.True(t, v.Checked)
	assert.Equal(t, 42, v.Score)
	assert.Equal(t, "D", v.Grade)
	assert.Equal(t, "/admin/icerikler/42/kalite", v.RefreshURL)
	assert.Equal(t, "80", v.Words)
	assert.Equal(t, "6", v.Sentences)
	assert.Equal(t, "13.3", v.AvgSentence)
	assert.Equal(t, "5.2", v.StdDev)
	assert.Equal(t, "0.81", v.MATTR)
	assert.Equal(t, "2", v.Cliches)
	assert.Equal(t, "1", v.Repeats)

	require.Len(t, v.Blockers, 1)
	assert.Equal(t, "Metin çok kısa", v.Blockers[0].Title)
	assert.Equal(t, "80 sözcük var.", v.Blockers[0].Detail)

	require.Len(t, v.Warnings, 1)
	assert.Equal(t, "Kalıplaşmış ifadeler", v.Warnings[0].Title)

	require.Len(t, v.Imla, 1)
	assert.Equal(t, "imla_birlesik", v.Imla[0].Title)
	assert.Equal(t, "“herşey” bitişik yazılmış; doğrusu “her şey”.", v.Imla[0].Detail)
}

func TestQualityViewEmptyContentIsBlocker(t *testing.T) {
	rep := quality.Analyze("   ", "tr")
	assert.False(t, rep.OK())

	v := qualityView(7, rep)
	assert.Equal(t, "E", v.Grade)
	assert.Equal(t, "0", v.Words)
	require.NotEmpty(t, v.Blockers)
	assert.Equal(t, "Metin boş", v.Blockers[0].Title)
}
