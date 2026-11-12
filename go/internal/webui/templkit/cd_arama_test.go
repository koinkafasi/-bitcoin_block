package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCDAramaTitremesiz: arama paneli açılınca sayfa TİTREMEMELİ.
//
// Kullanıcı bildirimi (12 Kasım 2026): "coindesk ... arka plan blur ve
// transparan şekilde arama kutusu yumuşak bir efekt ile açılıyor ve sitenin
// sağındaki scroll bir anda kaybolurken ... sayfa sabit kalıyor bizde scroll
// kaybolması sebebi ile sağda genişleme yaparak titretiyor sayfayı".
//
// KÖK NEDEN (ölçüldü): panel açılırken `html.cd-arama-acik { overflow:hidden }`
// kaydırma çubuğunu kaldırıyor, görünüm 15 px genişliyordu
// (önce 1425 px, sonra 1440 px). Çözüm: `scrollbar-gutter: stable`.
func TestCDAramaTitremesiz(t *testing.T) {
	css := cdCSS(t)

	assert.Contains(t, css, "html { scrollbar-gutter: stable; }",
		"kaydırma çubuğu yeri her zaman ayrılmalı; yoksa panel açılınca sayfa titrer")

	// Kilidin kendisi kalmalı (arka plan kaymasın) ama gutter sayesinde
	// genişlik değişmemeli.
	assert.Contains(t, css, "html.cd-arama-acik, html.cd-arama-acik body { overflow: hidden; }")
}

// TestCDAramaBlurEfekti: arka plan bulanık ve panel yarı saydam olmalı.
//
// Kullanıcı isteği: "arka plan blur ve transparan şekilde arama kutusu
// yumuşak bir efekt ile açılıyor". Önce perde düz siyahtı ve panel opaktı.
func TestCDAramaBlurEfekti(t *testing.T) {
	css := cdCSS(t)

	i := strings.Index(css, ".cd-arama__perde {")
	require.GreaterOrEqual(t, i, 0)
	perde := css[i:]
	perde = perde[:strings.Index(perde, "}")]
	assert.Contains(t, perde, "backdrop-filter: blur(", "perde arka planı bulanıklaştırmalı")

	i = strings.Index(css, ".cd-arama__panel {")
	require.GreaterOrEqual(t, i, 0)
	panel := css[i:]
	panel = panel[:strings.Index(panel, "}")]
	assert.Contains(t, panel, "backdrop-filter: blur(", "panel cam efektli olmalı")
	assert.Contains(t, panel, "color-mix(in srgb, var(--cd-surface-panel)",
		"panel yarı saydam olmalı (transparan)")

	// Yumuşak açılış animasyonu.
	assert.Contains(t, css, "@keyframes cd-arama-panel-gir")
	assert.Contains(t, css, "@keyframes cd-arama-perde-gir")
	// Hareket azaltma tercihi saygı görmeli.
	assert.Contains(t, css, ".cd-arama__perde, .cd-arama__panel { animation: none; }")
}

// TestCDAramaMobilSheet: mobilde panel alttan sheet olarak açılmalı.
func TestCDAramaMobilSheet(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, "@keyframes cd-arama-sheet-gir")
	assert.Contains(t, css, "from { transform: translateY(100%); }")
	// Panel içi kaydırma çubuğu görünmemeli (tasarım kuralı).
	assert.Contains(t, css, ".cd-arama__govde::-webkit-scrollbar { display: none; }")
}
