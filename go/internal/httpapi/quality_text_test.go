package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

func TestCleanForQualityDecodesEntities(t *testing.T) {
	in := "Borsa Gundem&amp;nbsp;&amp;nbsp;Uzmanpara"
	got := cleanForQuality(in)
	assert.NotContains(t, got, "nbsp")
	assert.NotContains(t, got, "amp")
	assert.Contains(t, got, "Borsa Gundem")
	assert.Contains(t, got, "Uzmanpara")
}

func TestCleanForQualityHandlesDoubleEscapedNbsp(t *testing.T) {
	got := cleanForQuality("Bir&amp;nbsp;iki")
	assert.Equal(t, "Bir iki", got)
}

func TestCleanForQualityStripsTags(t *testing.T) {
	got := cleanForQuality("<p>Birinci</p><p>İkinci</p>")
	assert.Equal(t, "Birinciİkinci", got)
}

func TestCleanForQualityCollapsesSpacesKeepsParagraphs(t *testing.T) {
	got := cleanForQuality("Bir   iki\tüç\n\n\n\nDört")
	assert.Equal(t, "Bir iki üç\n\nDört", got)
}

// Asıl regresyon: &nbsp; varlıkları kelime sayılıp tekrar metriğini
// şişiriyordu.
func TestQualityTextIgnoresHtmlEntitiesInMetrics(t *testing.T) {
	c := repo.Content{
		Title:  "Borsa",
		Spot:   "Spot&amp;nbsp;metni",
		Detail: []byte(`[{"type":"p","text":"Borsa Gündem&amp;nbsp;&amp;nbsp;Uzmanpara"}]`),
		Locale: "tr",
	}
	text := qualityText(c)
	assert.NotContains(t, text, "nbsp")
	assert.NotContains(t, text, "amp")
	assert.Equal(t, 1, strings.Count(text, "Uzmanpara"))
}
