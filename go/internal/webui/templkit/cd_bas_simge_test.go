package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCDBasSimgeStiliVar(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, ".cd-bas-simge", "başlık işaretinin stil kuralı silinmiş")
	// Ölçü kuralı şart: bu olmadan SVG kapsayıcı genişliğine şişer.
	assert.Regexp(t, `\.cd-bas-simge\s*\{[^}]*width:\s*1`, css,
		"işaretin genişlik kuralı yok — SVG kapsayıcıya şişer")
	// Tema kuralı: koyu temada marka sarısı.
	assert.Contains(t, css, "html.dark .cd-bas-simge",
		"koyu tema rengi tanımlı değil")
}
