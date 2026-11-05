package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Pano 2.0 panelin giriş sayfası oldu (/admin). Etkileşim katmanının tamamı
// pano.js'te: ray katlama, segment kontrolü, grafik balonu, sürüklenebilir
// yayın şeridi. Betik yüklenmezse sayfa DOĞRU GÖRÜNÜR ama hiçbir şey
// çalışmaz — sessiz bir kırılma, çünkü konsola hata da düşmez.
//
// Bu tam olarak bir kez oldu: giriş sayfası /admin/pano'dan /admin'e
// taşınınca eşleme güncellenmedi ve ray düğmesi tıklanınca hiçbir şey
// yapmaz hâle geldi.
func TestPanoBetigiGirisSayfasindaYukleniyor(t *testing.T) {
	for _, yol := range []string{"/admin", "/admin/pano"} {
		require.Containsf(t, adminScriptsFor(yol), "pano.js",
			"%s pano.js yüklemeli; yoksa pano görünür ama etkileşimsiz kalır", yol)
	}
}

// "/admin" önek olarak değerlendirilseydi bütün panel sayfaları pano.js
// alırdı. Tam eşleşme olduğunu sabitler.
func TestAdminOnekiSizmiyor(t *testing.T) {
	sizanlar := []string{
		"/admin/dosyalar",
		"/admin/ayarlar",
		"/admin/icerikler",
		"/admin/seo",
		"/admin/ozet", // eski pano: Pacely katmanı yok, pano.js gereksiz
	}
	for _, yol := range sizanlar {
		require.NotContainsf(t, adminScriptsFor(yol), "pano.js",
			"%s pano.js almamalı — /admin tam eşleşme olmalı, önek değil", yol)
	}
}

// İçerik formunun ağır paketi pano yollarına sızmamalı (ve tersi).
func TestBetikEslemeleriKarismiyor(t *testing.T) {
	require.NotContains(t, adminScriptsFor("/admin"), "editor.js")
	require.NotContains(t, adminScriptsFor("/admin/pano"), "editor.js")
	require.Contains(t, adminScriptsFor("/admin/icerikler/412"), "editor.js")
	require.NotContains(t, adminScriptsFor("/admin/icerikler/412"), "pano.js")
}
