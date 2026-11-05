package transcribe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// attachWords: kelimeler orta noktalarına göre doğru segmente yazılmalı.
func TestAttachWordsOrtaNoktayaGoreDagitir(t *testing.T) {
	segs := []Segment{
		{StartMS: 0, EndMS: 1000, Speaker: "A", Text: "Merhaba dünya"},
		{StartMS: 1000, EndMS: 2000, Speaker: "B", Text: "Nasılsın"},
	}
	words := []wordItem{
		{Text: "Merhaba", Start: 0, End: 400, Confidence: 0.9},
		{Text: "dünya", Start: 400, End: 900, Confidence: 0.8},
		{Text: "Nasılsın", Start: 1100, End: 1800, Confidence: 0.95},
	}

	attachWords(segs, words)

	require.Len(t, segs[0].Words, 2)
	assert.Equal(t, "Merhaba", segs[0].Words[0].Text)
	assert.Equal(t, int64(0), segs[0].Words[0].StartMS)
	assert.Equal(t, int64(400), segs[0].Words[0].EndMS)
	assert.InDelta(t, 0.9, segs[0].Words[0].Confidence, 0.001)
	assert.Equal(t, "dünya", segs[0].Words[1].Text)

	require.Len(t, segs[1].Words, 1)
	assert.Equal(t, "Nasılsın", segs[1].Words[0].Text)
}

// Segment aralıklarının dışındaki kelime hiçbir segmente yazılmamalı:
// uydurma zaman damgası editörü yanıltır.
func TestAttachWordsBosluktakiKelimeyiAtlar(t *testing.T) {
	segs := []Segment{
		{StartMS: 0, EndMS: 500},
		{StartMS: 2000, EndMS: 2500},
	}
	words := []wordItem{
		{Text: "bir", Start: 100, End: 200},
		{Text: "gurultu", Start: 900, End: 1100}, // iki segment arasındaki boşluk
		{Text: "iki", Start: 2100, End: 2200},
	}

	attachWords(segs, words)

	require.Len(t, segs[0].Words, 1)
	assert.Equal(t, "bir", segs[0].Words[0].Text)
	require.Len(t, segs[1].Words, 1)
	assert.Equal(t, "iki", segs[1].Words[0].Text)
}

func TestAttachWordsBosGirdiyeDayanir(t *testing.T) {
	assert.NotPanics(t, func() { attachWords(nil, nil) })

	segs := []Segment{{StartMS: 0, EndMS: 100, Text: "x"}}
	attachWords(segs, nil)
	assert.Empty(t, segs[0].Words)

	// Yalnız boşluktan oluşan kelime yazılmaz.
	attachWords(segs, []wordItem{{Text: "   ", Start: 10, End: 20}})
	assert.Empty(t, segs[0].Words)
}

// groupWordItems kelimeleri segmentin içinde taşımalı.
func TestGroupWordItemsKelimeleriTasir(t *testing.T) {
	words := []wordItem{
		{Text: "Yatırım", Start: 0, End: 400, Confidence: 0.98},
		{Text: "planı", Start: 400, End: 800, Confidence: 0.97},
		{Text: "hazır.", Start: 800, End: 1200, Confidence: 0.96},
	}

	segs := groupWordItems(words, 1200)

	require.Len(t, segs, 1, "cümle sonunda tek segment kapanır")
	assert.Equal(t, "Yatırım planı hazır.", segs[0].Text)
	require.Len(t, segs[0].Words, 3)
	assert.Equal(t, "hazır.", segs[0].Words[2].Text)
	assert.Equal(t, int64(1200), segs[0].Words[2].EndMS)
}

