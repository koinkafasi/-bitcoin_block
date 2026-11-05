package httpapi

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func int64Ptr(v int64) *int64 { return &v }

func TestKategoriPanelParseDefaults(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"p1","type":"kategori_panel","opts":{"kategori":"gastro"}}]`))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, "gastro", b.CatSlug)
	assert.Equal(t, 9, b.Count, "varsayılan manşet dışı haber sayısı 9")
	assert.Equal(t, "gastro", b.PanelDuzen, "gastro varsayılan düzeni")
	assert.Equal(t, "kirmizi", b.PanelTone)
}

func TestKategoriPanelParseExplicitAndInvalid(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[
		{"id":"p1","type":"kategori_panel","opts":{"kategori":"ekonomi","count":12,"duzen":"ekonomi","tone":"mavi"}},
		{"id":"p2","type":"kategori_panel","opts":{"kategori":"siyaset","count":99,"duzen":"gecersiz","tone":"gecersiz"}}
	]`))
	require.Len(t, blocks, 2)
	assert.Equal(t, 12, blocks[0].Count)
	assert.Equal(t, "ekonomi", blocks[0].PanelDuzen)
	assert.Equal(t, "mavi", blocks[0].PanelTone)
	assert.Equal(t, 30, blocks[1].Count, "count üst sınırı 30")
	assert.Equal(t, "solbuyuk", blocks[1].PanelDuzen, "geçersiz düzen kategori varsayılanına döner")
	assert.Equal(t, "kirmizi", blocks[1].PanelTone, "geçersiz renk kategori varsayılanına döner")
}

func TestCatBlockFromItems(t *testing.T) {
	items := []repo.Content{
		{ID: 1, Title: "kapaksız"},
		{ID: 2, Title: "manşet", CoverMediaID: int64Ptr(10)},
		{ID: 3, Title: "ikinci", CoverMediaID: int64Ptr(11)},
		{ID: 4, Title: "üçüncü"},
	}
	b, ok := catBlockFromItems("Ekonomi", "ekonomi", items, "ekonomi", "mavi", 2)
	require.True(t, ok)
	assert.Equal(t, int64(2), b.Lead.ID, "kapaklı ilk haber manşet olmalı")
	assert.Equal(t, "mavi", b.Tone)
	assert.Equal(t, "ekonomi", b.Duzen)
	require.Len(t, b.Items, 2)
	assert.Equal(t, int64(1), b.Items[0].ID)
	assert.Equal(t, int64(3), b.Items[1].ID)

	// Kapaklı manşet yoksa panel üretilmez.
	_, ok = catBlockFromItems("Ekonomi", "ekonomi", []repo.Content{{ID: 1}, {ID: 2}, {ID: 3}}, "ekonomi", "mavi", 2)
	assert.False(t, ok)

	// Manşet dışında iki kart kalmadıysa panel üretilmez.
	_, ok = catBlockFromItems("Ekonomi", "ekonomi", []repo.Content{
		{ID: 1, CoverMediaID: int64Ptr(1)}, {ID: 2},
	}, "ekonomi", "mavi", 2)
	assert.False(t, ok)
}

func TestKategoriPanelRendersSinglePanel(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_panel", HeadingStyle: "line",
		CatBlocks: []catBlock{{
			Name: "EKONOMİ", Slug: "ekonomi", Tone: "mavi", Duzen: "ekonomi",
			Lead: repo.Content{ID: 1, Title: "Büyük manşet"},
			Items: []repo.Content{
				{ID: 2, Title: "Küçük 1"}, {ID: 3, Title: "Küçük 2"},
				{ID: 4, Title: "Küçük 3"}, {ID: 5, Title: "Küçük 4"},
			},
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "kat-panel-blok mavi kp-ht")
	assert.Contains(t, out, "Büyük manşet")
	assert.Contains(t, out, "Küçük 4")
}

func TestExpandCategoryPanelBlocks(t *testing.T) {
	raw := []byte(`[
		{"id":"b1","type":"slider","opts":{"zone":"ust"}},
		{"id":"b-kat1","type":"kategori_bloklari","visible":true,"locked":true,"opts":{"zone":"genis","head_style":"line","hide_badge":true}},
		{"id":"b2","type":"galeri_video","opts":{"zone":"genis"}}
	]`)
	v := &panelsView{Adet: 9, HasCustom: true, Satirlar: []panelSatir{
		{Slug: "gundem", Name: "Gündem", Duzen: "ust3alt2", Tone: "kirmizi", Show: true},
		{Slug: "ekonomi", Name: "Ekonomi", Duzen: "ekonomi", Tone: "mavi", Show: true},
		{Slug: "spor", Name: "Spor", Duzen: "mozaik", Tone: "kirmizi", Show: false},
	}}
	out := expandCategoryPanelBlocks(raw, v)
	blocks := parseHomeBlocks(out)
	require.Len(t, blocks, 4, "slider + 2 panel + galeri")
	assert.Equal(t, "slider", blocks[0].Type)
	assert.Equal(t, "kategori_panel", blocks[1].Type)
	assert.Equal(t, "gundem", blocks[1].CatSlug)
	assert.Equal(t, "ust3alt2", blocks[1].PanelDuzen)
	assert.Equal(t, "kirmizi", blocks[1].PanelTone)
	assert.Equal(t, 9, blocks[1].Count)
	assert.Equal(t, "genis", blocks[1].Zone, "eski bloğun bölgesi korunmalı")
	assert.True(t, blocks[1].Visible)
	assert.Equal(t, "ekonomi", blocks[2].CatSlug)
	assert.Equal(t, "mavi", blocks[2].PanelTone)
	assert.Equal(t, "galeri_video", blocks[3].Type)
	for _, b := range blocks {
		assert.NotEqual(t, "spor", b.CatSlug, "gizli kategori modüle dönüşmemeli")
	}
}

func TestExpandCategoryPanelBlocksNoCustomLimit(t *testing.T) {
	rows := make([]panelSatir, 0, 8)
	for i := 0; i < 8; i++ {
		rows = append(rows, panelSatir{Slug: "k" + strconv.Itoa(i), Duzen: "mozaik", Tone: "kirmizi", Show: true})
	}
	v := &panelsView{Adet: 9, HasCustom: false, Satirlar: rows}
	raw := []byte(`[{"id":"legacy","type":"kategori_bloklari","opts":{"zone":"genis"}}]`)
	blocks := parseHomeBlocks(expandCategoryPanelBlocks(raw, v))
	require.Len(t, blocks, 6, "özel ayar yokken homeCatBlocks gibi 6 panel")
}

func TestExpandCategoryPanelBlocksIdempotent(t *testing.T) {
	raw := []byte(`[{"id":"p1","type":"kategori_panel","opts":{"kategori":"gundem"}},{"id":"legacy","type":"kategori_bloklari","opts":{}}]`)
	v := &panelsView{Adet: 9, HasCustom: true, Satirlar: []panelSatir{
		{Slug: "gundem", Duzen: "ust3alt2", Tone: "kirmizi", Show: true},
	}}
	out := expandCategoryPanelBlocks(raw, v)
	assert.JSONEq(t, string(raw), string(out), "yeni blok varsa dönüşüm tekrar uygulanmamalı")
}
