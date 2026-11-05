package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// ---- Modül kutu boşlukları: padding / margin / kart gap ----

func TestSanitizeBoxShorthand(t *testing.T) {
	cases := []struct {
		in        string
		allowAuto bool
		allowNeg  bool
		want      string
	}{
		{"1rem", false, false, "1rem"},
		{"10px 20px", false, false, "10px 20px"},
		{"1rem 0", false, false, "1rem 0px"},
		{"1rem 0 auto", true, true, "1rem 0px auto"},
		{"calc(1rem + 2px)", false, false, "calc(1rem + 2px)"},
		{"auto", false, false, ""},
		{"-1rem", false, false, ""},
		{"-1rem", true, true, "-1rem"},
		{"1 2 3 4 5", false, false, ""},
		{"url(evil)", false, false, ""},
		{"", false, false, ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, sanitizeBoxShorthand(c.in, c.allowAuto, c.allowNeg), "%q", c.in)
	}
}

func TestSanitizeLengthValue(t *testing.T) {
	assert.Equal(t, "1rem", sanitizeLengthValue("1rem", false, false))
	assert.Equal(t, "16px", sanitizeLengthValue("16", false, false))
	assert.Equal(t, "50%", sanitizeLengthValue("50%", false, false))
	assert.Equal(t, "", sanitizeLengthValue("auto", false, false))
	assert.Equal(t, "auto", sanitizeLengthValue("auto", true, false))
	assert.Equal(t, "", sanitizeLengthValue("-5px", false, false))
	assert.Equal(t, "-5px", sanitizeLengthValue("-5px", true, true))
	assert.Equal(t, "", sanitizeLengthValue("1rem;color:red", false, false))
}

func TestBuildStyleUzunlukVeKartGap(t *testing.T) {
	b := homeBlock{
		StPad:       "1rem",
		StMargin:    "0 auto",
		StGapTop:    "2rem",
		StGapBottom: "8px",
		StGapX:      "1rem",
		StGapY:      "12px",
	}
	b.buildStyle()
	s := string(b.Style)
	assert.True(t, b.Wrap, "uzunluk ayarı varsa sarmalayıcı gerekir")
	assert.Contains(t, s, "padding:1rem")
	assert.Contains(t, s, "margin:0 auto")
	assert.Contains(t, s, "margin-top:2rem")
	assert.Contains(t, s, "margin-bottom:8px")
	assert.Contains(t, s, "--blk-gap-x:1rem")
	assert.Contains(t, s, "--blk-gap-y:12px")
}

