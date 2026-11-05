package httpapi

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/piyasa"
	"theblock/site/internal/webui"
)

// Kur & Borsa modülü: filtre + şablon çıktısı.
//
// Referans düzen Habertürk ekonomi şeridi (birebir):
//   <a class="flex space-x-2">
//     <span class="flex flex-col">  etiket / fiyat / yüzde
//     <i class="setIcon h-8 w-3">   ok EN SAĞDA, altı noktalı
//   </a>
// Çerçeve yok, yalnız alt çizgi.

func ornekKalemler() []piyasa.Kalem {
	return []piyasa.Kalem{
		{Anahtar: "bist100", Kisa: "BIST 100", Ad: "BIST 100", Deger: 11151.03, Degisim: -0.01, Ondalik: 2, Yon: -1},
		{Anahtar: "usd", Kisa: "DOLAR", Ad: "Dolar", Deger: 49.8335, Degisim: 0.05, Ondalik: 4, Yon: 1},
		{Anahtar: "ceyrek-altin", Kisa: "ÇEYREK ALTIN", Ad: "Çeyrek Altın", Deger: 10558.61, Degisim: 0, Ondalik: 2, Yon: 0},
	}
}

func TestFilterPiyasaAyarYoksaTumu(t *testing.T) {
	out := filterPiyasa(ornekKalemler(), nil)
	assert.Len(t, out, 3, "ayar yoksa (eski düzen) tüm kalemler görünür")
}

func TestFilterPiyasaKalemKapatma(t *testing.T) {
	out := filterPiyasa(ornekKalemler(), map[string]any{
		"piyasa_bist100":      false,
		"piyasa_ceyrek-altin": false,
	})
	require.Len(t, out, 1)
	assert.Equal(t, "usd", out[0].Anahtar)
}

func TestFilterPiyasaAnahtarKimligiIleEslesir(t *testing.T) {
	// Görünen ad ("Dolar") değişse de ayar anahtarı kimliktir; kapalı kalır.
	k := []piyasa.Kalem{{Anahtar: "usd", Kisa: "USD/TRY", Ad: "Amerikan Doları", Deger: 49, Ondalik: 4}}
	out := filterPiyasa(k, map[string]any{"piyasa_usd": false})
	assert.Empty(t, out)
}

func TestFilterPiyasaCountSiniri(t *testing.T) {
	out := filterPiyasa(ornekKalemler(), map[string]any{"count": float64(2)})
	assert.Len(t, out, 2)
	// JSON'dan string gelse de çalışmalı
	out = filterPiyasa(ornekKalemler(), map[string]any{"count": "1"})
	assert.Len(t, out, 1)
}

func TestFilterPiyasaSifirCountSinirsiz(t *testing.T) {
	out := filterPiyasa(ornekKalemler(), map[string]any{"count": float64(0)})
	assert.Len(t, out, 3, "0 = tümü")
}

func TestFilterPiyasaBos(t *testing.T) {
	assert.Nil(t, filterPiyasa(nil, nil))
}

// ---- Şablon çıktısı ----

func renderKurBorsa(t *testing.T, kalemler []piyasa.Kalem) string {
	t.Helper()
	return renderKbStil(t, kalemler, "", "", "")
}

func renderKbStil(t *testing.T, kalemler []piyasa.Kalem, stil, ok, bg string) string {
	t.Helper()
	// NewServer'ın eklediği yardımcılar (mediaURL vb.) testte yer tutucuyla
	// tanımlanır; bu şablon yalnız ana blok yollarını gerektirir.
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	tmpl, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)
	s := &Server{tmpl: tmpl}
	b := &homeBlock{
		Type:         "kur_borsa",
		D:            &tmplData{LangBase: ""},
		Piyasa:       kalemler,
		PiyasaGuncel: time.Date(2026, 10, 28, 7, 51, 0, 0, time.UTC),
		PiyasaStil:   stil,
		PiyasaOk:     ok,
		PiyasaBg:     bg,
	}
	var buf bytes.Buffer
	require.NoError(t, s.tmpl.ExecuteTemplate(&buf, "blk_kur_borsa", b))
	return buf.String()
}

