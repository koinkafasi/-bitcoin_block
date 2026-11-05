package httpapi

// Yan menüdeki geçiş/animasyon farkları (kullanıcı isteği, 3 Kasım 2026):
// demo.codeandchill.store/pacely sol menüsüyle BİREBİR olmalı.
//
// Referans ölçümü (pacely/assets/index-BlFqaHp6.js + kit paketi `yo`):
//
//   • `.nav__pill`: `.side__scroll` içinde mutlak konumlu TEK span.
//     Aktif menü öğesinin KENDİ zemini yok
//     (`.nav__item[aria-current=page]{background:0 0}`).
//   • Yerleştirme `inset(t r b l round 12px)` + BASKIN EKSEN:
//       m = |Δt| > |Δl|  → dikey hareket
//       önde giden kenar   360 ms  power3.out
//       arkada kalan kenar 550 ms  power3.inOut
//       çapraz iki kenar   450 ms  power3.inOut
//   • Giriş (mount) efekti:
//       .side             x −28 → 0, opacity 0 → 1, 700 ms expo.out
//       .side__head/.nav__item…  x −12 → 0, 500 ms power3.out,
//                                kademe 25 ms, gecikme 100 ms
//       .nav__pill        scaleX .4 → 1, opacity 0 → 1, 600 ms back.out(2),
//                         origin 0% 50%, gecikme 450 ms
//   • Ray modunda pill GİZLENMEZ; ray değişince animasyonsuz yeniden yerleşir.
//
// theblock'ta karşılığı: `admin-ui.js` içindeki `sbKur` / `sbYerlestir` /
// `sbGiris` ve `admin-pano.css` içindeki `.tb-sb-hap`.

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func jsDosya(t *testing.T, ad string) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/"+ad)
	require.NoError(t, err)
	return string(b)
}

func TestYanMenuHapSureleriBirebir(t *testing.T) {
	js := jsDosya(t, "admin-ui.js")

	// Üç ayrı süre: önde 360, arkada 550, çapraz 450 (pacely `yo`).
	assert.Contains(t, js, "var SB_MS = { onde: 360, arkada: 550, capraz: 450 };")
	// Yarıçap 12 (`.nav__pill` border-radius / round 12px).
	assert.Contains(t, js, "var SB_YARICAP = 12;")

	// Basamak ekseni: dikey mi yatay mı?
	assert.Contains(t, js, "var dikey = Math.abs(hedefE.t - eski.t) > Math.abs(hedefE.l - eski.l);")
	// Önde/arkada/çapraz kenarlar.
	assert.Contains(t, js, "on = ileri ? 'b' : 't';")
	assert.Contains(t, js, "on = ileri ? 'r' : 'l';")
	assert.Contains(t, js, "capraz = ['l', 'r'];")
	assert.Contains(t, js, "capraz = ['t', 'b'];")

	// Eğriler: power3.out / power3.inOut (yurusun mevcut yardımcıları).
	assert.Contains(t, js, "SB_MS.onde, out3")
	assert.Contains(t, js, "SB_MS.arkada, inOut3")
	assert.Contains(t, js, "SB_MS.capraz, inOut3")
}

func TestYanMenuHapYerlesimi(t *testing.T) {
	js := jsDosya(t, "admin-ui.js")

	// Koordinatlar rayın GERÇEK ölçüsünden (taşan içerik varsa scrollWidth).
	assert.Contains(t, js, "ray.scrollWidth > ray.clientWidth")
	assert.Contains(t, js, "ray.scrollHeight > ray.clientHeight")
	// inset(... round 12px) yazımı.
	assert.Contains(t, js, "'inset(' + e.t + 'px ' + e.r + 'px ' + e.b + 'px ' + e.l + 'px round '")
	// Aynı hedefe animasyon sürerken yeniden yerleşme yok (sıçrama).
	assert.Contains(t, js, "if (animate && ayni && pill.__anim) return;")
	// Ray değişimi / yazı tipi yüklenmesi: animasyonsuz yerleşim.
	assert.Contains(t, js, "sbYerlestir(pill, ray, hedefBul(), false);")
}

