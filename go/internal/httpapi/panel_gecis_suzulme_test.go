package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func statikDosyaOku(t *testing.T, ad string) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/"+ad)
	require.NoError(t, err, "statik dosya okunamadı: %s", ad)
	return string(b)
}

func TestPanelIcerikSuzulme(t *testing.T) {
	js := statikDosyaOku(t, "admin-ui.js")

	// Modül var ve hap ile aynı süre/eğri.
	assert.Contains(t, js, "icerikSuzulme")
	assert.Contains(t, js, "var SURE = 320;")
	assert.Contains(t, js, "var EGRI = 'cubic-bezier(.22,1,.36,1)'") // --pn-ease
	assert.Contains(t, js, "var MESAFE = 28;")

	// Yön: sekme sırasından (window.tbWsListe).
	assert.Contains(t, js, "window.tbWsListe = WS;")
	assert.Contains(t, js, "if (b[0] < a[0]) yon = -1;")
	assert.Contains(t, js, "else if (b[0] === a[0] && b[1] < a[1]) yon = -1;")

	assert.Contains(t, js, "function suzul(")
	assert.Contains(t, js, "'translateX(' + (yonDeger * MESAFE) + 'px)'")
	assert.Contains(t, js, "opacity: 0.45")
	assert.Contains(t, js, "delay: Math.min(i * 12, 48)")

	// Yerinde kalanlar animasyonlanmamalı: üst çubuk, kırıntı, sekme hapı.
	assert.Contains(t, js, ".admin-top, .admin-crumb, .ws-tabs, script, style, link")

	// Hareketi azalt: animasyon kurulmaz.
	assert.Contains(t, js, "(prefers-reduced-motion: reduce)")
}

// hx-swap OUTERHTML olmalı: INNERHTML iken yanıttaki .admin-main mevcut
// main'in içine giriyor ve iç içe ikinci <main> oluşuyordu.
func TestPanelGezinmeOuterHTML(t *testing.T) {
	b, err := fs.ReadFile(webui.FS, "templates/partials.html")
	require.NoError(t, err)
	html := string(b)

	assert.Contains(t, html, `hx-swap="outerHTML show:window:top"`,
		"yan menü gezinmesi OUTERHTML ile değiştirmeli")
	assert.NotContains(t, html, `hx-swap="innerHTML show:window:top"`,
		"INNERHTML + hx-select çift <main> üretiyordu")

	js := statikDosyaOku(t, "admin-ui.js")
	assert.Contains(t, js, `nav.setAttribute('hx-swap', 'outerHTML show:window:top')`,
		"dinamik sekme hapı da OUTERHTML kullanmalı")
}

// Süzülme sırasında pn-gir (dikey 3 px) ve pn-yenile susturulmalı;
// yoksa içerik hem aşağı hem yana kayar (çift hareket).
func TestPanelSuzulmeCiftHareketYok(t *testing.T) {
	css := statikDosyaOku(t, "admin-pano.css")
	assert.Contains(t, css, ".admin-main.tb-suzuluyor")
	assert.Contains(t, css, "animation: none !important;")

	js := statikDosyaOku(t, "admin-ui.js")
	assert.Contains(t, js, "kok.classList.add('tb-suzuluyor')")
	assert.Contains(t, js, "kok.classList.remove('tb-suzuluyor')")
}

// Kısmi takaslar (tablo yenile, filtre, modal) süzülmeyi tetiklememeli.
func TestPanelSuzulmeYalnizKabukGezinmesinde(t *testing.T) {
	js := statikDosyaOku(t, "admin-ui.js")

	// htmx, boost'lu gezinmede detail.elt'e TAKAS HEDEFİNİ koyuyor
	// (ölçüldü). Guard iki durumu da kabul etmeli.
	assert.Contains(t, js, "tetik.classList.contains('admin-main')")
	assert.Contains(t, js, `tetik.closest('[hx-target=".admin-main"]')`)
	assert.Contains(t, js, "if (!kabukGezinmesi) return;")

	// outerHTML sonrası detail.target DOM'dan çıkmış oluyor; güncel öğe
	// DOM'dan sorgulanmalı, yoksa animasyon görünmez (ölçüldü).
	assert.Contains(t, js, "var kok = document.querySelector('.admin-main');")
	assert.Contains(t, js, "kok.isConnected")
	assert.NotContains(t, js, "var kok = e.detail && e.detail.target;\n      if (!kok || !kok.classList",
		"eski (DOM'dan çıkmış) hedefe animasyon uygulanmamalı")
}

// Sekme sırası listesi dışarı açık olmalı (yön hesabı başka modülde).
func TestPanelWsSirasiPaylasilir(t *testing.T) {
	js := statikDosyaOku(t, "admin-ui.js")
	idx := strings.Index(js, "window.tbWsListe = WS;")
	require.Greater(t, idx, 0)
	// WS tanımından SONRA gelmeli.
	assert.Greater(t, idx, strings.Index(js, "var WS = ["))
	// İçerik sekmesi sırası: İçerikler → Yeni İçerik → Kategoriler…
	assert.Contains(t, js, "['İçerikler', '/admin/icerikler']")
	assert.Contains(t, js, "['Kategoriler', '/admin/kategoriler']")
}