// Konuşmacı değişince segment kapanmalı — tek satırda iki kişi olmaz.
func TestGroupWordItemsKonusmaciDegisinceBoler(t *testing.T) {
	words := []wordItem{
		{Text: "Soruyu", Start: 0, End: 300, Speaker: "A"},
		{Text: "sordum", Start: 300, End: 600, Speaker: "A"},
		{Text: "Cevabım", Start: 700, End: 1000, Speaker: "B"},
		{Text: "şudur", Start: 1000, End: 1300, Speaker: "B"},
	}

	segs := groupWordItems(words, 1300)

	require.Len(t, segs, 2)
	assert.Equal(t, "A", segs[0].Speaker)
	assert.Equal(t, "Soruyu sordum", segs[0].Text)
	assert.Len(t, segs[0].Words, 2)
	assert.Equal(t, "B", segs[1].Speaker)
	assert.Equal(t, "Cevabım şudur", segs[1].Text)
	assert.Len(t, segs[1].Words, 2)
	assert.Equal(t, int64(700), segs[1].StartMS)
}

// 40 kelime sınırında bölme davranışı korunmalı ve kelimeler kaybolmamalı.
func TestGroupWordItemsKirkKelimedeBoler(t *testing.T) {
	var words []wordItem
	for i := 0; i < 85; i++ {
		words = append(words, wordItem{
			Text:  "kelime",
			Start: int64(i) * 100,
			End:   int64(i)*100 + 90,
		})
	}

	segs := groupWordItems(words, 8500)

	require.Len(t, segs, 3, "85 kelime → 40 + 40 + 5")
	assert.Len(t, segs[0].Words, 40)
	assert.Len(t, segs[1].Words, 40)
	assert.Len(t, segs[2].Words, 5)

	toplam := 0
	for _, s := range segs {
		toplam += len(s.Words)
	}
	assert.Equal(t, 85, toplam, "hiçbir kelime kaybolmamalı")
}

// AssemblyAI utterance döndürdüğünde kelimeler utterance'lara bağlanmalı.
func TestConvertTranscriptUtteranceKelimeleriBaglar(t *testing.T) {
	var tr assemblyTranscript
	tr.Text = "Merhaba dünya. Nasılsın?"
	tr.AudioDuration = 2
	tr.Utterances = append(tr.Utterances, assemblyUtterance{Start: 0, End: 1000, Text: "Merhaba dünya.", Speaker: "A", Confidence: 0.9})
	tr.Utterances = append(tr.Utterances, assemblyUtterance{Start: 1000, End: 2000, Text: "Nasılsın?", Speaker: "B", Confidence: 0.92})
	tr.Words = []assemblyWord{
		{Text: "Merhaba", Start: 0, End: 400, Speaker: "A", Confidence: 0.9},
		{Text: "dünya.", Start: 400, End: 900, Speaker: "A", Confidence: 0.88},
		{Text: "Nasılsın?", Start: 1100, End: 1900, Speaker: "B", Confidence: 0.92},
	}

	res := convertTranscript(tr, "job-1", "tr", "assemblyai", "best")

	require.Len(t, res.Segments, 2)
	require.Len(t, res.Segments[0].Words, 2)
	assert.Equal(t, "Merhaba", res.Segments[0].Words[0].Text)
	assert.Equal(t, "dünya.", res.Segments[0].Words[1].Text)
	require.Len(t, res.Segments[1].Words, 1)
	assert.Equal(t, "Nasılsın?", res.Segments[1].Words[0].Text)
	assert.Equal(t, []string{"A", "B"}, res.Speakers)
}

// Sağlayıcı kelime döndürmezse segmentler kelimesiz kalmalı (hata değil).
func TestConvertTranscriptKelimeYokItirazEtmez(t *testing.T) {
	var tr assemblyTranscript
	tr.Text = "Tek parça metin."
	tr.AudioDuration = 1
	tr.Utterances = append(tr.Utterances, assemblyUtterance{Start: 0, End: 1000, Text: "Tek parça metin.", Speaker: "A"})

	res := convertTranscript(tr, "job-2", "tr", "assemblyai", "best")

	require.Len(t, res.Segments, 1)
	assert.Empty(t, res.Segments[0].Words)
	assert.Equal(t, "Tek parça metin.", res.Segments[0].Text)
}
