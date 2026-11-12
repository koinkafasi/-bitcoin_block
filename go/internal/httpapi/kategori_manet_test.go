package httpapi

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestKategoriManetSliderOpts(t *testing.T) {
	raw := []byte(`[{
		"id":"k-liste","type":"kategori_liste","visible":true,
		"opts":{
			"zone":"govde","hero_style":"uclu","list_cols":"4",
			"overlay_gradient":"bottom","overlay_color":"#112233",
			"overlay_opacity_preset":"45","overlay_height":55,"overlay_blur":6,
			"pager_shape":"outline","pager_pos":"center","pager_size":32,
			"pager_color":"#00ff00","hero_shadow":true,
			"title_color":"#ffcc00","title_font":"serif"
		}
	}]`)
	blocks := parseHomeBlocks(raw)
	require.Len(t, blocks, 1)
	b := blocks[0]

	assert.Equal(t, "uclu", b.CatHeroStyle)
	assert.Equal(t, "outline", b.PagerShape)
	assert.Equal(t, "center", b.PagerPos)
	assert.Equal(t, 32, b.PagerSize)
	assert.Equal(t, "#00ff00", b.PagerColor)
	assert.True(t, b.HeroShadow, "başlık gölgesi bloğa taşınmalı")

	overlay := string(b.OverlayStyle)
	assert.Contains(t, overlay, "linear-gradient(to top", "aşağıdan yukarı geçiş uygulanmalı")
	assert.Contains(t, overlay, "rgba(17,34,51,0.45)", "renk + saydamlık birlikte uygulanmalı")
	assert.Contains(t, overlay, "height:55%", "başlık arkası yüksekliği uygulanmalı")
	assert.Contains(t, overlay, "blur(6px)", "arka plan bulanıklığı uygulanmalı")

	title := string(b.TitleStyle)
	assert.Contains(t, title, "color:#ffcc00")
	assert.Contains(t, title, "font-family:Georgia", "serif font eşlemesi uygulanmalı")
}

// TestKategoriManetPagerDefaults sayı biçimi seçilmemiş kategori manşetinde
// göstergenin yuvarlak kaldığını (mevcut tasarım) doğrular.
func TestKategoriManetPagerDefaults(t *testing.T) {
	raw := []byte(`[{"id":"k","type":"kategori_liste","visible":true,
		"opts":{"zone":"govde","hero_style":"portre"}}]`)
	blocks := parseHomeBlocks(raw)
	require.Len(t, blocks, 1)
	assert.Equal(t, "round", blocks[0].PagerShape, "kategori manşetinde varsayılan yuvarlak olmalı")
	assert.Equal(t, "right", blocks[0].PagerPos)
	assert.False(t, blocks[0].HeroShadow, "gölge varsayılan olarak kapalı olmalı")
	assert.Empty(t, string(blocks[0].OverlayStyle), "ayar yoksa başlık arkası stili üretilmemeli")
}

func heroCategoryItems(n int) []repo.Content {
	items := make([]repo.Content, n)
	for i := range items {
		items[i] = repo.Content{ID: int64(i + 1), Title: fmt.Sprintf("Manşet %d", i+1)}
	}
	return items
}

// TestCatListeTrioUsesHeroSliderOpts üçlü manşet şablonunun stil, gölge ve
// sayı göstergesi ayarlarını DOM'a bastığını doğrular.
func TestCatListeTrioUsesHeroSliderOpts(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_liste", CatHeroStyle: "uclu", CatHeroCount: 5,
		CatHeroWidth: 44, CatHeroRatio: "4/5", CatListCols: 4,
		OverlayStyle: `background:linear-gradient(to top,rgba(17,34,51,0.45),transparent);height:55%;`,
		TitleStyle:   `color:#ffcc00;`,
		PagerShape:   "outline", PagerPos: "center", PagerSize: 32,
		PagerColor: "#00ff00", HeroShadow: true,
	}
	d := &tmplData{LangBase: "", Category: &categoryData{
		Slug: "ekonomi", Items: heroCategoryItems(12), HeroCount: 5, ListCols: 4,
	}}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)
	require.Equal(t, "uclu", nd.Category.HeroStyle)
	assert.True(t, nd.Category.HeroShadow)
	assert.Contains(t, string(nd.Category.HeroPagerStyle), "--hero-dot-size:32px")
	assert.Contains(t, string(nd.Category.HeroPagerStyle), "--pager-color:#00ff00")

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()

	assert.Contains(t, out, `background:linear-gradient(to top,rgba(17,34,51,0.45),transparent);height:55%;`,
		"başlık arkası stili manşet katmanına yazılmalı")
	assert.Contains(t, out, `class="hero-title has-shadow"`, "gölge sınıfı başlığa eklenmeli")
	assert.Contains(t, out, `style="color:#ffcc00;"`, "başlık yazı stili uygulanmalı")
	assert.Contains(t, out, "hero-dots--shape-outline", "çerçeveli sayı biçimi basılmalı")
	assert.Contains(t, out, "hero-dots--pos-center", "sayı konumu uygulanmalı")
	assert.Contains(t, out, "--hero-dot-size:32px", "sayı boyutu göstergede olmalı")
	assert.Contains(t, out, "--pager-color:#00ff00", "gösterge rengi uygulanmalı")
}

// TestCatListePortreDefaultHeroOpts portre manşetinin ayar verilmediğinde
// eski görünümü (yuvarlak, sağ konum, gölgesiz) koruduğunu doğrular.
func TestCatListePortreDefaultHeroOpts(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_liste", CatHeroStyle: "portre", CatHeroCount: 5,
		CatHeroWidth: 58, CatHeroRatio: "4/5", CatListCols: 2,
		PagerShape: "round", PagerPos: "right",
	}
	d := &tmplData{Category: &categoryData{Slug: "ekonomi", Items: heroCategoryItems(9)}}
	nd := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)(d, b)

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", nd))
	out := buf.String()
	assert.Contains(t, out, "hero-dots--shape-round hero-dots--pos-right")
	assert.Contains(t, out, `class="hero-title"`, "gölge sınıfı eklenmemeli")
	assert.NotContains(t, out, "hero-title has-shadow")
}
