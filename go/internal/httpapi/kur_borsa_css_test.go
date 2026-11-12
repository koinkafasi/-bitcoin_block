package httpapi

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func kbCSS(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	s := string(b)
	// Yorumlar temizlenir: modülün kendi açıklamasında ".pys-item" gibi
	// metinler geçiyor, bunlar kural değildir.
	s = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(s, "")
	i := strings.Index(s, ".pys-strip {")
	require.Greater(t, i, 0, "pys-strip kuralı bulunmalı")
	// Modül bölgesi: pys-strip'ten hava/servis bandına kadar.
	end := strings.Index(s[i:], ".hp-ticker-bar")
	if end <= 0 {
		end = 9000
		if end > len(s)-i {
			end = len(s) - i
		}
	}
	return s[i : i+end]
}

// kural bloğunu süslü parantez dengesiyle çıkarır (iç içe @media için).
func kural(css, selector string) string {
	i := strings.Index(css, selector)
	if i < 0 {
		return ""
	}
	open := strings.Index(css[i:], "{")
	if open < 0 {
		return ""
	}
	start := i + open
	depth := 0
	for j := start; j < len(css); j++ {
		switch css[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[start : j+1]
			}
		}
	}
	return css[start:]
}

func TestKbCSSKaydirilabilirCubukGizli(t *testing.T) {
	css := kbCSS(t)
	scroll := kural(css, ".pys-scroll {")
	require.NotEmpty(t, scroll)

	assert.Contains(t, scroll, "overflow-x: auto", "şerit yatay kaydırılabilir olmalı")
	assert.Contains(t, scroll, "scrollbar-width: none", "kaydırma çubuğu gizli")
	assert.Contains(t, scroll, "-ms-overflow-style: none")
	assert.Contains(t, scroll, "scroll-snap-type: x proximity", "üst manşetle aynı snap")
	assert.Contains(t, scroll, "overscroll-behavior-x: contain",
		"şerit sonuna gelince sayfa yatay kaymasın")

	// WebKit çubuğu da gizli
	assert.Contains(t, css, ".pys-scroll::-webkit-scrollbar { display: none")
}

func TestKbCSSKalemlerEsitPaylasir(t *testing.T) {
	css := kbCSS(t)
	item := kural(css, ".pys-item {")
	scroll := kural(css, ".pys-scroll {")
	require.NotEmpty(t, item)
	require.NotEmpty(t, scroll)

	// Üst manşet deseni: kalem sabit genişlikte (flex: 0 0 auto), sığmayan
	// kısım KAYDIRILIR — küçülmez. Böylece fiyat 20px'te kalır, kırpılmaz.
	assert.Contains(t, item, "flex: 0 0 auto", "kalem küçülmemeli (üst manşet deseni)")
	assert.Regexp(t, regexp.MustCompile(`min-width:\s*132px`), item)
	assert.Contains(t, item, "scroll-snap-align: start")
	// Sürükleme imleci (piyasa.js ile fareyle çekilebilir)
	assert.Contains(t, item, "cursor: grab")
	// Sabit genişlik/esnek olmayan kalem olmamalı
	assert.NotContains(t, scroll, "flex-wrap: wrap",
		"satır SARMAZ; sığmayan kısım kaydırılır (üst manşet deseni)")
}

func TestKbCSSCerceveYok(t *testing.T) {
	css := kbCSS(t)
	// Referans şablonunda ne kenarlık ne alt çizgi.
	scroll := kural(css, ".pys-scroll {")
	item := kural(css, ".pys-item {")
	for ad, blok := range map[string]string{"pys-scroll": scroll, "pys-item": item} {
		assert.NotContains(t, blok, "border-bottom: 1px", "%s: alt çizgi olmamalı", ad)
		assert.NotRegexp(t, regexp.MustCompile(`border:\s*1px`), blok, "%s: çerçeve olmamalı", ad)
		assert.Contains(t, blok, "border: 0", "%s: kenarlık sıfırlanmalı", ad)
	}
	// Metin tek satırda kalmalı (kalem genişliği sabit, kaydırma var)
	assert.Contains(t, css, "white-space: nowrap")
}

