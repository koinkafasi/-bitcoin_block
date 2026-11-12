package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminScriptsIcerikFormuEditoruYukler(t *testing.T) {
	for _, path := range []string{
		"/admin/icerikler/yeni",
		"/admin/icerikler/128",
		"/admin/icerikler/128/onizleme",
	} {
		got := adminScriptsFor(path)
		assert.Contains(t, got, "editor.js", "yol: %s", path)
		assert.Contains(t, got, "ai_editor.js", "yol: %s", path)
		assert.Contains(t, got, "audio_panel.js", "yol: %s", path)
		assert.Contains(t, got, "ses_player.js", "yol: %s", path)
	}
}

func TestAdminScriptsMansetEditoruMotoruYukler(t *testing.T) {
	got := adminScriptsFor("/admin/manset-editor")
	assert.Contains(t, got, "manset.js")
	assert.NotContains(t, got, "editor.js", "manşet editörü TipTap paketini yüklememeli")
}

func TestAdminScriptsSesAyarlariOynaticiyiYukler(t *testing.T) {
	got := adminScriptsFor("/admin/ses-ayarlari")
	assert.Contains(t, got, "ses_player.js")
	assert.NotContains(t, got, "editor.js")
}

func TestAdminScriptsDigerSayfalarAgirPaketleriYuklemez(t *testing.T) {
	for _, path := range []string{
		"/admin",
		"/admin/kategoriler",
		"/admin/ayarlar",
		"/admin/raporlar",
		"/admin/yorumlar",
		"/admin/seo",
		"/admin/dosyalar",
		"/admin/icerikler", // liste sayfası — form değil
	} {
		got := adminScriptsFor(path)
		assert.NotContains(t, got, "editor.js", "yol: %s — 770 KB boşuna inmemeli", path)
		assert.NotContains(t, got, "ai_editor.js", "yol: %s", path)
		assert.NotContains(t, got, "manset.js", "yol: %s", path)
	}
}

// Bilinmeyen yol güvenli tarafa düşmeli: ağır paket yüklenmez.
func TestAdminScriptsBilinmeyenYolBosDoner(t *testing.T) {
	assert.Empty(t, adminScriptsFor("/admin/boyle-bir-sayfa-yok"))
	assert.Empty(t, adminScriptsFor(""))
	assert.Empty(t, adminScriptsFor("/"))
}

// Yol eşleme önek kaçağına izin vermemeli: "/admin/icerikler-arsiv" içerik
// formu değildir.
func TestAdminScriptsOnekKacagiYok(t *testing.T) {
	assert.NotContains(t, adminScriptsFor("/admin/icerikler-arsiv"), "editor.js")
	assert.NotContains(t, adminScriptsFor("/admin/manset-editor-eski"), "manset.js")
}
