package transcribe

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubTimeBicimi(t *testing.T) {
	assert.Equal(t, "00:00:00,000", srtTime(0))
	assert.Equal(t, "00:00:01,500", srtTime(1_500))
	assert.Equal(t, "00:01:05,250", srtTime(65_250))
	assert.Equal(t, "01:02:03,004", srtTime(3_723_004))
	// WebVTT ondalığı virgül değil nokta ile ayırır.
	assert.Equal(t, "00:01:05.250", vttTime(65_250))
	// Negatif zaman sıfıra çekilir (bozuk sağlayıcı verisi).
	assert.Equal(t, "00:00:00,000", srtTime(-5))
}

// Kısa segment tek kare olmalı, bölünmemeli.
func TestBuildCuesKisaSegmentTekKare(t *testing.T) {
	cues := BuildCues([]Segment{
		{StartMS: 1_000, EndMS: 3_000, Speaker: "A", Text: "Merhaba dünya."},
	})
	require.Len(t, cues, 1)
	assert.Equal(t, int64(1_000), cues[0].StartMS)
	assert.Equal(t, int64(3_000), cues[0].EndMS)
	assert.Equal(t, "A", cues[0].Speaker)
	assert.Equal(t, "Merhaba dünya.", cues[0].Text)
}

// Uzun segment kelime zamanlarıyla bölünmeli ve kare sınırları gerçek
// kelime zamanlarına oturmalı (tahmin edilmemeli).
func TestBuildCuesKelimeZamaniylaBoler(t *testing.T) {
	var words []Word
	// 30 kelime × "kelimeler" (9 harf) → 84 karakter sınırını birkaç kez aşar.
	for i := 0; i < 30; i++ {
		words = append(words, Word{
			StartMS: int64(i) * 500,
			EndMS:   int64(i)*500 + 450,
			Text:    "kelimeler",
		})
	}
	seg := Segment{StartMS: 0, EndMS: 15_000, Speaker: "B", Text: "uzun", Words: words}

	cues := BuildCues([]Segment{seg})

	require.Greater(t, len(cues), 1, "uzun segment bölünmeli")
	for i, c := range cues {
		assert.LessOrEqual(t, runeLen(c.Text), subMaxChars,
			"kare %d sınırı aşmamalı: %q", i, c.Text)
		assert.Greater(t, c.EndMS, c.StartMS, "kare %d sıfır süreli olmamalı", i)
		assert.Equal(t, "B", c.Speaker)
		if i > 0 {
			assert.GreaterOrEqual(t, c.StartMS, cues[i-1].EndMS,
				"kareler çakışmamalı (%d)", i)
		}
	}
	// İlk kare ilk kelimenin gerçek başlangıcında olmalı.
	assert.Equal(t, int64(0), cues[0].StartMS)

	// Hiçbir kelime kaybolmamalı.
	var toplam int
	for _, c := range cues {
		toplam += len(strings.Fields(c.Text))
	}
	assert.Equal(t, 30, toplam)
}

// Kelime zamanı yoksa metin karakter oranıyla bölünmeli; metin kaybolmamalı.
func TestBuildCuesKelimeYokKarakterOraniyla(t *testing.T) {
	text := strings.TrimSpace(strings.Repeat("ekonomi haberleri ", 20))
	cues := BuildCues([]Segment{{StartMS: 0, EndMS: 40_000, Text: text}})

	require.Greater(t, len(cues), 1)
	var parts []string
	for i, c := range cues {
		assert.LessOrEqual(t, runeLen(c.Text), subMaxChars, "kare %d", i)
		assert.Greater(t, c.EndMS, c.StartMS)
		if i > 0 {
			assert.GreaterOrEqual(t, c.StartMS, cues[i-1].EndMS, "kare %d çakışmamalı", i)
		}
		parts = append(parts, c.Text)
	}
	assert.Equal(t, text, strings.Join(parts, " "), "metnin tamamı korunmalı")
	assert.Equal(t, int64(0), cues[0].StartMS)
}

// Bitişi eksik/bozuk segment kullanılabilir süreye çekilmeli.
func TestBuildCuesBozukBitisiDuzeltir(t *testing.T) {
	cues := BuildCues([]Segment{
		{StartMS: 5_000, EndMS: 0, Text: "Bitiş yok."},
		{StartMS: 20_000, EndMS: 19_000, Text: "Bitiş başlangıçtan önce."},
	})
	require.Len(t, cues, 2)
	for _, c := range cues {
		assert.Greater(t, c.EndMS, c.StartMS)
		assert.GreaterOrEqual(t, c.EndMS-c.StartMS, int64(subMinMS))
	}
}

