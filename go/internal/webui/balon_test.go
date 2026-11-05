package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Seçim balonu (.be-bubble) — Faz 185.
//
// NEDEN: JS balonu <body> sonuna ekliyor. CSS'te `position` yoktu ve öğe
// varsayılan gizli değildi; akışta kaldığı için içerik formunun EN ALTINDA
// tam genişlikte duruyordu. Ölçüm:
//
//	konum static · display flex · inline stil yok · w 1440 · y 2249 (sayfa sonu)
//
// Kullanıcı bunu "sayfanın en altında boydan boya harf seçme çubuğu" olarak
// gördü (B / I / U / H2 / bağlantı düğmeleri).
//
// DOĞRUSU: konumlandırılmış + varsayılan `visibility:hidden`. `display:none`
// kullanılamaz: TipTap balonu gösterirken satır içi `visibility: visible`
// yazıyor, `display:none` olsa hiç görünmezdi.
func TestSecimBalonuKonumlanmis(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "icerikform.templ"))
	require.NoError(t, err)
	s := string(b)

	m := regexp.MustCompile(`\.be-bubble\{[^}]*\}`).FindString(s)
	require.NotEmpty(t, m, ".be-bubble kuralı bulunamadı")

	assert.Contains(t, m, "position:absolute",
		"balon konumlandırılmalı; yoksa sayfa akışında en alta düşer")
	assert.Contains(t, m, "visibility:hidden",
		"balon varsayılan gizli olmalı (TipTap satır içi stil ile gösterir)")
	assert.NotContains(t, m, "display:none",
		"display:none TipTap'in göstermesini engeller")
	assert.Contains(t, m, "width:max-content",
		"balon içeriği kadar geniş olmalı; tam genişlik hatası buydu")
	assert.Contains(t, m, "max-width:",
		"dar ekranda taşmamalı")
}
