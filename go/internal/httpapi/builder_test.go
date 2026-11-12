package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// ---- sayfa tasarımcısı: sayfa anahtarı ---------------------------

func TestBuilderPage(t *testing.T) {
	cases := []struct {
		name string
		q    string
		want string
	}{
		{"home", "home", "home"},
		{"article", "article", "article"},
		{"kategori", "kategori", "kategori"},
		{"category -> kategori", "category", "kategori"},
		{"bos -> home", "", "home"},
		{"gecersiz -> home", "hakkimizda", "home"},
		{"bosluk kirpilir", "  kategori  ", "kategori"},
		{"sql benzeri -> home", "kategori'; DROP--", "home"},
		{"buyuk harf -> home", "HOME", "home"},
		{"HOME bosluklu -> home", " HOME ", "home"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/admin/tasarim?sayfa="+url.QueryEscape(c.q), nil)
			assert.Equal(t, c.want, builderPage(r))
		})
	}
}

func TestBuilderPageNoQueryAtAll(t *testing.T) {
	r := httptest.NewRequest("GET", "/admin/tasarim", nil)
	assert.Equal(t, "home", builderPage(r))
}

func TestBuilderPageRepeatedParam(t *testing.T) {
	// url.Values.Get ilk değeri döndürür: yinelenen parametrede ilk kazanır.
	r := httptest.NewRequest("GET", "/admin/tasarim?sayfa=article&sayfa=home", nil)
	assert.Equal(t, "article", builderPage(r))

	r2 := httptest.NewRequest("GET", "/admin/tasarim?sayfa=home&sayfa=article", nil)
	assert.Equal(t, "home", builderPage(r2))
}

// ---- bölgeler (zones) --------------------------------------------

func zoneKeys(page string) []string {
	out := []string{}
	for _, z := range builderZonesFor(page) {
		out = append(out, z.Key)
	}
	return out
}

func TestBuilderZonesFor(t *testing.T) {
	cases := []struct {
		name     string
		page     string
		wantKeys []string
		absent   []string
	}{
		{
			name:     "home: dort bolge",
			page:     "home",
			wantKeys: []string{"ust", "ana", "yan", "genis"},
		},
		{
			name:     "article: ust/ana/yan/genis",
			page:     "article",
			wantKeys: []string{"ust", "ana", "yan", "genis"},
		},
		{
			name:     "kategori: ust/govde/yan/genis",
			page:     "kategori",
			wantKeys: []string{"ust", "govde", "yan", "genis"},
			absent:   []string{"ana"},
		},
		{
			name:     "bilinmeyen sayfa home gibi davranir",
			page:     "xyz",
			wantKeys: []string{"ust", "ana", "yan", "genis"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := zoneKeys(c.page)
			for _, k := range c.wantKeys {
				assert.Contains(t, got, k)
			}
			for _, k := range c.absent {
				assert.NotContains(t, got, k)
			}
			assert.Len(t, got, len(c.wantKeys), "bölge sayısı beklenenden farklı")
		})
	}
}

func TestBuilderZonesUniqueAndComplete(t *testing.T) {
	for _, page := range []string{"home", "article", "kategori", ""} {
		zones := builderZonesFor(page)
		require.NotEmpty(t, zones, "sayfa %q için bölge olmalı", page)
		seen := map[string]bool{}
		for _, z := range zones {
			assert.NotEmpty(t, z.Key, "bölge anahtarı boş olamaz")
			assert.NotEmpty(t, z.Name, "bölge adı boş olamaz")
			assert.NotEmpty(t, z.Hint, "bölge ipucu boş olamaz")
			assert.False(t, seen[z.Key], "sayfa %q: yinelenen bölge %q", page, z.Key)
			seen[z.Key] = true
		}
	}
}

// ---- blok paleti -------------------------------------------------

func typeKeys(types []blockType) []string {
	out := []string{}
	for _, b := range types {
		out = append(out, b.Type)
	}
	return out
}

func TestBuilderTypesFor(t *testing.T) {
	home := typeKeys(builderTypesFor("home"))
	art := typeKeys(builderTypesFor("article"))

	assert.Contains(t, home, "slider")
	assert.Contains(t, home, "manset6")
	assert.Contains(t, home, "kategori_panel")
	assert.Contains(t, home, "puan_durumu")
	assert.NotContains(t, home, "kategori_bloklari", "eski çoklu panel bloğu paletten kalkmalı")

	// Yazı sayfasında tam sayfa blokları sunulmamalı
	assert.NotContains(t, art, "slider")
	assert.NotContains(t, art, "manset6")
	assert.NotContains(t, art, "puan_durumu")
	assert.Contains(t, art, "canli_yayin")
	assert.Contains(t, art, "trend_numarali")
}

func TestBuilderTypesUnique(t *testing.T) {
	for _, page := range []string{"home", "article"} {
		keys := typeKeys(builderTypesFor(page))
		seen := map[string]bool{}
		for _, k := range keys {
			assert.False(t, seen[k], "sayfa %q: yinelenen blok tipi %q", page, k)
			seen[k] = true
		}
		assert.NotEmpty(t, keys)
	}
}

func TestBuilderTypesComplete(t *testing.T) {
	for _, page := range []string{"home", "article"} {
		for _, b := range builderTypesFor(page) {
			assert.NotEmpty(t, b.Type, "blok tipi boş olamaz")
			assert.NotEmpty(t, b.Name, "blok adı boş olamaz")
			assert.NotEmpty(t, b.Desc, "blok açıklaması boş olamaz")
		}
	}
}

// ---- önizleme temizleyici (Faz 2) --------------------------------

func TestSanitizePreviewHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "script govdesiyle birlikte cikar",
			in:   `<div class="x"><script>window.hikayeData = "a";</script></div>`,
			want: `<div class="x"></div>`,
		},
		{
			name: "uzaktaki script etiketi cikar",
			in:   `<div id="stories"></div>\n<script src="/v/zuck.js" defer></script>`,
			want: `<div id="stories"></div>\n`,
		},
		{
			name: "link etiketi cikar",
			in:   `<link rel="stylesheet" href="zuck.css">\n<div id="stories"></div>`,
			want: `\n<div id="stories"></div>`,
		},
		{
			name: "birden fazla script ve link",
			in:   `<link rel="stylesheet" href="a.css"><script>x</script><p>gövde</p><script src="b.js"></script>`,
			want: `<p>gövde</p>`,
		},
		{
			name: "buyuk harf duyarsiz",
			in:   `<SCRIPT>alert(1)</SCRIPT><LINK href="x">canlı`,
			want: `canlı`,
		},
		{
			name: "normal html korunur",
			in:   `<a href="/haber/1">Başlık Örneği</a>`,
			want: `<a href="/haber/1">Başlık Örneği</a>`,
		},
		{
			name: "kapanmamis script: gerisi dusurulur",
			in:   `<div>üst</div><script>totally broken`,
			want: `<div>üst</div>`,
		},
		{
			name: "link kapanmamis: gerisi korunur",
			in:   `<link rel="stylesheet" href="x.css"`,
			want: `<link rel="stylesheet" href="x.css"`,
		},
		{
			name: "script icine gomulu < metin sorunsuz",
			in:   `<script>if (a < b) {}</script><b>metin</b>`,
			want: `<b>metin</b>`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, sanitizePreviewHTML(c.in))
		})
	}
}

// Hikayeler bloğunun gerçek şablon çıktısı betik/linklerden arındırıldığında
// geriye yalnızca boş bir kabuk kalmalı → yer tutucu eşiği tetiklenmeli.
func TestSanitizePreviewHikayelerShell(t *testing.T) {
	raw := `<link rel="stylesheet" href="zuck/zuck.css">
<link rel="stylesheet" href="zuck/snapgram.css">
<div id="stories"></div>
<script>window.hikayeData = {"x":1};</script>
<script src="zuck/zuck.js" defer></script>`
	got := sanitizePreviewHTML(raw)
	assert.Equal(t, `<div id="stories"></div>`, strings.TrimSpace(got))
	assert.False(t, hasVisiblePreview(got), "hikayeler kabuğu görünür sayılmamalı")
}

