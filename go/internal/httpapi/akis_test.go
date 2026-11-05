package httpapi

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// akisItems sahte kategori haberleri üretir (ID 1..n).
func akisItems(n int) []repo.Content {
	items := make([]repo.Content, n)
	for i := range items {
		items[i] = repo.Content{
			ID:    int64(i + 1),
			Title: fmt.Sprintf("Enerji haberi %d", i+1),
		}
	}
	return items
}

func TestBuildAkisBatchesCardOrderAndTypes(t *testing.T) {
	batches := buildAkisBatches(akisItems(12), "/tr", false)
	require.Len(t, batches, 1)
	require.Len(t, batches[0], 3, "enerji vitrini 3 sütundan oluşmalı")

	for col, want := range akisColumnTypes {
		require.Len(t, batches[0][col], 4, "her sütunda 4 kart olmalı")
		for row, typ := range want {
			assert.Equal(t, typ, batches[0][col][row].Type, "sütun %d kart %d tipi", col, row)
		}
	}
	// Kartlar sütun-satır sırasıyla doldurulur (col*4+row).
	assert.Equal(t, int64(1), batches[0][0][0].C.ID)
	assert.Equal(t, int64(4), batches[0][0][3].C.ID)
	assert.Equal(t, int64(5), batches[0][1][0].C.ID)
	assert.Equal(t, int64(12), batches[0][2][3].C.ID)
	assert.Equal(t, "/tr", batches[0][0][0].LangBase)
}

func TestBuildAkisBatchesMultipleBlocksAndPartial(t *testing.T) {
	batches := buildAkisBatches(akisItems(20), "", false)
	require.Len(t, batches, 2, "20 haber 12+8 olarak iki bloğa bölünmeli")

	// İkinci blokta eksik kartlar ID=0 ile gelir; şablon bunları atlar.
	assert.Equal(t, int64(13), batches[1][0][0].C.ID)
	assert.Equal(t, int64(18), batches[1][1][1].C.ID)
	assert.Zero(t, batches[1][2][0].C.ID)
	assert.Zero(t, batches[1][2][3].C.ID)
}

func TestApplyAkisFlowFirstFourAndNextCursor(t *testing.T) {
	b := &homeBlock{AkisLimit: 60, AkisPer: 24, AkisAuto: true}
	c := &categoryData{Items: akisItems(24), Page: 1, PerPage: 24, HasNext: true}
	applyAkisFlow(c, b, "")

	assert.Equal(t, 60, c.FlowLimit)
	assert.Equal(t, 24, c.FlowPer)
	assert.True(t, c.FlowAuto)
	require.Len(t, c.FlowBatches, 2, "ilk 4 dışındaki 20 haber 12+8 iki blok olmalı")
	assert.Equal(t, int64(5), c.FlowBatches[0][0][0].C.ID,
		"ilk 4 kart 4'lü ızgarada basılır, akış 5. haberle başlar")
	assert.Equal(t, 24, c.FlowNext, "sonraki yükleme 24. haberden devam etmeli")
	assert.True(t, c.FlowMore, "kategori devam ettiği için yükleme sürmeli")
}

func TestApplyAkisFlowRespectsLimit(t *testing.T) {
	b := &homeBlock{AkisLimit: 16, AkisPer: 12}
	c := &categoryData{Items: akisItems(24), Page: 1, PerPage: 24, HasNext: true}
	applyAkisFlow(c, b, "")

	assert.Equal(t, 16, c.FlowNext, "akış haber sınırında durmalı")
	require.Len(t, c.FlowBatches, 1, "ilk 4 + 12 haber tek blok etmeli")
	assert.False(t, c.FlowMore, "sınır dolduğunda yükleme bitmeli")
}

func TestApplyAkisFlowAllLoaded(t *testing.T) {
	b := &homeBlock{AkisLimit: 60, AkisPer: 12}
	c := &categoryData{Items: akisItems(9), Page: 1, PerPage: 24, HasNext: false}
	applyAkisFlow(c, b, "")

	assert.Equal(t, 9, c.FlowNext)
	assert.False(t, c.FlowMore, "tüm haberler yüklendiyse akış bitmeli")
	require.Len(t, c.FlowBatches, 1)
}

