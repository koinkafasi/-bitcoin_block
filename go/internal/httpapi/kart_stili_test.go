package httpapi

import (
	"bytes"
	"html/template"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

// ---- Paylaşımlı kart stili (Bloomberg ve alternatifleri) ----

func kartStiliTestVerisi() *categoryData {
	return &categoryData{
		Title: "Gündem", Slug: "gundem", View: "klasik",
		HeroStyle: "klasik", HeroCount: 5, ListCols: 2,
		Items: []repo.Content{
			{ID: 1, Title: "Birinci haber", Spot: "spot 1", CategoryName: ptrStr("Gündem")},
			{ID: 2, Title: "İkinci haber", Spot: "spot 2", CategoryName: ptrStr("Gündem")},
		},
	}
}

func ptrStr(s string) *string { return &s }

func renderCatListe(t *testing.T, c *categoryData) string {
	t.Helper()
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.Category = c
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "cat_liste", d))
	return buf.String()
}

func TestKartStiliBloombergSiniflariVeRenkleri(t *testing.T) {
	c := kartStiliTestVerisi()
	c.CardClass = "cs-scope cs-bloomberg"
	c.CardCSS = template.CSS("--cs-light-bg:#0f172a;--cs-dark-bg:#f1f5f9;--cs-accent:#f59e0b;")
	out := renderCatListe(t, c)

	assert.Contains(t, out, "cat-cards", "kart ızgarası basılmalı")
	assert.Contains(t, out, "cs-scope cs-bloomberg", "Bloomberg sınıfı uygulanmalı")
	assert.Contains(t, out, "--cs-light-bg:#0f172a", "açık tema band rengi")
	assert.Contains(t, out, "--cs-dark-bg:#f1f5f9", "koyu tema band rengi")
	assert.Contains(t, out, "--cs-accent:#f59e0b", "vurgu rengi")
	// Başlıklar ve metinler korunur (SEO/erişilebilirlik).
	assert.Contains(t, out, "Birinci haber")
	assert.Contains(t, out, "İkinci haber")
}

func TestKartStiliYokkenKlasikKalir(t *testing.T) {
	out := renderCatListe(t, kartStiliTestVerisi())
	assert.NotContains(t, out, "cs-scope", "stil seçilmezse sınıf eklenmemeli")
	assert.NotContains(t, out, "--cs-light-bg", "stil seçilmezse renk değişkeni basılmamalı")
	assert.Contains(t, out, "cat-cards")
}

func TestTumKartStilleriSinifUretir(t *testing.T) {
	stiller := []string{
		"bloomberg", "bloomberg-cizgi", "bant-alt", "vurgu",
		"cerceve", "kose", "cam", "alt-cizgi",
	}
	for _, st := range stiller {
		c := kartStiliTestVerisi()
		c.CardClass = "cs-scope cs-" + st
		out := renderCatListe(t, c)
		assert.Contains(t, out, "cs-scope cs-"+st, "stil sınıfı basılmalı: "+st)
	}
}

