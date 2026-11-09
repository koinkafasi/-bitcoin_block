package httpapi

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func hisseOrnek() []repo.HisseRow {
	return []repo.HisseRow{
		{Code: "THYAO", Name: "Türk Hava Yolları", Value: 252.5, Delta: -0.198},
		{Code: "AKBNK", Name: "Akbank", Value: 63, Delta: 2.4},
		{Code: "AEFES", Name: "Anadolu Efes", Value: 15.9, Delta: -2.214},
		{Code: "ASELS", Name: "Aselsan", Value: 358.5, Delta: 0},
	}
}

func TestHisselerParseDefaults(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[{"id":"h1","type":"hisseler","opts":{"zone":"ana"}}]`))
	require.Len(t, blocks, 1)
	b := blocks[0]
	assert.Equal(t, hisseSayiVarsayilan, b.Count)
	assert.Equal(t, "serit", b.HisseYerlesim)
	assert.Equal(t, "referans", b.HisseStil, "Kur & Borsa ile aynı varsayılan şablon")
	assert.Equal(t, "noktali", b.HisseArrow, "Kur & Borsa ile aynı varsayılan gösterge")
	assert.Equal(t, "kart", b.HisseBg, "arka plan varsayılanı modül kartı (mevcut görünüm)")
	assert.Equal(t, "kod", b.HisseSirala)
	assert.Equal(t, hisseHizVarsayilan, b.HisseHiz)
	assert.False(t, b.HisseDegisim, "yüzde değişimi varsayılan kapalı (referans)")
}

func TestHisselerParseClampAndInvalid(t *testing.T) {
	blocks := parseHomeBlocks([]byte(`[
		{"id":"h1","type":"hisseler","opts":{"count":500,"hs_hiz":1,"hs_style":"gecersiz","hs_sirala":"gecersiz"}},
		{"id":"h2","type":"hisseler","opts":{"count":1,"hs_hiz":999,"hs_style":"izgara","hs_sirala":"yukselen","hs_degisim":true,"hs_ok":false}}
	]`))
	require.Len(t, blocks, 2)
	assert.Equal(t, hisseSayiAzami, blocks[0].Count, "üst sınır 40")
	// Eski hs_hiz=1 (sn) → px/sn'ye çevrilir (40*120/1) ve üst sınıra sıkışır.
	assert.Equal(t, hisseHizEnCok, blocks[0].HisseHiz, "en hızlı üst sınırı")
	assert.Equal(t, "serit", blocks[0].HisseYerlesim, "geçersiz yerleşim varsayılana döner")
	assert.Equal(t, "referans", blocks[0].HisseStil, "geçersiz şablon varsayılana döner")
	assert.Equal(t, "kod", blocks[0].HisseSirala, "geçersiz sıralama varsayılana döner")

	assert.Equal(t, 3, blocks[1].Count, "alt sınır 3")
	// Eski hs_hiz=999 sn → hesap 0'a yuvarlanır; varsayılan hız kullanılır.
	assert.Equal(t, hisseHizVarsayilan, blocks[1].HisseHiz, "anlamsız eski değer varsayılana döner")
	// Eski sürüm yerleşimi hs_style'da tutuyordu: "izgara" yerleşim sayılır.
	assert.Equal(t, "izgara", blocks[1].HisseYerlesim, "eski hs_style=izgara yerleşime taşınır")
	assert.Equal(t, "referans", blocks[1].HisseStil)
	assert.Equal(t, "yukselen", blocks[1].HisseSirala)
	assert.True(t, blocks[1].HisseDegisim)
	// Eski bool hs_ok=false göstergeyi kapatır.
	assert.Equal(t, "yok", blocks[1].HisseArrow, "eski hs_ok=false → gösterge yok")

	// Yeni sözlük: Kur & Borsa ile birebir aynı anahtarlar.
	yeni := parseHomeBlocks([]byte(`[{"id":"h3","type":"hisseler","opts":{"hs_layout":"izgara","hs_style":"kart","hs_arrow":"ucgen","hs_bg":"zemin"}}]`))
	require.Len(t, yeni, 1)
	assert.Equal(t, "izgara", yeni[0].HisseYerlesim)
	assert.Equal(t, "kart", yeni[0].HisseStil)
	assert.Equal(t, "ucgen", yeni[0].HisseArrow)
	assert.Equal(t, "zemin", yeni[0].HisseBg)
}

func TestHisseRowsSiralamaVeKirpma(t *testing.T) {
	rows := hisseOrnek()

	kod := hisseRows(rows, "kod", 0)
	require.Len(t, kod, 4)
	assert.Equal(t, []string{"AEFES", "AKBNK", "ASELS", "THYAO"}, hisseKodlar(kod))

	yuk := hisseRows(rows, "yukselen", 2)
	require.Len(t, yuk, 2)
	assert.Equal(t, "AKBNK", yuk[0].Code, "en çok yükselen başta")
	assert.Equal(t, "ASELS", yuk[1].Code)

	dus := hisseRows(rows, "dusen", 2)
	require.Len(t, dus, 2)
	assert.Equal(t, "AEFES", dus[0].Code, "en çok düşen başta")
	assert.Equal(t, "THYAO", dus[1].Code)

	// Girdi dilimi değişmemeli: aynı istekte iki modül farklı sıralar.
	assert.Equal(t, "THYAO", rows[0].Code, "kaynak dilim sıralanmamalı")
	assert.Len(t, hisseRows(nil, "kod", 5), 0)
}

func hisseKodlar(rows []repo.HisseRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Code)
	}
	return out
}

func hisselerBlogu(rows []repo.HisseRow, yerlesim string) *homeBlock {
	return &homeBlock{
		Type:          "hisseler",
		Hisseler:      rows,
		HisseYerlesim: yerlesim,
		HisseStil:     "referans",
		HisseArrow:    "noktali",
		HisseBg:       "yok",
		HisseHiz:      60,
		HisseGecis:    "surekli",
		HisseBekle:    4,
		HisseKayma:    700,
		HisseSure:     40,
		HisseAra:      true,
		HisseCSS:      " --hss-dur:40s",
		HisseGuncel:   time.Date(2026, 10, 28, 15, 4, 0, 0, time.UTC),
		HideHeading:   false,
		HeadingStyle:  "line",
		D:             &tmplData{LangBase: ""},
	}
}

func TestHisselerSeritRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := hisselerBlogu(hisseOrnek(), "serit")
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "hss-block hss-serit hss-style-referans hss-arrow-noktali hss-bg-yok")
	assert.Contains(t, out, "--hss-dur:40s")
	assert.Contains(t, out, "HİSSELER")
	assert.Contains(t, out, `href="/borsa/hisseler"`, "TÜMÜ tüm hisseler sayfasına")
	assert.Contains(t, out, `href="/borsa/hisseler?q=THYAO"`, "hisse bağlantısı aramaya gider")
	assert.Contains(t, out, ">TÜMÜ<")
	assert.Contains(t, out, ">THYAO<")
	assert.Contains(t, out, "252,50", "fiyat Türkçe biçimde (virgüllü)")
	assert.Contains(t, out, "15,90")
	assert.Contains(t, out, "hss-item hss-up", "yükselen hisse yön sınıfı")
	assert.Contains(t, out, "hss-item hss-down", "düşen hisse yön sınıfı")
	assert.Contains(t, out, "hss-item hss-flat", "değişmeyen hisse yön sınıfı")
	assert.Contains(t, out, "hss-ico", "yön göstergesi (noktalı ok) basılmalı")
	assert.Contains(t, out, "hss-ico-down", "düşen hissede ok 180° döner")
	assert.Contains(t, out, "hss-ico-noktali", "referans gösterge kesikli kuyruklu ok")
	assert.Contains(t, out, `title="Türk Hava Yolları"`, "tam şirket adı tooltip'te")

	// Kayan şerit: liste iki kez basılır, kopya ekran okuyucudan gizlenir.
	assert.Equal(t, 2, strings.Count(out, `class="hss-group"`))
	assert.Equal(t, 1, strings.Count(out, `class="hss-group" aria-hidden="true"`))
	// Varsayılan olarak yüzde değişimi basılmaz (referans yalnız fiyat + ok).
	assert.NotContains(t, out, "hss-chg")
	assert.Contains(t, out, "yatırım tavsiyesi değildir")
}

func TestHisselerIzgaraVeDegisim(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := hisselerBlogu(hisseOrnek(), "izgara")
	b.HisseDegisim = true
	b.HisseArrow = "yok"
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "hss-block hss-izgara")
	assert.Equal(t, 1, strings.Count(out, `class="hss-group"`), "ızgarada kopya grup basılmaz")
	assert.NotContains(t, out, `class="hss-group" aria-hidden="true"`)
	assert.Contains(t, out, "hss-chg", "değişim açıksa yüzde basılır")
	assert.Contains(t, out, "2,40")
	assert.Contains(t, out, "-0,20")
	assert.NotContains(t, out, `class="hss-ico`, "gösterge yok seçilince ikon basılmaz")
}

