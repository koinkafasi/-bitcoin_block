package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilSimgesiHepHesabimaGider(t *testing.T) {
	src := okuDosya(t, "public_handlers.go")

	i := strings.Index(src, `d.MemberURL = "/hesabim"`)
	require.Greater(t, i, 0, "profil simgesi /hesabim'e gitmiyor")

	// Personel için /admin'e yönlendiren eski blok kalmamalı.
	assert.NotContains(t, src, `d.MemberURL = "/admin"`,
		"personel profil simgesi yönetim paneline yönlendiriliyor (bildirilen hata)")

	// Personel bayrağı ayrı taşınır: kısayol gösterimi için.
	assert.Contains(t, src, "d.MemberStaff = auth.StaffRoles[u.Role]")
}

func TestHesapSayfasiPersonelKisayolu(t *testing.T) {
	// Hesap merkezi de bayrağı kurmalı, yoksa kısayol hiç görünmez.
	acc := okuDosya(t, "account_handlers.go")
	assert.Contains(t, acc, "d.MemberStaff = auth.StaffRoles[u.Role]")

	tpl := okuDosya(t, "../webui/templates/site_hesabim.html")
	assert.Contains(t, tpl, "{{if .MemberStaff}}", "personel kısayolu şablonda yok")
	assert.Contains(t, tpl, `href="/admin"`, "yönetim paneli bağlantısı yok")
}

func TestCekmecePersonelKisayoluVeCeviriler(t *testing.T) {
	tpl := okuDosya(t, "../webui/templates/partials.html")

	// Çekmecede açık yönetim kısayolu (gizli yönlendirme yerine).
	assert.Contains(t, tpl, `{{if .MemberStaff}}`)
	assert.Contains(t, tpl, `{{.T "staff_panel"}}`)

	// Sabit Türkçe metinler çeviriye bağlandı: İngilizce sitede de
	// "menü"/"Yazı boyutu" Türkçe kalıyordu.
	assert.Contains(t, tpl, `{{.T "menu"}}`)
	assert.Contains(t, tpl, `{{.T "font_size"}}`)
	assert.Contains(t, tpl, `{{.T "menu_close"}}`)
	assert.NotContains(t, tpl, ">Yazı boyutu</label>", "sabit Türkçe etiket kaldı")

	// Geçerli sayfa işareti JS'te konuyor (CSS kuralı boşa durmasın).
	js := okuDosya(t, "../webui/static/app.js")
	assert.Contains(t, js, "drawerIsaretle")
	assert.Contains(t, js, `setAttribute('aria-current', 'page')`)
}

func TestCekmeceCeviriAnahtarlari(t *testing.T) {
	up := okuDosya(t, "../../migrations/0104_ceviri_cekmece.up.sql")
	for _, k := range []string{"font_size", "staff_panel", "admin_group", "menu_close"} {
		assert.Contains(t, up, "'"+k+"'", "çeviri anahtarı yok: "+k)
	}
	assert.Contains(t, up, "ON CONFLICT (locale, key) DO NOTHING",
		"panelden düzenlenmiş çeviri ezilmemeli")

	down := okuDosya(t, "../../migrations/0104_ceviri_cekmece.down.sql")
	assert.Contains(t, down, "DELETE FROM translations")
}

func TestCekmeceEtiketleriBosKalmaz(t *testing.T) {
	up := okuDosya(t, "../../migrations/0105_ceviri_bos_degerler.up.sql")

	// Boş değerleri dolduran dosya bu anahtarları içermeli.
	for _, k := range []string{
		"footer.akis", "footer.bulten", "subscription", "footer.hava-durumu",
		"footer.namaz-vakitleri", "footer.borsa", "footer.fikstur",
		"footer.astroloji", "footer.nobetci-eczaneler",
	} {
		assert.Contains(t, up, "'"+k+"'", "çekmece anahtarı eksik: "+k)
	}
	// Panelden yazılmış metin ezilmesin.
	assert.Contains(t, up, "WHERE translations.value = ''")

	// Boş değeri "yok" sayan kural i18n tarafında.
	i18n := okuDosya(t, "../i18n/i18n.go")
	assert.Contains(t, i18n, `ok && v != ""`,
		"boş çeviri değeri etiketi silmemeli")
}

// TestCekmeceGecerliSayfaIsareti — açılışta da işaret konsun.
func TestCekmeceGecerliSayfaIsareti(t *testing.T) {
	js := okuDosya(t, "../webui/static/app.js")
	assert.Contains(t, js, "DOMContentLoaded', drawerIsaretle",
		"çekmece JS ile açıldığında geçerli sayfa işaretlenmiyor")
}
