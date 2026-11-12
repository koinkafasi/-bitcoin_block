package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCdDilKodu(t *testing.T) {
	assert.Equal(t, "en", cdDilKodu("en"))
	assert.Equal(t, "tr", cdDilKodu("tr"))
	// Boş ya da boşluk: sitenin ana dili.
	assert.Equal(t, "tr", cdDilKodu(""))
	assert.Equal(t, "tr", cdDilKodu("   "))
}

// x-default TÜM dil sürümlerinde aynı adresi göstermeli. Eskiden her
// sayfa kendi kanonik adresini basıyordu; İngilizce sayfa x-default'u
// kendisi, Türkçe sayfa da kendisi diye bildiriyordu.
func TestCdXDefaultHerDildeTurkceyiGosterir(t *testing.T) {
	trURL := "https://x.test/markets/2026/11/05/haber"
	enURL := "https://x.test/en/markets/2026/11/05/news"

	// Türkçe sayfanın head'i
	tr := CDHead{Canonical: trURL, Hreflang: map[string]string{"tr": trURL, "en": enURL}}
	// İngilizce sayfanın head'i — kanoniği farklı, x-default aynı olmalı
	en := CDHead{Canonical: enURL, Hreflang: map[string]string{"tr": trURL, "en": enURL}}

	assert.Equal(t, trURL, cdXDefault(tr))
	assert.Equal(t, trURL, cdXDefault(en), "iki sürüm de aynı x-default'u bildirmeli")
	assert.Equal(t, cdXDefault(tr), cdXDefault(en))
}

func TestCdXDefaultCevirisizSayfa(t *testing.T) {
	// Çevirisi olmayan sayfa kendi adresini gösterir.
	h := CDHead{Canonical: "https://x.test/tek"}
	assert.Equal(t, "https://x.test/tek", cdXDefault(h))

	// Hreflang haritası var ama tr boş: yine kendi adresi.
	h2 := CDHead{Canonical: "https://x.test/tek", Hreflang: map[string]string{"tr": "  "}}
	assert.Equal(t, "https://x.test/tek", cdXDefault(h2))
}
