package templkit

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCdLogoAcikTemaDosyasiYoksaKoyuKullanilir(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "BitCoŞaR", LogoURL: "/media/koyu.svg",
		LogoGenislik: 240, LogoYukseklik: 60,
	})
	// İKİ görsel de aynı dosyayı göstermeli (koyu + acik sınıfları).
	assert.Equal(t, 2, strings.Count(html, "/media/koyu.svg"),
		"açık tema yuvası da koyu dosyayla doldurulmalı:\n%s", html)
	assert.Contains(t, html, "cd-logo__img--koyu")
	assert.Contains(t, html, "cd-logo__img--acik")
}

func TestCdLogoAcikTemaDosyasiVarsaAyriKullanilir(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "BitCoŞaR",
		LogoURL:  "/media/koyu.svg", LogoAcikURL: "/media/acik.svg",
	})
	assert.Contains(t, html, "/media/koyu.svg")
	assert.Contains(t, html, "/media/acik.svg")
	assert.NotEqual(t, 2, strings.Count(html, "/media/koyu.svg"),
		"iki dosya ayrıysa koyu dosya iki kez basılmamalı")
}

func TestCdLogoOlcuCSSDegiskenleri(t *testing.T) {
	// Yükseklik/genişlik panelden: inline CSS değişkeni olarak geçmeli.
	html := logoRender(t, CDShell{
		SiteName: "X", LogoURL: "/media/x.svg",
		LogoHedefYukseklik: 48, LogoAzamiGenislik: 300,
	})
	assert.Contains(t, html, "--cd-logo-h:48px")
	assert.Contains(t, html, "--cd-logo-max:300px")
}

func TestCdLogoOlcuVarsayilanlari(t *testing.T) {
	// Panelden değer girilmediyse varsayılan 32 / 260 yazılır.
	html := logoRender(t, CDShell{SiteName: "X", LogoURL: "/media/x.svg"})
	assert.Contains(t, html, "--cd-logo-h:32px")
	assert.Contains(t, html, "--cd-logo-max:260px")
}

func TestCdLogoStilSinirlama(t *testing.T) {
	// Aralık dışı değerler kırpılır (bozuk ayar üst çubuğu taşırmasın).
	assert.Contains(t, cdLogoStil(0, 0, 0), "--cd-logo-h:32px")
	assert.Contains(t, cdLogoStil(-10, 99999, -5), "--cd-logo-h:32px")
	assert.Contains(t, cdLogoStil(-10, 99999, -5), "--cd-logo-max:260px")
	assert.Contains(t, cdLogoStil(48, 300, 0), "--cd-logo-h:48px")
	assert.Contains(t, cdLogoStil(48, 300, 0), "--cd-logo-max:300px")
}

func TestCdLogoSloganLogonunAltinda(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "X", LogoURL: "/media/x.svg", Tagline: "paranın geleceği",
	})
	grup := strings.Index(html, `class="cd-logo__grup"`)
	slogan := strings.Index(html, `class="cd-logo__slogan"`)
	img := strings.Index(html, "cd-logo__img--yuklenen")
	require.GreaterOrEqual(t, grup, 0, "dikey grup olmalı")
	assert.Greater(t, slogan, grup, "slogan grubun İÇİNDE olmalı")
	assert.Greater(t, slogan, img, "slogan görselden SONRA (altında) olmalı")
}

func TestCdHeadFaviconYuklenen(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, CDHeadParca(CDHead{
		Title: "t", SiteName: "s", Locale: "tr",
		FaviconURL: "/media/202611/fav.svg",
	}).Render(context.Background(), &b))
	html := b.String()

	// Yüklenen favicon `sizes="any"` ile basılmalı (SVG'ye özgü).
	assert.Contains(t, html, `href="/media/202611/fav.svg"`)
	assert.Contains(t, html, `sizes="any"`)
	assert.Contains(t, html, `rel="apple-touch-icon" href="/media/202611/fav.svg"`)
	// SVG'yi yok sayan botlar için PNG yedeği KALMALI.
	assert.Contains(t, html, "/static/icon-192.png")
}

