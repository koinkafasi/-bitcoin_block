package templkit

// Marka / logo testleri (11 Kasım 2026).
//
// NEDEN: kullanıcı "admin panelden logo değiştiremiyorum" dedi. Kök neden
// şablonda logonun SABİT olmasıydı. Bu testler üç şeyi sabitliyor:
//   1) LogoURL boşsa gömülü marka dosyaları basılır (geri uyumluluk)
//   2) LogoURL doluysa O basılır, statik PNG basılmaz
//   3) width/height GERÇEK ölçüden gelir, bilinmiyorsa 146×32'ye düşer (CLS)
//
// Ayrıca CSS'te bir özgüllük tuzağı vardı: `.cd img { height:auto }`
// kuralı yüklenen logonun 32 px yüksekliğini eziyordu (ölçüldü). Onun
// regresyon testi de burada.

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func logoRender(t *testing.T, sh CDShell) string {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, cdLogoMark(sh).Render(context.Background(), &b))
	return b.String()
}

func TestCdLogoMarkYuklenenLogo(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName:      "BitCoŞaR",
		LogoURL:       "/media/202611/abc.svg",
		LogoGenislik:  240,
		LogoYukseklik: 60,
		Tagline:       "paranın geleceği",
	})
	assert.Contains(t, html, "/media/202611/abc.svg", "yüklenen logo basılmalı")
	assert.Contains(t, html, "cd-logo__img--yuklenen")
	assert.NotContains(t, html, "logo-bitcosar-2x.png", "statik logo BASILMAMALI")
	// Gerçek ölçü (CLS).
	assert.Contains(t, html, `width="240"`)
	assert.Contains(t, html, `height="60"`)
	// Slogan logonun altında.
	assert.Contains(t, html, "paranın geleceği")
	assert.Contains(t, html, "cd-logo__slogan")
}

func TestCdLogoMarkYuklenenYoksaStatik(t *testing.T) {
	// Geri uyumluluk: media silinmiş/boşsa gömülü marka dosyaları.
	html := logoRender(t, CDShell{SiteName: "BitCoŞaR"})
	assert.Contains(t, html, "logo-bitcosar-2x.png")
	assert.Contains(t, html, "logo-bitcosar-acik-2x.png")
	assert.NotContains(t, html, "cd-logo__img--yuklenen")
	assert.NotContains(t, html, "cd-logo__slogan")
}

func TestCdLogoMarkOlcuYoksaVarsayilan(t *testing.T) {
	// SVG'de viewBox da yoksa ölçü 0 gelir; width="0" yazmak düzeni bozar.
	html := logoRender(t, CDShell{SiteName: "X", LogoURL: "/media/x.svg"})
	assert.Contains(t, html, `width="146"`)
	assert.Contains(t, html, `height="32"`)
	assert.NotContains(t, html, `width="0"`)
}

func TestCdLogoOlcuSinirlari(t *testing.T) {
	assert.Equal(t, "146", cdLogoOlcu(0, 146))
	assert.Equal(t, "32", cdLogoOlcu(-5, 32))
	assert.Equal(t, "146", cdLogoOlcu(99999, 146), "absürt değer varsayılana düşmeli")
	assert.Equal(t, "240", cdLogoOlcu(240, 146))
}

func TestCdLogoSloganAnahtarSozcukYoksaBasilmaz(t *testing.T) {
	html := logoRender(t, CDShell{SiteName: "X", LogoURL: "/media/x.svg", Tagline: "  "})
	assert.NotContains(t, html, "cd-logo__slogan")
}

func TestCdLogoCSSOzgulluk(t *testing.T) {
	// `.cd img { max-width:100%; height:auto }` (0,1,1) kuralı, yüklenen
	// logonun ölçüsünü veren kuralı (0,1,0) EZİYORDU → logo 32 px'e
	// inmiyor, üst çubuktan taşıyordu (tarayıcıda ölçüldü: 251×66).
	// Kural ata sınıfıyla yazılmalı (0,2,0).
	b, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err)
	css := string(b)
	assert.Contains(t, css, ".cd-logo .cd-logo__img--yuklenen {",
		"özgüllük 0,2,0 olmalı; yoksa `.cd img` kuralı ezer")
	assert.NotContains(t, css, "\n.cd-logo__img--yuklenen {",
		"çıplak sınıf seçicisi (0,1,0) ezilir")
}
