package quality

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func imlaCodes(fs []ImlaBulgu) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Code)
	}
	return out
}

func TestImlaDetectsMergedWords(t *testing.T) {
	got := CheckImla("Herşey yolunda. Birşey olmadı. Yada oldu.", "tr")
	assert.Contains(t, imlaCodes(got), "imla_birlesik")
	var joined string
	for _, f := range got {
		joined += f.Message
	}
	assert.Contains(t, joined, "her şey")
	assert.Contains(t, joined, "bir şey")
	assert.Contains(t, joined, "ya da")
}

// "birçok", "birkaç", "birtakım" doğru yazımlardır; uyarı üretmemeli.
func TestImlaDoesNotFlagCorrectMergedWords(t *testing.T) {
	got := CheckImla("Birçok kişi geldi. Birkaç gün kaldı. Birtakım sorunlar çıktı.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_birlesik")
}

func TestImlaDoesNotFlagEnglishTextForTurkishRules(t *testing.T) {
	got := CheckImla("Everyone arrived on time. Nothing else happened.", "en")
	assert.NotContains(t, imlaCodes(got), "imla_birlesik")
}

func TestImlaDetectsSpaceBeforePunctuation(t *testing.T) {
	got := CheckImla("Bu bir cümle , ve devamı var .", "tr")
	assert.Contains(t, imlaCodes(got), "imla_bosluk_once")
}

// Sayı ayırıcıları ve kısaltmalar yanlış pozitif üretmemeli.
func TestImlaAllowsNumbersAndAbbreviations(t *testing.T) {
	got := CheckImla("Fiyat 3,2 milyon lira. Oran %1.500 arttı. Elma, armut vb. ürünler var. Dr. Ahmet geldi.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_bosluk_sonra")
}

func TestImlaDetectsMissingSpaceAfterPunctuation(t *testing.T) {
	got := CheckImla("Bu cümle bitti.Sonraki cümle başladı.", "tr")
	assert.Contains(t, imlaCodes(got), "imla_bosluk_sonra")
}

func TestImlaDetectsRepeatedWord(t *testing.T) {
	got := CheckImla("Toplantı ve ve görüşme yapıldı.", "tr")
	assert.Contains(t, imlaCodes(got), "imla_tekrar_kelime")
	assert.Equal(t, 1, got[firstCode(got, "imla_tekrar_kelime")].Count)
}

// Tek harfli "a a" gibi gürültüler sayılmamalı.
func TestImlaRepeatedWordIgnoresShortNoise(t *testing.T) {
	got := CheckImla("A a harfi tabloda görünüyor.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_tekrar_kelime")
}

func TestImlaDetectsLowercaseSentenceStart(t *testing.T) {
	got := CheckImla("Birinci cümle bitti. ikinci cümle küçük başladı.", "tr")
	assert.Contains(t, imlaCodes(got), "imla_cumle_baslangici")
}

func TestImlaAcceptsCapitalizedSentences(t *testing.T) {
	got := CheckImla("Birinci cümle bitti. İkinci cümle büyük başladı. Üçüncü cümle de öyle.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_cumle_baslangici")
}

// Kısaltma sonrası küçük harf yanlış pozitif üretmemeli.
func TestImlaAbbreviationFollowedByLowercase(t *testing.T) {
	got := CheckImla("Elma, armut vb. ürünler satıldı.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_cumle_baslangici")
}

func TestImlaDetectsPunctuationStack(t *testing.T) {
	got := CheckImla("Bu gerçekten inanılmaz!! İnanamıyorum??", "tr")
	assert.Contains(t, imlaCodes(got), "imla_noktalama_yigini")
	assert.Equal(t, 2, got[firstCode(got, "imla_noktalama_yigini")].Count)
}

func TestImlaEllipsisAllowed(t *testing.T) {
	got := CheckImla("Sözü yarım kaldı… devamı gelmedi.", "tr")
	assert.NotContains(t, imlaCodes(got), "imla_noktalama_yigini")
}

func TestImlaEmptyText(t *testing.T) {
	assert.Empty(t, CheckImla("   ", "tr"))
}

func TestImlaCleanTextHasNoFindings(t *testing.T) {
	body := "İstanbul Büyükşehir Belediyesi yeni projeyi duyurdu. " +
		"Çalışmalar ekim sonunda tamamlanacak. " +
		"Belediye, 480 milyon liralık bütçe ayırdı."
	assert.Empty(t, CheckImla(body, "tr"), "temiz metinde bulgu olmamalı")
}

func firstCode(fs []ImlaBulgu, code string) int {
	for i, f := range fs {
		if f.Code == code {
			return i
		}
	}
	return -1
}
