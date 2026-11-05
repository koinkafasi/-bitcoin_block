package aieditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const atifTranscript = `Konuşmacı A: Yatırım planı 2027 yılında başlayacak. Bütçe detayları
önümüzdeki ay açıklanacak.
Konuşmacı B: Bu konuda henüz resmi bir açıklama yapılmadı.`

// TestQuoteWarningsVerifiesGroundedQuotes transkriptten birebir alınan
// alıntıların uyarı üretmediğini doğrular.
func TestQuoteWarningsVerifiesGroundedQuotes(t *testing.T) {
	r := &Result{Quotes: []string{"Yatırım planı 2027 yılında başlayacak"}}
	warns := quoteWarnings(r, atifTranscript)
	assert.Empty(t, warns, "doğrulanan alıntı uyarı üretmemeli")
	assert.Empty(t, r.UnverifiedQuotes)
}

// TestQuoteWarningsFlagsFabricatedQuote uydurulmuş alıntının işaretlendiğini
// ve Result.UnverifiedQuotes alanına yazıldığını doğrular.
func TestQuoteWarningsFlagsFabricatedQuote(t *testing.T) {
	r := &Result{Quotes: []string{
		"Yatırım planı 2027 yılında başlayacak",
		"Bakan ayrıca yeni bir vergi getirileceğini duyurdu",
	}}
	warns := quoteWarnings(r, atifTranscript)
	assert.Len(t, warns, 1)
	assert.Contains(t, warns[0], "1 alıntı")
	assert.Contains(t, warns[0], "doğrulanamadı")
	assert.Equal(t, []string{"Bakan ayrıca yeni bir vergi getirileceğini duyurdu"}, r.UnverifiedQuotes)
}

// TestQuoteWarningsNoQuotes alıntı yoksa hiçbir şey yapılmadığını doğrular.
func TestQuoteWarningsNoQuotes(t *testing.T) {
	r := &Result{}
	assert.Empty(t, quoteWarnings(r, atifTranscript))
	assert.Empty(t, r.UnverifiedQuotes)
}

// TestQuoteWarningsAllFabricated tümü uydurma olduğunda oranın doğru
// raporlandığını doğrular.
func TestQuoteWarningsAllFabricated(t *testing.T) {
	r := &Result{Quotes: []string{
		"Bakan yeni vergi getirileceğini duyurdu",
		"Yatırım 500 milyar lira olacak denildi",
	}}
	warns := quoteWarnings(r, atifTranscript)
	assert.Len(t, warns, 1)
	assert.Contains(t, warns[0], "2 alıntı")
	assert.Contains(t, warns[0], "0/2")
	assert.Len(t, r.UnverifiedQuotes, 2)
}

// TestNormalizeAudioDraftKeepsQuotes normalize çıktısının alıntıları
// Result.Quotes alanına taşıdığını doğrular (kapı bu alanı kullanır).
func TestNormalizeAudioDraftKeepsQuotes(t *testing.T) {
	act, ok := ActionByID("audio_draft")
	assert.True(t, ok)

	res, err := normalizeResult(act, map[string]any{
		"title":  "Başlık",
		"spot":   "Spot",
		"body":   "Gövde metni.",
		"quotes": []any{"Yatırım planı 2027 yılında başlayacak", "Bütçe önümüzdeki ay açıklanacak"},
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Yatırım planı 2027 yılında başlayacak", "Bütçe önümüzdeki ay açıklanacak"}, res.Quotes)
}