func TestHasVisiblePreview(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"bos", "", false},
		{"yalniz bos div", "<div id=\"stories\"></div>", false},
		{"yalniz bosluk", "   \n  ", false},
		{"kisa metin", "<span>az</span>", false},
		{"yeterli metin", "<h3>Haber Başlığı Denemesi</h3>", true},
		{"img", `<img src="/x.jpg" alt="foto">`, true},
		{"video", `<video src="/x.mp4"></video>`, true},
		{"svg", `<svg class="ti"><path d="M0"/></svg>`, true},
		{"background image", `<div style="background-image:url('/x.jpg')"></div>`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, hasVisiblePreview(c.in))
		})
	}
}

// ---- Faz 4: sürüm geçmişi yardımcıları -----------------------------

func TestLayoutBlockTypes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"bos dizi", `[]`, nil},
		{"tekli", `[{"type":"slider","opts":{"count":3}}]`, []string{"slider"}},
		{"coklu", `[{"type":"slider"},{"type":"manset"},{"type":"baslik"}]`, []string{"slider", "manset", "baslik"}},
		{"tipli+opts", `[{"type":"ust_manset","opts":{"zone":"ust"}}]`, []string{"ust_manset"}},
		{"tiysiz atla", `[{"type":"reklam"},{"id":"x"}]`, []string{"reklam"}},
		{"gecersiz json", `not-json`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, layoutBlockTypes([]byte(c.in)))
		})
	}
}

func TestParseHomeBlocksKapakTasarim(t *testing.T) {
	// Faz 3: blok opts.kapak_tasarim homeBlock.KapakTasarimID'ye okunmalı.
	in := `[{"id":"b-slider","type":"slider","visible":true,"opts":{"count":5,"kapak_tasarim":4}},{"id":"b-manset","type":"manset","opts":{"count":6}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, int64(4), blocks[0].KapakTasarimID)
	assert.False(t, blocks[0].ShowTitle, "kapak şablonu seçiliyken üst yazı varsayılan gizli olmalı")
	assert.False(t, blocks[0].ShowBadge)
	assert.False(t, blocks[0].ShowSpot)
	assert.True(t, blocks[1].ShowTitle)
	assert.True(t, blocks[1].ShowBadge)
	assert.Equal(t, int64(0), blocks[1].KapakTasarimID)
}

func TestHeadlinePresetsJSON(t *testing.T) {
	var arr []struct {
		Name   string          `json:"name"`
		Design json.RawMessage `json:"design"`
	}
	require.NoError(t, json.Unmarshal([]byte(headlinePresets), &arr))
	require.Len(t, arr, 6)
	for _, p := range arr {
		assert.NotEmpty(t, p.Name)
		assert.True(t, json.Valid(p.Design))
	}
}

func TestSliderHidesOverlayTextWithoutRemovingSEO(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		ShowTitle: false, ShowBadge: false, ShowSpot: false,
		TitleColor: "#ff0000", PagerType: "numbers", ShowAll: true,
		TitleCase:  "sentence",
		TitleStyle: template.CSS("color:#ff0000"),
		D:          &tmplData{LangBase: ""},
		Slides: []repo.Content{{
			ID: 1, Title: "SEO Başlığı", Spot: "SEO spotu", CategoryName: ptr("Gündem"),
		}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_slider", b))
	out := buf.String()
	assert.Contains(t, out, "hero-title sr-only title-case-sentence")
	assert.Contains(t, out, "hero-cat sr-only")
	assert.Contains(t, out, "hero-spot sr-only")
	assert.Contains(t, out, "hero-overlay--plain")
	assert.Contains(t, out, "SEO Başlığı")
	assert.Contains(t, out, "SEO spotu")
	assert.Contains(t, out, "hero-dots--numbers")
	assert.Contains(t, out, "hero-dot-all")
	assert.Contains(t, out, "hero-slide-all")
	assert.Contains(t, out, "Tüm manşetleri görüntüle")
	assert.Contains(t, out, `style="color:#ff0000"`)
	// Numaralı göstergede sayı .hero-dot-n içinde (sayılı şerit stili için).
	assert.Contains(t, out, `<span class="hero-dot-n">1</span></button>`)
}

func TestParseOverlayStyleOptions(t *testing.T) {
	in := `[{"id":"b-slider","type":"slider","opts":{"count":30,"allow_over_20":true,"slider_style":"edirne","pager_size":32,"title_color":"#ff0000","title_font":"serif","title_line_height":1.2,"title_letter_spacing":1,"title_word_spacing":2,"title_case":"sentence","overlay_color":"#112233","overlay_opacity":45,"overlay_height":50,"overlay_gradient":"bottom","overlay_blur":6,"cat_pos":"top-right","cat_size":18,"cat_color":"#abcdef","cat_shape":"pin","head_style":"bar","mod_style":"boxed","pager_type":"numbers","pager_shape":"round","pager_pos":"center","pager_color":"#00ff00"}},{"id":"b-m6","type":"manset6","opts":{"pager_type":"dots","pager_layout":"horizontal"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, "#ff0000", blocks[0].TitleColor)
	assert.Equal(t, "serif", blocks[0].TitleFont)
	assert.InDelta(t, 1.2, blocks[0].TitleLineHeight, 0.001)
	assert.Contains(t, string(blocks[0].TitleStyle), "font-family:Georgia")
	assert.Contains(t, string(blocks[0].TitleStyle), "line-height:1.20")
	assert.Contains(t, string(blocks[0].TitleStyle), "letter-spacing:1px")
	assert.Contains(t, string(blocks[0].TitleStyle), "word-spacing:2px")
	assert.Contains(t, string(blocks[0].CatStyle), "font-size:18px")
	assert.Contains(t, string(blocks[0].CatStyle), "background-color:#abcdef")
	assert.Contains(t, string(blocks[0].CatStyle), "right:16px")
	assert.Equal(t, "pin", blocks[0].CatShape)
	assert.Equal(t, "bar", blocks[0].HeadingStyle)
	assert.Equal(t, "boxed", blocks[0].ModuleStyle)
	assert.True(t, blocks[0].Wrap)
	assert.Equal(t, "numbers", blocks[0].PagerType)
	assert.Equal(t, "round", blocks[0].PagerShape)
	assert.Equal(t, "center", blocks[0].PagerPos)
	assert.Equal(t, "#00ff00", blocks[0].PagerColor)
	assert.Equal(t, "dots", blocks[1].PagerType)
	assert.Equal(t, "horizontal", blocks[1].PagerLayout)
	assert.Equal(t, 30, blocks[0].Count)
	assert.True(t, blocks[0].AllowOver20)
	assert.Equal(t, "edirne", blocks[0].SliderStyle)
	assert.Equal(t, 32, blocks[0].PagerSize)
	assert.Equal(t, "sentence", blocks[0].TitleCase)
	assert.Contains(t, string(blocks[0].OverlayStyle), "rgba(17,34,51,0.45)")
	assert.Contains(t, string(blocks[0].OverlayStyle), "linear-gradient(to top")
	assert.Contains(t, string(blocks[0].OverlayStyle), "height:auto")
	assert.Contains(t, string(blocks[0].OverlayStyle), "min-height:50%")
	assert.NotContains(t, string(blocks[0].OverlayStyle), "overflow:hidden")
	assert.Contains(t, string(blocks[0].OverlayStyle), "top:auto")
	assert.Contains(t, string(blocks[0].OverlayStyle), "blur(6px)")

	stil5 := parseHomeBlocks([]byte(`[{"id":"b-slider5","type":"slider","opts":{"slider_style":"stil5","m2_side":"left","m2_bg":"#112233","m2_fade":"#123456","m2_num_bg":"#445566","m2_num_active_bg":"#778899","m2_accent":"#ff0000"}}]`))
	require.Len(t, stil5, 1)
	assert.Equal(t, "stil5", stil5[0].SliderStyle)
	assert.Equal(t, "left", stil5[0].M2Side)
	assert.Equal(t, "#112233", stil5[0].M2Bg)
	assert.Equal(t, "#123456", stil5[0].M2Fade)
	assert.Equal(t, "#445566", stil5[0].M2NumBg)
	assert.Equal(t, "#778899", stil5[0].M2NumActiveBg)
	assert.Equal(t, "#ff0000", stil5[0].M2Accent)

	// allow_over_20 kapalıyken 20 üstü sayı 20'ye kırpılır.
	capped := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"count":30}}]`))
	require.Len(t, capped, 1)
	assert.Equal(t, 20, capped[0].Count)
	assert.False(t, capped[0].AllowOver20)
}

func TestPinTagDefaultsToBottomLeft(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"cat_shape":"pin"}}]`))
	require.Len(t, blocks, 1)
	assert.Equal(t, "pin", blocks[0].CatShape)
	assert.Equal(t, "bottom-left", blocks[0].CatPosition)
	assert.Contains(t, string(blocks[0].CatStyle), "bottom:-8px")
	assert.Contains(t, string(blocks[0].CatStyle), "top:auto")

	// Açıkça seçilen konum pin biçiminden önce gelir.
	top := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"cat_shape":"pin","cat_pos":"top-right"}}]`))
	require.Len(t, top, 1)
	assert.Equal(t, "top-right", top[0].CatPosition)
	assert.Contains(t, string(top[0].CatStyle), "right:16px")

	pin2 := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"cat_shape":"pin2"}}]`))
	require.Len(t, pin2, 1)
	assert.Equal(t, "bottom-center", pin2[0].CatPosition)
	assert.Contains(t, string(pin2[0].CatStyle), "left:50%")
	assert.Contains(t, string(pin2[0].CatStyle), "translateX(-50%)")
}

