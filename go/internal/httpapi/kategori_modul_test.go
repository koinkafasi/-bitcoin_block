package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCatItemSlice aralık dışına taşan isteklerin güvenli biçimde
// kırpıldığını doğrular.
func TestCatItemSlice(t *testing.T) {
	items := akisItems(5)
	assert.Len(t, catItemSlice(items, 0, 3), 3)
	assert.Equal(t, int64(3), catItemSlice(items, 2, 2)[0].ID)
	assert.Len(t, catItemSlice(items, 4, 4), 1)
	assert.Empty(t, catItemSlice(items, 5, 2))
	assert.Len(t, catItemSlice(items, -3, 2), 2, "negatif başlangıç 0'a çekilir")
	assert.Empty(t, catItemSlice(items, 0, 0))
}

// TestAssignCategoryModuleItemsExplicit siyaset dönüşümündeki açık
// aralıkları doğrular: slider 0-9, sol 10-12, sağ 13-14, 4'lü 15-18,
// akış 19'dan başlar.
func TestAssignCategoryModuleItemsExplicit(t *testing.T) {
	items := akisItems(25)
	blocks := []homeBlock{
		{Type: "kategori_slider", Count: 10, ItemOffset: 0},
		{Type: "kategori_secimli", Count: 3, ItemOffset: 10},
		{Type: "kategori_secimli", Count: 2, ItemOffset: 13},
		{Type: "kategori_4lu", Count: 4, ItemOffset: 15},
		{Type: "kategori_akis", ItemOffset: 19},
	}
	assignCategoryModuleItems(blocks, items)

	require.Len(t, blocks[0].CatItems, 10)
	assert.Equal(t, int64(1), blocks[0].CatItems[0].ID)
	assert.Equal(t, int64(10), blocks[0].CatItems[9].ID)
	require.Len(t, blocks[1].CatItems, 3)
	assert.Equal(t, int64(11), blocks[1].CatItems[0].ID)
	require.Len(t, blocks[2].CatItems, 2)
	assert.Equal(t, int64(14), blocks[2].CatItems[0].ID)
	require.Len(t, blocks[3].CatItems, 4)
	assert.Equal(t, int64(16), blocks[3].CatItems[0].ID)
	assert.Equal(t, 19, blocks[4].AkisStart)
	assert.True(t, blocks[4].AkisStartSet)
}

// TestAssignCategoryModuleItemsOutOfRange aralık dışı açık başlangıcın
// modülü boş bıraktığını, tam listeye geri düşmediğini doğrular.
func TestAssignCategoryModuleItemsOutOfRange(t *testing.T) {
	items := akisItems(5)
	blocks := []homeBlock{{Type: "kategori_4lu", Count: 4, ItemOffset: 50}}
	assignCategoryModuleItems(blocks, items)
	assert.NotNil(t, blocks[0].CatItems, "boş dilim nil olmamalı")
	assert.Empty(t, blocks[0].CatItems, "aralık dışı modül boş kalmalı")
}

// TestAssignCategoryModuleItemsAuto boş bırakılan modüllerin sırayla ilk
// kullanılmamış haberlerden beslendiğini doğrular.
func TestAssignCategoryModuleItemsAuto(t *testing.T) {
	items := akisItems(20)
	blocks := []homeBlock{
		{Type: "kategori_slider", Count: 5, ItemOffset: -1},
		{Type: "kategori_4lu", Count: 4, ItemOffset: -1},
		{Type: "kategori_akis", ItemOffset: -1},
	}
	assignCategoryModuleItems(blocks, items)

	require.Len(t, blocks[0].CatItems, 5)
	assert.Equal(t, int64(1), blocks[0].CatItems[0].ID)
	require.Len(t, blocks[1].CatItems, 4)
	assert.Equal(t, int64(6), blocks[1].CatItems[0].ID)
	assert.Equal(t, 9, blocks[2].AkisStart)
	assert.True(t, blocks[2].AkisStartSet)
}

