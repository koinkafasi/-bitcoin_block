package platform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeSVGSaglamLogoyuKorur(t *testing.T) {
	girdi := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="240" height="60" viewBox="0 0 240 60">
  <title>Bitcosar</title>
  <defs>
    <linearGradient id="g"><stop offset="0%" stop-color="#f5d06a"/><stop offset="100%" stop-color="#c9a227"/></linearGradient>
  </defs>
  <rect width="240" height="60" fill="url(#g)" rx="8"/>
  <path d="M10 10 L50 50" stroke="#000" fill="none"/>
  <text x="12" y="30" font-family="Inter">paranın geleceği</text>
</svg>`)
	temiz, w, h, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)

	// Ölçü okunur (admin panelinde önizleme boyutu için gerekli).
	assert.Equal(t, 240, w)
	assert.Equal(t, 60, h)

	s := string(temiz)
	// Çizim öğeleri ve belge içi referans KALIR.
	assert.Contains(t, s, "<svg")
	assert.Contains(t, s, "<path")
	assert.Contains(t, s, "<rect")
	assert.Contains(t, s, "linearGradient")
	assert.Contains(t, s, "url(#g)", "belge içi (#id) referans korunmalı")
	assert.Contains(t, s, "paranın geleceği")
}

func TestSanitizeSVGScriptVeOlaySilinir(t *testing.T) {
	girdi := []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)">
<script>alert(document.cookie)</script>
<rect width="10" height="10" onclick="steal()" onmouseover="x()"/>
</svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := strings.ToLower(string(temiz))
	assert.NotContains(t, s, "<script")
	assert.NotContains(t, s, "alert")
	assert.NotContains(t, s, "onload")
	assert.NotContains(t, s, "onclick")
	assert.NotContains(t, s, "onmouseover")
	assert.Contains(t, s, "<rect", "çizim öğesi kalmalı")
}

func TestSanitizeSVGForeignObjectAtilir(t *testing.T) {
	// foreignObject HTML kaçırma kapısıdır: içine <iframe>/<script> konur.
	girdi := []byte(`<svg xmlns="http://www.w3.org/2000/svg">
<foreignObject width="100" height="100"><iframe src="https://evil"></iframe></foreignObject>
<circle cx="5" cy="5" r="4"/>
</svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := strings.ToLower(string(temiz))
	assert.NotContains(t, s, "foreignobject")
	assert.NotContains(t, s, "iframe")
	assert.Contains(t, s, "<circle", "kardeş öğe korunmalı")
}

func TestSanitizeSVGDisaBaglantiReddedilir(t *testing.T) {
	// ESKİ SÜRÜMÜN AÇIĞI: dış image/href yakalanmıyordu (IP sızıntısı).
	girdi := []byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
<image href="https://evil.example/track.png" width="10" height="10"/>
<use xlink:href="https://evil.example/sprite.svg#x"/>
<rect width="5" height="5" href="https://evil.example/x"/>
</svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := string(temiz)
	assert.NotContains(t, s, "evil.example", "dış adres hiçbir biçimde kalmamalı")
	// image/use öğeleri beyaz listede yok → içerikleriyle atılır.
	assert.NotContains(t, strings.ToLower(s), "<image")
	assert.NotContains(t, strings.ToLower(s), "<use")
}

func TestSanitizeSVGStyleUrlReddedilir(t *testing.T) {
	// ESKİ SÜRÜMÜN AÇIĞI: style içindeki url() geçiyordu.
	girdi := []byte(`<svg xmlns="http://www.w3.org/2000/svg">
<style>@import url(https://evil.example/x.css); .a{fill:url(https://evil.example/y)}</style>
<rect width="5" height="5" style="fill:url(https://evil.example/z)"/>
</svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := strings.ToLower(string(temiz))
	assert.NotContains(t, s, "evil.example")
	assert.NotContains(t, s, "url(http")
	assert.NotContains(t, s, "@import")
}

func TestSanitizeSVGJavascriptHrefReddedilir(t *testing.T) {
	girdi := []byte(`<svg xmlns="http://www.w3.org/2000/svg">
<a href="javascript:alert(1)"><rect width="5" height="5"/></a>
<rect width="5" height="5" fill="javascript:alert(2)"/>
</svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := strings.ToLower(string(temiz))
	assert.NotContains(t, s, "javascript:")
}

func TestSanitizeSVGDoctypeReddedilir(t *testing.T) {
	// XML varlık şişmesi (billion laughs) / XXE yolu.
	girdi := []byte(`<?xml version="1.0"?>
<!DOCTYPE svg [<!ENTITY a "aaaa"><!ENTITY b "&a;&a;&a;&a;">]>
<svg xmlns="http://www.w3.org/2000/svg"><text>&b;</text></svg>`)
	temiz, _, _, err := SanitizeSVGBoyut(girdi)
	require.NoError(t, err)
	s := strings.ToLower(string(temiz))
	assert.NotContains(t, s, "<!doctype")
	assert.NotContains(t, s, "<!entity")
}

func TestSanitizeSVGHTMLKamuflajiReddedilir(t *testing.T) {
	// Kök öğe svg değilse dosya kabul edilmez.
	_, _, _, err := SanitizeSVGBoyut([]byte(`<html><body><script>alert(1)</script></body></html>`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kök öğe")
}

func TestSanitizeSVGBosVeCokBuyuk(t *testing.T) {
	_, _, _, err := SanitizeSVGBoyut(nil)
	assert.Error(t, err)

	// 2 MB üstü reddedilir.
	buyuk := append([]byte(`<svg xmlns="http://www.w3.org/2000/svg">`),
		[]byte(strings.Repeat(" ", maxSVGBytes))...)
	_, _, _, err = SanitizeSVGBoyut(buyuk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "çok büyük")
}

func TestSanitizeSVGViewBoxOlcusuOkunur(t *testing.T) {
	// width/height yoksa viewBox'tan ölçü çıkarılır (yaygın logo biçimi).
	_, w, h, err := SanitizeSVGBoyut([]byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 320 80"><rect width="1" height="1"/></svg>`))
	require.NoError(t, err)
	assert.Equal(t, 320, w)
	assert.Equal(t, 80, h)

	// Ölçü hiç yoksa 0 döner — çağıran uydurma sayı yazmaz.
	_, w, h, err = SanitizeSVGBoyut([]byte(
		`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`))
	require.NoError(t, err)
	assert.Zero(t, w)
	assert.Zero(t, h)
}

func TestSanitizeSVGEskiImzaKorunur(t *testing.T) {
	// background.go bu imzayı kullanıyor; bozulmamalı.
	temiz, err := SanitizeSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`))
	require.NoError(t, err)
	assert.Contains(t, string(temiz), "<svg")
}

func TestSanitizeSVGXmlnsTekrarlanmaz(t *testing.T) {
	// xml.Encoder ad alanını kendisi yazar; girdideki `xmlns` de korunursa
	// `<svg xmlns="…" xmlns="…">` gibi tekrarlı çıktı oluşuyordu.
	temiz, _, _, err := SanitizeSVGBoyut([]byte(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="10" height="10">` +
			`<rect width="5" height="5"/></svg>`))
	require.NoError(t, err)
	s := string(temiz)
	kok := s[:strings.Index(s, ">")]
	assert.Equal(t, 1, strings.Count(kok, "xmlns="),
		"kök öğede xmlns BİR KEZ olmalı: %s", kok)
}
