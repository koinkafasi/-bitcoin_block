package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func TestYaziSayfasiUstBolgesiVar(t *testing.T) {
	zones := builderZonesFor("article")
	require.NotEmpty(t, zones)

	// İlk bölge "ust" olmalı: panelde varsayılan bölge ilk sıradaki olur
	// (DEFAULT_ZONE = ZONES[0]), yani yeni eklenen modül doğrudan en üste düşer.
	assert.Equal(t, "ust", zones[0].Key, "yazı sayfasında ilk bölge Üst olmalı")
	assert.Equal(t, "Üst", zones[0].Name)
	assert.NotEmpty(t, zones[0].Hint)

	// Mevcut bölgeler korunmalı (geriye uyum)
	keys := map[string]bool{}
	for _, z := range zones {
		keys[z.Key] = true
	}
	assert.True(t, keys["yan"], "sağ sütun bölgesi korunmalı")
	assert.True(t, keys["genis"], "içerik altı bölgesi korunmalı")
}

func TestYaziSayfasiUstBolgesiSablonu(t *testing.T) {
	b, err := fs.ReadFile(webui.FS, "templates/site_public.html")
	require.NoError(t, err)
	s := string(b)

	// Makale şablonu içinde "ust" bölgesi, iki kolonlu düzenin (art-wrap) ve
	// başlığın (title-xl) ÖNÜNDE basılmalı.
	artStart := strings.Index(s, `{{define "article"}}`)
	require.Greater(t, artStart, 0, "article şablonu bulunmalı")

	bolge := strings.Index(s[artStart:], `eq .Zone "ust"`)
	wrap := strings.Index(s[artStart:], `class="art-wrap`)
	baslik := strings.Index(s[artStart:], `class="title-xl"`)

	require.Greater(t, bolge, 0, "article şablonunda ust bölgesi basılmalı")
	require.Greater(t, wrap, 0)
	require.Greater(t, baslik, 0)
	assert.Less(t, bolge, wrap, "üst bölge iki kolonlu düzenin önünde olmalı")
	assert.Less(t, bolge, baslik, "üst bölge başlıktan önce olmalı")
}
