package quality

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzeEmpty(t *testing.T) {
	r := Analyze("   ", "tr")
	assert.False(t, r.OK())
	assert.Equal(t, "empty", r.Blockers[0].Code)
	assert.Equal(t, 0, r.Metrics.Words)
}

func TestAnalyzeTooShort(t *testing.T) {
	r := Analyze("Kısa bir haber metni.", "tr")
	assert.False(t, r.OK())
	assert.Equal(t, "too_short", r.Blockers[0].Code)
}

func TestAnalyzeGoodTurkishText(t *testing.T) {
	body := strings.Join([]string{
		"İstanbul Büyükşehir Belediyesi, Kadıköy'deki sahil yolunda yürütülen yenileme çalışmalarının ekim sonunda tamamlanacağını açıkladı.",
		"Belediye yetkilileri, projenin toplam maliyetinin 480 milyon lira olduğunu ve yüklenici firmanın sözleşmede öngörülen takvime uyduğunu belirtti.",
		"Çalışma kapsamında 3,2 kilometrelik yürüyüş yolu yeniden döşendi, iki yeni bisiklet şeridi eklendi ve 140 ağaç dikildi.",
		"Mahalle sakinleri, çalışmalar sırasında oluşan gürültünün azalmasını istediklerini söyledi.",
		"Belediye, şikâyetlerin ardından gece çalışmalarını 22.00'de sonlandırma kararı aldı.",
		"Yeni düzenlemenin 2027 bütçesinde ayrılan ek kaynakla sürdürüleceği bildirildi.",
	}, "\n\n")

	r := Analyze(body, "tr")
	assert.True(t, r.OK(), "engelleyici bulgu olmamalı: %+v", r.Blockers)
	assert.GreaterOrEqual(t, r.Score, 75, "temiz metin B ve üzeri olmalı: %+v", r)
	assert.Contains(t, []string{"A", "B"}, r.Grade)
	assert.Greater(t, r.Metrics.Words, 60)
	assert.Greater(t, r.Metrics.Sentences, 5)
	assert.Equal(t, 6, r.Metrics.Paragraphs)
}

func TestAnalyzeDetectsCliches(t *testing.T) {
	body := strings.Repeat(
		"Günümüz dünyasında bu konu büyük önem taşıyor. Sonuç olarak herkes hemfikir. ", 8)

	r := Analyze(body, "tr")
	assert.Greater(t, r.Metrics.Cliques, 3)
	code := codes(r.Warnings)
	assert.Contains(t, code, "cliche")
}

func TestAnalyzeDetectsUniformRhythm(t *testing.T) {
	// Her cümle tam 8 sözcük: sapma 0 → tekdüze ritim uyarısı.
	s := "bir iki üç dört beş altı yedi sekiz"
	body := strings.Repeat(s+". ", 8)

	r := Analyze(body, "tr")
	assert.Less(t, r.Metrics.SentenceStdDev, 4.0)
	assert.Contains(t, codes(r.Warnings), "uniform_rhythm")
}

func TestAnalyzeDetectsRepeatedPhraseAsBlocker(t *testing.T) {
	block := "belediye meclisi yeni bütçe taslağını oybirliğiyle kabul etti"
	body := "Giriş cümlesi burada yer alıyor ve bağlam veriyor.\n\n" +
		strings.Join([]string{block, block, block, block, block}, ". ") + "."

	r := Analyze(body, "tr")
	assert.Contains(t, codes(r.Blockers), "repeated_phrase",
		"5 kez birebir tekrar yayını engellemeli: %+v", r.Blockers)
	assert.Greater(t, r.Metrics.RepeatedPhrases, 0)
}

func TestAnalyzeRepeatedPhraseWarningOnly(t *testing.T) {
	block := "kaymakamlık yeni sosyal yardım programının ayrıntılarını kamuoyuna duyurdu"
	body := "Bağlam cümlesi burada duruyor ve ayrıntı ekliyor.\n\n" +
		block + ". " + block + ". " + block + "."

	r := Analyze(body, "tr")
	assert.NotContains(t, codes(r.Blockers), "repeated_phrase")
	assert.Contains(t, codes(r.Warnings), "repeated_phrase")
}

