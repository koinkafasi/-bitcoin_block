package httpapi

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

func TestSayiliStilPagerNumarayaZorlanir(t *testing.T) {
	bs := parseHomeBlocks([]byte(`[
		{"id":"a","type":"slider","opts":{"slider_style":"sayili","pager_type":"dots"}},
		{"id":"b","type":"slider","opts":{"slider_style":"default","pager_type":"dots"}}
	]`))
	require.Len(t, bs, 2)

	assert.Equal(t, "sayili", bs[0].SliderStyle)
	assert.Equal(t, "numbers", bs[0].PagerType,
		"sayılı şerit stili numaralı göstergenin parçası")
	assert.True(t, bs[0].ShowAll, "son hücredeki + için show_all açılmalı")

	assert.Equal(t, "dots", bs[1].PagerType, "diğer stiller etkilenmemeli")
	assert.False(t, bs[1].ShowAll)
}

func TestSayiliStilSablonSiniflari(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		Type:        "slider",
		SliderStyle: "sayili",
		PagerType:   "numbers",
		ShowAll:     true,
		D:           &tmplData{LangBase: ""},
		Slides: []repo.Content{
			{ID: 1, Title: "Bir"},
			{ID: 2, Title: "İki"},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_slider", b))
	out := buf.String()

	assert.Contains(t, out, "hero-row hero-row-sayili",
		"stil sınıfı satırda da olmalı: .hero-row, section'ın ÜST öğesidir")
	assert.Contains(t, out, "hero-slider hero-style-sayili")
	assert.Contains(t, out, `class="hero-dot-n"`,
		"şerit hücresinde sayı ayrı sarmalayıcıda olmalı")
	assert.Contains(t, out, "hero-dot-all", "son hücre + hücresi")
	assert.Contains(t, out, `<span class="hero-dot-n">+</span>`,
		"referansta son hücre T değil +")
}

func TestSayiliStilCSSKurallari(t *testing.T) {
	raw, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	css := string(raw)

	// 50/50 düzen ve kare oran. Kart boşluğu değişkenleri modül bazında
	// ayarlanabilsin diye iki değerli gap kullanılır (varsayılan 20px).
	assert.Contains(t, css, ".hero-row-sayili { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: var(--blk-gap-y, 20px) var(--blk-gap-x, 20px); }",
		"stil sınıfı .hero-row'un üstünde olduğu için satır sınıfı ayrı olmalı")
	assert.Contains(t, css, "aspect-ratio: 1 / 1;")

	// Şerit: 32px yükseklik, siyah/15 zemin, beyaz/20 hücre.
	i := strings.Index(css, ".hero-style-sayili .hero-dots {")
	require.Greater(t, i, 0, "şerit kuralı bulunmalı")
	blok := css[i : i+420]
	assert.Contains(t, blok, "height: 32px")
	assert.Contains(t, blok, "rgba(0, 0, 0, .15)")

	j := strings.Index(css, ".hero-style-sayili .hero-dot {")
	require.Greater(t, j, 0)
	assert.Contains(t, css[j:j+420], "rgba(255, 255, 255, .20)")

	// Aktif hücrenin eğik zemini: kaynaktaki .bg-transfer-white kuralı.
	assert.Contains(t, css, ".hero-style-sayili .hero-dot.on::before { background: hsla(0, 0%, 100%, .30); }")

	// Eğik zemin + sağ çizgi.
	k := strings.Index(css, ".hero-style-sayili .hero-dot::before {")
	require.Greater(t, k, 0)
	assert.Contains(t, css[k:k+320], "skewX(-12deg)")
	assert.Contains(t, css[k:k+320], "border-right: 1px solid #1f1f1f")
}

func TestSayiliStilPanelSecenegi(t *testing.T) {
	raw, err := fs.ReadFile(webui.FS, "templates/site_builder.html")
	require.NoError(t, err)
	assert.Contains(t, string(raw), `['sayili','Stil 6 · Sayılı Şerit']`,
		"panelde stil seçeneği görünmeli")
}
