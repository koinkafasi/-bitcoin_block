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

// Medya galerisi — kart görünümü, sürükleme alanı, yükleme halkası (Faz 186).
//
// Referans (Murmura /library/media) yalnız ÖLÇÜLDÜ; değerler:
//
//	kart yarıçapı      20px
//	kart gölgesi       inset 0 1px rgba(255,255,255,.5), 0 18px 30px -22px
//	ızgara             grid-auto-rows 4px, column-gap 14px
//	masonry            gridRowEnd = yükseklik / 4
//	sürükleme kutusu   1.5px dashed, radius 20px, .2s geçiş
//	ikon dairesi       52x52, tam yuvarlak
//	yükleme halkası    64x64 SVG, r=22, stroke 5, dasharray 2π·22 = 138.23
//
// Kod/varlık alınmadı; yalnız bu ölçüler kendi uygulamamıza taşındı.

func medyaCSS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "admin-only.css"))
	require.NoError(t, err)
	return string(b)
}

func medyaTempl(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("templkit", "medya.templ"))
	require.NoError(t, err)
	return string(b)
}

// medyaKural, verilen SEÇİCİYLE BAŞLAYAN kuralın gövdesini döndürür.
//
// Alt dize aramak yetmiyor: `.media-panel:not(.media-list-view) .media-item`
// metni kart kuralı kaldırılsa bile diğer (`.media-open`, `.mi-actions`)
// seçicilerde geçmeye devam ediyor, yani mutasyon yakalanmıyordu.
func medyaKural(t *testing.T, css, secici string) string {
	t.Helper()
	for _, satir := range strings.Split(css, "\n") {
		if strings.TrimSpace(satir) != secici+" {" {
			continue
		}
		i := strings.Index(css, satir)
		require.GreaterOrEqual(t, i, 0)
		son := strings.Index(css[i:], "}")
		require.Greater(t, son, 0)
		return css[i : i+son]
	}
	require.Failf(t, "kural bulunamadı", "seçici: %s", secici)
	return ""
}

// TestMedyaKartGorunumu: kart yarıçapı/gölgesi ve ölçülen masonry ızgarası.
func TestMedyaKartGorunumu(t *testing.T) {
	s := medyaCSS(t)

	grid := medyaKural(t, s, ".media-panel:not(.media-list-view) .media-grid")
	assert.Contains(t, grid, "grid-auto-rows: 4px",
		"masonry satır yüksekliği 4px olmalı (ölçüldü)")
	assert.Contains(t, grid, "column-gap: 14px", "sütun aralığı 14px")

	// Kapsam: liste görünümü aynı DOM'u kullanıyor. Kart kuralı kapsamsız
	// olursa `.media-item img{height:100%}` listedeki 82px'lik satırları
	// da bozardı (ölçüldü: itemH 800, imgH 0).
	kart := medyaKural(t, s, ".media-panel:not(.media-list-view) .media-item")
	assert.Contains(t, kart, "border-radius: 20px", "kart yarıçapı 20px")
	assert.Contains(t, kart, "0 18px 30px -22px", "kart gölgesi ölçülen değer")
	assert.Contains(t, kart, "overflow: hidden", "kart görseli taşmamalı")
}

// TestMedyaSuruklemeKutusu: galerinin ilk kutusu, ölçülen stil.
func TestMedyaSuruklemeKutusu(t *testing.T) {
	s := medyaCSS(t)
	kirp := func(sel string) string {
		i := strings.Index(s, sel+" {")
		require.Greaterf(t, i, 0, "kural bulunamadı: %s", sel)
		son := strings.Index(s[i:], "}")
		require.Greater(t, son, 0)
		return s[i : i+son]
	}

	btn := kirp(".mgal-drop__btn")
	assert.Contains(t, btn, "border: 1.5px dashed", "1.5px kesikli kenar (ölçüldü)")
	assert.Contains(t, btn, "border-radius: 20px")
	assert.Contains(t, btn, "transition:")

	ico := kirp(".mgal-drop__ico")
	assert.Contains(t, ico, "width: 52px")
	assert.Contains(t, ico, "height: 52px")
	assert.Contains(t, ico, "border-radius: 50%")

	// Sürükleme durumu: marka tonuyla dolgu + düz kenar.
	assert.Contains(t, s, ".media-panel.is-drag .mgal-drop__btn")
	assert.Contains(t, s, ".media-panel.is-drag .mgal-drop__ico")
}

