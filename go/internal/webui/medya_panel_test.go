package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Medya paneli — Faz 187: sonsuz kaydırma, gruplu kategori çipleri, sade üst
// alan.
//
// Üç ayrı kullanıcı isteği:
//  1. "medya yüklemeyi sayfalama değil lazy load süreli otomatik yüklenir" →
//     sayfalama kaldırıldı, `revealed` gözcüsü geldi.
//  2. "Arama Kategoriler gruplayarak seçenek aksiyon ve efektif efektleri" →
//     kategori select'i yerine sayaçlı, geçişli çipler.
//  3. "Dosyaları seç Butonu fazla kaldır ... üste başlık alt metin gibi
//     seçenekler resim detayında zaten var olan alanları birdaha tekrar
//     olmasın panelde" → üstteki yükleme kartı tamamen kaldırıldı.

func medyaJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "media.js"))
	require.NoError(t, err)
	return string(b)
}

// medyaUstDilim, templ dosyasının modal ÖNCESİ kısmını döndürür: "üstte"
// olan alanları modal alanlarından ayırmak için.
func medyaUstDilim(t *testing.T, s string) string {
	t.Helper()
	i := strings.Index(s, `id="mediaEditModal"`)
	require.Greater(t, i, 0, "modal bulunamadı")
	return s[:i]
}

// TestMedyaTekrarAlanYok: dosya seç düğmesi ve başlık/alt metin/fotoğraf
// altı/açıklama alanları panelin üstünde TEKRAR EDİLMEMELİ.
//
// Bu alanlar resim detayında (modal) zaten var: aynı bilgiyi iki yerde
// sormak, yüklerken doldurulan alanın sonradan düzenlenmesi gerektiğinde
// hangisinin geçerli olduğu belirsizliğini doğuruyordu.
func TestMedyaTekrarAlanYok(t *testing.T) {
	ust := medyaUstDilim(t, medyaTempl(t))

	for _, yasak := range []string{
		`class="card card-pad media-upload"`,
		`class="media-upload"`,
		`id="medyaDosya"`,
		`name="baslik"`,
		`name="alt"`,
		`name="caption"`,
		`name="aciklama"`,
		`id="medyaOnizleme"`,
	} {
		assert.NotContains(t, ust, yasak,
			"üst alanda tekrar eden yükleme alanı kalmamalı: %s", yasak)
	}

	// Alanlar modal'da DURMALI — kaldırma değil, tekrarı kaldırma istendi.
	modal := medyaTempl(t)
	for _, gerekli := range []string{
		`id="mediaEditName"`, `id="mediaEditAlt"`,
		`id="mediaEditCaption"`, `id="mediaEditDescription"`,
		`id="mediaEditCategory"`,
	} {
		assert.Contains(t, modal, gerekli, "resim detayı alanı korunmalı: %s", gerekli)
	}

	// Yükleme hâlâ mümkün: galerideki sürükleme alanı.
	assert.Contains(t, medyaTempl(t), `data-mg-file`,
		"yükleme galerideki sürükleme alanından yapılmalı")
}

// TestMedyaSayfalamaYok: pager kalkmalı, yerine sonsuz kaydırma gözcüsü
// gelmeli.
func TestMedyaSayfalamaYok(t *testing.T) {
	s := medyaTempl(t)

	assert.NotContains(t, s, `media-pager`, "sayfalama düğmeleri kalkmalı")
	assert.NotContains(t, s, "Önceki", "önceki sayfa bağlantısı kalkmalı")
	assert.NotContains(t, s, "Sonraki", "sonraki sayfa bağlantısı kalkmalı")

	// Gözcü: görünür olunca sonraki dilimi ister, yerine geçer.
	assert.Contains(t, s, `id="medyaSon"`)
	assert.Contains(t, s, `hx-trigger="revealed"`)
	assert.Contains(t, s, `hx-swap="outerHTML"`)
	assert.Contains(t, s, `hx-get={ v.ParcaURL }`)
	assert.Contains(t, s, `Tümü gösterildi`,
		"liste bittiğinde kullanıcıya haber verilmeli")

	// Dilim yolu: sunucu `parca=1` isteğinde yalnız grupları basmalı.
	h, err := os.ReadFile(filepath.Join("..", "httpapi", "admin_handlers.go"))
	require.NoError(t, err)
	assert.Contains(t, string(h), `r.URL.Query().Get("parca") == "1"`,
		"parça isteği ayrı yoldan gitmeli")
	assert.Contains(t, string(h), "s.renderMedyaParca(")

	v, err := os.ReadFile(filepath.Join("..", "httpapi", "medya_view.go"))
	require.NoError(t, err)
	assert.Contains(t, string(v), `q.Set("parca", "1")`,
		"dilim adresi parca=1 taşımalı")
}

