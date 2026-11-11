package templkit

// Slogan girintisi testleri (11 Kasım 2026).
//
// Kullanıcı isteği: "yuvarlak simgenin altına ekledin, o simgenin kendi
// BİTCOŞAR yazısındaki B harfinin altına getir".
//
// Logo dosyasının içinde solda simge, sağda yazı var; sunucu SVG'nin
// içeriğini yorumlayamaz. Bu yüzden girinti PANELDEN ayarlanıyor ve
// CSS değişkeniyle uygulanıyor. Bu testler değişkenin uçtan uca
// taşındığını doğrular: ayar → cdLogoStil → inline style → CSS.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func TestSloganGirintisiStileYazilir(t *testing.T) {
	html := logoRender(t, CDShell{
		SiteName: "X", LogoURL: "/media/x.svg",
		Tagline: "paranın geleceği", SloganGirinti: 64,
	})
	assert.Contains(t, html, "--cd-logo-indent:64px",
		"girinti inline stile yazılmalı")
}

func TestSloganGirintisiVarsayilanSifir(t *testing.T) {
	html := logoRender(t, CDShell{SiteName: "X", LogoURL: "/media/x.svg"})
	assert.Contains(t, html, "--cd-logo-indent:0px")
}

func TestSloganGirintisiSinirlama(t *testing.T) {
	// Negatif girinti sloganı logonun dışına taşırır; üst sınır absurd
	// değerleri engeller.
	assert.Equal(t, "--cd-logo-h:32px;--cd-logo-max:260px;--cd-logo-indent:0px",
		cdLogoStil(0, 0, -50))
	assert.Contains(t, cdLogoStil(32, 260, 9999), "--cd-logo-indent:0px")
	assert.Contains(t, cdLogoStil(32, 260, 64), "--cd-logo-indent:64px")
}

func TestSloganGirintisiCSSUygulanir(t *testing.T) {
	b, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err)
	css := string(b)

	i := strings.Index(css, ".cd-logo__slogan {")
	require.Greater(t, i, 0, ".cd-logo__slogan kuralı olmalı")
	son := i + 600
	if son > len(css) {
		son = len(css)
	}
	blok := css[i:son]
	assert.Contains(t, blok, "margin-left: var(--cd-logo-indent, 0px)",
		"girinti CSS'te uygulanmalı")
}

func TestSloganGirintisiTekBasinaSloganYoksaEtkisiz(t *testing.T) {
	// Slogan boşsa girinti de basılmamalı (boş span olmasın).
	html := logoRender(t, CDShell{
		SiteName: "X", LogoURL: "/media/x.svg",
		Tagline: "", SloganGirinti: 64,
	})
	assert.NotContains(t, html, "cd-logo__slogan")
}