// Boş ve yalnız boşluktan oluşan segmentler kare üretmemeli.
func TestBuildCuesBosSegmentleriAtar(t *testing.T) {
	assert.Empty(t, BuildCues(nil))
	assert.Empty(t, BuildCues([]Segment{{StartMS: 0, EndMS: 1_000, Text: "   "}}))
	assert.Len(t, BuildCues([]Segment{
		{StartMS: 0, EndMS: 1_000, Text: ""},
		{StartMS: 1_000, EndMS: 2_000, Text: "Tek kare."},
	}), 1)
}

// Çakışan segmentler normalize edilmeli: oynatıcıda kare kaybolmasın.
func TestNormalizeCuesCakismayiCozer(t *testing.T) {
	cues := normalizeCues([]Cue{
		{StartMS: 0, EndMS: 5_000, Text: "bir"},
		{StartMS: 3_000, EndMS: 6_000, Text: "iki"},
	})
	require.Len(t, cues, 2)
	assert.LessOrEqual(t, cues[0].EndMS, cues[1].StartMS)
	assert.Greater(t, cues[0].EndMS, cues[0].StartMS)
}

func TestSRTBicimi(t *testing.T) {
	out := SRT([]Cue{
		{StartMS: 0, EndMS: 2_000, Speaker: "A", Text: "Merhaba dünya."},
		{StartMS: 2_500, EndMS: 4_000, Text: "Konuşmacısız kare."},
	})

	assert.Equal(t, "1\n00:00:00,000 --> 00:00:02,000\nA: Merhaba dünya.\n\n"+
		"2\n00:00:02,500 --> 00:00:04,000\nKonuşmacısız kare.\n\n", out)
}

func TestWebVTTBicimi(t *testing.T) {
	out := WebVTT([]Cue{
		{StartMS: 0, EndMS: 2_000, Speaker: "A", Text: "Merhaba dünya."},
	})

	assert.True(t, strings.HasPrefix(out, "WEBVTT\n\n"), "WEBVTT başlığı şart")
	assert.Contains(t, out, "00:00:00.000 --> 00:00:02.000")
	assert.Contains(t, out, "<v A>Merhaba dünya.")
}

// WebVTT'de "<" kaçırılmazsa metin etiket gibi yorumlanır.
func TestWebVTTKacirir(t *testing.T) {
	out := WebVTT([]Cue{
		{StartMS: 0, EndMS: 1_000, Speaker: "<b>", Text: `5 < 7 & "ok" <i>eğik</i>`},
	})
	assert.NotContains(t, out, "<i>")
	assert.NotContains(t, out, "<b>")
	assert.Contains(t, out, "&lt;i&gt;")
	assert.Contains(t, out, "5 &lt; 7 &amp;")
	assert.Contains(t, out, "<v &lt;b&gt;>")
}

// Boş kare çıktıya girmemeli ve numaralandırma kaymamalı.
func TestSRTBosKareyiAtar(t *testing.T) {
	out := SRT([]Cue{
		{StartMS: 0, EndMS: 1_000, Text: "bir"},
		{StartMS: 1_000, EndMS: 2_000, Text: "   "},
		{StartMS: 2_000, EndMS: 3_000, Text: "iki"},
	})
	assert.Contains(t, out, "1\n")
	assert.Contains(t, out, "2\n00:00:02,000")
	assert.NotContains(t, out, "3\n")
}

// Satır sonu ve sekme gibi artıklar tek boşluğa inmeli: aksi hâlde SRT
// ayrıştırıcısı kareyi erken bitirmiş sayar.
func TestSRTSatirSonunuTemizler(t *testing.T) {
	out := SRT([]Cue{{StartMS: 0, EndMS: 1_000, Text: "iki\n\nsatır\tve\nboşluk"}})
	assert.Contains(t, out, "iki satır ve boşluk")
	// Zaman satırından sonra tek metin bloğu, sonra tek boş satır olmalı.
	assert.Equal(t, "1\n00:00:00,000 --> 00:00:01,000\niki satır ve boşluk\n\n", out)
}

func TestWrapLines(t *testing.T) {
	// 42 karaktere sığan metin tek satır kalır.
	assert.Equal(t, "kısa satır", wrapLines("kısa satır"))

	long := strings.TrimSpace(strings.Repeat("abcde ", 20)) // 119 karakter
	wrapped := wrapLines(long)
	for _, line := range strings.Split(wrapped, "\n") {
		assert.LessOrEqual(t, runeLen(line), subMaxLineChars, "satır: %q", line)
	}
	// Metin düşürülmemeli.
	assert.Equal(t, strings.Fields(long), strings.Fields(strings.ReplaceAll(wrapped, "\n", " ")))
	assert.Equal(t, "", wrapLines("   "))
}
