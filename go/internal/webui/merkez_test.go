package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Site Yönetim Merkezi (/admin/merkez) kaldırıldı (kullanılmıyordu:
// her bölüm kendi gerçek sayfasına gidiyordu, kabuk yalnız iframe sarıyordu).
//
// Bu testler kalıntı bağlantı bırakmadığımızı doğruluyor: kaldırılan bir
// sayfaya yan menüden/panodan/sekme çubuğundan giden bağlantı kalırsa
// kullanıcı 404'e düşer — üstelik 301 yönlendirme olduğu için bunu
// "çalışıyor" sanarak fark etmeyiz.

func TestMerkezBaglantisiKalmadi(t *testing.T) {
	// 1) Şablonlarda /admin/merkez bağlantısı olmamalı.
	var kalan []string
	err := filepath.Walk(filepath.Join("templates"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "/admin/merkez") {
			kalan = append(kalan, p)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Emptyf(t, kalan, "şu şablonlarda kaldırılan sayfaya bağlantı var: %v", kalan)

	// 2) Şablon dosyası silinmiş olmalı.
	_, err = os.Stat(filepath.Join("templates", "site_center.html"))
	assert.Truef(t, os.IsNotExist(err), "site_center.html hâlâ duruyor")

	// 3) Yan menüde "Site Merkezi" etiketi kalmamalı.
	b, err := os.ReadFile(filepath.Join("templates", "partials.html"))
	require.NoError(t, err)
	assert.NotContains(t, string(b), "Site Merkezi", "yan menüde etiket kalmış")
}

func TestMerkezSekmesiKalmadi(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// ws-tabs tanımında merkez alanı olmamalı: olsaydı /admin/merkez'e
	// giden bir "Site Merkezi" sekmesi çizilirdi.
	assert.NotContains(t, s, "id: 'merkez'",
		"sekme tanımında merkez alanı kalmış")
	assert.NotContains(t, s, "/admin/merkez",
		"JS'te kaldırılan sayfaya bağlantı kalmış")
}
