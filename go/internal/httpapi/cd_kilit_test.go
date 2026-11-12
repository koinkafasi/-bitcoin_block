package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Önizleme kesme: kilitli içerikte gövdenin ilk N paragrafı açık kalır.
// Kesme HTML'i bozmamalı — yarım etiket bırakmak sayfayı dağıtır.

func TestOnizlemeKesIlkParagraflariAlir(t *testing.T) {
	govde := "<p>Bir</p><p>İki</p><p>Üç</p><p>Dört</p>"
	kesik := cdOnizlemeKes(govde, 3)

	assert.Equal(t, "<p>Bir</p><p>İki</p><p>Üç</p>", kesik)
	assert.NotContains(t, kesik, "Dört", "sınırdan sonrası alınmamalı")
	assert.Equal(t, 3, strings.Count(kesik, "<p>"))
	assert.Equal(t, 3, strings.Count(kesik, "</p>"), "açık etiket kalmamalı")
}

func TestOnizlemeKesGovdeKisaysaTamaminiVerir(t *testing.T) {
	govde := "<p>Tek paragraf</p>"
	assert.Equal(t, govde, cdOnizlemeKes(govde, 5))
}

func TestOnizlemeKesSinirDurumlari(t *testing.T) {
	assert.Empty(t, cdOnizlemeKes("<p>Bir</p>", 0), "sıfır paragraf istenirse boş döner")
	assert.Empty(t, cdOnizlemeKes("", 3))
	// Paragraf etiketi olmayan gövde: kesilecek sınır yok, boş döner —
	// kilit kutusu yine basıldığı için sayfa boş kalmıyor.
	assert.Empty(t, cdOnizlemeKes("düz metin", 3))
}

func TestOnizlemeKesAraEtiketleriKorur(t *testing.T) {
	// Paragraf içindeki biçimlendirme bozulmamalı.
	govde := `<p>Bir <strong>kalın</strong> ve <a href="/x">bağlantı</a></p><p>İki</p>`
	kesik := cdOnizlemeKes(govde, 1)
	assert.Equal(t, `<p>Bir <strong>kalın</strong> ve <a href="/x">bağlantı</a></p>`, kesik)
}

// Kilit kararının temel kuralları (yardımcılar abonelik akışından geliyor;
// burada yeni tasarımın onlara doğru bağlandığını sabitliyoruz).
func TestKilitKurallari(t *testing.T) {
	// Kademe yoksa kilit yok.
	assert.Equal(t, 0, contentRequiredWeight(false, 0))
	// `is_exclusive` tek başına birinci kademeyi gerektirir.
	assert.Equal(t, 1, contentRequiredWeight(true, 0))
	// Açık kademe her zaman kazanır.
	assert.Equal(t, 3, contentRequiredWeight(true, 3))

	// Erişim: üye ağırlığı gerekli kademeye eşit ya da üstündeyse açık.
	assert.True(t, tierAllows(3, 3))
	assert.True(t, tierAllows(5, 2))
	assert.False(t, tierAllows(1, 2))
	// Gereksiz kilit yok: 0 gereklilik herkese açık.
	assert.True(t, tierAllows(0, 0))
}
