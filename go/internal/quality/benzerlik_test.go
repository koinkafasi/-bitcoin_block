package quality

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJaccard(t *testing.T) {
	a := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	b := map[string]struct{}{"b": {}, "c": {}, "d": {}}
	// kesişim 2, birleşim 4
	assert.InDelta(t, 0.5, Jaccard(a, b), 0.0001)
	assert.Equal(t, 1.0, Jaccard(a, a))
	assert.Equal(t, 0.0, Jaccard(a, map[string]struct{}{}))
	assert.Equal(t, 0.0, Jaccard(map[string]struct{}{}, b))
}

func TestShinglesCount(t *testing.T) {
	set := Shingles("bir iki üç dört beş altı", 5)
	assert.Len(t, set, 2) // bir iki üç dört beş / iki üç dört beş altı
	assert.Contains(t, set, "bir iki üç dört beş")
}

func TestShinglesShortText(t *testing.T) {
	set := Shingles("kısa metin", 5)
	assert.Len(t, set, 1)
	assert.Contains(t, set, "kısa metin")
}

func TestShinglesEmpty(t *testing.T) {
	assert.Empty(t, Shingles("   ", 5))
}

func TestCheckDuplicatesIdenticalIsMatch(t *testing.T) {
	body := strings.Repeat("belediye meclisi yeni bütçe taslağını oybirliğiyle kabul etti ", 5)
	r := CheckDuplicates(body, []Candidate{{Ref: "1", Title: "Bütçe", Text: body}})
	assert.False(t, r.OK())
	assert.Len(t, r.Matches, 1)
	assert.GreaterOrEqual(t, r.MaxScore, 0.99)
	assert.Equal(t, 0, r.Score)
	assert.Contains(t, r.Matches[0].Detail, "benzer")
}

func TestCheckDuplicatesSlightEditStillMatch(t *testing.T) {
	// Gerçekçi metin: tekrar eden kalıp yok.
	base := "İstanbul Büyükşehir Belediyesi Kadıköy'deki sahil yolunda yürütülen yenileme " +
		"çalışmalarının ekim sonunda tamamlanacağını açıkladı. Projenin toplam maliyeti " +
		"480 milyon lira olarak açıklandı ve yüklenici firmanın takvime uyduğu belirtildi. " +
		"Çalışma kapsamında 3,2 kilometrelik yürüyüş yolu yeniden döşendi ve iki yeni " +
		"bisiklet şeridi eklendi. Mahalle sakinleri çalışmalar sırasında oluşan gürültünün " +
		"azalmasını istedi; belediye gece çalışmalarını 22.00'de sonlandırma kararı aldı."
	edited := strings.ReplaceAll(base,
		"Mahalle sakinleri çalışmalar sırasında oluşan gürültünün azalmasını istedi",
		"Bölge halkı çalışmalardan doğan gürültünün azaltılmasını talep etti")
	edited = strings.ReplaceAll(edited, "480 milyon lira", "480 milyon TL")
	r := CheckDuplicates(edited, []Candidate{{Ref: "7", Title: "Bütçe haberi", Text: base}})
	assert.False(t, r.OK(), "küçük düzenleme kopyayı gizlememeli (skor %.3f)", r.MaxScore)
	assert.Greater(t, r.MaxScore, 0.55)
}

func TestCheckDuplicatesDifferentTextIsClean(t *testing.T) {
	a := "Borsa İstanbul günü yükselişle kapattı. Bankacılık hisseleri öne çıktı."
	b := "Kadıköy'de sahil yolu yenilendi. Bisiklet şeridi eklendi ve 140 ağaç dikildi."
	r := CheckDuplicates(a, []Candidate{{Ref: "2", Title: "Sahil yolu", Text: b}})
	assert.True(t, r.OK())
	assert.Equal(t, 100, r.Score)
}

func TestCheckDuplicatesEmptyCandidates(t *testing.T) {
	r := CheckDuplicates("Bir metin burada.", nil)
	assert.Equal(t, 0, r.Checked)
	assert.True(t, r.OK())
}

func TestCheckDuplicatesEmptyText(t *testing.T) {
	r := CheckDuplicates("   ", []Candidate{{Ref: "1", Text: "başka metin"}})
	assert.True(t, r.OK())
	assert.Equal(t, 100, r.Score)
}

func TestCheckDuplicatesPicksHighest(t *testing.T) {
	text := "Borsa İstanbul günü yükselişle kapattı. Bankacılık hisseleri öne çıktı ve " +
		"işlem hacmi bir önceki güne göre arttı. Analistler kısa vadede dalgalanmanın " +
		"sürebileceğini belirtiyor. Döviz kurları yatay seyretti, tahvil faizleri geriledi."
	near := strings.ReplaceAll(text, "Analistler kısa vadede dalgalanmanın sürebileceğini belirtiyor",
		"Uzmanlar kısa vadede dalgalanmanın devam edebileceğini söylüyor")
	far := "Kadıköy'de sahil yolu yenilendi. Bisiklet şeridi eklendi ve 140 ağaç dikildi. " +
		"Belediye çalışmaların ekim sonunda biteceğini açıkladı."
	r := CheckDuplicates(text, []Candidate{
		{Ref: "far", Title: "Uzak", Text: far},
		{Ref: "near", Title: "Yakın", Text: near},
	})
	assert.Len(t, r.Matches, 1, "yalnız yakın metin eşleşmeli (max %.3f)", r.MaxScore)
	assert.Equal(t, "near", r.Matches[0].Ref)
	assert.GreaterOrEqual(t, r.MaxScore, DuplicateThreshold)
}

func TestNormalizeTextForCompare(t *testing.T) {
	assert.Equal(t, "istanbul da hava güzel", normalizeTextForCompare("İstanbul'da hava güzel!"))
	assert.Equal(t, "a b", normalizeTextForCompare("a—b"))
}

func TestShortTitle(t *testing.T) {
	assert.Equal(t, "Kısa", shortTitle("Kısa", "ref"))
	assert.Equal(t, "ref", shortTitle("  ", "ref"))
	long := strings.Repeat("a", 80)
	assert.LessOrEqual(t, len([]rune(shortTitle(long, "x"))), 61)
}
