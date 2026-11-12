package httpapi

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

func tnItems() []repo.Content {
	slug := func(s string) *string { return &s }
	return []repo.Content{
		{ID: 1, Title: "Birinci haber", Slug: slug("birinci-haber")},
		{ID: 2, Title: "İkinci haber", Slug: slug("ikinci-haber")},
	}
}

func renderTrendNumarali(t *testing.T, stil string, radius int, css template.CSS) string {
	t.Helper()
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	tmpl, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)

	b := &homeBlock{
		Type:           "trend_numarali",
		D:              &tmplData{},
		Items:          tnItems(),
		TrendNumStil:   stil,
		TrendNumRadius: radius,
		TrendNumCSS:    css,
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_trend_numarali", b))
	return buf.String()
}

// Varsayılan görünüm korunur: tn-blok, köşe değişkeni basılmaz.
func TestTrendNumaraliVarsayilan(t *testing.T) {
	h := renderTrendNumarali(t, "blok", -1, "")

	assert.Contains(t, h, `class="trend-num side-trend tn-blok"`)
	assert.NotContains(t, h, "--tn-radius", "ayar verilmedikçe inline değişken yok")
	// Sayı, iç öğe ile sarılır (stil kuralları buradan beslenir).
	assert.Contains(t, h, `<span class="st-n"><span class="st-n-i">1</span></span>`)
	assert.Contains(t, h, `<span class="st-n"><span class="st-n-i">2</span></span>`)
}

func TestTrendNumaraliStilleri(t *testing.T) {
	for _, stil := range []string{"blok", "duz", "serit", "rozet", "buyuk", "kontur"} {
		h := renderTrendNumarali(t, stil, -1, "")
		assert.Contains(t, h, `side-trend tn-`+stil, "stil sınıfı basılmalı: %s", stil)
	}
}

// Köşe yarıçapı: 0 geçerli (tamamen düz), 40 üst sınır.
func TestTrendNumaraliKoseYaricapi(t *testing.T) {
	h := renderTrendNumarali(t, "blok", 0, template.CSS("--tn-radius:0px"))
	assert.Contains(t, h, `style="--tn-radius:0px"`, "0px de basılmalı (düz köşe)")

	h = renderTrendNumarali(t, "blok", 14, template.CSS("--tn-radius:14px;--tn-bg:#123456"))
	assert.Contains(t, h, `--tn-radius:14px`)
	assert.Contains(t, h, `--tn-bg:#123456`)

	// Ayar bloğu: -1 varsayılan, 0 kabul, üst sınır 40, geçersiz stil → blok.
	bloklar := parseHomeBlocks([]byte(`[
		{"id":"a","type":"trend_numarali","opts":{"zone":"ana","tn_stil":"rozet","tn_radius":0}},
		{"id":"b","type":"trend_numarali","opts":{"zone":"ana","tn_radius":99}},
		{"id":"c","type":"trend_numarali","opts":{"zone":"ana","tn_stil":"yok-boyle-bir-sey"}},
		{"id":"d","type":"trend_numarali","opts":{"zone":"ana"}}
	]`))
	require.Len(t, bloklar, 4)

	assert.Equal(t, "rozet", bloklar[0].TrendNumStil)
	assert.Equal(t, 0, bloklar[0].TrendNumRadius, "0 = düz köşe, ayarlanmamış sayılmaz")
	assert.Contains(t, string(bloklar[0].TrendNumCSS), "--tn-radius:0px")

	assert.Equal(t, 40, bloklar[1].TrendNumRadius, "üst sınır 40px")
	assert.Equal(t, "blok", bloklar[2].TrendNumStil, "geçersiz stil varsayılana döner")
	assert.Equal(t, -1, bloklar[3].TrendNumRadius, "-1 = tema yarıçapı")
	assert.Empty(t, string(bloklar[3].TrendNumCSS), "ayar yoksa inline stil yok")
}

