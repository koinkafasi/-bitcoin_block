package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Koyu tema TEK sözleşmeyle yazılır: `html.dark`.
//
// `data-tema` bir DÜĞME seçicisidir (csp-delegated.js onu dinler), tema
// durumu değil. Oraya yazılan kural sessizce hiç çalışmaz — 12 Kasım
// 2026'da görsel ton kuralı tam bu yüzden etkisiz kaldı. Yanlış seçici
// derlemeyi kırmaz, test kırsın.
func TestCDKoyuTemaTekSozlesme(t *testing.T) {
	css := cdCSS(t)

	assert.NotContains(t, css, `[data-tema="koyu"]`,
		`koyu tema seçicisi html.dark olmalı; [data-tema] düğme seçicisidir`)
	assert.NotContains(t, css, `[data-theme="dark"]`,
		`cd.css data-theme sözleşmesini kullanmıyor (üyelik sayfası kullanıyordu)`)

	assert.True(t, strings.Contains(css, "html.dark"),
		"koyu tema kuralları html.dark ile yazılmalı")
}
