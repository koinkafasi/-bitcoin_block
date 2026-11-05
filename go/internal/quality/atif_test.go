package quality

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const transcriptTR = `Konuşmacı A: Yatırım planı 2027 yılında başlayacak. Bütçe detayları
önümüzdeki ay açıklanacak.
Konuşmacı B: Bu konuda henüz resmi bir açıklama yapılmadı, biz de bekliyoruz.`

func TestVerifyQuotesVerbatim(t *testing.T) {
	r := VerifyQuotes([]string{"Yatırım planı 2027 yılında başlayacak"}, transcriptTR)
	assert.True(t, r.OK(), "birebir alıntı doğrulanmalı: %+v", r.Unverified)
	assert.Equal(t, 1, r.Checked)
	assert.Equal(t, 1, r.Verified)
	assert.Equal(t, 100, r.Score)
}

// Noktalama, büyük/küçük harf ve tırnak farkı tolere edilir.
func TestVerifyQuotesToleratesPunctuationAndCase(t *testing.T) {
	r := VerifyQuotes([]string{"“yatırım planı 2027 yılında başlayacak!”"}, transcriptTR)
	assert.True(t, r.OK(), "%+v", r.Unverified)
}

func TestVerifyQuotesDetectsFabricatedQuote(t *testing.T) {
	r := VerifyQuotes([]string{"Bakan yatırımın 500 milyar lira olacağını söyledi"}, transcriptTR)
	assert.False(t, r.OK(), "uydurma alıntı yakalanmalı")
	assert.Len(t, r.Unverified, 1)
	assert.Equal(t, 0, r.Score)
	assert.Contains(t, r.Unverified[0].Detail, "bulunamadı")
}

func TestVerifyQuotesDetectsAlteredNumbers(t *testing.T) {
	src := "Bütçe 480 milyon lira olarak açıklandı ve çalışmalar ekim sonunda bitecek."
	r := VerifyQuotes([]string{"Bütçe 840 milyon lira olarak açıklandı ve çalışmalar ekim sonunda bitecek"}, src)
	assert.False(t, r.OK(), "değiştirilmiş sayı yakalanmalı: %+v", r)
}

func TestVerifyQuotesShortExclamationsSkipped(t *testing.T) {
	r := VerifyQuotes([]string{"Evet", "Hayır."}, transcriptTR)
	assert.True(t, r.OK())
	assert.Equal(t, 2, r.Checked)
}

func TestVerifyQuotesEmptyInput(t *testing.T) {
	r := VerifyQuotes(nil, transcriptTR)
	assert.Equal(t, 0, r.Checked)
	assert.Equal(t, 100, r.Score)
	assert.True(t, r.OK())
}

func TestVerifyQuotesMultipleMixed(t *testing.T) {
	r := VerifyQuotes([]string{
		"Bütçe detayları önümüzdeki ay açıklanacak",
		"Bakan ayrıca yeni bir vergi getirileceğini duyurdu",
	}, transcriptTR)
	assert.Equal(t, 2, r.Checked)
	assert.Equal(t, 1, r.Verified)
	assert.Equal(t, 50, r.Score)
	assert.Len(t, r.Unverified, 1)
}

// Alıntı kaynağın içinde ama başka bir cümleden alınmış sözcüklerle
// birleştirilmişse yakalanmalı.
func TestVerifyQuotesRejectsStitchedQuote(t *testing.T) {
	src := "Ali bugün geldi. Veli yarın gidecek."
	r := VerifyQuotes([]string{"Ali yarın gidecek dedi"}, src)
	assert.False(t, r.OK(), "birleştirilmiş alıntı yakalanmalı: %+v", r)
}

func TestNormalizeForQuote(t *testing.T) {
	// Tırnak/kesme işareti kaldırılır (sözcük bölünmez), Türkçe harfler korunur,
	// bölünmez boşluk normal boşluğa döner.
	assert.Equal(t, "istanbulda hava güzel", normalizeForQuote("İstanbul’da hava güzel!"))
	assert.Equal(t, "a b", normalizeForQuote("A\u00a0B"))
}

func TestSequentialOverlap(t *testing.T) {
	src := tokenize(normalizeForQuote("bir iki üç dört beş"))
	assert.Equal(t, 1.0, sequentialOverlap([]string{"bir", "iki", "üç"}, src))
	assert.Less(t, sequentialOverlap([]string{"altı", "yedi"}, src), 0.5)
}

func TestVerifyQuotesEmptySource(t *testing.T) {
	r := VerifyQuotes([]string{"Bu cümle kaynakta yok"}, "")
	assert.False(t, r.OK())
	assert.Equal(t, 0, r.Score)
}

func TestVerifyQuotesLongQuotePartialMatch(t *testing.T) {
	src := strings.Repeat("uzun bir kaynak cümlesi burada yer alıyor ve devam ediyor ", 3)
	r := VerifyQuotes([]string{"uzun bir kaynak cümlesi burada yer alıyor ve devam ediyor"}, src)
	assert.True(t, r.OK(), "%+v", r.Unverified)
}
