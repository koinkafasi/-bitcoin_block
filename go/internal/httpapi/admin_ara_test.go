package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kısa sorgu 200 + boş grup döner (hata değil): palet açılır açılmaz
// boş sonuç göstermemesi için istemci de bunu bekler.
func TestAdminAraKisaSorguBosDoner(t *testing.T) {
	s := &Server{}
	for _, q := range []string{"", " ", "a", "  a  "} {
		r := httptest.NewRequest(http.MethodGet, "/admin/ara?q="+url.QueryEscape(q), nil)
		w := httptest.NewRecorder()
		s.handleAdminAra(w, r)

		require.Equal(t, http.StatusOK, w.Code, "sorgu: %q", q)
		var d struct {
			OK      bool  `json:"ok"`
			Gruplar []any `json:"gruplar"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d))
		assert.True(t, d.OK)
		assert.Empty(t, d.Gruplar, "kısa sorguda grup olmamalı: %q", q)
	}
}

// Uç nokta JSON döner ve HTML üretmez: arama terimi istemcide textContent
// ile basılıyor, burada kaçış gerekmiyor.
func TestAdminAraJSONDoner(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest(http.MethodGet, "/admin/ara?q=a", nil)
	w := httptest.NewRecorder()
	s.handleAdminAra(w, r)

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.NotContains(t, w.Body.String(), "<")
}

// Palet betiği varlık aramasına bağlı ve emojisiz olmalı.
func TestPaletVarlikAramasiYapar(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "static", "app.js"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, "/admin/ara?q=",
		"palet sunucu aramasına bağlanmalı; yalnız rota listesi kalmamalı")

	// Emoji yerine sprite ikonu.
	assert.Contains(t, s, "tabler-icons.svg?v=b89545a6#ti-")
	for _, e := range []string{"⚡", "🔎", "📌", "🎤"} {
		assert.NotContainsf(t, s, e, "palette emoji kalmamalı: %s", e)
	}
}

// Sonuçlar gruplu ve klavyeyle gezilebilir olmalı.
func TestPaletGrupluVeKlavyeli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "static", "app.js"))
	require.NoError(t, err)
	s := string(b)

	for _, iz := range []string{"cmdk-grup", "cmdk-oge", "tb_cmdk_recent",
		"ArrowDown", "ArrowUp", "Escape", "sonKullanilanlar"} {
		assert.Containsf(t, s, iz, "palet %q davranışını içermeli", iz)
	}

	i := strings.Index(s, "var RECENT = 'tb_cmdk_recent'")
	require.Greater(t, i, 0)
	blok := s[i:]
	j := strings.Index(blok, "document.querySelectorAll('.hero-slider')")
	require.Greater(t, j, 0)
	blok = blok[:j]
	assert.NotContains(t, blok, ".innerHTML =", "arama terimi innerHTML ile basılmamalı")
}

// Arama terimi vurgulanırken HTML kaçışı gerekmez: <mark> düğümü kurulur.
func TestPaletVurguDugumleKurulur(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "static", "app.js"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "createElement('mark')")
	assert.Contains(t, s, "document.createTextNode")
}

func TestAdminAramaAlanlariSabit(t *testing.T) {
	r := regexp.MustCompile(`json:"(id|baslik|alt|href|rozet|tur|ad|ikon|oge)"`)
	b, err := os.ReadFile(filepath.Join("..", "repo", "admin_search.go"))
	require.NoError(t, err)
	bulunan := map[string]bool{}
	for _, m := range r.FindAllStringSubmatch(string(b), -1) {
		bulunan[m[1]] = true
	}
	for _, a := range []string{"id", "baslik", "alt", "href", "rozet", "ad", "ikon", "oge"} {
		assert.Truef(t, bulunan[a], "arama cevabında %q alanı olmalı", a)
	}
}