// TestMedyaYuklemeHalkasi: SVG halka ve dasharray.
func TestMedyaYuklemeHalkasi(t *testing.T) {
	s := medyaCSS(t)
	assert.Contains(t, s, ".mgal-up__ring", "yükleme halkası tanımlı olmalı")
	assert.Contains(t, s, "transform: rotate(-90deg)",
		"halka -90° dönmeli (12 yönünden başlar)")
	assert.Contains(t, s, "stroke-width: 5px", "halka kalınlığı 5px (ölçüldü)")

	js, err := os.ReadFile(filepath.Join("static", "media.js"))
	require.NoError(t, err)
	j := string(js)

	// dasharray = 2π·22. Sabit yazmak yerine hesap: referansla aynı.
	assert.Contains(t, j, "const CEVRE = 2 * Math.PI * 22",
		"halka çevresi r=22 ile hesaplanmalı (138.23)")
	assert.Contains(t, j, "stroke-dashoffset",
		"ilerleme dashoffset ile gösterilmeli")
	assert.Contains(t, j, "xhr.upload.onprogress",
		"gerçek yükleme ilerlemesi dinlenmeli")
	assert.Contains(t, j, "'/admin/dosyalar?json=1'",
		"yükleme JSON ucuna gitmeli (panel tazelemesi ayrı)")
}

// TestMedyaMasonryDongusu: görünüm değişince span tazelenmeli; TDZ hatası
// geri gelmemeli.
func TestMedyaMasonryDongusu(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "media.js"))
	require.NoError(t, err)
	s := string(b)

	// applyView masonry çağırmalı (liste görünümünde span kalırsa kart
	// 800px yüksekliğe uzuyordu — ölçüldü).
	i := strings.Index(s, "const applyView = (view) => {")
	require.Greater(t, i, 0)
	son := strings.Index(s[i:], "};")
	require.Greater(t, son, 0)
	assert.Contains(t, s[i:i+son], "masonry()",
		"görünüm değişince masonry span'ları tazelenmeli")

	// Görünüm bloğu, masonry sabitlerinden SONRA gelmeli: init sırasında
	// `Cannot access 'GAP' before initialization` alınmıştı.
	iMasonry := strings.Index(s, "const GAP = 14;")
	iApply := strings.Index(s, "const applyView = (view) => {")
	assert.Greater(t, iApply, iMasonry,
		"görünüm bloğu masonry sabitlerinden önce çalışmamalı (TDZ)")

	// Kırık görsel yedeği: dinleyici çağrısı ve yer tutucu sınıfı ayrı ayrı
	// aranır. Yalnız "'error'" aramak yetmiyordu — açıklama yorumunda da
	// aynı metin geçtiği için dinleyici silinse bile test yeşil kalıyordu.
	assert.Contains(t, s, "panel.addEventListener('error'",
		"kırık görsel yakalanmalı (kırık ikon + alt metin kartın üstüne biniyordu)")
	assert.Contains(t, s, "mi-kirik",
		"kırık görselde nötr yer tutucu sınıfı eklenmeli")
}

// TestMedyaDropKutusuIsaretleme: galerinin ilk kutusu + erişilebilir dosya
// girdisi.
func TestMedyaDropKutusuIsaretleme(t *testing.T) {
	s := medyaTempl(t)

	assert.Contains(t, s, "templ medyaDropKutu()")
	assert.Contains(t, s, `class="mgal-drop"`)
	assert.Contains(t, s, `data-mg-file`)
	assert.Contains(t, s, `data-mg-browse`)
	// Kart görünümünde alt yazı; eski metin gövdesi korunmalı (liste).
	assert.Contains(t, s, `class="mi-cap"`)
	assert.Contains(t, s, `class="mi-body"`)
	// Panel kapsamı
	assert.Contains(t, s, `class="media-panel"`)

	// Sürükleme kutusu galeri içinde olmalı (referansta ilk kutu).
	i := strings.Index(s, `class="media-grid"`)
	require.Greater(t, i, 0)
	assert.Contains(t, s[i:], "@medyaDropKutu()",
		"sürükleme kutusu galerinin içinde olmalı")
}

