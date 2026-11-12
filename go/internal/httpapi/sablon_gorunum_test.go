package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func builderHTML(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "templates/site_builder.html")
	require.NoError(t, err)
	return string(b)
}

// yaziDali buildBP() içindeki yazı sayfası (son else) dalını döndürür.
func yaziDali(t *testing.T, s string) string {
	t.Helper()
	i := strings.Index(s, "function buildBP()")
	require.Greater(t, i, 0, "buildBP bulunmalı")
	seg := s[i:]
	// Son "} else {" yazı sayfası dalıdır (home, kategori, yazı sırası).
	j := strings.LastIndex(seg[:strings.Index(seg, "bp.innerHTML")], "} else {")
	require.Greater(t, j, 0, "yazı sayfası dalı bulunmalı")
	dal := seg[j:]
	if k := strings.Index(dal, "bp.innerHTML"); k > 0 {
		dal = dal[:k]
	}
	return dal
}

func TestSablonGorunumundeUstKutusuVar(t *testing.T) {
	s := builderHTML(t)
	dal := yaziDali(t, s)

	assert.Contains(t, dal, "zoneBox('ust')",
		"yazı sayfası şablon görünümünde Üst kutusu olmalı")
	// Sıra: ust, YAZI İÇERİĞİ kutusundan ÖNCE gelmeli.
	// (Yorumda da geçtiği için HTML etiketi aranır: >YAZI İÇERİĞİ<)
	ust := strings.Index(dal, "zoneBox('ust')")
	yazi := strings.Index(dal, ">YAZI İÇERİĞİ<")
	require.Greater(t, ust, 0)
	require.Greater(t, yazi, 0)
	assert.Less(t, ust, yazi, "Üst kutusu yazı içeriğinin üstünde olmalı")
}

func TestSablonGorunumundeUstuSuruklenebilir(t *testing.T) {
	s := builderHTML(t)
	// zoneBox data-slots ile sürükleme hedefi üretir; "ust" için de aynı
	// mekanizma kullanılmalı (özel durum yok).
	i := strings.Index(s, "function zoneBox(z, extraCls){")
	require.Greater(t, i, 0)
	blok := s[i : i+700]
	assert.Contains(t, blok, `data-zone="'+z+'"`)
	assert.Contains(t, blok, `data-slots="'+z+'"`,
		"her bölge sürükleme hedefi olmalı (ust dahil)")

	// Ust rozeti mavi (z-ust) tanımlı olmalı.
	assert.Contains(t, s, ".zone-chip.z-ust{")
}