func TestKbCSSDuyarliKirilimlar(t *testing.T) {
	css := kbCSS(t)
	// Ekran daraldıkça kalem daralır (aynı anda 2-3 kalem görünür, gerisi
	// kaydırılır). Üst manşetin flex-basis kademeleriyle aynı yaklaşım.
	for _, bp := range []string{"max-width: 1200px", "max-width: 900px", "max-width: 600px", "max-width: 380px"} {
		assert.Contains(t, css, bp, "kırılım noktası olmalı: %s", bp)
	}
	// Kaydırma olduğu için satır sarması ve sütun hesabı YOK.
	assert.NotContains(t, css, "calc(50% - 6px)", "sütun bölme yerine kaydırma")
	assert.NotContains(t, css, "flex-basis: 100%")
}

func TestKbCSSTipografiReferanslaAyni(t *testing.T) {
	css := kbCSS(t)

	name := kural(css, ".pys-name {")
	price := kural(css, ".pys-price {")
	chg := kural(css, ".pys-chg {")

	assert.Contains(t, name, "font-size: .75rem", "etiket 12px (text-xs)")
	// Kaydırma olduğu için fiyat TAM 20px — referansla birebir (text-xl).
	assert.Contains(t, price, "font-size: 1.25rem", "fiyat 20px (text-xl)")
	assert.Contains(t, price, "font-weight: 700", "fiyat kalın (font-bold)")
	assert.Contains(t, chg, "font-size: .625rem", "yüzde ~10px (text-xxs)")
	// Etiket ve yüzde normal ağırlık (referansta 400)
	assert.Contains(t, name, "font-weight: 400")
	assert.Contains(t, chg, "font-weight: 400")
	// Sayısal hizalama: fiyatlar sütun halinde kaymasın
	assert.Contains(t, price, "font-variant-numeric: tabular-nums")
}

// Mobilde yatay kayma olmamalı: taşma bu üç kuralla engellenir.
func TestKbCSSMobilTasmaEngeli(t *testing.T) {
	css := kbCSS(t)

	strip := kural(css, ".pys-strip {")
	assert.Contains(t, strip, "max-width: 100%", "şerit kapsayıcıdan taşmaz")

	scroll := kural(css, ".pys-scroll {")
	assert.Contains(t, scroll, "max-width: 100%")
	assert.Contains(t, scroll, "overscroll-behavior-x: contain",
		"şerit sonunda sayfa yatay kaymamalı")

	// Kalemler küçülmez (flex: 0 0 auto) → sığmayan kısım kaydırılır,
	// içerik kırpılmaz. Böylece mobilde metin kesilmiyor.
	item := kural(css, ".pys-item {")
	assert.Contains(t, item, "flex: 0 0 auto")
	assert.NotContains(t, item, "flex-wrap")
}

func TestKbCSSSinifCakismasiYok(t *testing.T) {
	b, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	tum := string(b)

	// Yorumlar temizlenir: modülün kendi açıklamasında ".pys-item" gibi
	// metinler geçiyor, bunlar kural değildir.
	yorumsuz := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(tum, "")

	start := strings.Index(yorumsuz, ".pys-strip {")
	require.Greater(t, start, 0, "pys-strip kuralı bulunmalı")
	end := strings.Index(yorumsuz[start:], ".hp-ticker-bar")
	require.Greater(t, end, 0, "modül bölgesinin sonu bulunmalı")
	end += start

	benim := yorumsuz[start:end]
	disarisi := yorumsuz[:start] + yorumsuz[end:]

	// Benim bölgemde pys- sınıfları var
	assert.Contains(t, benim, ".pys-item")
	assert.Contains(t, benim, ".pys-ico")

	// Dışarıda pys- HİÇ geçmemeli: başka modül aynı sınıfı ezmemeli.
	assert.NotContains(t, disarisi, "pys-",
		"pys- sınıfları yalnız Kur & Borsa modülünde tanımlı olmalı "+
			"(başka modül aynı sınıfa kural yazarsa görünüm bozulur)")

	// Referans şablonunda çerçeve yok: temel .pys-item kuralında border:0.
	base := kural(benim, ".pys-item {")
	assert.Contains(t, base, "border: 0")
	assert.NotContains(t, base, "border: 1px")
}
