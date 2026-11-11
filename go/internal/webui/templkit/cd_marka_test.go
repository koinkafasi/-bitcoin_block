package templkit

// Marka: açık/koyu tema logosu + favicon testleri (11 Kasım 2026).
//
// Kullanıcı bildirimleri:
//   "Dark moda görünen logo light moda göstermiyor"
//   "Favicon svg yükledim ama browserda göstermedi"
//
// İkisi de sessizce bozulur: hata vermez, sadece görünüm yanlış olur.

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCdLogoAcikTemaDosyasiYoksaKoyuKullanilir(t *testing.T) {
	// KRİTİK REGRESYON: açık tema dosyası yüklenmediyse şablon eskiden
	// yalnız `--koyu` basıyordu; CSS açık temada onu gizleyince HİÇBİR
	// logo kalmıyordu ("light moda göstermiyor").
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
	// Slogan `.cd-logo__grup` (dikey sütun) İÇİNDE olmalı; doğrudan
	// `.cd-logo` çocuğu olursa flex satırında logonun YANINA düşer
	// (ölçüldü: görsel 12–60, slogan 32–43 = aynı satır).
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
