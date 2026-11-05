package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Admin varlık yükleme kapsamı.
//
// Durum tespiti (28 Ekim 2026): `admin_head` 3 CSS + 16 JS dosyasını
// KOŞULSUZ yüklüyordu — her admin sayfasında 1.113.576 bayt. Bunun
// 770.939 baytı `editor.js` (TipTap + Yjs). Kontrol paneli, kategoriler ve
// ayarlar sayfaları içerik editörünü ve manşet motorunu boşuna indiriyordu.
//
// Çözüm: ağır betikler yalnız ihtiyaç duyan sayfalara. Eşleme tek bir saf
// fonksiyonda tutulur ki 40 handler'a dokunmadan, tablo ile test edilebilsin.

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

// Asıl kazanç: editörle ilgisi olmayan sayfalar 770 KB'lık paketi
// indirmemeli. Bu testin kırılması, birinin betiği yeniden koşulsuz
// yüklemeye aldığı anlamına gelir.
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
