package repo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kelime zamanları segments JSONB'sinin içinde saklandığı için serileştirme
// biçimi sözleşmedir: alan adları değişirse eski kayıtlar okunamaz.
func TestAudioSegmentKelimeYuvarlakTur(t *testing.T) {
	in := []AudioSegment{{
		StartMS: 0, EndMS: 900, Speaker: "A", Text: "Merhaba dünya", Confidence: 0.91,
		Words: []AudioWord{
			{StartMS: 0, EndMS: 400, Text: "Merhaba", Confidence: 0.93},
			{StartMS: 400, EndMS: 900, Text: "dünya", Confidence: 0.89},
		},
	}}

	raw, err := json.Marshal(in)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"words"`)
	assert.Contains(t, string(raw), `"start_ms":400`)

	var out []AudioSegment
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Len(t, out, 1)
	require.Len(t, out[0].Words, 2)
	assert.Equal(t, "Merhaba", out[0].Words[0].Text)
	assert.Equal(t, int64(900), out[0].Words[1].EndMS)
	assert.InDelta(t, 0.89, out[0].Words[1].Confidence, 0.001)
}

// Kelimesiz segment "words" anahtarı basmamalı: Faz 125 öncesi kayıtların
// boyutu büyümesin ve eski satırlar olduğu gibi okunabilsin.
func TestAudioSegmentKelimesizOmitempty(t *testing.T) {
	raw, err := json.Marshal([]AudioSegment{{StartMS: 0, EndMS: 500, Text: "tek parça"}})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"words"`)
}

// Faz 126 öncesi yazılmış segment JSON'u (words alanı yok) sorunsuz okunmalı.
func TestAudioSegmentEskiKayitGeriyeUyumlu(t *testing.T) {
	eski := `[{"start_ms":0,"end_ms":1200,"speaker":"A","text":"Eski kayıt","confidence":0.8}]`

	var out []AudioSegment
	require.NoError(t, json.Unmarshal([]byte(eski), &out))
	require.Len(t, out, 1)
	assert.Equal(t, "Eski kayıt", out[0].Text)
	assert.Equal(t, "A", out[0].Speaker)
	assert.Nil(t, out[0].Words, "kelime yoksa nil kalır, arayüz segment seviyesine düşer")
}