// TestAssignCategoryModuleItemsSkipsUsed açık aralıkların otomatik
// modüllerden önce tüketildiğini doğrular.
func TestAssignCategoryModuleItemsSkipsUsed(t *testing.T) {
	items := akisItems(12)
	blocks := []homeBlock{
		{Type: "kategori_secimli", Count: 2, ItemOffset: 4},
		{Type: "kategori_slider", Count: 3, ItemOffset: -1},
	}
	assignCategoryModuleItems(blocks, items)

	require.Len(t, blocks[0].CatItems, 2)
	assert.Equal(t, int64(5), blocks[0].CatItems[0].ID)
	require.Len(t, blocks[1].CatItems, 3)
	assert.Equal(t, int64(1), blocks[1].CatItems[0].ID)
}

// TestAssignCategoryModuleItemsAutoAkisAfterExplicit açık aralıklı
// modüllerden sonra gelen otomatik akışın kullanılmış haberleri atladığını
// doğrular; aksi halde slider'daki haberler akışta tekrar basılırdı.
func TestAssignCategoryModuleItemsAutoAkisAfterExplicit(t *testing.T) {
	items := akisItems(20)
	blocks := []homeBlock{
		{Type: "kategori_slider", Count: 5, ItemOffset: 0},
		{Type: "kategori_4lu", Count: 4, ItemOffset: 10},
		{Type: "kategori_akis", ItemOffset: -1},
	}
	assignCategoryModuleItems(blocks, items)
	require.Len(t, blocks[0].CatItems, 5)
	require.Len(t, blocks[1].CatItems, 4)
	assert.Equal(t, 14, blocks[2].AkisStart, "akış kullanılmamış ilk haberden başlamalı")
	assert.True(t, blocks[2].AkisStartSet)
}

// TestAssignCategoryModuleItemsAfterExplicitAkis açık aralıklı akışın
// sonraki otomatik modüllere haber bırakmadığını doğrular; aksi halde aynı
// haberler akışta ve ızgarada iki kez basılırdı.
func TestAssignCategoryModuleItemsAfterExplicitAkis(t *testing.T) {
	items := akisItems(20)
	blocks := []homeBlock{
		{Type: "kategori_slider", Count: 5, ItemOffset: 0},
		{Type: "kategori_akis", ItemOffset: 5},
		{Type: "kategori_4lu", Count: 4, ItemOffset: -1},
	}
	assignCategoryModuleItems(blocks, items)
	require.Len(t, blocks[0].CatItems, 5)
	assert.Equal(t, 5, blocks[1].AkisStart)
	assert.True(t, blocks[1].AkisStartSet)
	assert.Empty(t, blocks[2].CatItems, "akış aralığı sonraki otomatik modüle bırakılmamalı")
}

// TestGroupCatRows hero-* modüllerinin tek satırda birleştiğini, araya giren
// normal blokların satırı böldüğünü ve orta slider ölçüsünün satıra
// taşındığını doğrular.
func TestGroupCatRows(t *testing.T) {
	blocks := []homeBlock{
		{Type: "manset2", Zone: "govde", Visible: true},
		{Type: "kategori_slider", Zone: "govde", Visible: true, RowSlot: "center", CatHeroWidth: 44, CatHeroRatio: "4/5"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, RowSlot: "left"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, RowSlot: "right"},
		{Type: "kategori_4lu", Zone: "govde", Visible: true},
		{Type: "kategori_akis", Zone: "govde", Visible: true},
	}
	rows := groupCatRows(blocks)
	require.Len(t, rows, 4)

	assert.False(t, rows[0].Group)
	assert.Equal(t, "manset2", rows[0].Blocks[0].Type)

	require.True(t, rows[1].Group)
	assert.Contains(t, rows[1].Class, "cat-row-lcr")
	assert.Contains(t, rows[1].Class, "cat-ratio-4-5")
	assert.Contains(t, string(rows[1].Style), "--cat-hero-w:44%")
	require.Len(t, rows[1].Blocks, 3)

	assert.False(t, rows[2].Group)
	assert.Equal(t, "kategori_4lu", rows[2].Blocks[0].Type)
	assert.False(t, rows[3].Group)
	assert.Equal(t, "kategori_akis", rows[3].Blocks[0].Type)
}