func TestAnalyzeEnglishCliches(t *testing.T) {
	body := strings.Repeat("In today's fast-paced world, it is important to note the shift. ", 12)
	r := Analyze(body, "en")
	assert.GreaterOrEqual(t, r.Metrics.Cliques, 10)
	assert.Contains(t, codes(r.Warnings), "cliche")
}

func TestAnalyzeUnknownLanguageUsesBothLists(t *testing.T) {
	body := "Günümüz dünyasında ve in today's fast-paced world aynı cümlede geçiyor. "
	r := Analyze(strings.Repeat(body, 6), "de")
	assert.GreaterOrEqual(t, r.Metrics.Cliques, 6)
}

func TestMATTRPenalizesRepetition(t *testing.T) {
	unique := make([]string, 120)
	for i := range unique {
		unique[i] = "kelime" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	repeat := make([]string, 120)
	for i := range repeat {
		repeat[i] = "aynı"
	}
	assert.Greater(t, mattr(unique, mattrWindow), 0.9)
	assert.Less(t, mattr(repeat, mattrWindow), 0.1)
}

func TestMATTRShortTextFallsBackToTTR(t *testing.T) {
	got := mattr([]string{"a", "b", "a", "c"}, 50)
	assert.InDelta(t, 0.75, got, 0.0001)
}

func TestTokenizeSplitsPunctuationAndKeepsUnicode(t *testing.T) {
	got := tokenize("Kadıköy'de 3,2 kilometre — “yeni” şerit!")
	assert.Equal(t, []string{"kadıköy", "de", "3", "2", "kilometre", "yeni", "şerit"}, got)
}

func TestSplitSentencesAndParagraphs(t *testing.T) {
	text := "Birinci cümle. İkinci cümle!\n\nÜçüncü cümle?"
	assert.Len(t, splitSentences(text), 3)
	assert.Len(t, splitParagraphs(text), 2)
}

func TestAnalyzeIncludesImlaFindings(t *testing.T) {
	body := strings.Repeat("Herşey yolunda gidiyor ve ve sorun görünmüyor. ", 6)
	r := Analyze(body, "tr")
	assert.NotEmpty(t, r.Imla, "imla bulguları rapora eklenmeli")
	codes := map[string]bool{}
	for _, f := range r.Imla {
		codes[f.Code] = true
	}
	assert.True(t, codes["imla_birlesik"], "birleşik yazım bulunmalı: %+v", r.Imla)
	assert.True(t, codes["imla_tekrar_kelime"], "yinelenen kelime bulunmalı")
	assert.Less(t, r.Score, 100, "imla bulgusu puanı düşürmeli")
	assert.NotEqual(t, "A", r.Grade)
}

func TestAnalyzeCleanTextHasNoImla(t *testing.T) {
	body := "İstanbul Büyükşehir Belediyesi yeni projeyi duyurdu. " +
		"Çalışmalar ekim sonunda tamamlanacak. " +
		"Belediye 480 milyon liralık bütçe ayırdı."
	r := Analyze(body, "tr")
	assert.Empty(t, r.Imla, "temiz metinde imla bulgusu olmamalı: %+v", r.Imla)
}

func TestGradeScales(t *testing.T) {
	assert.Equal(t, "A", grade(95))
	assert.Equal(t, "B", grade(80))
	assert.Equal(t, "C", grade(65))
	assert.Equal(t, "D", grade(45))
	assert.Equal(t, "E", grade(10))
}

// TestWarningCapsGradeAtB uyarı varken A verilmediğini doğrular: uyarı,
// düzeltilmesi gereken bir şey olduğunu gösterir.
func TestWarningCapsGradeAtB(t *testing.T) {
	body := strings.Repeat("Kısa ama düzgün bir cümle yazıldı buraya. ", 10)
	r := Analyze(body, "tr")
	assert.NotEmpty(t, r.Warnings)
	assert.NotEqual(t, "A", r.Grade, "uyarı varken A verilmemeli: %+v", r)
	assert.LessOrEqual(t, r.Score, 89)
}

func codes(fs []Finding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Code)
	}
	return out
}