func TestKurBorsaSablonuKalemleriBasar(t *testing.T) {
	h := renderKurBorsa(t, ornekKalemler())

	assert.Contains(t, h, `class="pys-strip`)
	assert.Contains(t, h, "pys-style-referans", "varsayılan şablon referans olmalı")
	assert.Contains(t, h, "pys-arrow-noktali", "varsayılan gösterge noktalı ok olmalı")
	assert.Contains(t, h, "pys-bg-yok", "varsayılan arka plan yok (saydam)")
	assert.Contains(t, h, "pys-scroll")
	// Türkçe biçim: binlik nokta + ondalık virgül
	assert.Contains(t, h, "11.151,03")
	assert.Contains(t, h, "49,8335")
	assert.Contains(t, h, "10.558,61")
	// Yön sınıfları
	assert.Contains(t, h, "pys-down")
	assert.Contains(t, h, "pys-up")
	// Son güncelleme + yasal uyarı
	assert.Contains(t, h, "Son güncelleme")
	assert.Contains(t, h, "yatırım tavsiyesi değildir")
}

func TestKurBorsaSablonuBosVerideBasilmaz(t *testing.T) {
	h := renderKurBorsa(t, nil)
	assert.Empty(t, h, "veri yoksa modül hiç basılmamalı (kırık 0,00 yerine)")
}

func TestKurBorsaSablonuDegisimIsareti(t *testing.T) {
	h := renderKurBorsa(t, ornekKalemler())
	// Artı işareti HTML varlığı olarak çıkabilir (&#43;); ikisi de kabul.
	assert.True(t,
		bytes.Contains([]byte(h), []byte("+0,05")) || bytes.Contains([]byte(h), []byte("&#43;0,05")),
		"artı işaretli değişim beklenirdi: %s", h)
	assert.Contains(t, h, "-0,01")
	// Yatay kalemde ok başı yok: yalnız kesikli dikey çizgi (referans
	// icon-graphic-fixed davranışı).
	assert.Contains(t, h, "pys-ico-flat")
	assert.Contains(t, h, `<rect x="8.13" y="6" width="2.115" height="3.172"/>`)
}

// Referansın yapısal özellikleri: ok metnin SAĞINDA, çerçeve yok.
func TestKurBorsaReferansYapisi(t *testing.T) {
	h := renderKurBorsa(t, ornekKalemler())

	// Sol sütun + sağda ikon: ikon, fiyat sütunundan SONRA gelmeli.
	colIdx := bytes.Index([]byte(h), []byte(`class="pys-col"`))
	icoIdx := bytes.Index([]byte(h), []byte(`class="pys-ico`))
	require.True(t, colIdx > 0 && icoIdx > colIdx,
		"ok metnin sağında olmalı (col=%d ico=%d)", colIdx, icoIdx)

	// Noktalı kuyruk: referans SVG'sindeki kesikli rect'ler bulunmalı.
	assert.Contains(t, h, `<rect x="15.42" y="35.768" width="2.109" height="3.164"/>`)
	// Çerçeve yok: pys-scroll üzerinde inline border yok.
	assert.NotContains(t, h, "border:1px solid", "referansta çerçeve yok")
}

func TestKurBorsaArkaPlanSecenekleri(t *testing.T) {
	// yok → hiçbir stil basılmaz (saydam)
	h := renderKbStil(t, ornekKalemler(), "referans", "noktali", "yok")
	assert.Contains(t, h, "pys-bg-yok")
	assert.NotContains(t, h, `style="background`, "saydam seçimde arka plan basılmamalı")

	// zemin → yumuşak zemin
	h = renderKbStil(t, ornekKalemler(), "referans", "noktali", "zemin")
	assert.Contains(t, h, "pys-bg-zemin")
}

