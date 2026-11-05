package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelevanceIdealBaslikVeSpot(t *testing.T) {
	// 40–90 karakter başlık (+20), >=120 karakter spot (+25), 3 etiket (+15),
	// kapak (+10), kaynak (+10), taban 20 → 100.
	score, note := Relevance(Input{
		Title:     "Merkez Bankası faiz kararını açıkladı, piyasalar hareketlendi",
		Spot:      "Merkez Bankası politika faizini beklentilerin altında artırdı. Karar sonrası döviz kurları ve borsa sert hareket etti; ekonomistler yıl sonu tahminlerini güncelledi.",
		TagCount:  4,
		HasCover:  true,
		HasSource: true,
	})
	assert.Equal(t, 100, score)
	assert.Equal(t, Note, note)
}

func TestRelevanceBosIcerikDusukPuan(t *testing.T) {
	// Boş başlık (-10), boş spot (-10), taban 20 → 0.
	score, _ := Relevance(Input{})
	assert.Equal(t, 0, score)
}

func TestRelevanceYalnizBaslikVeKaynak(t *testing.T) {
	// Kısa başlık (<10) → -10; boş spot → -10; taban 20; kaynak +10 → 10.
	score, _ := Relevance(Input{Title: "Kısa", HasSource: true})
	assert.Equal(t, 10, score)
}

func TestRelevanceEtiketEsikleri(t *testing.T) {
	base := Input{Title: "Belediye meclisi yeni bütçeyi oy çokluğuyla kabul etti"}
	zero, _ := Relevance(base)
	base.TagCount = 1
	one, _ := Relevance(base)
	base.TagCount = 3
	three, _ := Relevance(base)
	assert.Equal(t, 8, one-zero)
	assert.Equal(t, 7, three-one)
}

func TestRelevanceSinirlariAsmaz(t *testing.T) {
	score, _ := Relevance(Input{
		Title:     "Test test test test test test test test test test test test",
		Spot:      "uzun bir spot metni burada duruyor ve yüz yirmi karakterden fazla olması gerekiyor ki tam puan versin, devam ediyoruz",
		TagCount:  10,
		HasCover:  true,
		HasSource: true,
	})
	assert.LessOrEqual(t, score, 100)
	assert.GreaterOrEqual(t, score, 0)
}

func TestKeywordDensityKisaKelimeleriAtlar(t *testing.T) {
	// "ve" 2 harf, "bir" 3 harf → sayılmaz. "ekonomi" iki kez → 5 puan.
	assert.Equal(t, 5, KeywordDensity("ekonomi ve bir ekonomi"))
	assert.Equal(t, 0, KeywordDensity("tek kelime"))
}

func TestRelevanceDeterministik(t *testing.T) {
	in := Input{Title: "Aynı girdi aynı puanı verecek", Spot: "Bu spot yeterince uzun değil ama sabit.", TagCount: 2}
	a, _ := Relevance(in)
	b, _ := Relevance(in)
	assert.Equal(t, a, b)
}
