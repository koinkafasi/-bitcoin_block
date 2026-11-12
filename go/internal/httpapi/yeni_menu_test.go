package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adminTop(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "go", "internal", "webui", "templates", "partials.html"))
	if err != nil {
		b, err = os.ReadFile(filepath.Join("..", "webui", "templates", "partials.html"))
	}
	require.NoError(t, err)
	return string(b)
}

// Üst barda "Yeni" menüsü olmalı ve her hedef var olan bir rota olmalı.
func TestYeniMenusuRotali(t *testing.T) {
	s := adminTop(t)
	require.Contains(t, s, `class="new-dd"`)

	i := strings.Index(s, `class="new-dd"`)
	son := strings.Index(s[i:], "</details>")
	require.Greater(t, son, 0)
	blok := s[i : i+son]

	// Menüdeki hedefler sunucuda tanımlı olmalı: ölü bağlantı olmasın.
	sunucu, err := os.ReadFile("server.go")
	require.NoError(t, err)
	hedefler := []string{
		"/admin/icerikler/yeni", "/admin/gazete", "/admin/dosyalar",
		"/admin/kategoriler", "/admin/yazarlar", "/admin/reklamlar",
	}
	for _, h := range hedefler {
		assert.Containsf(t, blok, `href="`+h+`"`, "menüde %s olmalı", h)
		assert.Containsf(t, string(sunucu), `"`+h+`"`, "hedef rota tanımlı değil: %s", h)
	}

	// Erişilebilirlik: menü rolü + öğe rolü + özet etiketi.
	assert.Contains(t, blok, `role="menu"`)
	assert.Contains(t, blok, `role="menuitem"`)
	assert.Contains(t, blok, `aria-label="Yeni oluştur"`)
}

// Menü emoji taşımamalı; simgeler yerel sprite'tan gelmeli.
func TestYeniMenusuSimgeli(t *testing.T) {
	s := adminTop(t)
	i := strings.Index(s, `class="new-dd"`)
	son := strings.Index(s[i:], "</details>")
	blok := s[i : i+son]

	assert.NotContains(t, blok, "➕", "artı işareti metin değil simge olmalı")
	assert.Contains(t, blok, "tabler-icons.svg?v=b89545a6#ti-plus")
	// 6 menü öğesi + 1 özet düğmesi = 7 simge.
	assert.Equal(t, 7, strings.Count(blok, `class="ti"`),
		"özet düğmesi ve altı menü öğesinin her biri simge taşımalı")
}

// Kapatma davranışı JS'te olmalı: dış tıklama ve Esc.
func TestYeniMenusuKapanir(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	i := strings.Index(s, "tbYeniMenu")
	require.Greater(t, i, 0)
	blok := s[i:]
	if j := strings.Index(blok, "function tbAdminInit"); j > 0 {
		blok = blok[:j]
	}
	assert.Contains(t, blok, "'Escape'", "Esc ile kapanmalı")
	assert.Contains(t, blok, "contains(e.target)", "dış tıklama kapatmalı")
	assert.Contains(t, blok, "__tbKuruldu", "HTMX değişiminde yeniden kurulmamalı")
}

// Yorum listesinde metin araması: repo katmanında ve şablonda.
func TestYorumAramasi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "repo", "analytics.go"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "func ListCommentsAra(")
	assert.Contains(t, s, "immutable_unaccent",
		"Türkçe katlama yorum aramasında da geçerli olmalı")

	// Geriye dönük uyum: eski imza korunmalı (çağıranlar kırılmasın).
	assert.Contains(t, s, "func ListComments(")

	tmpl, err := os.ReadFile(filepath.Join("..", "webui", "templkit", "topluluk.templ"))
	require.NoError(t, err)
	ts := string(tmpl)
	assert.Contains(t, ts, `name="q"`, "arama alanı olmalı")
	assert.Contains(t, ts, `aria-label="Yorumlarda ara"`)
	// Sekme durumu korunmalı: arama yapınca sekme sıfırlanmamalı.
	assert.Contains(t, ts, `name="durum" value={ v.Tab }`)
	// Boş durum bileşeni (Faz 163 tutarlılığı).
	assert.Contains(t, ts, `@BosDurum("search-off"`)
}

// Handler arama terimini geçirmeli.
func TestYorumAramasiHandler(t *testing.T) {
	b, err := os.ReadFile("admin_handlers.go")
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `r.URL.Query().Get("q")`)
	assert.Contains(t, s, "repo.ListCommentsAra(")
	assert.Contains(t, s, "Ara: ara")
}

// Üye listesinde metin araması (Faz 168, N-11).
func TestUyeAramasi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "repo", "users.go"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "func ListUsersAra(")
	assert.Contains(t, s, "immutable_unaccent",
		"Türkçe katlama üye aramasında da geçerli olmalı")
	// Geriye dönük uyum: eski imza korunmalı.
	assert.Contains(t, s, "func ListUsers(")

	tmpl, err := os.ReadFile(filepath.Join("..", "webui", "templkit", "topluluk.templ"))
	require.NoError(t, err)
	ts := string(tmpl)
	assert.Contains(t, ts, `name="q"`, "arama alanı olmalı")
	assert.Contains(t, ts, `aria-label="Üyelerde ara"`)
	assert.Contains(t, ts, `@BosDurum("search-off"`)

	h, err := os.ReadFile("members_handlers.go")
	require.NoError(t, err)
	hs := string(h)
	assert.Contains(t, hs, "repo.ListUsersAra(")
	assert.Contains(t, hs, `r.URL.Query().Get("q")`)
}