func TestCustomCategoryTagGeometry(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"cat_shape":"pin","cat_clip":"polygon(0 0, 100% 0, 50% 100%)","cat_left":"30px","cat_bottom":"-14px"}}]`))
	require.Len(t, blocks, 1)
	assert.Equal(t, "polygon(0 0, 100% 0, 50% 100%)", blocks[0].CatClip)
	assert.Equal(t, "30px", blocks[0].CatLeft)
	assert.Equal(t, "-14px", blocks[0].CatBottom)
	assert.Contains(t, string(blocks[0].CatStyle), "clip-path:polygon(0 0, 100% 0, 50% 100%)")
	assert.Contains(t, string(blocks[0].CatStyle), "left:30px")
	assert.Contains(t, string(blocks[0].CatStyle), "bottom:-14px")

	// Güvenli olmayan değerler sessizce yok sayılır.
	bad := parseHomeBlocks([]byte(`[{"id":"b-slider","type":"slider","opts":{"cat_shape":"pin","cat_clip":"url(javascript:alert(1))","cat_left":"0;color:red","cat_bottom":"10px"}}]`))
	require.Len(t, bad, 1)
	assert.Empty(t, bad[0].CatClip)
	assert.Empty(t, bad[0].CatLeft)
	assert.Equal(t, "10px", bad[0].CatBottom)

	media := parseHomeBlocks([]byte(`[{"id":"b-ust","type":"ust_manset","opts":{"cat_shape":"pin2","cat_left":"20%","cat_bottom":"-10px"}}]`))
	require.Len(t, media, 1)
	assert.Contains(t, string(media[0].CatStyle), "--cat-left:20%")
	assert.Contains(t, string(media[0].CatStyle), "--cat-bottom:-10px")
	assert.NotContains(t, string(media[0].CatStyle), "position:absolute")
}

func TestPinTagInMediaCardsUsesMediaWrapper(t *testing.T) {
	for _, typ := range []string{"ust_manset", "kari_kartlar"} {
		blocks := parseHomeBlocks([]byte(`[{"id":"b-` + typ + `","type":"` + typ + `","opts":{"cat_shape":"pin"}}]`))
		require.Len(t, blocks, 1)
		assert.Equal(t, "bottom-left", blocks[0].CatPosition)
		assert.NotContains(t, string(blocks[0].CatStyle), "position:absolute",
			"%s pin etiketi görsel sarmalayıcısında konumlanmalı", typ)
	}
}

func TestHideMobileOption(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"b-baslik","type":"baslik","opts":{"hide_mobile":true,"text":"Sadece masaüstü"}}]`))
	require.Len(t, blocks, 1)
	assert.True(t, blocks[0].HideMobile)
	assert.True(t, blocks[0].Wrap)
}

func TestModuleStyleAndTagShapeRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "one_cikanlar", HeadingStyle: "bar", CatShape: "pin",
		ModuleStyle: "boxed", HideMobile: true, Wrap: true, ShowTitle: true, ShowBadge: true,
		CatStyle: template.CSS("background-color:#abcdef"),
		D:        &tmplData{LangBase: ""},
		Items: []repo.Content{{
			ID: 7, Title: "Modül başlığı", CategoryName: ptr("Gündem"),
		}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, `class="blk-st blk-mod-boxed blk-hide-mobile"`)
	assert.Contains(t, out, "blk-title head-mark-bar")
	assert.Contains(t, out, "one-cat cat-shape-pin")
	assert.Contains(t, out, "background-color:#abcdef")
	assert.Contains(t, out, "Modül başlığı")
}

func TestParseHomeBlocksHeadingStylesAndHideFlags(t *testing.T) {
	styles := []string{"line", "bar", "underline", "pill", "dot", "corner", "tab", "boxed", "gradient", "none"}
	for _, st := range styles {
		blocks := parseHomeBlocks([]byte(`[{"id":"b","type":"trend_numarali","opts":{"head_style":"` + st + `"}}]`))
		require.Len(t, blocks, 1)
		assert.Equal(t, st, blocks[0].HeadingStyle, "geçerli başlık stili korunmalı: "+st)
	}
	bad := parseHomeBlocks([]byte(`[{"id":"b","type":"trend_numarali","opts":{"head_style":"gecersiz"}}]`))
	require.Len(t, bad, 1)
	assert.Equal(t, "line", bad[0].HeadingStyle, "geçersiz stil varsayılana dönmeli")

	flags := parseHomeBlocks([]byte(`[{"id":"b","type":"trend_numarali","opts":{"head_style":"corner","hide_heading":true,"hide_badge":true}}]`))
	require.Len(t, flags, 1)
	assert.True(t, flags[0].HideHeading)
	assert.True(t, flags[0].HideBadge)
}

func TestBlockCategorySelectionFallsBackWhenUnknown(t *testing.T) {
	s := &Server{}
	h := &homeData{
		SiteID: 1,
		Slides: []repo.Content{{ID: 1, Title: "Genel manşet"}},
		Lead:   []repo.Content{{ID: 2, Title: "Genel öne çıkan"}},
	}
	blocks := s.buildBlocksFrom(context.Background(), &tmplData{}, []byte(
		`[{"id":"s1","type":"slider","opts":{"kategori":"silinmis-kategori","count":5}}]`), h)
	require.Len(t, blocks, 1)
	assert.Equal(t, "silinmis-kategori", blocks[0].CatSlug, "seçim blokta saklanmalı")
	require.Len(t, blocks[0].Slides, 1, "kategori bulunamazsa varsayılan akış kullanılmalı")
	assert.Equal(t, "Genel manşet", blocks[0].Slides[0].Title)
}