func TestCdHeadFaviconVarsayilan(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, CDHeadParca(CDHead{
		Title: "t", SiteName: "s", Locale: "tr",
	}).Render(context.Background(), &b))
	html := b.String()
	assert.Contains(t, html, "/static/favicon.svg")
	assert.Contains(t, html, "/static/icon-192.png")
	assert.NotContains(t, html, `sizes="any"`, "varsayılanda yüklenen favicon yok")
}

func TestCdLogosimgeKilidi(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "BitCoŞaR",
		LogoURL:  "/media/yazi.svg", LogoSimgeURL: "/media/simge.svg",
		Tagline: "paranın geleceği",
	})
	require.Contains(t, html, `class="cd-logo__kilit"`, "üçlü kilit olmalı")
	require.Contains(t, html, "/media/simge.svg", "simge basılmalı")
	require.Contains(t, html, "/media/yazi.svg", "yazı basılmalı")

	kilit := strings.Index(html, `class="cd-logo__kilit"`)
	simge := strings.Index(html, `class="cd-logo__simge"`)
	grup := strings.Index(html, `class="cd-logo__grup"`)
	yazi := strings.Index(html, "/media/yazi.svg")
	slogan := strings.Index(html, `class="cd-logo__slogan"`)
	assert.Greater(t, simge, kilit, "simge kilidin içinde")
	assert.Greater(t, grup, simge, "yazı sütunu simgeden SONRA")
	assert.Greater(t, yazi, grup, "yazı sütunun içinde")
	assert.Greater(t, slogan, yazi,
		"slogan yazıdan sonra → aynı sütunda ALTINDA (B hizası)")
}

func TestCdLogoYalnizSimgeKirikGorselYok(t *testing.T) {
	// Yalnız simge yüklendiyse yazı yuvası boş kalır. `src=""` basmak
	// tarayıcıda kırık görsel simgesi çizer — şablon bunu atlamalı.
	html := logoRender(t, CDShell{
		SiteName: "X", LogoSimgeURL: "/media/simge.svg",
		Tagline: "slogan",
	})
	assert.Contains(t, html, "/media/simge.svg")
	// Yalnız simge görseli olmalı; boş src'li ikinci <img> kırık simge çizer.
	assert.Equal(t, 1, strings.Count(html, "<img"), "tek görsel (simge) olmalı")
	assert.NotContains(t, html, "cd-logo__img--koyu cd-logo__img--yuklenen",
		"yazı yoksa yazı görseli basılmamalı")
	assert.Contains(t, html, "cd-logo__slogan", "slogan yine görünmeli")
}

func TestCdSimgeStilSinirlari(t *testing.T) {
	// Simge yüksekliği: ayrı değer yoksa logo yüksekliğine düşer.
	assert.Equal(t, "--cd-simge-h:32px", cdSimgeStil(0, 0))
	assert.Equal(t, "--cd-simge-h:48px", cdSimgeStil(0, 48))
	assert.Equal(t, "--cd-simge-h:40px", cdSimgeStil(40, 48))
	assert.Equal(t, "--cd-simge-h:32px", cdSimgeStil(9999, 48), "üst sınır kırpılır")
	assert.Equal(t, "--cd-simge-h:32px", cdSimgeStil(-5, 0), "negatif varsayılana döner")
}

func TestCdLogosimgeYuksekligiStileYazilir(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "X", LogoURL: "/m.svg", LogoSimgeURL: "/s.svg",
		LogoHedefYukseklik: 40, LogoSimgeYukseklik: 56,
	})
	assert.Contains(t, html, "--cd-simge-h:56px", "simge kendi yüksekliğini kullanmalı")
	assert.Contains(t, html, "--cd-logo-h:40px", "yazı kendi yüksekliğini korumalı")
}