func TestParseHomeBlocksUzunlukOpts(t *testing.T) {
	in := `[{"id":"b-manset","type":"manset","opts":{"stPad":"1rem","stMargin":"1rem 0","stGapX":"1rem","stGapY":"2rem"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, "1rem", b.StPad)
	assert.Equal(t, "1rem 0px", b.StMargin)
	assert.Equal(t, "1rem", b.StGapX)
	assert.Equal(t, "2rem", b.StGapY)
	assert.Contains(t, string(b.Style), "--blk-gap-x:1rem")
	assert.Contains(t, string(b.Style), "--blk-gap-y:2rem")
}

// Eski düzenlerde boşluklar JSON sayısı olarak kayıtlıydı; bunlar px'e
// çevrilmeye devam etmeli (geriye uyum).
func TestParseHomeBlocksEskiSayisalBosluk(t *testing.T) {
	in := `[{"id":"b-manset","type":"manset","opts":{"stPad":12,"stGapTop":4,"stGapBottom":6}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, "12px", b.StPad)
	assert.Equal(t, "4px", b.StGapTop)
	assert.Equal(t, "6px", b.StGapBottom)
	assert.Contains(t, string(b.Style), "padding:12px")

	// 0 eski düzende "ayar yok" demekti; boş sayılmalı ve sarmalayıcı
	// üretilmemeli (public DOM değişmesin).
	zero := parseHomeBlocks([]byte(`[{"id":"b-manset","type":"manset","opts":{"stPad":0,"stGapTop":0}}]`))
	require.Len(t, zero, 1)
	assert.Equal(t, "", zero[0].StPad)
	assert.Equal(t, "", zero[0].StGapTop)
	assert.False(t, zero[0].Wrap)
}

func TestPageLayoutBlockGap(t *testing.T) {
	g := "1rem"
	v := pageLayoutViewFrom("home", pageLayoutConfig{BlockGap: &g}, true, false)
	assert.Equal(t, "1rem", v.BlockGap)

	bad := "url(x)"
	v = pageLayoutViewFrom("home", pageLayoutConfig{BlockGap: &bad}, true, false)
	assert.Equal(t, "", v.BlockGap, "geçersiz uzunluk boş olmalı")

	// Kategoriye özel kayıtta alan yoksa genel düzenden miraslanır.
	base := pageLayoutConfig{BlockGap: &g}
	m := mergePageLayout(base, pageLayoutConfig{})
	require.NotNil(t, m.BlockGap)
	assert.Equal(t, "1rem", *m.BlockGap)
}

// ---- Şablon ve panel bağlantıları ----

// cssRule, verilen seçicinin gövdesini döndürür (ilk eşleşme).
func cssRule(t *testing.T, css, sel string) string {
	t.Helper()
	i := strings.Index(css, sel+" {")
	require.Greater(t, i, 0, "%s kuralı bulunmalı", sel)
	j := strings.Index(css[i:], "}")
	require.Greater(t, j, 0, "%s kuralı kapanmalı", sel)
	return css[i : i+j]
}

func TestKartGapCSSDegiskenleri(t *testing.T) {
	raw, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	css := string(raw)

	// Modül kart ızgaraları blok bazlı değişkenleri tüketmeli. Varsayılan
	// değerler korunur; ayar yoksa görünüm değişmez.
	for _, sel := range []string{
		".grid-2", ".hp-grid-2", ".hp-grid-3", ".hero-row",
		".manset-grid", ".kat-grid", ".galeri-grid", ".kb-grid",
		".rss-grid", ".kari-izgara", ".kat-yap", ".kat-kucuk",
	} {
		assert.Contains(t, cssRule(t, css, sel), "--blk-gap-x",
			"%s kart boşluğu ayarlanabilmeli", sel)
		assert.Contains(t, cssRule(t, css, sel), "--blk-gap-y",
			"%s dikey boşluğu ayarlanabilmeli", sel)
	}

	// Sayfa şablonu: modüller arası boşluk bölge sarmalayıcılarına uygulanır.
	assert.Contains(t, css, ".blk-zone { display: contents; }")
	assert.Contains(t, css, ".pg-blk-gap .blk-zone,")
	assert.Contains(t, css, "gap: var(--pg-blk-gap, 0px);")
	assert.Contains(t, css, ".pg-blk-gap .home-side,")
}

func TestBlkZoneSablonlari(t *testing.T) {
	raw, err := fs.ReadFile(webui.FS, "templates/site_public.html")
	require.NoError(t, err)
	html := string(raw)

	// Ana sayfa, yazı ve kategori sayfalarında bölge sarmalayıcıları var.
	assert.Contains(t, html, `class="blk-zone blk-zone-ust"`)
	assert.Contains(t, html, `class="blk-zone blk-zone-genis"`)
	assert.Contains(t, html, `class="home-main blk-zone-main"`)

	// Modüller arası boşluk yalnız ayarlandığında devreye girer.
	assert.Contains(t, html, "pg-blk-gap")
	assert.Contains(t, html, "--pg-blk-gap:")
}

func TestPanelBlokGapAlanlari(t *testing.T) {
	raw, err := fs.ReadFile(webui.FS, "templates/site_builder.html")
	require.NoError(t, err)
	html := string(raw)

	// Modül ayarlarında kutu boşlukları uzunluk alanı olarak yer almalı.
	assert.Contains(t, html, "['stPad','İç dolgu (padding)','length'")
	assert.Contains(t, html, "['stMargin','Dış boşluk (margin)','length'")
	assert.Contains(t, html, "['stGapX','Kart yan boşluk (sütun gap)','length'")
	assert.Contains(t, html, "['stGapY','Kart dikey boşluk (satır gap)','length'")
	assert.Contains(t, html, "dataset.type==='length'")

	// Sayfa şablonu panelinde modüller arası boşluk alanı.
	assert.Contains(t, html, `id="plBlockGap"`)
	assert.Contains(t, html, "PAGELAYOUT.blockGap")
	assert.Contains(t, html, "savePageLayout({blockGap:v})")
}