func TestHisselerBosVerideBasilmaz(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := hisselerBlogu(nil, "serit")
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	assert.Empty(t, strings.TrimSpace(buf.String()), "veri yoksa modül hiç basılmamalı")
}

func TestHisselerPalette(t *testing.T) {
	for _, page := range []string{"home", "kategori", "article"} {
		assert.Contains(t, typeKeys(builderTypesFor(page)), "hisseler", "%s paletinde hisseler olmalı", page)
	}
}

func TestHisseHizOf(t *testing.T) {
	// Yeni ayar doğrudan px/sn.
	assert.Equal(t, 90, hisseHizOf(90, 0, 20))
	// Sınırlar.
	assert.Equal(t, hisseHizEnAz, hisseHizOf(1, 0, 20))
	assert.Equal(t, hisseHizEnCok, hisseHizOf(9999, 0, 20))
	// Ayar yoksa varsayılan.
	assert.Equal(t, hisseHizVarsayilan, hisseHizOf(0, 0, 20))
	// Eski düzen: hs_hiz = tur süresi (sn) → px/sn'ye çevrilir.
	assert.Equal(t, 20*120/40, hisseHizOf(0, 40, 20))
	// Eski değer sınırlara sıkışır.
	assert.Equal(t, hisseHizEnAz, hisseHizOf(0, 600, 20))
}

