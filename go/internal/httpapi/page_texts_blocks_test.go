package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Faz 142: yazı gövdesi bloğu ---------------------------------

func TestNormalizeYaziGovdeTekilVeAna(t *testing.T) {
	s := &Server{}
	in := json.RawMessage(`[
		{"id":"g1","type":"yazi_govde","opts":{"zone":"yan"}},
		{"id":"g2","type":"yazi_govde","opts":{"zone":"ust"}},
		{"id":"m","type":"metin","opts":{"zone":"ana","html":"<p>x</p>"}}]`)
	_, arr, err := s.normalizeBlocksForPage("article", in)
	require.NoError(t, err)
	require.Len(t, arr, 2, "ikinci yazi_govde düşürülmeli")
	opts0, _ := arr[0]["opts"].(map[string]any)
	assert.Equal(t, "ana", opts0["zone"], "gövde bloğu her zaman ana bölgesindedir")
	assert.Equal(t, "metin", arr[1]["type"], "diğer bloklar korunur")
}

func TestNormalizeYaziGovdeArticleDisiDuser(t *testing.T) {
	s := &Server{}
	in := json.RawMessage(`[{"id":"g","type":"yazi_govde","opts":{"zone":"ana"}},{"id":"b","type":"baslik","opts":{"zone":"ana"}}]`)
	_, arr, err := s.normalizeBlocksForPage("home", in)
	require.NoError(t, err)
	require.Len(t, arr, 1)
	assert.Equal(t, "baslik", arr[0]["type"], "yazi_govde yalnız yazı sayfasında tutulur")
}

func TestBuilderTypesArticleGovdeVeMetin(t *testing.T) {
	var hasGovde, hasMetin bool
	for _, typ := range builderTypesFor("article") {
		switch typ.Type {
		case "yazi_govde":
			hasGovde = true
		case "metin":
			hasMetin = true
		}
	}
	assert.True(t, hasGovde, "article paletinde yazi_govde olmalı")
	assert.True(t, hasMetin, "article paletinde metin olmalı")
}
