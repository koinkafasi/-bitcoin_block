package httpapi

import (
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Kategori varsayılan abonelik katmanı: alan ayrıştırma + şablon varlığı.

func TestKategoriKatmanAlaniAyristirma(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"0", 0}, {"1", 1}, {"3", 3}, {"5", 5},
		{"6", 0}, // üst sınır dışı → 0
		{"abc", 0},
	} {
		form := url.Values{
			"kategori_katman": []string{c.in},
			"min_skor":        []string{"0"},
			"ana_skor":        []string{"0"},
		}
		r := httptest.NewRequest("POST", "/admin/kategoriler", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		require.NoError(t, r.ParseForm())
		var cat repo.Category
		applyCategoryGateFields(r, &cat)
		assert.Equal(t, c.want, cat.MinTierWeight, "girdi: %s", c.in)
	}
}

func TestKategoriKatmanAlanYoksaSifir(t *testing.T) {
	form := url.Values{"min_skor": []string{"0"}, "ana_skor": []string{"0"}}
	r := httptest.NewRequest("POST", "/admin/kategoriler", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())
	var cat repo.Category
	applyCategoryGateFields(r, &cat)
	assert.Equal(t, 0, cat.MinTierWeight, "yeni kategoride alan yoksa kilit yok")
}

func TestKategoriKatmanGuncellemedeAlanYoksaKorunur(t *testing.T) {
	form := url.Values{"min_skor": []string{"0"}, "ana_skor": []string{"0"}}
	r := httptest.NewRequest("POST", "/admin/kategoriler/23", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())
	cat := repo.Category{ID: 23, MinTierWeight: 3}
	applyCategoryGateFields(r, &cat)
	assert.Equal(t, 3, cat.MinTierWeight, "güncellemede alan yoksa mevcut katman korunmalı")
}

func TestKategoriSablonuKatmanSeceneginiIcerir(t *testing.T) {
	src, err := os.ReadFile("../webui/templkit/kategoriler.templ")
	require.NoError(t, err, "kategori templ kaynağı okunmalı")
	s := string(src)
	assert.Contains(t, s, `name="kategori_katman"`, "kategori formunda katman alanı olmalı")
	assert.Contains(t, s, "Katman kilidi yok")
	assert.Contains(t, s, "Elmas (5)")
	assert.Contains(t, s, "Katmanlar []IOpt")
	assert.Contains(t, s, `name="footer"`, "kategori satırında footer alanı olmalı")
	assert.Contains(t, s, "InFooter", "footer durumu forma taşınmalı")
}

func TestIcerikFormundaKatmanAlaniVar(t *testing.T) {
	src, err := os.ReadFile("../webui/templkit/icerikform.templ")
	require.NoError(t, err)
	s := string(src)
	assert.Contains(t, s, `name="min_tier_weight"`, "içerik formunda katman ağırlığı alanı olmalı")
	assert.Contains(t, s, "Elmas (5)")
}