func TestUstMansetKartStili(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		D:    headerData(),
		Type: "ust_manset", CardStyle: "bloomberg",
		CardCSS:  template.CSS("--cs-dark-bg:#111827;"),
		UMLayout: "izgara", UMCols: 4, UMStyle: "overlay",
		ShowTitle: true,
		Items:     []repo.Content{{ID: 5, Title: "Manşet", CategoryName: ptrStr("Gündem")}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_ust_manset", b))
	out := buf.String()

	assert.Contains(t, out, "cs-scope cs-bloomberg")
	assert.Contains(t, out, "--cs-dark-bg:#111827")
	assert.Contains(t, out, "Manşet")
	// Kart stili seçiliyken overlay sınıfı kullanılmaz (bant üstte olmalı).
	assert.NotContains(t, out, "um-style-overlay")
	assert.NotContains(t, out, "um-card um-ov")
}

func TestUstMansetStilsizOverlayKorunur(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		D:    headerData(),
		Type: "ust_manset", UMLayout: "serit", UMStyle: "band", UMCols: 4,
		ShowTitle: true,
		Items:     []repo.Content{{ID: 6, Title: "Manşet 2"}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_ust_manset", b))
	out := buf.String()
	assert.Contains(t, out, "um-style-band")
	assert.Contains(t, out, "um-card um-ov")
	assert.NotContains(t, out, "cs-scope")
}

func TestOneCikanlarKartStili(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := &homeBlock{
		D:    headerData(),
		Type: "one_cikanlar", CardStyle: "cerceve",
		CardCSS:   template.CSS("--cs-accent:#d10015;"),
		ShowTitle: true,
		Items:     []repo.Content{{ID: 7, Title: "Öne çıkan", CategoryName: ptrStr("Gündem")}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_one_cikanlar", b))
	out := buf.String()
	assert.Contains(t, out, "one-cikanlar cs-scope cs-cerceve")
	assert.Contains(t, out, "--cs-accent:#d10015")
	assert.Contains(t, out, "Öne çıkan")
}

// publicCSS gömülü public.css içeriğini döndürür.
func publicCSS(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(webui.FS, "static/public.css")
	require.NoError(t, err)
	return string(b)
}

func TestKartStiliCSSDosyasiTumStilleriIcerir(t *testing.T) {
	css := publicCSS(t)
	for _, st := range []string{
		".cs-scope", ".cs-bloomberg", ".cs-bloomberg-cizgi", ".cs-bant-alt",
		".cs-vurgu", ".cs-cerceve", ".cs-kose", ".cs-cam", ".cs-alt-cizgi",
	} {
		assert.True(t, strings.Contains(css, st), "public.css stil içermeli: "+st)
	}
	// Dark mode band renkleri tema ile değişmeli.
	assert.Contains(t, css, `[data-theme="dark"] .cs-scope`)
	assert.Contains(t, css, "--cs-dark-bg")
	// Kart gövdesi (kabuk) rengi ve gölgesi ayrı değişkenlerden beslenir;
	// koyu tema ayrı değer kullanır ve gölge kapatılabilir.
	assert.Contains(t, css, "--cs-card-light-bg")
	assert.Contains(t, css, "--cs-card-dark-bg")
	assert.Contains(t, css, "--cs-card-bg: var(--cs-card-dark-bg)")
	assert.Contains(t, css, "--cs-shadow")
	assert.Contains(t, css, "--cs-shadow-hover")
	assert.Contains(t, css, ".cs-scope .card,")
	assert.Contains(t, css, "background: var(--cs-card-bg)")
	assert.Contains(t, css, "box-shadow: var(--cs-shadow)")
	// Bloomberg: başlık bandı görselin üstünde (order:-1).
	assert.Contains(t, css, "order: -1")
	// Bloomberg II: tip5'in taşan başlık kutusu tüm kartlara uygulanır.
	assert.Contains(t, css, ".cs-bloomberg-cizgi .ak-cap5")
	assert.Contains(t, css, "width: 91.6667%")
	assert.Contains(t, css, "margin: -48px 0 0")
	// Bloomberg II varsayılanı tema yüzeyi: koyu temada kategorideki tip5
	// kartıyla aynı renge geçer.
	assert.Contains(t, css, "--cs-light-bg: var(--hp-surface)")
	// Akış (ilk 4 karttan sonraki) kartları da aynı stil ailesini kullanmalı.
	for _, sec := range []string{
		".cs-scope .ak-fig",
		".cs-scope .ak-fig5",
		".cs-scope .ak-cap",
		".cs-scope .ak-cap5",
		".cs-scope .ak-img5",
		".cs-scope .ak-img5 img",
		".cs-bloomberg .ak-cap",
		".cs-bloomberg-cizgi .ak-cap",
		".cs-vurgu .ak-cap",
		".cs-cerceve .ak-cap",
		".cs-kose .ak-cap",
		".cs-cam .ak-cap",
		".cs-alt-cizgi .ak-cap",
	} {
		assert.Contains(t, css, sec, "akış kartı stili eksik: "+sec)
	}
}

// Akış görünümünde ilk 4 karttan sonraki ak-flow kabı da kart stilini
// almalı; AJAX ile gelen parçalar bu kabın altında render edilir.
func TestKartStiliAkisKartlarinaUygulanir(t *testing.T) {
	items := make([]repo.Content, 0, 8)
	for i := 1; i <= 8; i++ {
		items = append(items, repo.Content{
			ID: int64(i), Title: "Akış " + itoa(i), CategoryName: ptrStr("Gündem"),
		})
	}
	c := &categoryData{
		Title: "Gündem", Slug: "gundem", View: "akis",
		ListCols:  4,
		Items:     items,
		CardClass: "cs-scope cs-bloomberg",
		CardCSS:   template.CSS("--cs-light-bg:#0f172a;--cs-dark-bg:#e5e7eb;"),
	}
	out := renderCatListe(t, c)
	assert.Contains(t, out, "cat-cards cat-cols-4 ak-first cs-scope cs-bloomberg",
		"ilk 4 kart stili almalı")
	assert.Contains(t, out, "ak-flow cs-scope cs-bloomberg",
		"ilk 4'ten sonraki akış kabı da stili almalı")
	assert.Contains(t, out, "--cs-light-bg:#0f172a")
	assert.Contains(t, out, "--cs-dark-bg:#e5e7eb")
	assert.Contains(t, out, "Akış 4", "ilk 4 kart basılmalı")
}

// Akış kartının başlığı ve kategori etiketi DOM'da kalmalı (SEO/erişilebilirlik);
// bant yalnız görsel sıralamayı değiştirir.
func TestAkisKartiBasligiVeEtiketiKorunur(t *testing.T) {
	tmpl := parseTestTemplates(t)
	card := akisCard{
		C:        repo.Content{ID: 9, Title: "Akış manşeti", CategoryName: ptrStr("Gündem")},
		Type:     "type2",
		LangBase: "/tr",
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ak_card", card))
	out := buf.String()
	assert.Contains(t, out, "Akış manşeti")
	assert.Contains(t, out, "Gündem")
	assert.Contains(t, out, "ak-cap", "bant sınıfı korunmalı")
}

// tip5 (ak-fig5/ak-cap5) de aynı bant ailesine girer: görsel kabı ve başlık
// bandı sınıfları korunur; böylece .cs-scope kuralları bu kartta da çalışır.
func TestAkisTip5KartiBantYapisiniKorur(t *testing.T) {
	tmpl := parseTestTemplates(t)
	card := akisCard{
		C:        repo.Content{ID: 10, Title: "Tip5 manşeti"},
		Type:     "type5",
		LangBase: "/tr",
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ak_card", card))
	out := buf.String()
	assert.Contains(t, out, "ak-fig5")
	assert.Contains(t, out, "ak-cap5")
	assert.Contains(t, out, "ak-img5")
	assert.Contains(t, out, "Tip5 manşeti")
}

// Manşet (portre) sonrası alt kart ızgarası da kart stilini almalı:
// kullanıcının "4 karttan sonra gelen yapı" dediği bölüm burasıdır.
func TestKartStiliManşetSonrasiIzgara(t *testing.T) {
	items := make([]repo.Content, 0, 10)
	for i := 1; i <= 10; i++ {
		items = append(items, repo.Content{
			ID: int64(i), Title: "Haber " + itoa(i), CategoryName: ptrStr("Gündem"),
		})
	}
	c := &categoryData{
		Title: "Gündem", Slug: "gundem", View: "klasik",
		HeroStyle: "portre", HeroCount: 5, ListCols: 4,
		Items:     items,
		CardClass: "cs-scope cs-bloomberg",
		CardCSS:   template.CSS("--cs-light-bg:#111827;--cs-dark-bg:#e5e7eb;"),
	}
	out := renderCatListe(t, c)
	assert.Contains(t, out, "cat-hero", "portre manşet basılmalı")
	assert.Contains(t, out, "cat-cards cat-cols-4 cs-scope cs-bloomberg",
		"manşet sonrası ızgara kart stilini almalı")
	assert.Contains(t, out, "--cs-light-bg:#111827")
	assert.Contains(t, out, "Haber 10", "ızgara kartları basılmalı")
}