func TestBlockCategorySelectionStored(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"m1","type":"manset","opts":{"kategori":"ekonomi","count":6}}]`))
	require.Len(t, blocks, 1)
	assert.Equal(t, "ekonomi", blocks[0].CatSlug)
	assert.Equal(t, 6, blocks[0].Count)
}

func TestServisBandiHomeOlmadanRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "servis_bandi",
		D:    &tmplData{LangBase: ""},
		ServisKurlar: []repo.Kur{
			{Name: "USD", Value: 49.97, Delta: 0.12},
			{Name: "ALTIN", Value: 6268.21, Delta: -0.4},
		},
		ServisHava:   &WeatherS{Slug: "ankara", Sicak: 16, Aciklama: "Parçalı bulutlu"},
		ServisCities: []repo.City{{Name: "Ankara", Slug: "ankara"}, {Name: "İzmir", Slug: "izmir"}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "hp-ticker-bar")
	assert.Contains(t, out, "tk-USD")
	assert.Contains(t, out, "tk-ALTIN")
	assert.Contains(t, out, "Parçalı bulutlu")
	assert.Contains(t, out, `<option value="ankara" selected>`)
	assert.Contains(t, out, `<option value="izmir"`)
}

// Servis bandı veri yoksa hiç basılmamalı (boş kutu gösterme).
func TestServisBandiVeriYoksaBasilmaz(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{Type: "servis_bandi", D: &tmplData{}}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	assert.NotContains(t, buf.String(), "hp-ticker-bar")
}

func TestKategoriPanelGastroDuzeni(t *testing.T) {
	// Habertürk GASTRO vitrini seçilebilir düzen olmalı; gastro kategorisinin
	// varsayılanı bu düzen olmalı.
	assert.True(t, validPanelDuzen("gastro"))
	assert.Equal(t, "gastro", defaultPanelDuzen("gastro"))
	assert.Equal(t, "ekonomi", defaultPanelDuzen("ekonomi"), "ekonomi varsayılanı bozulmamalı")

	tmpl := parseTestTemplates(t)
	items := make([]repo.Content, 0, 9)
	for i := 0; i < 9; i++ {
		items = append(items, repo.Content{ID: int64(i + 2), Title: fmt.Sprintf("Gastro %d", i+2)})
	}
	b := &homeBlock{
		Type: "kategori_bloklari", HeadingStyle: "line",
		CatBlocks: []catBlock{{
			Name: "GASTRO", Slug: "gastro", Tone: "turuncu", Duzen: "gastro",
			Lead:  repo.Content{ID: 1, Title: "Gastro manşet"},
			Items: items,
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "kat-panel-blok turuncu kp-ht kp-gs")
	assert.Contains(t, out, ">TÜMÜ<", "Habertürk başlığında TÜMÜ büyük harf olmalı")
	assert.Contains(t, out, `class="gs-scroll"`)
	assert.Equal(t, 10, strings.Count(out, `class="gs-card`), "1 manşet + 9 kart = 10 kart olmalı")
	// Dönüşümlü başlık konumu: 0,2,4… altta; 1,3,5… üstte (gs-rev).
	assert.Equal(t, 5, strings.Count(out, "gs-card gs-rev"), "tek sıradaki 5 kart ters (başlık üstte) olmalı")
	assert.Contains(t, out, "Gastro manşet")
	assert.Contains(t, out, "Gastro 10")
	// Tüm görseller 1:2 oran sınıfıyla aynı kapsayıcıyı kullanır.
	assert.Equal(t, 10, strings.Count(out, `class="gs-img"`), "her kartta görsel kapsayıcısı olmalı")
	assert.Equal(t, 10, strings.Count(out, `class="gs-tx"`), "kırpma sarmalayıcısı her başlıkta olmalı")

	// Diğer düzenlerde GASTRO sınıfları basılmamalı.
	b.CatBlocks[0].Duzen = "mozaik"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out = buf.String()
	assert.NotContains(t, out, "kp-gs")
	assert.NotContains(t, out, "gs-scroll")
	assert.Contains(t, out, "Tümü")
}