// TestGroupCatRowsDuplicateSlot aynı yuvanın ikinci kez gelmesinin yeni
// satır başlattığını doğrular.
func TestGroupCatRowsDuplicateSlot(t *testing.T) {
	blocks := []homeBlock{
		{Type: "kategori_secimli", Zone: "govde", Visible: true, RowSlot: "left"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, RowSlot: "left"},
	}
	rows := groupCatRows(blocks)
	require.Len(t, rows, 2)
	assert.Contains(t, rows[0].Class, "cat-row-l")
	assert.Contains(t, rows[1].Class, "cat-row-l")
}

// TestGroupCatRowsVariants tüm yuva bileşimlerinin doğru satır sınıfını
// ürettiğini doğrular. Sınıf adları CSS'teki grid tanımlarıyla birebirdir.
func TestGroupCatRowsVariants(t *testing.T) {
	slot := func(name string) homeBlock {
		return homeBlock{Type: "kategori_secimli", Zone: "govde", Visible: true, RowSlot: name}
	}
	cases := []struct {
		name  string
		slots []string
		class string
	}{
		{"üçlü", []string{"left", "center", "right"}, "cat-row-lcr"},
		{"sol+orta", []string{"left", "center"}, "cat-row-lc"},
		{"orta+sağ", []string{"center", "right"}, "cat-row-cr"},
		{"sol+sağ", []string{"left", "right"}, "cat-row-lr"},
		{"yalnız orta", []string{"center"}, "cat-row-c"},
		{"yalnız sol", []string{"left"}, "cat-row-l"},
		{"yalnız sağ", []string{"right"}, "cat-row-r"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blocks := make([]homeBlock, 0, len(tc.slots))
			for _, s := range tc.slots {
				blocks = append(blocks, slot(s))
			}
			rows := groupCatRows(blocks)
			require.Len(t, rows, 1)
			require.True(t, rows[0].Group)
			assert.Contains(t, rows[0].Class, tc.class)
			assert.Len(t, rows[0].Blocks, len(tc.slots))
		})
	}
}

// TestGroupCatRowsIgnoresHiddenAndOtherZones gizli ya da gövde dışındaki
// modüllerin manşet satırına katılmadığını doğrular.
func TestGroupCatRowsIgnoresHiddenAndOtherZones(t *testing.T) {
	blocks := []homeBlock{
		{Type: "kategori_slider", Zone: "govde", Visible: true, RowSlot: "center"},
		{Type: "kategori_secimli", Zone: "govde", Visible: false, RowSlot: "left"},
		{Type: "kategori_secimli", Zone: "ust", Visible: true, RowSlot: "right"},
	}
	rows := groupCatRows(blocks)
	require.Len(t, rows, 1)
	assert.Contains(t, rows[0].Class, "cat-row-c")
	assert.Len(t, rows[0].Blocks, 1)
}

// TestBuilderTypesForPage kategori modüllerinin yalnız kategori sayfası
// paletinde göründüğünü, ana sayfa ve makale paletlerinde bulunmadığını
// doğrular.
func TestBuilderTypesForPage(t *testing.T) {
	hasType := func(list []blockType, want string) bool {
		for _, t := range list {
			if t.Type == want {
				return true
			}
		}
		return false
	}

	home := builderTypesFor("home")
	assert.False(t, hasType(home, "kategori_slider"))
	assert.False(t, hasType(home, "kategori_secimli"))
	assert.False(t, hasType(home, "kategori_4lu"))
	assert.False(t, hasType(home, "kategori_akis"))
	assert.True(t, hasType(home, "kategori_liste"), "eski liste ana sayfada kalır")

	cat := builderTypesFor("kategori")
	for _, ty := range []string{"kategori_slider", "kategori_secimli", "kategori_4lu", "kategori_akis", "kategori_liste"} {
		assert.True(t, hasType(cat, ty), "kategori paletinde %s olmalı", ty)
	}

	specific := builderTypesFor("kategori:siyaset")
	assert.True(t, hasType(specific, "kategori_slider"))

	article := builderTypesFor("article")
	assert.False(t, hasType(article, "kategori_slider"))
	assert.True(t, hasType(article, "canli_yayin"))
}