func TestCatListeAkisRendersFlow(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{ListView: "akis", AkisLimit: 60, AkisPer: 12, AkisAuto: true}
	d := &tmplData{LangBase: "", Category: &categoryData{
		Slug: "enerji", Items: akisItems(20), Page: 1, PerPage: 24, HasNext: true,
	}}
	fn, ok := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)
	require.True(t, ok)
	nd := fn(d, b)
	require.Equal(t, "akis", nd.Category.View)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()

	assert.Contains(t, out, `class="cat-cards cat-cols-4 ak-first"`,
		"ilk 4 haber 4 sütunlu ızgara olmalı")
	assert.Equal(t, 4, strings.Count(out, `class="card cat-card"`),
		"ilk ızgarada tam 4 kart olmalı")
	assert.Contains(t, out, `class="ak-flow"`)
	assert.Contains(t, out, `data-ak-flow`)
	assert.Contains(t, out, `data-ak-bas="20"`)
	assert.Contains(t, out, `data-ak-sinir="60"`)
	assert.Contains(t, out, `data-ak-oto="1"`)
	assert.Contains(t, out, `class="ak-grid"`, "enerji blokları basılmalı")
	assert.Contains(t, out, "ak-type5", "enerji kart tipleri kullanılmalı")
	assert.Contains(t, out, "ak-type2")
	assert.Contains(t, out, "data-ak-more", "elle yükleme düğmesi bulunmalı")
	assert.NotContains(t, out, `class="pager"`, "akışta sayfalama çubuğu olmamalı")
	assert.NotContains(t, out, "cat-hero", "akışta manşet düzeni basılmamalı")
}

func TestCatListeAkisWithoutMoreHidesButton(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{ListView: "akis", AkisLimit: 12, AkisPer: 12}
	d := &tmplData{LangBase: "", Category: &categoryData{
		Slug: "enerji", Items: akisItems(12), Page: 1, PerPage: 24, HasNext: false,
	}}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()
	assert.NotContains(t, out, "data-ak-more", "haber kalmadıysa yükleme düğmesi basılmamalı")
	assert.Contains(t, out, "Tüm haberler gösterildi")
}

// TestCatHeroEndMansetTuketimi manşet düzenlerinin akıştan önce tükettiği
// haber sayısını doğrular.
func TestCatHeroEndMansetTuketimi(t *testing.T) {
	items := heroCategoryItems(30)
	assert.Equal(t, 10, catHeroEnd(&categoryData{HeroStyle: "uclu", HeroCount: 5, Items: items}, 30),
		"üçlü: 5 slayt + 3 sol + 2 sağ = 10 haber")
	assert.Equal(t, 7, catHeroEnd(&categoryData{HeroStyle: "portre", HeroCount: 5, Items: items}, 30),
		"portre: 5 slayt + 2 yan = 7 haber")
	assert.Equal(t, 3, catHeroEnd(&categoryData{HeroStyle: "klasik", Items: items}, 30),
		"klasik: 1 büyük + 2 yan = 3 haber")
	assert.Equal(t, 3, catHeroEnd(&categoryData{Items: items}, 30),
		"stil seçilmemişse klasik varsayılır")
	assert.Equal(t, 0, catHeroEnd(&categoryData{HeroStyle: "uclu", HeroCount: 5}, 8),
		"yeterli haber yoksa manşet basılmaz")
	assert.Equal(t, 0, catHeroEnd(nil, 30))
}