func TestKategoriPanelEkonomiDuzeni(t *testing.T) {
	// Habertürk Ekonomi vitrini seçilebilir bir düzen olmalı ve ekonomi
	// kategorisinin varsayılanı bu düzen olmalı.
	assert.True(t, validPanelDuzen("ekonomi"))
	assert.Equal(t, "ekonomi", defaultPanelDuzen("ekonomi"))
	assert.False(t, validPanelDuzen("gecersiz"))

	tmpl := parseTestTemplates(t)
	items := []repo.Content{
		{ID: 2, Title: "Küçük kare"}, {ID: 3, Title: "Küçük geniş"},
		{ID: 4, Title: "Küçük geniş 2"}, {ID: 5, Title: "Küçük kare 2"},
	}
	b := &homeBlock{
		Type: "kategori_bloklari", HeadingStyle: "line", CatStyle: "",
		CatBlocks: []catBlock{{
			Name: "EKONOMİ", Slug: "ekonomi", Tone: "mavi", Duzen: "ekonomi",
			Lead:  repo.Content{ID: 1, Title: "Büyük manşet"},
			Items: items,
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "kat-panel-blok mavi kp-ht")
	assert.Contains(t, out, ">TÜMÜ<", "Habertürk başlığında TÜMÜ büyük harf olmalı")
	assert.Contains(t, out, `class="ek-grid"`)
	assert.Contains(t, out, "ek-card ek-big")
	assert.Equal(t, 5, strings.Count(out, `class="ek-card`), "1 büyük + 4 küçük kart basılmalı")
	assert.Equal(t, 2, strings.Count(out, `class="ek-col"`), "sağda iki sütun olmalı")
	assert.Equal(t, 2, strings.Count(out, "ek-img ek-wide"), "iki kart 16:9 olmalı")
	assert.Equal(t, 3, strings.Count(out, "ek-img ek-sq"), "üç kart kare olmalı")
	assert.Equal(t, 5, strings.Count(out, `class="ek-tx"`), "kırpma sarmalayıcısı her başlıkta olmalı")
	assert.Contains(t, out, "Büyük manşet")
	assert.Contains(t, out, "Küçük kare 2")
	// Başlıklar görselin ALTINDA: ek-t, ek-img kapanışından sonra gelir.
	sq := strings.Index(out, "ek-img ek-sq")
	ttl := strings.Index(out[sq:], `class="ek-t"`)
	assert.Greater(t, ttl, 0, "başlık görsel kapsayıcısından sonra gelmeli")

	// Diğer düzenlerde Habertürk sınıfı ve TÜMÜ biçimi uygulanmamalı.
	b.CatBlocks[0].Duzen = "buyuk"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out = buf.String()
	assert.NotContains(t, out, "kp-ht")
	assert.Contains(t, out, "Tümü")
}

func TestUstMansetCardStyles(t *testing.T) {
	for _, st := range []string{"klasik", "overlay", "overlay-top", "band", "frame", "corner"} {
		blocks := parseHomeBlocks([]byte(`[{"id":"u1","type":"ust_manset","opts":{"um_style":"` + st + `","um_layout":"izgara","um_cols":4}}]`))
		require.Len(t, blocks, 1)
		assert.Equal(t, st, blocks[0].UMStyle, "geçerli kart stili korunmalı: "+st)
		assert.Equal(t, "izgara", blocks[0].UMLayout)
		assert.Equal(t, 4, blocks[0].UMCols)
	}
	bad := parseHomeBlocks([]byte(`[{"id":"u1","type":"ust_manset","opts":{"um_style":"gecersiz","um_layout":"gecersiz","um_cols":9,"um_title_color":"url(x)","um_title_bg":"#112233","um_title_size":99,"um_grad_opacity":300}}]`))
	require.Len(t, bad, 1)
	b := bad[0]
	assert.Equal(t, "klasik", b.UMStyle, "geçersiz stil varsayılana dönmeli")
	assert.Equal(t, "serit", b.UMLayout, "geçersiz düzen varsayılana dönmeli")
	assert.Equal(t, 4, b.UMCols, "sütun sayısı 2-4 aralığına çekilmeli")
	assert.Empty(t, b.UMTitleColor, "geçersiz renk yok sayılmalı")
	assert.Equal(t, 40, b.UMTitleSize, "başlık boyutu üst sınırı uygulanmalı")
	assert.Equal(t, 100, b.UMGradOpacity, "saydamlık %100'de kırpılmalı")
	assert.Contains(t, string(b.UMStyleCSS), "--um-title-bg:rgba(17,34,51,0.82)")
	assert.True(t, b.UMShadow, "gölge varsayılan açık olmalı")

	off := parseHomeBlocks([]byte(`[{"id":"u1","type":"ust_manset","opts":{"um_shadow":false,"um_title_color":"#ffcc00","um_grad_color":"#001122","um_grad_opacity":70}}]`))
	require.Len(t, off, 1)
	assert.Contains(t, string(off[0].UMStyleCSS), "--um-shadow:none")
	assert.Contains(t, string(off[0].UMStyleCSS), "--um-title-color:#ffcc00")
	assert.Contains(t, string(off[0].UMStyleCSS), "--um-grad-opacity:0.70")
}

func TestUstMansetKartKabuguAyarlari(t *testing.T) {
	b := parseHomeBlocks([]byte(`[{"id":"u1","type":"ust_manset","opts":{
		"um_card_bg":"#123456",
		"um_card_dark_bg":"#654321",
		"um_card_shadow":false,
		"um_title_pad":24
	}}]`))
	require.Len(t, b, 1)
	assert.Equal(t, "#123456", b[0].UMCardBg)
	assert.Equal(t, "#654321", b[0].UMCardDarkBg)
	assert.False(t, b[0].UMCardShadow)
	assert.Equal(t, 24, b[0].UMTitlePad)
	css := string(b[0].UMStyleCSS)
	assert.Contains(t, css, "--um-card-bg-light:#123456")
	assert.Contains(t, css, "--um-card-bg-dark:#654321")
	assert.Contains(t, css, "--um-card-shadow:none")
	assert.Contains(t, css, "--um-card-shadow-hover:none")
	assert.Contains(t, css, "--um-title-pad-x:24px")

	// 0 geçerli (yan boşluk yok); geçersiz renkler yok sayılır.
	zero := parseHomeBlocks([]byte(`[{"id":"u2","type":"ust_manset","opts":{
		"um_title_pad":0,
		"um_card_bg":"url(x)",
		"um_card_dark_bg":"javascript:alert(1)"
	}}]`))
	require.Len(t, zero, 1)
	assert.Equal(t, 0, zero[0].UMTitlePad)
	assert.Empty(t, zero[0].UMCardBg)
	assert.Empty(t, zero[0].UMCardDarkBg)
	assert.Contains(t, string(zero[0].UMStyleCSS), "--um-title-pad-x:0px")
	assert.NotContains(t, string(zero[0].UMStyleCSS), "javascript")

	// Üst sınır 60px; varsayılanlar gölge açık, arka plan tema yüzeyi.
	cap := parseHomeBlocks([]byte(`[{"id":"u3","type":"ust_manset","opts":{"um_title_pad":999}}]`))
	require.Len(t, cap, 1)
	assert.Equal(t, 60, cap[0].UMTitlePad)
	assert.Contains(t, string(cap[0].UMStyleCSS), "--um-title-pad-x:60px")

	def := parseHomeBlocks([]byte(`[{"id":"u4","type":"ust_manset","opts":{}}]`))
	require.Len(t, def, 1)
	assert.True(t, def[0].UMCardShadow, "kart gölgesi varsayılan açık")
	assert.Empty(t, def[0].UMCardBg)
	assert.Empty(t, def[0].UMCardDarkBg)
	assert.Equal(t, -1, def[0].UMTitlePad, "ayarlanmamış yan boşluk -1 olmalı")
}

func TestUstMansetOverlayRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	base := &homeBlock{
		Type: "ust_manset", UMStyle: "overlay", UMLayout: "izgara", UMCols: 4, UMShadow: true,
		UMStyleCSS: template.CSS("--um-title-color:#ffcc00;--um-grad-opacity:0.70"),
		ShowTitle:  true, ShowBadge: true, CatShape: "badge", D: &tmplData{LangBase: ""},
		Items: []repo.Content{{ID: 7, Title: "Görsel üstü başlık", CategoryName: ptr("Ekonomi")}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", base))
	out := buf.String()
	assert.Contains(t, out, "ust-manset um-style-overlay um-layout-izgara um-cols-4")
	assert.Contains(t, out, "--um-title-color:#ffcc00")
	assert.Contains(t, out, "um-card um-ov")
	assert.Contains(t, out, "um-shade", "görsel üstü stilde gradyan katmanı basılmalı")
	// Başlık görselin İÇİNDE (medya span'i içinde) olmalı.
	mediaStart := strings.Index(out, `class="um-media"`)
	cardEnd := strings.Index(out[mediaStart:], "</a>")
	inner := out[mediaStart : mediaStart+cardEnd]
	assert.Contains(t, inner, `class="um-t`, "başlık medya bloğunun içinde olmalı")
	// Kırpma iç elemanda; dış elemanda dolgu var (dolgu bandına sızma olmasın).
	assert.Contains(t, out, `class="um-tx"`, "başlık metni kırpma sarmalayıcısında olmalı")

	// Klasik stil: başlık görselin altında, gradyan yok.
	base.UMStyle = "klasik"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", base))
	out = buf.String()
	assert.Contains(t, out, "um-style-klasik")
	assert.NotContains(t, out, "um-card um-ov")
	assert.NotContains(t, out, "um-shade")

	// Başlık kapatılsa da metin DOM'da kalmalı (SEO/erişilebilirlik).
	base.UMStyle = "overlay"
	base.ShowTitle = false
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", base))
	out = buf.String()
	assert.Contains(t, out, "um-t sr-only")
	assert.Contains(t, out, "Görsel üstü başlık")
}

func TestParseKategoriSekmeliOptions(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"ks","type":"kategori_sekmeli","opts":{"sekmeler":"ekonomi, siyaset,ekonomi,,gecersiz slug","tab_style":"square","tab_bg":"#112233","tab_color":"#445566","tab_active_bg":"#778899","tab_active_color":"#abcdef","tab_radius":4}}]`))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, []string{"ekonomi", "siyaset"}, b.CatTabs, "slug'lar temizlenmeli ve tekrarlar atılmalı")
	assert.Equal(t, "square", b.TabStyle)
	assert.Equal(t, "#112233", b.TabBg)
	assert.Equal(t, 4, b.TabRadius)
	assert.Contains(t, string(b.TabCSS), "--kt-bg:#112233")
	assert.Contains(t, string(b.TabCSS), "--kt-active-bg:#778899")
	assert.Contains(t, string(b.TabCSS), "--kt-radius:4px")

	// Dosya görünümünün iki varyantı: yumuşak köşe (tab) ve köşeli (tabflat).
	for _, st := range []string{"tab", "tabflat"} {
		got := parseHomeBlocks([]byte(`[{"id":"ks","type":"kategori_sekmeli","opts":{"tab_style":"` + st + `"}}]`))
		require.Len(t, got, 1)
		assert.Equal(t, st, got[0].TabStyle, "geçerli sekme stili korunmalı: "+st)
	}

	bad := parseHomeBlocks([]byte(`[{"id":"ks","type":"kategori_sekmeli","opts":{"tab_style":"gecersiz","tab_bg":"url(javascript:alert(1))","tab_radius":99,"count":9}}]`))
	require.Len(t, bad, 1)
	assert.Equal(t, "segmented", bad[0].TabStyle, "geçersiz stil varsayılana dönmeli")
	assert.Empty(t, bad[0].TabBg, "geçersiz renk yok sayılmalı")
	assert.Equal(t, 28, bad[0].TabRadius, "yarıçap üst sınırı uygulanmalı")
}

func TestKategoriSekmeliRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := []repo.Content{
		{ID: 1, Title: "Piyasa"}, {ID: 2, Title: "Borsa"}, {ID: 3, Title: "Dolar"},
		{ID: 4, Title: "Euro"}, {ID: 5, Title: "Altın"}, {ID: 6, Title: "Faiz"},
	}
	b := &homeBlock{
		Type: "kategori_sekmeli", TabStyle: "square", TabNav: true,
		TabCSS: template.CSS("--kt-active-bg:#abcdef;--kt-radius:4px"),
		CatSections: []catSection{{
			Name: "Ekonomi", Slug: "ekonomi", Items: items,
			Pages: [][]repo.Content{items[:5], items[5:]},
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "kat-sekmeli kt-square")
	assert.Contains(t, out, "style=\"--kt-active-bg:#abcdef;--kt-radius:4px\"")
	assert.Contains(t, out, "data-tab=\"ekonomi\"")
	assert.Contains(t, out, "Piyasa")
	assert.Contains(t, out, "Faiz", "sonraki sayfa haberleri de DOM'da olmalı (SEO)")
	assert.Contains(t, out, "kat-nav kat-prev")
	assert.Contains(t, out, "kat-nav kat-next")
	assert.Equal(t, 2, strings.Count(out, "class=\"kat-yap kat-page"), "iki haber sayfası basılmalı")
	assert.Contains(t, out, "kat-page on")
	assert.Contains(t, out, "kat-page\" hidden", "ikinci sayfa gizli başlamalı")
	assert.Equal(t, 2, strings.Count(out, "class=\"kat-dot"), "sayfa noktaları basılmalı")
}

func TestKategoriSekmeliTabflatRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type: "kategori_sekmeli", TabStyle: "tabflat", TabNav: false,
		CatSections: []catSection{{
			Name: "Ekonomi", Slug: "ekonomi",
			Items: []repo.Content{{ID: 1, Title: "Piyasa"}},
			Pages: [][]repo.Content{{{ID: 1, Title: "Piyasa"}}},
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	assert.Contains(t, buf.String(), "kat-sekmeli kt-tabflat")
}

func TestKategoriSekmeliNavOff(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := []repo.Content{{ID: 1, Title: "Tek"}}
	b := &homeBlock{
		Type: "kategori_sekmeli", TabStyle: "segmented", TabNav: false,
		CatSections: []catSection{{
			Name: "Ekonomi", Slug: "ekonomi", Items: items,
			Pages: [][]repo.Content{items},
		}},
		D: &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.NotContains(t, out, "kat-nav kat-prev", "oklar kapalıyken basılmamalı")
	assert.NotContains(t, out, "kat-nav-dots")
}

func TestHeadingStyleHideRender(t *testing.T) {
	tmpl := parseTestTemplates(t)

	// Başlık stili tüm modül başlıklarında sınıf olarak basılmalı.
	b := &homeBlock{
		Type: "trend_numarali", HeadingStyle: "corner", Wrap: true,
		ShowTitle: true, ShowBadge: true, D: &tmplData{LangBase: ""},
		Items: []repo.Content{{ID: 1, Title: "Trend"}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()
	assert.Contains(t, out, "side-h head-mark-corner")

	// Başlık kapat: görsel olarak gizlenir, metin DOM'da kalır.
	b.HideHeading = true
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out = buf.String()
	assert.Contains(t, out, "side-h head-mark-corner sr-only")
	assert.Contains(t, out, "Trend Haberler", "kapalı başlık SEO için DOM'da kalmalı")

	// Etiket kapat: kategori rozeti sr-only olur, metin kaybolmaz.
	one := &homeBlock{
		Type: "one_cikanlar", HeadingStyle: "bar", HideBadge: true, CatShape: "badge",
		ShowTitle: true, ShowBadge: true, Wrap: true, D: &tmplData{LangBase: ""},
		Items: []repo.Content{{ID: 2, Title: "Haber", CategoryName: ptr("Ekonomi")}},
	}
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", one))
	out = buf.String()
	assert.Contains(t, out, "one-cat cat-shape-badge sr-only")
	assert.Contains(t, out, "Ekonomi", "gizli etiket DOM'da kalmalı")

	one.HideBadge = false
	one.ShowBadge = false
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", one))
	out = buf.String()
	assert.Contains(t, out, "one-cat cat-shape-badge sr-only", "ShowBadge=false da etiketi gizlemeli")
}

func TestGalleryHeadingStyleRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	for _, st := range []string{"line", "bar", "underline", "pill", "dot", "corner", "tab", "boxed", "gradient", "none"} {
		b := &homeBlock{
			Type: "galeri_foto", HeadingStyle: st, Wrap: true, D: &tmplData{LangBase: ""},
			Items: []repo.Content{{ID: 1, Title: "Galeri"}},
		}
		var buf bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
		assert.Contains(t, buf.String(), "galeri-etiket head-mark-"+st,
			"galeri başlığında stil sınıfı olmalı: "+st)
	}
}

func TestSliderRailStaysFixedWhenCountChanges(t *testing.T) {
	h := &homeData{Slides: []repo.Content{
		{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5},
	}}
	s := &Server{}
	blocks := s.buildBlocksFrom(context.Background(), &tmplData{}, []byte(
		`[{"id":"b-slider","type":"slider","opts":{"count":3}}]`), h)
	require.Len(t, blocks, 1)
	require.Len(t, blocks[0].Slides, 3)
	require.Len(t, blocks[0].Rail, 2)
	assert.Equal(t, int64(3), blocks[0].Rail[0].ID)
	assert.Equal(t, int64(4), blocks[0].Rail[1].ID)
}

func TestManset2DefaultsAndRender(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"b-m2","type":"manset2","opts":{}}]`))
	require.Len(t, blocks, 1)
	assert.Equal(t, 23, blocks[0].Count)
	assert.Equal(t, "alternate", blocks[0].M2Side)
	assert.Equal(t, 330, blocks[0].M2Height)
	assert.Equal(t, "#0c1021", blocks[0].M2Bg)
	assert.Equal(t, "#c90914", blocks[0].M2Accent)

	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		D: &tmplData{LangBase: ""}, M2Bg: "#0c1021", M2Accent: "#c90914",
		M2Side: "alternate", M2Height: 330, Arrows: true,
		Slides: []repo.Content{{ID: 1, Title: "Manşet II test"}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_manset2", b))
	out := buf.String()
	assert.Contains(t, out, "m2-side-alternate")
	assert.Contains(t, out, "Manşet II test")
	assert.Contains(t, out, "m2-num")
	assert.Contains(t, out, "m2-prev")
}