// TestMedyaKategoriCipleri: kategori select'i yerine sayaçlı çipler.
func TestMedyaKategoriCipleri(t *testing.T) {
	s := medyaTempl(t)
	ust := medyaUstDilim(t, s)

	assert.Contains(t, ust, `class={ "media-cip"`)
	assert.Contains(t, ust, "media-cipler")
	// Sayaç chip'in içinde: "genel 495" gibi.
	assert.Contains(t, ust, `<b>{ itoaT(c.Count) }</b>`)
	// Seçili chip erişilebilir olarak işaretli.
	assert.Contains(t, ust, `aria-current={ c.Aria }`)
	assert.Contains(t, ust, `aria-current={ v.TumuAria }`)

	// Eski select kalktı: kategori artık yalnız chip ile seçiliyor.
	assert.NotContains(t, ust, `<select class="input" name="kategori">`,
		"kategori select'i chip'lere taşındı")

	// Arama formu kategoriyi taşımalı; aksi hâlde arama yapınca filtre
	// kaybolurdu (canlı ölçümde yakalandı).
	assert.Contains(t, ust, `<input type="hidden" name="kategori" value={ v.Category }/>`)

	// Arama yazarken kendiliğinden çalışır ve odağı korur.
	assert.Contains(t, ust, `hx-trigger="input changed delay:350ms, search"`)
	assert.Contains(t, ust, `hx-preserve="true"`,
		"arama kutusu panel değişiminde korunmalı (yazarken odak kaybolmasın)")

	// CSS: chip geçişi tanımlı ve özgüllüğü yeterli (0,1,0 yetmiyordu).
	css := medyaCSS(t)
	i := strings.Index(css, ".media-panel .media-cip {")
	require.Greater(t, i, 0, "çip geçişi `.media-panel .media-cip` ile tanımlanmalı")
	assert.Contains(t, css[i:i+260], "transition:")
	assert.Contains(t, css[i:i+260], "transform")
}

// TestMedyaSonsuzKaydirmaBakimi: dilim sonrası gruplar birleşmeli, masonry
// tazelenmeli. Ayrıca JS öncesi ızgara çökmemeli.
func TestMedyaSonsuzKaydirmaBakimi(t *testing.T) {
	js := medyaJS(t)

	assert.Contains(t, js, "gruplariBirlestir",
		"sayfa sınırı bir ayın ortasına düşerse aynı grup birleştirilmeli")
	assert.Contains(t, js, "dataset.grup")
	assert.Contains(t, js, "window.__tbMediaTazele")

	// Birleştirme çağrısı, dilim sonrası bakımın İÇİNDE olmalı: yalnız
	// fonksiyonun tanımlı olması yetmez (çağrılmazsa grup iki kez görünür).
	iT := strings.Index(js, "const listeTazele = () => {")
	require.Greater(t, iT, 0, "listeTazele bulunamadı")
	tSon := strings.Index(js[iT:], "};")
	require.Greater(t, tSon, 0)
	taze := js[iT : iT+tSon]
	assert.Contains(t, taze, "gruplariBirlestir()",
		"dilim sonrası gruplar birleştirilmeli")
	assert.Contains(t, taze, "masonry()", "dilim sonrası ızgara tazelenmeli")

	// Dilim tespiti yanıt adresinden yapılmalı: outerHTML sonrası
	// detail.target ayrılmış düğüm olabiliyor.
	assert.Contains(t, js, "parca=1")
	assert.Contains(t, js, "xhr.responseURL")

	// JS öncesi varsayılan yükseklik: span yokken ızgara 4px satırla
	// çöküyor, gözcü hemen görünüyor ve sayfa açılır açılmaz ikinci dilim
	// isteniyordu (ölçüldü: 48 değil 96 kart).
	css := medyaCSS(t)
	i := strings.Index(css, ".media-panel:not(.media-list-view) .media-item {")
	require.Greater(t, i, 0)
	son := strings.Index(css[i:], "}")
	require.Greater(t, son, 0)
	assert.Contains(t, css[i:i+son], "grid-row-end: span",
		"JS öncesi kart makul yükseklikte olmalı")

	// Gözcü iskeleti ve animasyonu.
	assert.Contains(t, css, ".media-son__iskelet")
	assert.Contains(t, css, "@keyframes media-son-nabiz")
	assert.Contains(t, css, "prefers-reduced-motion")
}