// TestCatListeAkisWithTrioHero akış vitrininin üstünde üçlü manşet
// (3 + slider + 2) alanının basıldığını ve akışın manşetten sonra
// başladığını doğrular.
func TestCatListeAkisWithTrioHero(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_liste", ListView: "akis", CatHeroStyle: "uclu", CatHeroCount: 5,
		AkisLimit: 60, AkisPer: 12, AkisAuto: true, AkisHero: true,
	}
	d := &tmplData{LangBase: "", Category: &categoryData{
		Slug: "ekonomi", Items: heroCategoryItems(30), Page: 1, PerPage: 24, HasNext: true,
	}}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)
	require.Equal(t, "akis", nd.Category.View)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()

	assert.Contains(t, out, `class="cat-hero cat-hero-trio`, "akışın üstünde üçlü manşet alanı olmalı")
	assert.Equal(t, 1, strings.Count(out, `class="cat-hero-rail cat-trio-left"`), "solda 3 manşet rayı")
	assert.Equal(t, 1, strings.Count(out, `class="cat-hero-rail cat-trio-right"`), "sağda 2 manşet rayı")
	assert.Contains(t, out, `class="hero-slider cat-hero-slider"`, "ortada slider olmalı")

	// Manşet 10 haber tüketir: 4'lü kart satırı 11..14, akış 15. haberle başlar.
	assert.Equal(t, 4, strings.Count(out, `class="card cat-card"`),
		"manşetten sonra 4'lü kart satırı olmalı")
	assert.Contains(t, out, `data-ak-bas="30"`, "akış imleci manşetten sonraki haberden devam etmeli")
	assert.Equal(t, 2, strings.Count(out, `class="ak-grid"`),
		"akış 30-14=16 haber için iki enerji bloğu basmalı")
	assert.Equal(t, 16, strings.Count(out, `class="ak-item ak-type`), "akışta 16 kart olmalı")
	assert.NotContains(t, out, `class="pager"`)
}

// TestCatListeAkisHeroOff üst manşet kapatıldığında akışın ilk haberle
// başladığını doğrular.
func TestCatListeAkisHeroOff(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_liste", ListView: "akis", CatHeroStyle: "uclu", CatHeroCount: 5,
		AkisLimit: 60, AkisPer: 12, AkisHero: false,
	}
	d := &tmplData{Category: &categoryData{
		Slug: "ekonomi", Items: heroCategoryItems(30), Page: 1, PerPage: 24,
	}}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()
	assert.NotContains(t, out, "cat-hero", "manşet kapalıyken akışın üstünde manşet olmamalı")
	assert.Equal(t, 4, strings.Count(out, `class="card cat-card"`), "ilk 4 haber kart ızgarasında")
	assert.Equal(t, 26, strings.Count(out, `class="ak-item ak-type`), "akış 30-4=26 haberle başlamalı")
}

// TestCatListeAkisFont tek yazı ailesinin tüm akışa (ilk 4 kart + enerji
// blokları) uygulandığını ve geri dönüşün (coinhit serif) mümkün olduğunu
// doğrular.
func TestCatListeAkisFont(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := &tmplData{Category: &categoryData{
		Slug: "enerji", Items: heroCategoryItems(20), Page: 1, PerPage: 24,
	}}

	// Seçili font: tüm akış tek CSS değişkeniyle değişir.
	b := &homeBlock{ListView: "akis", AkisLimit: 60, AkisPer: 12, AkisFont: "mono"}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)
	require.Contains(t, string(nd.Category.FlowFontStyle), "--ak-font:")
	assert.Contains(t, string(nd.Category.FlowFontStyle), "monospace")

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()
	assert.Contains(t, out, `style="--ak-font:ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;"`,
		"akış fontu hem kart satırına hem enerji bloklarına uygulanmalı")
	assert.Equal(t, 2, strings.Count(out, "--ak-font:"),
		"tek ayar: kart satırı + akış konteynerinde aynı değişken")

	// Font seçilmezse tema başlık fontu (slider ile aynı) kullanılır.
	nd2 := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d,
		&homeBlock{ListView: "akis", AkisLimit: 60, AkisPer: 12})
	assert.Empty(t, string(nd2.Category.FlowFontStyle), "boş seçimde ek stil basılmamalı")
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd2))
	assert.NotContains(t, buf.String(), "--ak-font:",
		"boş seçimde tema başlık fontuna (slider ile aynı) dönülmeli")

	// Coinhit serif'i seçilebilir; istenen an geri dönülür.
	nd3 := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d,
		&homeBlock{ListView: "akis", AkisLimit: 60, AkisPer: 12, AkisFont: "serif"})
	assert.Contains(t, string(nd3.Category.FlowFontStyle), "--ak-font:Georgia")
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd3))
	assert.Contains(t, buf.String(), `--ak-font:Georgia, Times New Roman, Times, serif;`,
		"coinhit serif'i seçilebilmeli")

	// Geçersiz font değeri yok sayılır.
	nd4 := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d,
		&homeBlock{ListView: "akis", AkisLimit: 60, AkisPer: 12, AkisFont: "yok-boyle-font"})
	assert.Empty(t, string(nd4.Category.FlowFontStyle))
}