func TestKbArkaPlanCSS(t *testing.T) {
	assert.Equal(t, "", kbArkaPlanCSS("yok", ""), "yok = saydam")
	assert.Equal(t, "", kbArkaPlanCSS("yok", "#ff0000"), "yok renk verilse de saydam")
	assert.Contains(t, kbArkaPlanCSS("kart", ""), "var(--hp-surface")
	assert.Contains(t, kbArkaPlanCSS("zemin", ""), "var(--hp-surface-2")
	assert.Contains(t, kbArkaPlanCSS("ozel", "#123456"), "background:#123456")
	// Yarıçap tema değişkeninden gelir: köşeli temada modül de köşeli olur.
	assert.Contains(t, kbArkaPlanCSS("zemin", ""), "border-radius:var(--hp-radius")
	assert.Contains(t, kbArkaPlanCSS("ozel", "#123456"), "border-radius:var(--hp-radius")
	// Renk seçilmeden "özel" seçilirse saydam kalır
	assert.Equal(t, "", kbArkaPlanCSS("ozel", ""))
	// Geçersiz renk sessizce yok sayılır
	assert.Equal(t, "", kbArkaPlanCSS("ozel", "javascript:alert(1)"))
}

func TestKbBgAyarOkuma(t *testing.T) {
	assert.Equal(t, "yok", kbBgOf(nil))
	assert.Equal(t, "zemin", kbBgOf(map[string]any{"kb_bg": "zemin"}))
	assert.Equal(t, "yok", kbBgOf(map[string]any{"kb_bg": "bilinmeyen"}))
}

// Ortak "Arka plan rengi" (stBg) alanı finans modüllerinde modülün kendi
// arka plan seçimiyle yönetilir; sarmalayıcıya asla basılmaz.
//
// Regresyon: kb_bg=yok seçilse bile stBg basılıyordu; kullanıcı "saydam"
// istediği hâlde kutulu görünüyordu. Artık özel renk bölümün kendi inline
// stiline (kbArkaPlanCSS) gider, sarmalayıcıya değil.
func TestFinansBgOrtakRengiSusturur(t *testing.T) {
	for _, tip := range []string{"kur_borsa", "hisseler", "kripto_para", "kur_cevirici", "servis_bandi"} {
		for _, bg := range []string{"kart", "yok", "zemin", "ozel"} {
			b := homeBlock{Type: tip, StBg: "#102030", Opts: map[string]any{"kb_bg": bg}}
			b.buildStyle()
			assert.NotContains(t, string(b.Style), "#102030",
				"%s/%s: ortak renk sarmalayıcıya basılmamalı", tip, bg)
		}
	}
	// Özel renk bölüm stilinde yaşamaya devam eder.
	assert.Contains(t, kbArkaPlanCSS("ozel", "#102030"), "background:#102030")
}

// kb_bg ayarı hiç yoksa (eski düzen) saydam kabul edilir.
func TestKbBgAyarYoksaSaydam(t *testing.T) {
	b := homeBlock{Type: "kur_borsa", StBg: "#102030"}
	b.buildStyle()
	assert.NotContains(t, string(b.Style), "#102030",
		"ayar yoksa varsayılan saydam olmalı")
}

func TestBuilderPaletindeKurBorsaVar(t *testing.T) {
	// Her iki palet de (ana sayfa + yazı) modülü içermeli: kullanıcı "her
	// yerde" kullanabilsin.
	for _, page := range []string{"home", "article"} {
		var bulundu bool
		for _, tp := range builderTypesFor(page) {
			if tp.Type == "kur_borsa" {
				bulundu = true
				assert.Equal(t, "Kur & Borsa", tp.Name)
				assert.NotEmpty(t, tp.Desc)
			}
		}
		assert.True(t, bulundu, "%s paletinde kur_borsa olmalı", page)
	}
}
