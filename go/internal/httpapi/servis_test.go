package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

func TestFilterServisKurlar(t *testing.T) {
	items := []repo.Kur{
		{Name: "BIST", Value: 1},
		{Name: "USD", Value: 2},
		{Name: "EUR", Value: 3},
		{Name: "ALTIN", Value: 4},
		{Name: "BTC", Value: 5},
		{Name: "ONS", Value: 6},
	}
	all := filterServisKurlar(items, nil)
	assert.Len(t, all, 6)

	got := filterServisKurlar(items, map[string]any{
		"kur_usd": false, "kur_btc": false, "kur_ons": false,
	})
	assert.Len(t, got, 3)
	assert.Equal(t, []string{"BIST", "EUR", "ALTIN"}, []string{got[0].Name, got[1].Name, got[2].Name})

	// Tümü kapatılabilir.
	none := filterServisKurlar(items, map[string]any{
		"kur_bist": false, "kur_usd": false, "kur_eur": false,
		"kur_altin": false, "kur_btc": false, "kur_ons": false,
	})
	assert.Empty(t, none)
}

func TestParseHomeBlocksKeepsServiceOpts(t *testing.T) {
	raw := []byte(`[{"id":"b-servis","type":"servis_bandi","opts":{"kur_usd":false,"kur_bist":true}}]`)
	blocks := parseHomeBlocks(raw)
	if assert.Len(t, blocks, 1) {
		assert.False(t, blockOptBool(blocks[0].Opts, "kur_usd", true))
		assert.True(t, blockOptBool(blocks[0].Opts, "kur_bist", true))
	}
}

func TestBlockOptBool(t *testing.T) {
	assert.True(t, blockOptBool(nil, "kur_usd", true))
	assert.False(t, blockOptBool(map[string]any{"kur_usd": false}, "kur_usd", true))
	assert.True(t, blockOptBool(map[string]any{"kur_usd": "hayır"}, "kur_usd", true))
}
