package httpapi

// İçerikler sayfasındaki DURUM sekmeleri (Yayında / Taslak / Planlanmış /
// Arşiv / Çöp / Tümü) üst menüdeki hap geçişinin AYNISINI yapmalı
// (kullanıcı isteği, 3 Kasım 2026).
//
// Ölçülen sorunlar (canlı panel, Puppeteer):
//   1. Hap hangi sekme seçili olursa olsun HEP İLK sekmeye ("Yayında")
//      oturuyordu: clip 4 sekme boyunca hiç değişmiyordu.
//      Sebep: `aktifOge` içinde "tam yol eşleşmesi her zaman kazanır"
//      kuralı ilk sıradaydı; bütün durum sekmeleri `/admin/icerikler?durum=…`
//      adresine gittiği ve sorgu atıldığında `/admin/icerikler` =
//      location.pathname olduğu için ilk aday dönüyordu.
//   2. Hap kaymıyor, SOLUYORDU: panel `hx-swap` ile tümüyle değişiyor,
//      htmx `.htmx-added` veriyor, pn-yenile (opacity .35→1) hapı da
//      kapsıyordu. Ölçüm: opacity 1 → 0 → 1, clip sabit.
//   3. Sekme çubuğu her takasta yeni DOM olduğu için kimlik kaydı
//      eşleşmiyordu (`ray:adsiz`) ve tıklama işleyicisi `hx-get`
//      bağlantılarını atlıyordu → konum devri hiç yazılmıyordu.
//
// Düzeltmeden sonra ölçüm (aynı araç):
//   Taslak   hap clip çeşidi 18 · opaklık min 1.00 · içerik −28 px
//   Arşiv    hap clip çeşidi 18 · opaklık min 1.00 · içerik +28 px
//   Yayında  hap clip çeşidi 18 · opaklık min 1.00 · içerik −28 px
//   Tümü     hap clip çeşidi 18 · opaklık min 1.00 · içerik +28 px

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func jsOku(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/admin-ui.js")
	require.NoError(t, err)
	return string(b)
}

func cssOku(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/admin-pano.css")
	require.NoError(t, err)
	return string(b)
}

// Sunucunun işaretlediği sekme, adres eşleşmesinden ÖNCE gelmeli.
// (Aksi hâlde ?durum=… sekmelerinde hap hep ilk sekmede kalıyordu.)
func TestDurumSekmeAktifOgeSirasi(t *testing.T) {
	js := jsOku(t)

	bas := strings.Index(js, "function aktifOge(")
	require.Greater(t, bas, 0)
	son := strings.Index(js[bas:], "\n    }\n")
	require.Greater(t, son, 0)
	govde := js[bas : bas+son]

	// İşaretli (aria-current / .active / .on) toplama, yol eşleşmesinden önce.
	isaretIdx := strings.Index(govde, "a.getAttribute('aria-current') === 'page' ||")
	yolIdx := strings.Index(govde, "href && href === yol")
	require.Greater(t, isaretIdx, 0, "işaretli sekme toplama bulunamadı")
	require.Greater(t, yolIdx, 0, "tam yol eşleşmesi bulunamadı")
	assert.Less(t, isaretIdx, yolIdx,
		"önce sunucunun işaretlediği sekme gelmeli; ?durum= sekmelerinde yol eşleşmesi hepsine uyuyor")

	// Tek işaretli varsa doğrudan o dönmeli.
	assert.Contains(t, govde, "if (isaretli.length === 1) return isaretli[0];")
	// Birden fazla işaretli: tam yol ayırt edici ( /admin/icerikler/yeni ).
	assert.Contains(t, govde, "if (isaretli.length > 1) {")
}

// Kimlik: id'siz çubuklar için de kararlı olmalı (yol tabanlı).
func TestDurumSekmeKimlikKararli(t *testing.T) {
	js := jsOku(t)
	bas := strings.Index(js, "function kimlik(")
	require.Greater(t, bas, 0)
	govde := js[bas : bas+900]

	assert.Contains(t, govde, "if (ray.dataset && ray.dataset.hap) return 'hap:' + ray.dataset.hap;")
	assert.Contains(t, govde, "if (ray.id) return 'ray:' + ray.id;")
	assert.Contains(t, govde, "var a = ray.querySelector('a[href]');")
	assert.Contains(t, govde, "return 'yol:' + (yol || 'adsiz');")
	assert.NotContains(t, govde, "'ray:' + (ray.id || 'adsiz')",
		"id'siz çubuklar 'adsiz' anahtarını paylaşmamalı")
}

// hx-get/hx-post sekmeleri de gezinmedir: konum devri yazılmalı.
func TestDurumSekmeTiklamadaKonumDevri(t *testing.T) {
	js := jsOku(t)

	bas := strings.Index(js, "// Tıklama: hem anında tepki hem SONRAKİ sayfaya konum devri.")
	require.Greater(t, bas, 0)
	govde := js[bas : bas+1400]

	assert.NotContains(t, govde, "if (a.hasAttribute('hx-get') || a.hasAttribute('hx-post')) return;",
		"hx-get sekmeleri atlanmamalı; aksi hâlde hap kaymıyordu")
	assert.Contains(t, govde, "yaz(ray.dataset.hapId, {")
}

// Hap geri yüklenirken kabın VE rayın belirme animasyonu susturulmalı,
// ama susturma kuralı TORUNLARA uygulanmamalı.
func TestDurumSekmeBelirmeSusturma(t *testing.T) {
	js := jsOku(t)
	assert.Contains(t, js, "function rayonBelirmeSustur(")
	assert.Contains(t, js, "ray.closest('.htmx-added, .htmx-settling')")
	assert.Contains(t, js, "el.classList.add('tb-hap-geliyor')")

	css := cssOku(t)
	assert.Contains(t, css, ".tb-hap-geliyor {")
	assert.Contains(t, css, "animation: none !important;")
	// `transform: none !important` torunlara bulaşırsa içerik süzülmesi
	// (WAAPI) ezilir — bu satır regresyonun ta kendisiydi.
	assert.NotContains(t, css, ".tb-hap-geliyor .tabs {",
		"susturma kuralı yalnız sınıfı taşıyan öğeye uygulanmalı")
	// Hap eski yerinden ANINDA görünmeli (solma değil, kayma).
	assert.Contains(t, css, ".tb-hap-geliyor .tb-hap,")
	assert.Contains(t, css, "transition: none !important;")
}

// Durum sekmesinde İÇERİK de hapın yönünden süzülmeli.
func TestDurumSekmeIcerikSuzulmesi(t *testing.T) {
	js := jsOku(t)

	// Yön, istek adresi ile çubuk sırası karşılaştırılarak bulunur
	// (detail.elt güvenilmez: htmx gösterge DIV'ini veriyor).
	assert.Contains(t, js, "function sekmeIndeksi(")
	assert.Contains(t, js, "var yeniIdx = sekmeIndeksi(ray, istek);")
	assert.Contains(t, js, "var eskiIdx = sekmeIndeksi(ray, null);")
	assert.Contains(t, js, "sekmeYonu = yeniIdx < eskiIdx ? -1 : 1;")

	// Panel değişiminde sekme çubuğu hariç içerik süzülür.
	assert.Contains(t, js, "if (!kabukGezinmesi && sekmeYonu !== 0) {")
	assert.Contains(t, js, "suzul(panel, sekmeYonu, '.tabs');")
	assert.Contains(t, js, "function suzul(")
}