// TestParseCategoryModuleBlocks yeni blok tiplerinin, satır yerleşiminin ve
// 0 değerli item_offset'in ayrıştırıldığını doğrular.
func TestParseCategoryModuleBlocks(t *testing.T) {
	raw := []byte(`[
		{"id":"a","type":"kategori_slider","visible":true,"opts":{"zone":"govde","count":10,"row_layout":"hero-center","item_offset":0,"hero_width":44,"hero_ratio":"4/5"}},
		{"id":"b","type":"kategori_secimli","visible":true,"opts":{"zone":"govde","count":3,"row_layout":"hero-left","item_offset":10,"rail_style":"overlay"}},
		{"id":"c","type":"kategori_4lu","visible":true,"opts":{"zone":"govde","count":4,"item_offset":15,"list_cols":4}},
		{"id":"d","type":"kategori_akis","visible":true,"opts":{"zone":"govde","item_offset":19,"akis_limit":60,"akis_per":12}}
	]`)
	blocks := parseHomeBlocks(raw)
	require.Len(t, blocks, 4)

	assert.Equal(t, "hero-center", blocks[0].RowLayout)
	assert.Equal(t, "cat-slot-center", blocks[0].RowSlotClass)
	assert.Equal(t, 0, blocks[0].ItemOffset, "0 açık bir başlangıçtır")
	assert.Equal(t, 10, blocks[0].Count)
	assert.Equal(t, 44, blocks[0].CatHeroWidth)

	assert.Equal(t, "hero-left", blocks[1].RowLayout)
	assert.Equal(t, "cat-slot-left", blocks[1].RowSlotClass)
	assert.Equal(t, 10, blocks[1].ItemOffset)
	assert.Equal(t, "overlay", blocks[1].RailStyle)

	assert.Equal(t, 15, blocks[2].ItemOffset)
	assert.Equal(t, 4, blocks[2].CatListCols)
	assert.Equal(t, 19, blocks[3].ItemOffset)
	assert.Equal(t, 60, blocks[3].AkisLimit)
}

// TestApplyAkisFlowExplicitStart bağımsız akış modülünün 4'lü ızgarayı
// atlayıp doğrudan verilen sıradan başladığını doğrular.
func TestApplyAkisFlowExplicitStart(t *testing.T) {
	b := &homeBlock{AkisLimit: 60, AkisPer: 12, AkisAuto: true, AkisStart: 19, AkisStartSet: true}
	c := &categoryData{Items: akisItems(24), Page: 1, PerPage: 24, HasNext: true}
	applyAkisFlow(c, b, "")

	require.NotEmpty(t, c.FlowBatches)
	assert.Equal(t, int64(20), c.FlowBatches[0][0][0].C.ID,
		"akış 19. sıradaki haberden (ID 20) başlamalı")
	assert.Equal(t, 24, c.FlowNext)
	assert.True(t, c.FlowMore)
}