func TestYanMenuHapKurulumuGecYukleniyor(t *testing.T) {
	js := jsDosya(t, "admin-ui.js")

	// sbKur aktif öğeyi izlemeli (sınıf/aria-current değişimi).
	assert.Contains(t, js, "attributeFilter: ['class', 'aria-current']")
	// MutationObserver + ResizeObserver.
	assert.Contains(t, js, "new MutationObserver(")
	assert.Contains(t, js, "new ResizeObserver(")

	// KRİTİK: ilk kurulum dosyanın EN SONUNDA olmalı. `tbAdminInit()` satır
	// ~952'de çağrılıyor; `tbSbHapKur` ise kayganHap IIFE'sinde (~1920)
	// tanımlanıyor. Erken çağrıda hap sayfa yüklenirken hiç kurulmuyordu
	// (ölçüldü: hap ancak bir gezinmeden sonra ortaya çıkıyordu).
	kaygan := indexOf(js, "window.tbSbHapKur = sbKur;")
	bootstrap := indexOf(js, "if (window.tbSbHapKur) window.tbSbHapKur();\n  if (window.tbSbGiris) window.tbSbGiris();")
	require.Greater(t, kaygan, 0)
	require.Greater(t, bootstrap, 0)
	assert.Greater(t, bootstrap, kaygan,
		"ilk kurulum sbKur tanımından SONRA çağrılmalı")
}

func TestYanMenuGirisEfekti(t *testing.T) {
	js := jsDosya(t, "admin-ui.js")

	// .side: x −28 → 0, 700 ms, expo.out
	assert.Contains(t, js, "transform: 'translateX(-28px)', opacity: 0")
	assert.Contains(t, js, "duration: 700, easing: EXPO")
	// Öğeler: x −12 → 0, 500 ms power3.out, kademe 25, gecikme 100.
	assert.Contains(t, js, "transform: 'translateX(-12px)', opacity: 0")
	assert.Contains(t, js, "duration: 500, easing: P3OUT, delay: 100 + i * 25")
	// Pill: scaleX .4 → 1, 600 ms back.out(2), origin 0% 50%, gecikme 450.
	assert.Contains(t, js, "'scaleX(.4)'")
	assert.Contains(t, js, "duration: 600, easing: BACK, delay: 450")
	assert.Contains(t, js, "transformOrigin = '0% 50%'")
	// Pill'de opacity satır içi 1 KALMALI (pacely clearProps yalnız transform).
	assert.Contains(t, js, "hap.style.opacity = '1';")
	// Dar ekranda atlanır (pacely: innerWidth <= 760).
	assert.Contains(t, js, "if (window.innerWidth <= 760) return;")
}

func TestYanMenuHapCss(t *testing.T) {
	css := jsDosya(t, "admin-pano.css")

	assert.Contains(t, css, ".tb-sb-hap {")
	assert.Contains(t, css, "position: absolute;")
	assert.Contains(t, css, "pointer-events: none;")
	assert.Contains(t, css, "border-radius: 12px;")
	// Aktif öğenin kendi zemini YOK (zemini hap veriyor).
	assert.Contains(t, css, ".admin-side .menu a.active,\n.admin-side .menu a[aria-current=\"page\"] {\n\tbackground: transparent;")
	// offsetParent kayması: `.menu` (ul) konumlanmamalı, yalnız <a>.
	assert.Contains(t, css, ".tb-sb-hap-var .menu a { position: relative; z-index: 1; }")
	assert.NotContains(t, css, ".tb-sb-hap-var .menu,\n.tb-sb-hap-var .menu a")
	// Ray modunda hap GÖRÜNÜR kalır.
	assert.NotContains(t, css, "html.admin-rail .tb-sb-hap { display: none; }")
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
