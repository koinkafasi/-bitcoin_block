package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func statikDosya(t *testing.T, ad string) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/"+ad)
	require.NoError(t, err, "statik dosya okunamadı: %s", ad)
	return string(b)
}

func TestTemaGecisDaireEfekti(t *testing.T) {
	js := statikDosya(t, "app.js")

	// View Transitions API kullanılmalı — düz sınıf değişimi değil.
	assert.Contains(t, js, "document.startViewTransition")
	assert.Contains(t, js, `pseudoElement: "::view-transition-new(root)"`)

	// Daire, tıklanan düğmenin merkezinden açılır ve en uzak köşeyi kaplar.
	assert.Contains(t, js, "circle(0px at ")
	assert.Contains(t, js, "Math.hypot(")

	// Süre ve eğri kaynakla birebir.
	assert.Contains(t, js, "TEMA_GECIS_MS = 450")
	assert.Contains(t, js, `TEMA_GECIS_EGRI = "cubic-bezier(.65,0,.35,1)"`)

	// Hareketi azalt tercihine saygı: animasyonsuz, tema yine değişir.
	assert.Contains(t, js, "prefers-reduced-motion: reduce")
}

func TestTemaGecisVarsayilanSonsumeyiKapatir(t *testing.T) {
	css := statikDosya(t, "app.css")

	// Bu iki kural olmadan tarayıcı eski/yeni temayı çapraz söndürür ve
	// daire açılımı bulanık görünür.
	assert.Contains(t, css, "::view-transition-old(root),")
	assert.Contains(t, css, "::view-transition-new(root)")
	assert.Contains(t, css, "animation: none;")
	assert.Contains(t, css, "mix-blend-mode: normal;")
}

// Tema geçişi düğme tıklamasından geçmeli ki daire merkezi doğru olsun.
func TestTemaGecisDugmedenGecer(t *testing.T) {
	js := statikDosya(t, "app.js")
	assert.Contains(t, js, `const btn = e.target.closest("[data-theme-btn]")`)
	assert.Contains(t, js, "temaGecis(next, btn)")
	// Merkezsiz (btn'siz) çağrı kalmamalı.
	assert.NotContains(t, strings.ReplaceAll(js, "temaGecis(next, btn)", ""), "temaGecis(next)")
}