// ---- Faz 5: serbest (absolute) konumlandırma -----------------------

func TestParseHomeBlocksAbsPos(t *testing.T) {
	in := `[{"id":"b-baslik","type":"baslik","opts":{"stAbs":true,"stX":40,"stY":-12,"stW":320,"stZ":5}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.True(t, b.StAbs)
	assert.Equal(t, 40, b.StX)
	assert.Equal(t, -12, b.StY)
	assert.Equal(t, 320, b.StW)
	assert.Equal(t, 5, b.StZ)
	assert.True(t, b.Wrap, "absolute blok sarmalayıcıya alınmalı")
	assert.Contains(t, string(b.Style), "position:absolute")
	assert.Contains(t, string(b.Style), "left:40px")
	assert.Contains(t, string(b.Style), "top:-12px")
	assert.Contains(t, string(b.Style), "width:320px")
	assert.Contains(t, string(b.Style), "z-index:5")
}

func TestParseHomeBlocksAbsPosKapalı(t *testing.T) {
	in := `[{"id":"b-manset","type":"manset","opts":{"count":5}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	assert.False(t, blocks[0].StAbs)
	assert.False(t, blocks[0].Wrap, "stil yoksa sarmalayıcı olmamalı")
	assert.Equal(t, "", string(blocks[0].Style))
}

func TestBuildStyleAbsClamp(t *testing.T) {
	// sınır dışı değerler kırpılmalı
	b := homeBlock{StAbs: true, StX: 99999, StY: -99999, StW: -5}
	b.buildStyle()
	assert.Contains(t, string(b.Style), "left:4000px")
	assert.Contains(t, string(b.Style), "top:-2000px")
	assert.NotContains(t, string(b.Style), "width:", "geçersiz genişlik atlanmalı")
}

// ---- Faz 6: zengin metin bloğu --------------------------------

func TestParseHomeBlocksRichText(t *testing.T) {
	// metin bloğu opts.html → RichHTML (sanitize edilmiş)
	in := `[{"id":"b-metin","type":"metin","opts":{"html":"<p>Merhaba <strong>dünya</strong></p><script>alert(1)</script>"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Contains(t, string(b.RichHTML), "<strong>dünya</strong>")
	assert.Contains(t, string(b.RichHTML), "<p>Merhaba")
	assert.NotContains(t, string(b.RichHTML), "script", "script etiketi temizlenmeli")
}

func TestParseHomeBlocksRichTextBoş(t *testing.T) {
	in := `[{"id":"b-manset","type":"manset","opts":{"count":5}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 1)
	assert.Equal(t, template.HTML(""), blocks[0].RichHTML)
}

// ---- Faz 7: A/B testi --------------------------------

func TestRollABVariantSınırlar(t *testing.T) {
	assert.Equal(t, "a", rollABVariant(0), "ağırlık 0 → daima A")
	assert.Equal(t, "b", rollABVariant(100), "ağırlık 100 → daima B")
	// aralık sınır dışı dağılım (repo.SaveLayoutAB kırpıyor; burada beklenen)
	m := map[string]int{}
	for i := 0; i < 4000; i++ {
		m[rollABVariant(30)]++
	}
	assert.Greater(t, m["b"], 500, "ağırlık 30 ile B ~%30 olmalı (rasgele ama b>500)")
	assert.Greater(t, m["a"], 1000, "ağırlık 30 ile A çoğunluk olmalı")
}

func TestABCookieOkuma(t *testing.T) {
	s := &Server{}
	// home çerezi
	r := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "", s.abVariantFromCookie(r, "home"), "çerez yoksa boş")
	r.AddCookie(&http.Cookie{Name: "tb_ab_home", Value: "b"})
	assert.Equal(t, "b", s.abVariantFromCookie(r, "home"), "home çerezi okunmalı")
	// başka sayfa için ayrı çerez — home etkilenmez
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(&http.Cookie{Name: "tb_ab_kategori", Value: "a"})
	assert.Equal(t, "", s.abVariantFromCookie(r2, "home"), "başka sayfanın çerezi home'a yansımamalı")
	assert.Equal(t, "a", s.abVariantFromCookie(r2, "kategori"))
	// hatalı değer yok sayılır
	r3 := httptest.NewRequest("GET", "/", nil)
	r3.AddCookie(&http.Cookie{Name: "tb_ab_home", Value: "x"})
	assert.Equal(t, "", s.abVariantFromCookie(r3, "home"))
}

// ---- Faz 8: düzen kopyalama (zon uyarlama) ---------------------

func TestNormalizeBlocksForPageZon(t *testing.T) {
	s := &Server{}
	in := json.RawMessage(`[
		{"id":"b-a","type":"slider","opts":{"zone":"ust"}},
		{"id":"b-b","type":"baslik","opts":{"zone":"ana","text":"Merhaba"}},
		{"id":"b-c","type":"baslik","opts":{"zone":"genis"}},
		{"id":"b-d","type":"baslik","opts":{}},
		{"id":"e","type":"gecersiz"}]
	`)
	// article sayfasına kopya: geçerli zonlar ust + ana + yan + genis
	norm, arr, err := s.normalizeBlocksForPage("article", in)
	require.NoError(t, err)
	require.Len(t, arr, 5)
	var m []map[string]any
	require.NoError(t, json.Unmarshal(norm, &m))
	got := map[string]string{}
	for _, b := range m {
		opts, _ := b["opts"].(map[string]any)
		z, _ := opts["zone"].(string)
		got[b["id"].(string)] = z
	}
	// Yazı şablonunda "ust" bölgesi 15ff2a3 ile eklendi; artık geçerli.
	assert.Equal(t, "ust", got["b-a"], "ust article'da geçerli, korunmalı")
	// Faz 142: "ana" yazı gövdesi bölgesi olarak geçerli.
	assert.Equal(t, "ana", got["b-b"], "ana article'da geçerli, korunmalı")
	// geçersiz/boş zonlar yazı sayfasının güvenli zonuna (yan) sabitlenmeli
	assert.Equal(t, "yan", got["b-d"], "boş → yan")
	// geçerli zon korunur
	assert.Equal(t, "genis", got["b-c"], "genis article'da geçerli, korunmalı")
}

func TestNormalizeBlocksForPageKoruma(t *testing.T) {
	s := &Server{}
	// home: ust/ana/yan/genis geçerli — dokunulmamalı
	in := json.RawMessage(`[{"id":"x","type":"baslik","opts":{"zone":"ana","text":"t"}}]`)
	_, arr, err := s.normalizeBlocksForPage("home", in)
	require.NoError(t, err)
	opts, _ := arr[0]["opts"].(map[string]any)
	assert.Equal(t, "ana", opts["zone"], "geçerli zon korunmalı")
}

// ---- Faz 10: kategori gövdesi modüler -------------------------

func TestParseHomeBlocksGovdeKorunur(t *testing.T) {
	// parseHomeBlocks "govde" zonunu "ana"ya çevirmemeli (Faz 10 düzeltmesi).
	in := `[{"id":"k-liste","type":"kategori_liste","opts":{"zone":"govde","count":24}},{"id":"k-b","type":"baslik","opts":{"zone":"govde","text":"t"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, "govde", blocks[0].Zone, "kategori_liste govde korunmalı")
	assert.Equal(t, "govde", blocks[1].Zone, "baslik govde korunmalı")
	assert.Equal(t, "kategori_liste", blocks[0].Type)
}

func TestParseHomeBlocksListView(t *testing.T) {
	// Faz 11: kategori_liste bloğunun view ops'u ListView alanına okunur.
	in := `[{"id":"k-l","type":"kategori_liste","opts":{"zone":"govde","view":"liste"}},{"id":"k-l2","type":"kategori_liste","opts":{"zone":"govde"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, "liste", blocks[0].ListView, "bloğa özel görünüm okunmalı")
	assert.Equal(t, "", blocks[1].ListView, "belirtilmemiş görünüm boş kalmalı (genel ayar)")
}

func TestParseHomeBlocksCategoryHeroDefaults(t *testing.T) {
	// Yeni kategori gövdesi varsayılanı: portre slider, 5 slayt, %58
	// genişlik, 4:5 oran ve iki sütunlu alt kartlar.
	blocks := parseHomeBlocks([]byte(`[{"id":"k-l","type":"kategori_liste","opts":{"zone":"govde"}}]`))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, "portre", b.CatHeroStyle)
	assert.Equal(t, 5, b.CatHeroCount)
	assert.Equal(t, 58, b.CatHeroWidth)
	assert.Equal(t, "4/5", b.CatHeroRatio)
	assert.Equal(t, 2, b.CatListCols)
}

func TestParseHomeBlocksCategoryHeroOptions(t *testing.T) {
	// Formdan sayısal string de gelebilir; seçim alanları string gelir.
	in := `[{"id":"k-l","type":"kategori_liste","opts":{"hero_style":"portre","hero_count":"7","hero_width":"64","hero_ratio":"3/4","list_cols":"4"}},{"id":"k-k","type":"kategori_liste","opts":{"hero_style":"klasik"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, "portre", blocks[0].CatHeroStyle)
	assert.Equal(t, 7, blocks[0].CatHeroCount)
	assert.Equal(t, 64, blocks[0].CatHeroWidth)
	assert.Equal(t, "3/4", blocks[0].CatHeroRatio)
	assert.Equal(t, 4, blocks[0].CatListCols)
	assert.Equal(t, "klasik", blocks[1].CatHeroStyle)
	assert.Equal(t, 5, blocks[1].CatHeroCount, "klasik düzende de varsayılan slayt sayısı korunur")
	assert.Equal(t, 2, blocks[1].CatListCols)
}

func TestParseHomeBlocksCategoryTrio(t *testing.T) {
	// Üçlü manşet: solda 3 + ortada slider + sağda 2; alt kartlar 4 sütun.
	in := `[{"id":"k-l","type":"kategori_liste","opts":{"hero_style":"uclu"}},{"id":"k-l2","type":"kategori_liste","opts":{"hero_style":"uclu","hero_width":70,"list_cols":"2"}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, "uclu", blocks[0].CatHeroStyle)
	assert.Equal(t, 5, blocks[0].CatHeroCount)
	assert.Equal(t, 44, blocks[0].CatHeroWidth, "üçlü düzende orta slider daha dar başlar")
	assert.Equal(t, 4, blocks[0].CatListCols, "üçlü şablon 4'lü kart ızgarasıyla gelir")
	assert.Equal(t, 60, blocks[1].CatHeroWidth, "üçlü düzende genişlik en fazla %60")
	assert.Equal(t, 2, blocks[1].CatListCols, "açıkça seçilen 2 sütun korunur")
}

func TestParseHomeBlocksCategoryHeroClamp(t *testing.T) {
	in := `[{"id":"k-l","type":"kategori_liste","opts":{"hero_count":1,"hero_width":10,"hero_ratio":"gecersiz","list_cols":5}},{"id":"k-l2","type":"kategori_liste","opts":{"hero_count":99,"hero_width":99}}]`
	blocks := parseHomeBlocks([]byte(in))
	require.Len(t, blocks, 2)
	assert.Equal(t, 2, blocks[0].CatHeroCount)
	assert.Equal(t, 45, blocks[0].CatHeroWidth)
	assert.Equal(t, "4/5", blocks[0].CatHeroRatio)
	assert.Equal(t, 2, blocks[0].CatListCols)
	assert.Equal(t, 10, blocks[1].CatHeroCount)
	assert.Equal(t, 75, blocks[1].CatHeroWidth)
}

func TestCategoryPortraitHeroRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := make([]repo.Content, 10)
	for i := range items {
		items[i] = repo.Content{ID: int64(i + 1), Title: fmt.Sprintf("Haber %d", i+1)}
	}
	d := &tmplData{
		LangBase: "",
		Category: &categoryData{
			Items: items, View: "klasik", HeroStyle: "portre",
			HeroCount: 5, HeroWidth: 58, HeroRatio: "4/5", ListCols: 2,
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", d))
	out := buf.String()
	assert.Contains(t, out, `class="cat-hero cat-ratio-4-5"`)
	assert.Contains(t, out, `style="--cat-hero-w:58%"`)
	assert.Equal(t, 5, strings.Count(out, `class="hero-slide `), "5 portre slayt olmalı")
	assert.Equal(t, 2, strings.Count(out, `class="chr-t"`), "sağ rayda 2 manşet olmalı")
	assert.Contains(t, out, `class="cat-cards cat-cols-2"`)
	assert.Equal(t, 3, strings.Count(out, `class="card cat-card"`), "ilk 7 içerikten sonrası alt kartlarda olmalı")
	assert.NotContains(t, out, `class="cat-lead"`)
}

func TestCategoryTrioHeroRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := make([]repo.Content, 12)
	for i := range items {
		items[i] = repo.Content{ID: int64(i + 1), Title: fmt.Sprintf("Haber %d", i+1)}
	}
	d := &tmplData{
		LangBase: "",
		Category: &categoryData{
			Items: items, View: "klasik", HeroStyle: "uclu",
			HeroCount: 5, HeroWidth: 44, HeroRatio: "4/5", ListCols: 4,
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", d))
	out := buf.String()
	assert.Contains(t, out, `class="cat-hero cat-hero-trio cat-ratio-4-5"`)
	assert.Contains(t, out, `style="--cat-hero-w:44%"`)
	assert.Equal(t, 1, strings.Count(out, `<div class="cat-hero-rail cat-trio-left"`), "sol ray bir kez basılmalı")
	assert.Equal(t, 1, strings.Count(out, `<div class="cat-hero-rail cat-trio-right"`), "sağ ray bir kez basılmalı")
	assert.Equal(t, 5, strings.Count(out, `class="hero-slide `), "ortada 5 slayt olmalı")
	assert.Equal(t, 5, strings.Count(out, `class="chr-t"`), "3 sol + 2 sağ manşet olmalı")
	assert.Contains(t, out, `class="cat-cards cat-cols-4"`)
	assert.Equal(t, 2, strings.Count(out, `class="card cat-card"`), "ilk 10 içerikten sonrası alt kartlarda olmalı")
	assert.NotContains(t, out, `class="cat-lead"`)
}

func TestCategoryPortraitHeroFallsBackForFewItems(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := &tmplData{
		Category: &categoryData{
			Items: []repo.Content{{ID: 1, Title: "Tek haber"}, {ID: 2, Title: "İkinci haber"}},
			View:  "klasik", HeroStyle: "portre", HeroCount: 5, HeroWidth: 58, HeroRatio: "4/5", ListCols: 2,
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", d))
	out := buf.String()
	assert.NotContains(t, out, "cat-hero", "az içerikte boş portre rayı basılmamalı")
	assert.Contains(t, out, `class="cat-cards cat-cols-2"`)
	assert.Equal(t, 2, strings.Count(out, `class="card cat-card"`))
}

func TestCategoryClassicStillRenders(t *testing.T) {
	tmpl := parseTestTemplates(t)
	items := make([]repo.Content, 7)
	for i := range items {
		items[i] = repo.Content{ID: int64(i + 1), Title: fmt.Sprintf("Haber %d", i+1)}
	}
	d := &tmplData{
		Category: &categoryData{
			Items: items, View: "klasik", HeroStyle: "klasik", ListCols: 3,
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", d))
	out := buf.String()
	assert.Contains(t, out, `class="cat-lead"`)
	assert.NotContains(t, out, `class="cat-hero"`)
	assert.Contains(t, out, `grid-2 cat-cols-3`)
}

func TestWithCatBlockCategoryHero(t *testing.T) {
	fn, ok := templateFuncs()["withCatBlock"].(func(*tmplData, *homeBlock) *tmplData)
	require.True(t, ok, "withCatBlock şablon fonksiyonu bulunmalı")
	d := &tmplData{LangBase: "", Category: &categoryData{Slug: "ekonomi", View: "izgara"}}
	b := &homeBlock{
		CatHeroStyle: "portre", CatHeroCount: 7, CatHeroWidth: 64,
		CatHeroRatio: "3/4", CatListCols: 4, HideBadge: true,
	}
	nd := fn(d, b)
	require.NotSame(t, d, nd)
	assert.Equal(t, "klasik", nd.Category.View, "portre, blok görünümü seçilmemişse klasik varyantı kullanmalı")
	assert.Equal(t, "izgara", d.Category.View, "kaynak kategori verisi değişmemeli")
	assert.Equal(t, "portre", nd.Category.HeroStyle)
	assert.Equal(t, 7, nd.Category.HeroCount)
	assert.Equal(t, 64, nd.Category.HeroWidth)
	assert.Equal(t, "3/4", nd.Category.HeroRatio)
	assert.Equal(t, 4, nd.Category.ListCols)
	assert.True(t, nd.Category.HideBadge, "etiket kapatma kategori listesine taşınmalı")
	assert.False(t, d.Category.HideBadge, "kaynak kategori verisi değişmemeli")

	b.ListView = "liste"
	nd = fn(d, b)
	assert.Equal(t, "liste", nd.Category.View, "blokta açık görünüm seçimi portre varsayılanını ezmeli")

	b.ListView = ""
	b.CatHeroStyle = "uclu"
	nd = fn(d, b)
	assert.Equal(t, "klasik", nd.Category.View, "üçlü manşet de klasik varyantını kullanmalı")
	assert.Equal(t, "uclu", nd.Category.HeroStyle)

	b.CatHeroStyle = "klasik"
	nd = fn(d, b)
	assert.Equal(t, "izgara", nd.Category.View, "klasik manşet genel görünümü zorla değiştirmemeli")
}
