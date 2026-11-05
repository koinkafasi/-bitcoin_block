package httpapi

import (
	"bytes"
	"html/template"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

// Finans modüllerinin köşe yarıçapı ve arka plan sözleşmesi.
//
// Kullanıcı isteği: modüllerin yuvarlak köşeleri değiştirilebilir olmalı;
// site geneli köşeli seçildiğinde finans modülleri de düzleşmeli. Arka plan
// kapatılabilmeli, değiştirilebilmeli ve saydam yapılabilmeli.

func TestBuildStyleKoseYuvarlakligi(t *testing.T) {
	// Açıkça 0: tamamen köşeli. Yerel değişkenler iç kartlara da geçer.
	b := homeBlock{StRadiusSet: true, StRadius: 0}
	b.buildStyle()
	require.True(t, b.Wrap)
	s := string(b.Style)
	assert.Contains(t, s, "--hp-radius:0px")
	assert.Contains(t, s, "--hp-radius-sm:0px")
	assert.Contains(t, s, "border-radius:0px")

	// 12px: küçük yarıçap 10px'e iner.
	b = homeBlock{StRadiusSet: true, StRadius: 12}
	b.buildStyle()
	s = string(b.Style)
	assert.Contains(t, s, "--hp-radius:12px")
	assert.Contains(t, s, "--hp-radius-sm:10px")
	assert.Contains(t, s, "border-radius:12px")

	// Alan hiç doldurulmadıysa tema değeri miras alınır; inline kural yok.
	b = homeBlock{StRadiusSet: false, StRadius: 0}
	b.buildStyle()
	assert.False(t, b.Wrap, "boş alan sarmalayıcı üretmemeli")
	assert.NotContains(t, string(b.Style), "--hp-radius")

	// Sınır dışı değer kırpılır.
	b = homeBlock{StRadiusSet: true, StRadius: 999}
	b.buildStyle()
	assert.Contains(t, string(b.Style), "--hp-radius:60px")
}

func TestParseStRadiusSifirAyarlanmis(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[
		{"id":"a","type":"kripto_para","opts":{"stRadius":0}},
		{"id":"b","type":"kripto_para","opts":{}},
		{"id":"c","type":"kripto_para","opts":{"stRadius":null}},
		{"id":"d","type":"kripto_para","opts":{"stRadius":""}}
	]`))
	require.Len(t, blocks, 4)
	assert.True(t, blocks[0].StRadiusSet, "0 açıkça verildiğinde ayarlanmış sayılır")
	assert.Equal(t, 0, blocks[0].StRadius)
	assert.False(t, blocks[1].StRadiusSet, "alan yoksa tema yarıçapı geçerli")
	assert.False(t, blocks[2].StRadiusSet, "null boş sayılır")
	assert.False(t, blocks[3].StRadiusSet, "boş string boş sayılır")
}

func TestFinansArkaPlanSecenekleri(t *testing.T) {
	// Hisseler: mevcut görünümü korumak için varsayılan kart.
	assert.Equal(t, "kart", hisseBgOf(""))
	assert.Equal(t, "kart", hisseBgOf("bilinmeyen"))
	for _, bg := range []string{"kart", "yok", "zemin", "ozel"} {
		assert.Equal(t, bg, hisseBgOf(bg))
		assert.Equal(t, bg, kriptoBgOf(bg))
		assert.Equal(t, bg, kurCeviriciBgOf(bg))
		assert.Equal(t, bg, servisBgOf(bg))
	}
	// Kripto, çevirici ve servis bandı da kart varsayılanıyla mevcut görünümü korur.
	assert.Equal(t, "kart", kriptoBgOf(""))
	assert.Equal(t, "kart", kurCeviriciBgOf(""))
	assert.Equal(t, "kart", servisBgOf(""))
	// Kur & Borsa referansı saydam kalır.
	assert.Equal(t, "yok", kbBgOf(nil))
}

func TestFinansArkaPlanSiniflariRender(t *testing.T) {
	tmpl := parseTestTemplates(t)

	// Kripto: yok seçimi bölüm sınıfına ve saydamlık CSS'ine yansır.
	kb := &homeBlock{
		Type:           "kripto_para",
		Kripto:         kriptoOrnek(),
		KriptoYerlesim: "izgara",
		KriptoStil:     "referans",
		KriptoEtkin:    "referans",
		KriptoKolon:    4,
		KriptoDegisim:  true,
		KriptoIkon:     true,
		KriptoBg:       "yok",
		D:              &tmplData{LangBase: ""},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", kb))
	out := buf.String()
	assert.Contains(t, out, "krp-bg-yok")
	assert.Contains(t, out, "krp-block")

	// Çevirici: özel renk bölümün inline stiline yazılır.
	kc := &homeBlock{
		Type: "kur_cevirici",
		KurCevirici: &kurCeviriciView{
			Baslik: "KUR ÇEVİRİCİ",
			Para: []kurPara{
				{Code: "USD", Label: "Amerikan Doları", Rate: 49.84},
				{Code: "TRY", Label: "Türk Lirası", Rate: 1},
			},
			From: "USD", To: "TRY", Miktar: 100, Sonuc: 4984, Var: true,
		},
		KCBg:    "ozel",
		KCBgCSS: template.CSS(kbArkaPlanCSS("ozel", "#102030")),
		D:       &tmplData{LangBase: ""},
	}
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", kc))
	out = buf.String()
	assert.Contains(t, out, "kcr-bg-ozel")
	assert.Contains(t, out, "background:#102030")

	// Hisseler: saydam seçim sınıfı.
	hs := hisselerBlogu(hisseOrnek(), "serit")
	hs.HisseBg = "yok"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", hs))
	assert.Contains(t, buf.String(), "hss-bg-yok")

	// Servis bandı: saydam seçim sınıfı.
	sv := &homeBlock{
		Type:         "servis_bandi",
		ServisBg:     "yok",
		ServisKurlar: []repo.Kur{{Name: "USD", Value: 49.84, Delta: 0.12}},
		D:            &tmplData{LangBase: "", Home: &homeData{}},
	}
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", sv))
	assert.Contains(t, buf.String(), "sv-bg-yok")
}

func TestFinansCSSYaricapDegiskeni(t *testing.T) {
	b, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	css := string(b)
	for _, selector := range []string{
		".hss-frame {",
		".krp-card {",
		".krp-frame {",
		".krp-list-rail {",
		".kcr-block {",
	} {
		block := kural(css, selector)
		require.NotEmpty(t, block, "%s bulunmalı", selector)
		assert.Contains(t, block, "border-radius: var(--hp-radius",
			"%s tema yarıçapını izlemeli", selector)
	}
	// Saydam arka plan sınıfları gerçekten yüzeyi kaldırır.
	assert.Contains(t, css, ".hss-bg-yok .hss-frame")
	assert.Contains(t, css, ".krp-bg-yok .krp-card")
	assert.Contains(t, css, ".kcr-block.kcr-bg-yok")
	assert.Contains(t, css, ".sv-bg-yok .tk-inner")
	assert.Contains(t, css, "var(--sv-bg")
	assert.Contains(t, css, "border-radius: var(--hp-radius-sm)")
}

func TestFinansModulArkaPlanAnahtarlari(t *testing.T) {
	// Kripto ve çevirici blok ayarları kayıtlı düzenlerden okunur.
	blocks := parseHomeBlocks([]byte(`[
		{"id":"k","type":"kripto_para","opts":{"kp_bg":"yok"}},
		{"id":"c","type":"kur_cevirici","opts":{"kc_bg":"zemin"}},
		{"id":"h","type":"hisseler","opts":{"hs_bg":"ozel"}}
	]`))
	require.Len(t, blocks, 3)
	assert.Equal(t, "yok", blocks[0].KriptoBg)
	assert.Equal(t, "zemin", blocks[1].KCBg)
	assert.Equal(t, "ozel", blocks[2].HisseBg)

	// Bilinmeyen değer kart varsayılanına döner.
	blocks = parseHomeBlocks([]byte(`[{"id":"k","type":"kripto_para","opts":{"kp_bg":"bozuk"}}]`))
	require.Len(t, blocks, 1)
	assert.Equal(t, "kart", blocks[0].KriptoBg)
}

func TestFinansKriptoOrnekSatirlar(t *testing.T) {
	// kriptoOrnek yardımcısının boş olmadığını doğrula (render testi veri ister).
	rows := kriptoOrnek()
	require.NotEmpty(t, rows)
	assert.True(t, strings.EqualFold(rows[0].Symbol, "BTC"))
}