func TestHisseSureTur(t *testing.T) {
	// Süre = kalem*sabir / hız → hız kalem sayısından bağımsız sabit kalır.
	assert.Equal(t, (20*hisseItemPx)/60, hisseSureTur(20, 60))
	assert.Equal(t, (40*hisseItemPx)/120, hisseSureTur(40, 120))
	// Alt/üst sınırlar: çok kısa tur göz kırpmasına, çok uzun tur donmaya dönüşmesin.
	assert.Equal(t, 6, hisseSureTur(3, 300))
	assert.Equal(t, 480, hisseSureTur(40, 10), "40*120/10 = 480 sn (üst sınırın altında)")
}

func TestHisseAramaNorm(t *testing.T) {
	for _, c := range []struct{ in, out string }{
		{"ŞİŞE", "sise"},
		{"Türk Hava Yolları", "turk hava yollari"},
		{"  AEFES  ", "aefes"},
		{"ÇİMSA", "cimsa"},
		{"İş Bankası", "is bankasi"},
	} {
		assert.Equal(t, c.out, hisseAramaNorm(c.in), "girdi: %q", c.in)
	}
}

func TestHisseSayfaOlustur(t *testing.T) {
	hd := repo.HisseData{
		Rows: []repo.HisseRow{
			{Code: "THYAO", Name: "Türk Hava Yolları", Value: 252.5, Delta: -0.198},
			{Code: "AEFES", Name: "ANADOLU EFES", Value: 15.9, Delta: -2.214},
		},
		Updated: time.Date(2026, 10, 29, 9, 30, 0, 0, time.UTC),
	}

	hepsi := hisseSayfaOlustur(hd, "", "")
	assert.Equal(t, len(repo.HisseEvreni()), hepsi.Toplam)
	assert.Equal(t, hepsi.Toplam, hepsi.Adet)
	assert.NotEmpty(t, hepsi.Harfler)

	// Koda göre arama (küçük harf yazımı da bulur).
	kod := hisseSayfaOlustur(hd, "thyao", "")
	require.Len(t, kod.Satir, 1)
	assert.Equal(t, "THYAO", kod.Satir[0].Code)
	assert.True(t, kod.Satir[0].Var, "fiyat verisi varsa Var=true")
	assert.InDelta(t, 252.5, kod.Satir[0].Value, 0.001)

	// Ada göre Türkçe duyarsız arama.
	ad := hisseSayfaOlustur(hd, "türk hava", "")
	require.Len(t, ad.Satir, 1)
	assert.Equal(t, "THYAO", ad.Satir[0].Code)

	// Harf filtresi.
	harfA := hisseSayfaOlustur(hd, "", "A")
	require.NotEmpty(t, harfA.Satir)
	for _, r := range harfA.Satir {
		assert.Equal(t, "A", r.Ilk, "%s A ile başlamalı", r.Code)
	}

	// Veri olmayan sembol de listelenir (fiyatı yok).
	eksik := hisseSayfaOlustur(repo.HisseData{}, "THYAO", "")
	require.Len(t, eksik.Satir, 1)
	assert.False(t, eksik.Satir[0].Var, "fiyat yoksa Var=false (— basılır)")

	// BAĞLANTI SÖZLEŞMESİ: fiyatı olan satır KENDİ detay sayfasına gider
	// (".IS" ekli — Yahoo BIST sembolü), fiyatı olmayan arama linkinde
	// kalır (detay ucu veri bulamayınca 404 olurdu).
	assert.Equal(t, "/price/stock/THYAO.IS", kod.Satir[0].Href)
	assert.Equal(t, "/borsa/hisseler?q=THYAO", eksik.Satir[0].Href)
	for _, r := range hepsi.Satir {
		assert.NotEmpty(t, r.Href, "%s satırında bağlantı boş", r.Code)
	}

	// Sonuç yok.
	assert.Empty(t, hisseSayfaOlustur(hd, "zzzzz", "").Satir)
}

func TestHisselerModulAramaVeGecis(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := hisselerBlogu(hisseOrnek(), "serit")
	b.HisseGecis = "adim"
	b.HisseBekle = 3
	b.HisseKayma = 500
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, `data-hss`)
	assert.Contains(t, out, `data-model="adim"`)
	assert.Contains(t, out, `data-speed="60"`)
	assert.Contains(t, out, `data-hold="3"`)
	assert.Contains(t, out, `data-glide="500"`)
	assert.Contains(t, out, `class="hss-search"`, "modül altında hisse arama kutusu")
	assert.Contains(t, out, `action="/borsa/hisseler"`)
	assert.Contains(t, out, `href="/borsa/hisseler"`, "TÜMÜ tüm hisseler sayfasına gitmeli")

	// Arama kutusu kapatılabilir.
	b.HisseAra = false
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	assert.NotContains(t, buf.String(), `class="hss-search"`)
}
