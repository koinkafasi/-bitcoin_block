package httpapi

// Panel yan menüsünün daralt/genişlet geçişi (kullanıcı isteği):
// demo.codeandchill.store/pacely sol kolon menüsünün BİREBİR aynısı.
//
// Kaynak ölçümü (3 Kasım 2026, pacely paketi: assets/index-BlFqaHp6.js →
// qt() ve Jt()):
//
//   DARALTMA
//     etiketler  opacity 1→0, x 0→-8, 140 ms, power2.in
//     menü       clip-path inset(0 0 0 0) → inset(0 (w-88)px 0 0),
//                300 ms, power3.in
//     bitince    sınıf değişir (width ANLIK), stiller temizlenir
//   GENİŞLETME
//     sınıf hemen değişir; clip-path
//                inset(0 (w-88)px 0 0) → inset(0 0 0 0), 500 ms, expo.out
//     etiketler  opacity 0→1, x -10→0, 550 ms, power3.out,
//                gecikme 120 ms, kademe 10 ms
//   ANA KOLON  her iki yönde FLIP telafisi, 600 ms, expo.out
//
// theblock'ta GSAP yok: aynı çağrılar Web Animations API ile yapılıyor.
// Eğri karşılıkları:
//   power2.in  cubic-bezier(.55,.085,.68,.53)
//   power3.in  cubic-bezier(.55,.055,.675,.19)
//   power3.out cubic-bezier(.215,.61,.355,1)
//   expo.out   cubic-bezier(.19,1,.22,1)

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func statikOku(t *testing.T, ad string) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/"+ad)
	require.NoError(t, err, "statik dosya okunamadı: %s", ad)
	return string(b)
}

// Genişlik CSS geçişiyle DEĞİL, clip-path perdesiyle animasyonlanmalı.
// İkisi birlikte olursa menü iki kez hareket eder (çift animasyon).
func TestPanelRailCssGenislikGecisiYok(t *testing.T) {
	css := statikOku(t, "admin-pano.css")

	// .admin-side ve mark kutusu için `transition: width` kalmamalı.
	assert.NotContains(t, css, "transition: width .26s var(--pn-ease)",
		"genişlik geçişi clip-path animasyonuyla çakışır")

	// Ray genişliği Pacely --rail-w ile aynı olmalı.
	assert.Contains(t, css, "html.admin-rail .admin-side { width: 88px; }")
}

func TestPanelRailJsPacelyDegerleri(t *testing.T) {
	js := statikOku(t, "admin-ui.js")

	// Ray genişliği ve eğriler birebir.
	assert.Contains(t, js, "var RAY_GENISLIK = 88;")
	assert.Contains(t, js, "power2in: 'cubic-bezier(.55,.085,.68,.53)'")
	assert.Contains(t, js, "power3in: 'cubic-bezier(.55,.055,.675,.19)'")
	assert.Contains(t, js, "power3out: 'cubic-bezier(.215,.61,.355,1)'")
	assert.Contains(t, js, "expoOut: 'cubic-bezier(.19,1,.22,1)'")

	// Daraltma: 140 ms etiket soldurma + 300 ms perde.
	assert.Contains(t, js, "duration: 140, easing: RAY_EGRI.power2in")
	assert.Contains(t, js, "RAY_EGRI.power3in, function ()")

	// Genişletme: 500 ms perde + 550 ms etiket, 120 ms gecikme, 10 ms kademe.
	assert.Contains(t, js, "500, RAY_EGRI.expoOut, null")
	assert.Contains(t, js, "duration: 550, easing: RAY_EGRI.power3out")
	assert.Contains(t, js, "delay: 120 + i * 10")

	// Ana kolon FLIP telafisi 600 ms expo.out.
	assert.Contains(t, js, "duration: 600, easing: RAY_EGRI.expoOut")

	// Perde inset() ile: clip-path perdesi (Pacely birebir).
	assert.Contains(t, js, "'inset(0px 0px 0px 0px)'")
	assert.Contains(t, js, "'inset(0px ' + perdeDar + 'px 0px 0px)'")
	assert.Contains(t, js, "'inset(0px ' + perdeGen + 'px 0px 0px)'")

	// Hareketi azalt: animasyon kurulmaz, sınıf anında değişir.
	assert.Contains(t, js, "(prefers-reduced-motion: reduce)")
}

// Perde miktarı DARALTMA öncesi, GENİŞLETME sonrası ölçülmeli.
// Aksi hâlde genişletmede menü hâlâ 88 px olduğu için perde 0 px çıkar ve
// geçiş görünmez olur (ölçüldü: inset(0px)→inset(0px), hiç animasyon yok).
func TestPanelRailPerdeYonu(t *testing.T) {
	js := statikOku(t, "admin-ui.js")

	daraltIdx := strings.Index(js, "var perdeDar = yan.offsetWidth - RAY_GENISLIK;")
	sinifIdx := strings.Index(js, "sinifiUygula();\n      // 2) perde")
	genIdx := strings.Index(js, "var perdeGen = yan.offsetWidth - RAY_GENISLIK;")

	require.Greater(t, daraltIdx, 0, "daraltma perdesi bulunamadı")
	require.Greater(t, genIdx, 0, "genişletme perdesi bulunamadı")
	require.Greater(t, sinifIdx, 0, "genişletmede sınıf uygulaması bulunamadı")

	// Genişletmede sınıf, perde hesabından ÖNCE gelmeli (Pacely: o(); e=...).
	assert.Less(t, sinifIdx, genIdx,
		"genişletmede önce sınıf değişmeli, perde genişliği sonra ölçülmeli")

	// Bitince animasyon iptal edilmeli: fill:both DOM'da kalırsa sonraki
	// geçişte eski clip-path değeri yarışır.
	assert.Contains(t, js, "try { a.cancel(); } catch (e) {}")
}

// Etiketler ayrı düğüm olmalı: Pacely yalnız .nav__label solduruyor,
// ikonlar yerinde kalıyor. theblock'ta etiket çıplak metin düğümüydü.
func TestPanelRailEtiketDugumu(t *testing.T) {
	b, err := fs.ReadFile(webui.FS, "templates/partials.html")
	require.NoError(t, err)
	html := string(b)

	etiket := regexp.MustCompile(`</svg> <span class="mn">`)
	bulunan := etiket.FindAllStringIndex(html, -1)
	assert.GreaterOrEqual(t, len(bulunan), 14,
		"yan menüdeki tüm etiketler <span class=\"mn\"> içinde olmalı (bulunan: %d)", len(bulunan))

	// Menü bağlantılarının hepsi kapsanmış olmalı: kalan çıplak etiket yok.
	assert.NotContains(t, html, "</svg> Pano</a>")
	assert.NotContains(t, html, "</svg> İçerik</a>")

	// JS bu düğümleri hedefliyor.
	js := statikOku(t, "admin-ui.js")
	assert.Contains(t, js, "yan.querySelectorAll('.menu a .mn, .brand')")
}
