package httpapi

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSeoDoldurRotaKayitli: rota panelin içe aktarma listesinde olmalı,
// yoksa düğme 404 alır.
func TestSeoDoldurRotaKayitli(t *testing.T) {
	src := mustRead(t, "server.go")
	assert.Contains(t, src, `"/admin/icerikler/{id}/ai/seo-doldur"`,
		"SEO doldurma rotası kayıtlı olmalı")
	assert.Contains(t, src, "s.handleSeoDoldur")
}

// TestSeoDoldurHataMetni: sağlayıcı yoksa teknik ayrıntı yerine ne
// yapılacağını söyleyen metin dönmeli.
func TestSeoDoldurHataMetni(t *testing.T) {
	// aieditor'ın ham hatası kullanıcıya gösterilmemeli.
	msg := seoHataMetni(errString("AI sağlayıcısı yapılandırılmadı"))
	assert.Contains(t, msg, "Ayarlar")
	assert.NotContains(t, msg, "yapılandırılmadı:")

	msg = seoHataMetni(errString("SEO yanıtı JSON değil: unexpected"))
	assert.Contains(t, msg, "yeniden deneyin")

	msg = seoHataMetni(errString("SEO için başlık veya gövde gerekli"))
	assert.Contains(t, msg, "başlığı veya gövdesi")
}

func TestSeoDoldurOnizlemeYazmaz(t *testing.T) {
	h := mustRead(t, "seo_doldur_handlers.go")

	assert.NotContains(t, h, "aieditor.SeoYaz(",
		"önizleme uç noktası içeriğe yazmamalı")
	assert.NotContains(t, h, `FormValue("force")`,
		"force bayrağı kalkmalı: yazma yalnız arka plan işinde")

	// Üretim ve öneri doldurma hâlâ yapılıyor.
	i := strings.Index(h, "aieditor.SeoUret(")
	require.Greater(t, i, 0, "SeoUret çağrısı bulunamadı")
	j := strings.Index(h, "s.renderSeoOneri(w, r, v)")
	assert.Greater(t, j, i)
	arada := h[i:j]
	assert.Contains(t, arada, "v.Baslik = oneri.Baslik")
	assert.Contains(t, arada, "v.Aciklama = oneri.Aciklama")

	// Yazma yalnız arka plan işinde.
	w := mustRead(t, "../aieditor/seo_worker.go")
	assert.Contains(t, w, "SeoYaz(ctx, pool, c.ID, oneri, false)",
		"arka plan işi dolu alanı ezmemeli")
}

// TestSeoDoldurSablonBaglantisi: form düğmesi hedef div'e bağlı olmalı ve
// panel boşken görünmemeli.
func TestSeoDoldurSablonBaglantisi(t *testing.T) {
	form := mustRead(t, "../webui/templkit/icerikform.templ")
	assert.Contains(t, form, "/ai/seo-doldur")
	assert.Contains(t, form, `hx-target="#seoOneri"`)
	assert.Contains(t, form, "@SeoOneriPanel(")

	panel := mustRead(t, "../webui/templkit/seo_oneri.templ")
	assert.Contains(t, panel, `id="seoOneri"`)
	assert.Contains(t, panel, "data-seo-uygula")
	assert.Contains(t, panel, "data-baslik={ v.Baslik }")
	assert.Contains(t, panel, "data-aciklama={ v.Aciklama }")
	// Boş durumda panel gövdesi basılmamalı (sayfada boş kutu görünmesin).
	assert.Contains(t, panel, `v.Baslik == "" && v.Aciklama == "" && v.Hata == ""`)

	js := mustRead(t, "../webui/static/admin-ui.js")
	assert.Contains(t, js, "[data-seo-uygula]")
	assert.Contains(t, js, "[name=seo_baslik]")
	assert.Contains(t, js, "[name=seo_aciklama]")
	// SERP/sayaç önizlemesi `input` dinliyor: olay tetiklenmeli.
	assert.Contains(t, js, "new Event('input'")
}

// TestSeoBotAyarAnahtari: panelden açılıp kapanabilmeli.
func TestSeoBotAyarAnahtari(t *testing.T) {
	bot := mustRead(t, "bot_handlers.go")
	assert.Contains(t, bot, `"seo_otomatik"`)
	assert.Contains(t, bot, `"seo_dakika"`,
		"hız dakika cinsinden olmalı (kuyruk erisin)")
	assert.Contains(t, bot, "aieditor.SeoDoldurAyarlari")

	tpl := mustRead(t, "../webui/templkit/trend.templ")
	assert.Contains(t, tpl, `name="seo_otomatik"`)
	assert.Contains(t, tpl, `name="seo_dakika"`)

	main := mustRead(t, "../../cmd/site/main.go")
	assert.Contains(t, main, "aieditor.RunSeoDoldurma",
		"arka plan işi başlatılmalı")
}

// errString basit hata üreticisi (test yardımcısı).
type errString string

func (e errString) Error() string { return string(e) }

// mustRead test dosyasını okur (yol paket dizinine göre).
func mustRead(t *testing.T, yol string) string {
	t.Helper()
	b, err := os.ReadFile(yol)
	require.NoError(t, err, "okunamadı: %s", yol)
	return string(b)
}

func TestSeoOnizlemeKapsayicisi(t *testing.T) {
	form := mustRead(t, "../webui/templkit/icerikform.templ")
	assert.Contains(t, form, "data-seo-preview",
		"canlı önizleme kapsayıcısı formda olmalı")

	// Kapsayıcı, önizlenen üç bloğu (gauge, SERP, sosyal kart) içermeli.
	i := strings.Index(form, "data-seo-preview")
	require.Greater(t, i, 0)
	j := strings.Index(form[i:], "</details>")
	require.Greater(t, j, 0)
	ic := form[i : i+j]
	assert.Contains(t, ic, `class="gauge"`, "puan göstergesi kapsayıcı içinde")
	assert.Contains(t, ic, `class="serp"`, "SERP önizlemesi kapsayıcı içinde")
	assert.Contains(t, ic, "data-social-preview", "sosyal kart kapsayıcı içinde")

	// SEO alanları `data-src` taşımalı; app.js bunları tercih ediyor.
	assert.Contains(t, form, `name="seo_baslik" value={ v.SeoTitle } data-src="seo_title"`)
	assert.Contains(t, form, `data-src="seo_desc"`)

	js := mustRead(t, "../webui/static/app.js")
	assert.Contains(t, js, `alanDeger("[data-src=seo_title]")`,
		"SEO başlığı doluysa önizlemede o gösterilmeli")
	assert.Contains(t, js, `alanDeger("[data-src=seo_desc]")`)
}
