package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kullanıcı arayüz tercihleri (Faz 173, A-3).

// Anahtarlar istemciden geliyor; sınırlar sunucuda.
func TestTercihAnahtarGecerli(t *testing.T) {
	gecerli := []string{
		"pn:rail",
		"tb_form_grup:Künye",
		"tb_form_grup:Yayın ve görünürlük",
		"a.b-c_d 1",
	}
	for _, k := range gecerli {
		assert.Truef(t, tercihAnahtarGecerli(k), "%q geçerli olmalı", k)
	}
	gecersiz := []string{
		"",                        // boş
		"x/y",                     // yol benzeri
		"<script>",                // etiket
		"a\"b",                    // tırnak
		"k;DROP TABLE user_prefs", // noktalı virgül
		string(make([]byte, 90)),  // çok uzun
	}
	for _, k := range gecersiz {
		assert.Falsef(t, tercihAnahtarGecerli(k), "%q geçersiz olmalı", k)
	}
}

// Sayfaya gömülen meta etiketi istemcide bulunmalı: tohumlama buna dayanıyor.
func TestTercihMetaEtiketi(t *testing.T) {
	s := okuDosya(t, "..", "webui", "templates", "partials.html")
	assert.Contains(t, s, `name="tb-prefs"`)
	assert.Contains(t, s, `.Tercihler`)

	// Menü daralma betiği meta'yı okumalı: yoksa yeni cihazda ilk boyama
	// geniş menüyle yapılır ve her sayfa geçişinde zıplama görülür.
	i := indexOfStr(s, `classList.add('admin-rail')`)
	if assert.Greater(t, i, 0) {
		bas := i - 1200
		if bas < 0 {
			bas = 0
		}
		assert.Contains(t, s[bas:i], "tb-prefs",
			"menü daralma betiği sunucudaki tercihi okumuyor (FOUC)")
	}
}

// İstemci: yalnız iş akışı tercihleri sunucuya yazılmalı, görüntü
// tercihleri (tema, yazı boyu) yerel kalmalı.
func TestTercihSenkronKapsami(t *testing.T) {
	js := okuDosya(t, "..", "webui", "static", "admin-ui.js")

	assert.Contains(t, js, "SENKRON_ONEK", "senkron kapsamı tanımlı olmalı")
	assert.Contains(t, js, "'tb_form_grup:'")
	assert.Contains(t, js, "'pn:'")
	assert.Contains(t, js, "/admin/tercih", "sunucuya yazma ucu çağrılmalı")

	// Tema/boyut yerel kalmalı: bu anahtarlar senkron listesinde olmamalı.
	i := indexOfStr(js, "var SENKRON_ONEK")
	if assert.Greater(t, i, 0) {
		son := i + 120
		if son > len(js) {
			son = len(js)
		}
		blok := js[i:son]
		assert.NotContains(t, blok, "tb_theme")
		assert.NotContains(t, blok, "tb_font_scale")
	}
}

func indexOfStr(s, alt string) int { return strings.Index(s, alt) }

// okuDosya test için dosya okur (paket kökü: go/internal/httpapi).
func okuDosya(t *testing.T, parcalar ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parcalar...))
	require.NoError(t, err)
	return string(b)
}