// TestCatGovdeRendersModules dört bağımsız modülün tek gövdede, doğru
// yuvalarda ve tekrar etmeden basıldığını doğrular.
func TestCatGovdeRendersModules(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := akisItems(24)
	blocks := []homeBlock{
		{Type: "kategori_slider", Zone: "govde", Visible: true, Count: 10, ItemOffset: 0, RowSlot: "center", RowSlotClass: "cat-slot-center", CatHeroWidth: 44, CatHeroRatio: "1/1", CatHeroStyle: "uclu", PagerShape: "round", PagerPos: "center"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, Count: 3, ItemOffset: 10, RowSlot: "left", RowSlotClass: "cat-slot-left", RailStyle: "overlay"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, Count: 2, ItemOffset: 13, RowSlot: "right", RowSlotClass: "cat-slot-right", RailStyle: "overlay"},
		{Type: "kategori_4lu", Zone: "govde", Visible: true, Count: 4, ItemOffset: 15, CatListCols: 4, CatHeroStyle: "portre"},
		{Type: "kategori_akis", Zone: "govde", Visible: true, ItemOffset: 19, AkisLimit: 60, AkisPer: 12, AkisAuto: true, CatHeroStyle: "portre"},
	}
	assignCategoryModuleItems(blocks, items)
	d := &tmplData{
		LangBase: "", Blocks: blocks,
		Category: &categoryData{Slug: "siyaset", Items: items, Page: 1, PerPage: 24, HasNext: true},
	}
	for i := range blocks {
		blocks[i].D = d
	}
	d.CatRows = groupCatRows(blocks)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_govde", d))
	out := buf.String()

	assert.Contains(t, out, `class="cat-hero cat-row-lcr cat-ratio-1-1"`,
		"slider ve iki ray tek manşet satırında birleşmeli")
	assert.Contains(t, out, `--cat-hero-w:44%`)
	assert.Contains(t, out, `class="hero-slider cat-hero-slider cat-ratio-1-1 cat-slot-center"`)
	assert.Contains(t, out, `cat-hero-rail cat-slot-left cat-rail-overlay`)
	assert.Contains(t, out, `cat-hero-rail cat-slot-right cat-rail-overlay`)
	assert.Contains(t, out, `cat-cards cat-cols-4 ak-first`)
	assert.Contains(t, out, `data-ak-bas="24"`, "bir sonraki yükleme 24. haberden sürmeli")
	assert.Contains(t, out, `data-ak-sinir="60"`)

	// Tekrar yok: slider 1-10, sol 11-13, sağ 14-15, 4'lü 16-19, akış 20-24.
	assert.Contains(t, out, "Enerji haberi 1</h2>")
	assert.NotContains(t, out, "Enerji haberi 11</h2>")
	assert.Contains(t, out, "Enerji haberi 11</span>")
	assert.Contains(t, out, "Enerji haberi 14</span>")
	assert.Contains(t, out, "Enerji haberi 16</h3>")
	assert.Contains(t, out, "Enerji haberi 20")
}

// TestRenderBlockPointerCategoryModules tasarımcı önizlemesinin kullandığı
// doğrudan render_block çağrısını doğrular. Şablonlar `$` üzerinden
// withCatBlock'a *homeBlock beklediğinden blok adresi verilmelidir; değer
// verilirse "wrong type for value" hatası oluşur.
func TestRenderBlockPointerCategoryModules(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := akisItems(20)
	blocks := []homeBlock{
		{Type: "kategori_slider", Zone: "govde", Visible: true, Count: 5, ItemOffset: 0, RowSlot: "center", RowSlotClass: "cat-slot-center", CatHeroWidth: 44, CatHeroRatio: "1/1"},
		{Type: "kategori_secimli", Zone: "govde", Visible: true, Count: 3, ItemOffset: 5, RowSlot: "left", RowSlotClass: "cat-slot-left", RailStyle: "overlay"},
		{Type: "kategori_4lu", Zone: "govde", Visible: true, Count: 4, ItemOffset: 8, CatListCols: 4},
		{Type: "kategori_akis", Zone: "govde", Visible: true, ItemOffset: 12, AkisLimit: 60, AkisPer: 12},
	}
	assignCategoryModuleItems(blocks, items)
	d := &tmplData{LangBase: "", Category: &categoryData{Slug: "siyaset", Items: items, Page: 1, PerPage: 24, HasNext: true}}
	markers := []string{"hero-slide", "cat-hero-rail", "cat-cards", "ak-flow"}
	for i := range blocks {
		blocks[i].D = d
		var buf bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", &blocks[i]),
			"blok adresiyle render edilebilmeli: %s", blocks[i].Type)
		assert.Contains(t, buf.String(), markers[i], "%s önizlemesi dolu olmalı", blocks[i].Type)
	}
}