// TestMedyaGridSabitDegerleri: ölçülen ızgara değerleri kaynakta.
func TestMedyaGridSabitDegerleri(t *testing.T) {
	s := medyaCSS(t)
	i := strings.Index(s, ".media-panel:not(.media-list-view) .media-grid {")
	require.Greater(t, i, 0)
	son := strings.Index(s[i:], "}")
	require.Greater(t, son, 0)
	blok := s[i : i+son]

	assert.Contains(t, blok, "grid-template-columns: repeat(auto-fill, minmax(190px, 1fr))",
		"ızgara dar ekranda sarmalı")
	re := regexp.MustCompile(`grid-auto-rows:\s*4px`)
	assert.Regexpf(t, re, blok, "masonry satır yüksekliği")
}

// TestMedyaKartOraniSinirli: kart yüksekliği özgün dosya oranından değil
// EKRANDAKİ görselin oranından gelmeli ve sınırlanmalı.
//
// Gerçek vaka: özgün ölçü 750×15873 (uzun şerit ekran görüntüsü), thumb
// 422×422. Özgün oranla kart 215×4554 px oluyordu (ölçüldü: span 1142,
// ızgarada tek başına bir sütun boyu boşluk).
func TestMedyaKartOraniSinirli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "media.js"))
	require.NoError(t, err)
	s := string(b)

	i := strings.Index(s, "function oran(el) {")
	require.Greater(t, i, 0, "oran() bulunamadı")
	son := strings.Index(s[i:], "\n    }")
	require.Greater(t, son, 0)
	govde := s[i : i+son]

	// naturalWidth/naturalHeight öncelikli olmalı: data-dimensions özgün
	// dosyanın ölçüsü, thumb ise kırpılmış olabiliyor.
	assert.Contains(t, govde, "naturalWidth",
		"oran ekrandaki görselin gerçek ölçüsünden gelmeli")
	iNat := strings.Index(govde, "naturalWidth")
	iDim := strings.Index(govde, "dataset.dimensions")
	assert.Less(t, iNat, iDim,
		"gerçek görsel ölçüsü, sunucunun özgün ölçüsünden önce denenmeli")

	// Sınır: sabitler tanımlı, sinirla() gerçekten kırpıyor ve oran()
	// her dönüşte onu uyguluyor.
	assert.Contains(t, s, "const ORAN_MIN = 0.5;")
	assert.Contains(t, s, "const ORAN_MAX = 2.5;")
	assert.Contains(t, govde, "sinirla(",
		"her dönüş sınırlanmalı (4554 px kart geri gelmemeli)")

	// sinirla() passthrough olursa sınır yok sayılır: iki ucun da
	// kırpıldığı ayrıca doğrulanır.
	iS := strings.Index(s, "function sinirla(r) {")
	require.Greater(t, iS, 0, "sinirla() bulunamadı")
	sSon := strings.Index(s[iS:], "\n    }")
	require.Greater(t, sSon, 0)
	sGovde := s[iS : iS+sSon]
	assert.Contains(t, sGovde, "Math.min(ORAN_MAX",
		"üst sınır uygulanmalı (çok geniş görsel)")
	assert.Contains(t, sGovde, "Math.max(ORAN_MIN",
		"alt sınır uygulanmalı (çok uzun görsel)")

	// Görseller yüklendikçe ızgara tazelenmeli.
	assert.Contains(t, s, "panel.addEventListener('load'",
		"görsel yüklenince oran düzeltilmeli")
	assert.Contains(t, s, "item.dataset.ar",
		"ölçülen oran karta yazılmalı")
}