// CSS sözleşmesi: köşe kontrolü hem iç (--tn-radius) hem dış (--tn-sag)
// köşeyi kapsamalı. Dış köşe yalnız kartın kırpmasıyla kontrol edilebilir.
func TestTrendNumaraliStilSozlesmesi(t *testing.T) {
	css, err := webui.FS.ReadFile("static/public.css")
	require.NoError(t, err)
	s := string(css)

	assert.Contains(t, s, ".st-n {", "sayısal alan tabanı")
	assert.Contains(t, s, "border-start-start-radius: var(--tn-radius, 0)",
		"iç köşe ayardan gelir")
	// Dış köşe: kartın sağ köşeleri değişkene bağlı olmalı.
	assert.Contains(t, s, "border-start-end-radius: var(--tn-sag",
		"dış köşe karttan kontrol edilir")
	assert.Contains(t, s, ".tn-duz { --tn-radius: 0px; --tn-sag: 0px; }",
		"düz stil hem iç hem dış köşeyi sıfırlar")

	// Altı stil de tanımlı olmalı.
	for _, stil := range []string{"tn-serit", "tn-rozet", "tn-buyuk", "tn-kontur"} {
		assert.Contains(t, s, "."+stil+" .st-n", "stil kuralı eksik: %s", stil)
	}
	assert.Contains(t, s, "prefers-reduced-motion", "hareket azaltma korunur")
	// Renk ayarları değişkenle gelir; sabit renk yazılmaz.
	assert.Contains(t, s, "var(--tn-bg,", "arka plan ayardan gelir")
	assert.Contains(t, s, "var(--tn-fg,", "yazı rengi ayardan gelir")
}

// Panel: palet şemasında stil + yarıçap + renk alanları olmalı; yeni blok
// varsayılanı tn_stil taşımalı.
func TestBedrockTrendNumaraliPanelAlanlari(t *testing.T) {
	ham, err := webui.FS.ReadFile("templates/site_builder.html")
	require.NoError(t, err)
	s := string(ham)

	assert.Contains(t, s, "['tn_stil','Sayısal alan stili','choice'")
	assert.Contains(t, s, "['tn_radius','Sayısal alan köşe yarıçapı (0 = düz, boş = tema)','number'")
	assert.Contains(t, s, "['tn_bg','Sayı arka planı / konturu','color'")
	assert.Contains(t, s, "['tn_fg','Sayı rengi','color']")
	assert.Contains(t, s, "type==='trend_numarali') nb.opts={zone:zone,count:6,tn_stil:'blok'}",
		"yeni blok varsayılanı")

	// Altı stil seçeneğinin tamamı sunulmalı (CSS'te karşılığı var).
	for _, stil := range []string{"'blok'", "'duz'", "'serit'", "'rozet'", "'buyuk'", "'kontur'"} {
		assert.Contains(t, s, stil, "seçenek eksik: %s", stil)
	}
}

// Modül başka bölgeye taşınsa da sayısal alan bozulmamalı: stil sınıfı
// bölgeden bağımsızdır.
func TestTrendNumaraliBolgedenBagimsiz(t *testing.T) {
	for _, zone := range []string{"ana", "yan", "genis", "ust"} {
		bloklar := parseHomeBlocks([]byte(`[{"id":"x","type":"trend_numarali","opts":{"zone":"` + zone + `","tn_stil":"rozet","tn_radius":0}}]`))
		require.Len(t, bloklar, 1)
		assert.Equal(t, zone, bloklar[0].Zone)
		assert.Equal(t, "rozet", bloklar[0].TrendNumStil)
		h := renderTrendNumarali(t, bloklar[0].TrendNumStil, bloklar[0].TrendNumRadius, bloklar[0].TrendNumCSS)
		assert.True(t, strings.Contains(h, "tn-rozet"), "bölge %s: stil korunur", zone)
	}
}
