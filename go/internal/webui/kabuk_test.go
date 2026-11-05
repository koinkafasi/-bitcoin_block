package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tek kabuk testleri (Faz 154).
//
// Arka plan: pano kendi kabuğunu (.pn-app/.pn-side/.pn-top) çiziyordu, diğer
// sayfalar ortak kabuğu (.admin-wrap/.admin-side/.admin-top) kullanıyordu.
// Sonuç: iki sayfa arasında yan menü zemini, arama hapının biçimi ve gövde
// boşlukları farklıydı. Ölçüm: menü rgb(246,245,242) vs rgb(231,228,221);
// hap 340×42 r13 vs 340×38 r999.
//
// Artık tek kabuk var. Bu testler geri dönüşü engelliyor.

// TestTekKabuk_PanoKendiKabugunuCizmez: pano içeriği ortak kabuğun içinde
// yaşamalı. pn-app geri gelirse iki menü iki görünüm demektir.
func TestTekKabuk_PanoKendiKabugunuCizmez(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "pano.templ"))
	require.NoError(t, err)
	s := string(b)

	for _, yasak := range []string{"pn-app", "pn-side", "pn-top", "pn-brand"} {
		assert.NotContainsf(t, s, `"`+yasak+`"`,
			"pano şablonu kendi kabuğunu çiziyor (%s); ortak kabuk kullanılmalı", yasak)
	}
	assert.Contains(t, s, `class="pn-view"`,
		"pano içeriği .pn-view sarmalayıcısında olmalı")
}

// TestTekKabuk_PanoCSSKabukTasimaz: kabuk kuralları tek yerde
// (admin-pano.css). pano.css'te ikinci bir yan menü/üst çubuk tanımı
// olursa iki stil kaynağı yeniden doğar.
func TestTekKabuk_PanoCSSKabukTasimaz(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "pano.css"))
	require.NoError(t, err)
	s := string(b)

	for _, yasak := range []string{
		".pn-app {", ".pn-side {", ".pn-top {", ".pn-main {",
		".pn-item {", ".pn-brand {", ".pn-search {",
	} {
		assert.NotContainsf(t, s, yasak,
			"pano.css kabuk kuralı taşıyor (%s); admin-pano.css'e ait", yasak)
	}

	// Renk değişkenleri .pn-view kapsamında olmalı: .pn-app üzerinde
	// kalsaydı (o öge artık yok) grafikler renksiz kalırdı.
	assert.Contains(t, s, ".pn-view {", "değişkenler .pn-view kapsamında tanımlanmalı")
	assert.Contains(t, s, "--pn-sig", "pano renk değişkeni eksik")
}

// TestHareketKatmani: geçişler tüm panelde ortak ve erişilebilir olmalı.
func TestHareketKatmani(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	s := string(css)

	// 1) Kademeli giriş animasyonu her panel sayfasında geçerli.
	assert.Contains(t, s, "@keyframes pn-gir")
	assert.Contains(t, s, ".admin-body > *", "animasyon tüm gövde çocuklarını kapsamalı")
	assert.Contains(t, s, "animation-delay", "kademeli gecikme olmalı")

	// 2) HTMX ile yerinde güncellenen paneller yumuşak geçmeli.
	assert.Contains(t, s, ".htmx-settling")
	assert.Contains(t, s, ".htmx-added")

	// 3) Tam sayfa görünüm geçişi BİLİNÇLİ OLARAK YOK.
	//
	// Eski sürümde @view-transition vardı. Ölçüm gösterdi ki tam sayfa
	// görünüm geçişi ile sekme hapının animasyonu çakışıyordu: tarayıcı
	// geçiş boyunca DONDURULMUŞ görüntüyü gösterdiği için hap animasyonu
	// o karelerde görünmüyor, geçiş bitince hap çoktan varmış oluyordu.
	// Sonuç kullanıcının bildirdiği "tutarsız kayma"ydı.
	//
	// Tek geçiş sistemi: hap (admin-ui.js kayganHap) + kartların kademeli
	// girişi (pn-gir). Referans da SPA olduğu için tam sayfa geçişi
	// kullanmıyor. Kural geri gelirse bu test kırılır.
	// Yorumlar ayıklanır: bu kararın GEREKÇESİ dosyada yorum olarak
	// duruyor ve kaldırılan kuralın adı geçiyor. Aradığımız gerçek kural.
	kural := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(s, " ")
	assert.NotContains(t, kural, "@view-transition",
		"tam sayfa görünüm geçişi geri gelmemeli — hap animasyonuyla çakışır")
	assert.NotContains(t, kural, "view-transition-name",
		"adlandırılmış görünüm geçişi öğesi kalmamalı")

	// 4) Hareket hassasiyeti: hepsi reduced-motion altında kapanmalı.
	//    Birden çok reduced-motion bloğu var (hareket katmanı, hap);
	//    bu yüzden SON bloğu değil, hareket katmanının bloğunu arıyoruz.
	i := strings.Index(s, "prefers-reduced-motion: reduce) {\n\t.admin-body > *")
	require.Greater(t, i, 0, "hareket katmanının reduced-motion bloğu bulunamadı")
	son := s[i:]
	assert.Contains(t, son, ".admin-body > *", "giriş animasyonu hedefi durmalı")
	assert.Contains(t, son, "animation: none", "animasyonlar kapatılmalı")
	assert.Contains(t, son, ".htmx-settling", "HTMX geçişi reduced-motion'da durmalı")
	assert.Contains(t, son, ".table tbody tr", "tablo geçişi reduced-motion'da durmalı")
}

// TestHareketJS_TamSayfaGecisiYok: manuel startViewTransition çağrısı
// kalmamalı. O çağrı HTMX bağlantılarında preventDefault ederek paneli
// "tıklıyorum, olmuyor" hâline getirebiliyordu ve hap animasyonuyla
// çakışıyordu.
func TestHareketJS_TamSayfaGecisiYok(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	// Yalnız yorumda geçebilir; ÇAĞRI olarak bulunmamalı.
	assert.NotContains(t, s, "document.startViewTransition(",
		"manuel tam sayfa görünüm geçişi çağrısı kalmamalı")
	require.Contains(t, s, "window.tbToast",
		"ortak balon sistemi tanımlı olmalı")
}
