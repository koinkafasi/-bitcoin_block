package httpapi

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

func TestTakimKisaReferansKodlari(t *testing.T) {
	cases := map[string]string{
		"Galatasaray":              "GS",
		"Fenerbahçe":               "FB",
		"Beşiktaş":                 "BJK",
		"Trabzonspor":              "TS",
		"Kasımpaşa":                "KSM",
		"Göztepe":                  "GÖZ",
		"Konyaspor":                "KON",
		"Samsunspor":               "SAMS",
		"Eyüpspor":                 "EYP",
		"Alanyaspor":               "ALNY",
		"İstanbul Başakşehir":      "İBFK",
		"Çaykur Rizespor":          "ÇRZ",
		"Gaziantep FK":             "GFK",
		"Kocaelispor":              "KOC",
		"Amed Sportif Faaliyetler": "ASF",
		"Erzurumspor FK":           "EFK",
		"Gençlerbirliği":           "GB",
	}
	for ad, want := range cases {
		assert.Equal(t, want, takimKisa(ad), ad)
	}
}

func TestTakimKisaBilinmeyenTakimGenelKural(t *testing.T) {
	// Tabloda olmayan takım: kelime baş harfleri (≤4) ya da ilk üç harf.
	assert.Equal(t, "ZZT", takimKisa("Zztrspor"))
	assert.Equal(t, "ABC", takimKisa("Alfa Beta Centauri"))
	assert.Equal(t, "", takimKisa(""))
}

func TestTakimKisaTurkceDuyarli(t *testing.T) {
	// Büyük/küçük harf ve aksan farkı eşleşmeyi bozmamalı.
	assert.Equal(t, "GS", takimKisa("GALATASARAY"))
	assert.Equal(t, "BJK", takimKisa("besiktas"))
	assert.Equal(t, "GS", takimKisa("  Galatasaray  "))
}

// ---- Şablon ----

func pssKalemler() []repo.StandingRow {
	return []repo.StandingRow{
		{Rank: 1, Team: "Galatasaray", Badge: "/takim-logo/e-432.png", Points: 22},
		{Rank: 2, Team: "Amed Sportif Faaliyetler", Badge: "", Points: 20},
		{Rank: 3, Team: "Trabzonspor", Badge: "/takim-logo/e-436.png", Points: 19},
	}
}

func renderPuanSerit(t *testing.T, rows []repo.StandingRow, baslik string, hideHeading bool) string {
	t.Helper()
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	tmpl, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)

	b := &homeBlock{
		Type:        "puan_serit",
		D:           &tmplData{},
		Standings:   rows,
		PuanBaslik:  baslik,
		HideHeading: hideHeading,
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_puan_serit", b))
	return buf.String()
}

func TestPuanSeritYapisi(t *testing.T) {
	h := renderPuanSerit(t, pssKalemler(), "PUAN DURUMU", false)

	assert.Contains(t, h, `class="pss-block"`)
	assert.Contains(t, h, `class="pss-head"`)
	assert.Contains(t, h, `<h2 class="pss-title">PUAN DURUMU</h2>`)
	assert.Contains(t, h, `class="pss-rule"`, "başlığın yanında yatay çizgi")
	assert.Contains(t, h, `class="pss-scroll" data-pys-scroll`, "yatay kaydırma (sürüklenebilir)")
	assert.Equal(t, 3, bytes.Count([]byte(h), []byte(`class="pss-card"`)))
	// Logo + kısa kod + puan
	assert.Contains(t, h, `src="/takim-logo/e-432.png"`)
	assert.Contains(t, h, `<span class="pss-code">GS</span>`)
	assert.Contains(t, h, `<span class="pss-pts">22</span>`)
	assert.Contains(t, h, `<span class="pss-code">ASF</span>`)
	// Logosuz takımda kısa kod yer tutucu olarak basılır (boş kutu yok)
	assert.Contains(t, h, `class="pss-logo-yok"`)
}

func TestPuanSeritBaslikGizlenebilir(t *testing.T) {
	h := renderPuanSerit(t, pssKalemler(), "PUAN DURUMU", true)
	assert.NotContains(t, h, `class="pss-head"`, "başlık gizlenince başlık bloğu basılmaz")
	assert.Contains(t, h, `class="pss-scroll"`, "kartlar yine basılır")
}

func TestPuanSeritVeriYoksaBasilmaz(t *testing.T) {
	h := renderPuanSerit(t, nil, "", false)
	assert.Empty(t, h, "puan verisi yoksa modül hiç basılmamalı")
}

func TestPuanSeritPaletteVar(t *testing.T) {
	for _, page := range []string{"home", "article"} {
		var bulundu bool
		for _, tp := range builderTypesFor(page) {
			if tp.Type == "puan_serit" {
				bulundu = true
				assert.Equal(t, "Puan Durumu Şeridi", tp.Name)
			}
		}
		assert.True(t, bulundu, "%s paletinde puan_serit olmalı", page)
	}
}
