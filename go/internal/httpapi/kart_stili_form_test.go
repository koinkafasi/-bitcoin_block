package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseTekBlok(t *testing.T, opts string) homeBlock {
	t.Helper()
	raw := []byte(`[{"id":"b1","type":"ust_manset","visible":true,"opts":` + opts + `}]`)
	blocks := parseHomeBlocks(raw)
	require.Len(t, blocks, 1, "tek blok ayrıştırılmalı")
	return blocks[0]
}

func TestKartStiliJSONAyristirma(t *testing.T) {
	b := parseTekBlok(t, `{
		"card_style":"bloomberg",
		"card_title_bg":"#0f172a",
		"card_title_fg":"#ffffff",
		"card_dark_bg":"#f1f5f9",
		"card_dark_fg":"#181d28",
		"card_accent":"#d10015",
		"card_title_size":18
	}`)

	assert.Equal(t, "bloomberg", b.CardStyle)
	css := string(b.CardCSS)
	assert.Contains(t, css, "--cs-light-bg:#0f172a")
	assert.Contains(t, css, "--cs-light-fg:#ffffff")
	assert.Contains(t, css, "--cs-dark-bg:#f1f5f9")
	assert.Contains(t, css, "--cs-dark-fg:#181d28")
	assert.Contains(t, css, "--cs-accent:#d10015")
	assert.Contains(t, css, "--cs-size:18px")
}

func TestKartGovdesiRenkleriVeGolgeKapatma(t *testing.T) {
	b := parseTekBlok(t, `{
		"card_style":"bloomberg-cizgi",
		"card_box_bg":"#ff0000",
		"card_box_dark_bg":"#00ff00",
		"card_shadow":false
	}`)
	css := string(b.CardCSS)
	assert.Contains(t, css, "--cs-card-light-bg:#ff0000", "açık tema kart gövdesi")
	assert.Contains(t, css, "--cs-card-dark-bg:#00ff00", "koyu tema kart gövdesi")
	assert.Contains(t, css, "--cs-shadow:none;--cs-shadow-hover:none;", "gölge ve hover gölgesi kapanmalı")
}

func TestKartGolgesiVarsayilanAcik(t *testing.T) {
	b := parseTekBlok(t, `{"card_style":"bloomberg"}`)
	assert.NotContains(t, string(b.CardCSS), "--cs-shadow:none", "gölge varsayılan olarak açık kalmalı")
}

func TestKartGovdesiGecersizRenkYokSayilir(t *testing.T) {
	b := parseTekBlok(t, `{"card_style":"bloomberg","card_box_bg":"javascript:alert(1)","card_box_dark_bg":"#12"}`)
	assert.NotContains(t, string(b.CardCSS), "javascript", "geçersiz renk enjekte edilmemeli")
	assert.NotContains(t, string(b.CardCSS), "--cs-card-light-bg", "geçersiz açık renk yok sayılmalı")
	assert.NotContains(t, string(b.CardCSS), "--cs-card-dark-bg", "geçersiz koyu renk yok sayılmalı")
}

func TestKartGovdesiKartStiliYokkenUygulanmaz(t *testing.T) {
	b := parseTekBlok(t, `{"card_box_bg":"#ff0000","card_box_dark_bg":"#00ff00","card_shadow":false}`)
	assert.Empty(t, b.CardStyle)
	assert.Empty(t, string(b.CardCSS), "kart stili seçilmeden gövde/gölge değişkeni üretilmemeli")
}

func TestKartStiliGecersizDegerYokSayilir(t *testing.T) {
	b := parseTekBlok(t, `{"card_style":"zararli","card_title_bg":"javascript:alert(1)"}`)
	assert.Empty(t, b.CardStyle, "bilinmeyen stil yok sayılmalı")
	assert.NotContains(t, string(b.CardCSS), "javascript", "geçersiz renk enjekte edilmemeli")
}

func TestKartStiliBoyutSinirlari(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`"card_title_size":9`, ""}, // alt sınır altı → yok sayılır
		{`"card_title_size":10`, "--cs-size:10px"},
		{`"card_title_size":40`, "--cs-size:40px"},
		{`"card_title_size":41`, ""}, // üst sınır üstü → yok sayılır
	}
	for _, c := range cases {
		b := parseTekBlok(t, `{"card_style":"kose",`+c.in+`}`)
		if c.want == "" {
			assert.NotContains(t, string(b.CardCSS), "--cs-size", "girdi: "+c.in)
		} else {
			assert.Contains(t, string(b.CardCSS), c.want, "girdi: "+c.in)
		}
	}
}

func TestTumKartStilleriJSONKabul(t *testing.T) {
	for _, st := range []string{
		"bloomberg", "bloomberg-cizgi", "bant-alt", "vurgu",
		"cerceve", "kose", "cam", "alt-cizgi",
	} {
		b := parseTekBlok(t, `{"card_style":"`+st+`"}`)
		assert.Equal(t, st, b.CardStyle, "stil kabul edilmeli: "+st)
	}
	// Boş = klasik
	b := parseTekBlok(t, `{"card_style":""}`)
	assert.Empty(t, b.CardStyle)
}
