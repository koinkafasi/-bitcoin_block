package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderConfigNormalize(t *testing.T) {
	c := HeaderConfig{
		Model: "bozuk", Skin: "neon", Width: "wide",
		Buttons: []HeaderButton{
			{Label: "  Abone  ", URL: "/abone", Style: "ghost"},
			{Label: "2", URL: "/2"}, {Label: "3", URL: "/3"},
			{Label: "4", URL: "/4"}, {Label: "5", URL: "/5"},
		},
	}
	c.normalize()
	assert.Equal(t, "model1", c.Model)
	assert.Equal(t, "auto", c.Skin)
	assert.Equal(t, "wide", c.Width)
	assert.Equal(t, "Abone", c.Buttons[0].Label)
	assert.Equal(t, "ghost", c.Buttons[0].Style)
	assert.Len(t, c.Buttons, 4)
}

func TestFooterConfigNormalizeLegacy(t *testing.T) {
	c := FooterConfig{Layout: "bozuk", LogoPos: "middle"}
	c.normalize()
	assert.Equal(t, "klasik", c.Layout)
	assert.Equal(t, "right", c.LogoPos)
	assert.True(t, c.ShowSocial)
	assert.True(t, c.ShowBrand)
	assert.True(t, c.ShowTagline)
	assert.True(t, c.ShowBottom)
}

func TestEnsureFooterRSSInCorporateColumn(t *testing.T) {
	c := FooterConfig{Cols: []FootCol{
		{Title: "Kurumsal", Links: []FootLink{{Label: "Künyemiz", URL: "/sayfa/kunye"}}},
	}}
	ensureFooterRSS(&c)
	found := false
	for _, col := range c.Cols {
		if col.Title != "Kurumsal" {
			continue
		}
		for _, l := range col.Links {
			if l.URL == "/rss-listesi" && l.Label == "RSS Akışları" {
				found = true
			}
		}
	}
	assert.True(t, found, "RSS Akışları Kurumsal sütununa eklenmeli")

	// Aynı bağlantı ikinci kez eklenmemeli.
	ensureFooterRSS(&c)
	count := 0
	for _, col := range c.Cols {
		for _, l := range col.Links {
			if l.URL == "/rss-listesi" {
				count++
			}
		}
	}
	assert.Equal(t, 1, count)
}

func TestFooterSadeRenders(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := &tmplData{
		SiteName: "Test", SiteSlogan: "Slogan", Year: "2026", LangBase: "",
		HomeURL: "/", Brand: &BrandConfig{Mode: "text", Text: "Test"},
		Footer: &FooterConfig{
			Layout: "sade", Skin: "light", LogoPos: "center",
			ShowSocial: true, ShowBrand: true, ShowTagline: true, ShowBottom: false,
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "site_footer", d))
	out := buf.String()
	assert.Contains(t, out, "foot-layout-sade")
	assert.Contains(t, out, "foot-skin-light")
	assert.Contains(t, out, "hp-logo-box")
}
