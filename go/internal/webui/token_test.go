package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cssMetni(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "admin-pano.css"))
	require.NoError(t, err)
	return string(b)
}

func tokenDegeri(t *testing.T, ad string) string {
	t.Helper()
	s := cssMetni(t)
	i := strings.Index(s, ad+":")
	require.Greaterf(t, i, 0, "token bulunamadı: %s", ad)
	bas := i + len(ad) + 1
	son := strings.IndexAny(s[bas:], ";}")
	require.Greater(t, son, 0)
	return strings.TrimSpace(s[bas : bas+son])
}

// blokBul `seçici {` ile başlayan kuralın gövdesini döndürür.
func blokBul(t *testing.T, secici string) string {
	t.Helper()
	s := cssMetni(t)
	i := strings.Index(s, "\n"+secici+" {")
	require.Greaterf(t, i, 0, "kural bulunamadı: %s", secici)
	bas := i + len(secici) + 2
	son := strings.Index(s[bas:], "}")
	require.Greater(t, son, 0)
	return s[bas : bas+son]
}

func TestTasarimTokenlariPacelyIleAyni(t *testing.T) {
	// Ölçülen değerler (Pacely'nin :root bloğundan).
	beklenen := map[string]string{
		"--sh-bg":         "#f3f2ef",
		"--sh-fg":         "#1b1a18",
		"--sh-card":       "#fdfdfb",
		"--sh-muted":      "#f6f5f2",
		"--sh-muted-fg":   "#67635b",
		"--sh-sidebar":    "#e7e4dd",
		"--fs-display":    "34px",
		"--fs-3xl":        "22px",
		"--fs-2xl":        "20px",
		"--fs-xl":         "17px",
		"--fs-lg":         "15px",
		"--fs-base":       "14px",
		"--fs-md":         "13.5px",
		"--fs-sm":         "12.5px",
		"--fs-xs":         "12px",
		"--fs-2xs":        "11px",
		"--r-sm":          "10px",
		"--r-md":          "14px",
		"--r-lg":          "18px",
		"--r-xl":          "24px",
		"--pn-t1":         "#f5e0d5",
		"--pn-t2":         "#e1e9da",
		"--pn-t3":         "#f1e6cd",
		"--pn-t4":         "#efdfe3",
		"--pn-side-ink":   "#1b1a18",
		"--pn-side-muted": "#5f5a51",
		"--pn-card-3":     "#e9e7e2",
	}
	for ad, istenen := range beklenen {
		assert.Equalf(t, istenen, tokenDegeri(t, ad), "token kaymış: %s", ad)
	}
}

// TestGovdeTipografisiPacelyIleAyni: Pacely `font: 500 14px/1.45`.
func TestGovdeTipografisiPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, "body")
	assert.Contains(t, blok, "font-weight: 500",
		"Pacely gövdesi 500 ağırlık; 400 Urbanist'te ince kalıyordu")
	assert.Contains(t, blok, "line-height: 1.45")
	assert.Contains(t, blok, "tabular-nums",
		"sayı sütunları hizalansın (KPI, tablo, para)")
}

// TestKartPacelyIleAyni: Kart r-lg (18px) ve iç 1px çizgi taşımalı.
func TestKartPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".card")
	assert.Contains(t, blok, "border-radius: var(--r-lg)",
		"kart yarıçapı r-lg olmalı (14px bir basamak küçüktü)")
	assert.Contains(t, blok, "inset 0 0 0 1px var(--border)",
		"kart iç 1px çizgi taşımalı; yoksa kenarlar yumuşak kalıyor")
}

// TestUstCubukPacelyIleAyni: düz zemin, 13/28/11 boşluk.
func TestUstCubukPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".admin-top")
	assert.Contains(t, blok, "padding: 13px 28px 11px")
	assert.Contains(t, blok, "background: var(--bg)",
		"üst çubuk düz zemin kullanmalı (yarı saydam + blur değil)")
	assert.NotContains(t, blok, "backdrop-filter")
}

// TestRozetPacelyIleAyni: Pacely .badge = 20px, card-3 zemin, 0 6px.
func TestRozetPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".badge")
	assert.Contains(t, blok, "height: 20px")
	assert.Contains(t, blok, "padding: 0 6px")
	assert.Contains(t, blok, "background: var(--pn-card-3)",
		"rozet zemini üçüncü yüzey olmalı; surface-2 çok soluk kalıyordu")
	assert.Contains(t, blok, "color: var(--text-2)",
		"rozet metni tam siyah değil, ikincil metin rengi olmalı")
}

// TestYanMenuPacelyIleAyni: 44px, r12, 600, tek çizgi.
func TestYanMenuPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".admin-side .menu a")
	assert.Contains(t, blok, "height: 44px")
	assert.Contains(t, blok, "border-radius: 12px")
	assert.Contains(t, blok, "font-weight: 600",
		"menü etiketi gövdeyle aynı ağırlıkta kalırsa içerikten ayrışmıyor")

	// Kenarlık tek olmalı: border + inset gölge birlikte çizgiyi iki kat
	// koyulaştırıyor ve 1px yer kaplıyordu.
	yan := blokBul(t, ".admin-side")
	assert.NotContains(t, yan, "border-right",
		"yan menüde border-right olmamalı; Pacely yalnız inset gölge kullanıyor")
	assert.Contains(t, yan, "inset -1px 0 0 var(--pn-side-line)")
}

// TestDugmeOlculeriPacelyIleAyni: taban 0 16px, küçük 0 12px.
func TestDugmeOlculeriPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".btn")
	assert.Contains(t, blok, "padding: 0 16px")
	assert.Contains(t, blok, "height: 40px")

	s := cssMetni(t)
	i := strings.Index(s, ".btn.small {")
	require.Greater(t, i, 0)
	son := strings.Index(s[i:], "}")
	kucuk := s[i : i+son]
	assert.Contains(t, kucuk, "padding: 0 12px")
	assert.Contains(t, kucuk, "height: 32px")
}

// TestSabitleDugmesiEmojiKullanmaz: emoji <a> içinde metne giriyordu.
func TestSabitleDugmesiEmojiKullanmaz(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	assert.NotContains(t, s, "📌", "sabitle düğmesi emoji kullanmamalı")
	assert.NotContains(t, s, "📍", "sabitle düğmesi emoji kullanmamalı")
	assert.Contains(t, s, "ti-pin", "sabitle simgesi yerel sprite'tan gelmeli")
	assert.Contains(t, s, "aria-label", "düğmenin erişilebilir adı olmalı")
}

func TestFormAlanlariPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, ".input, .select, .textarea")
	assert.Contains(t, blok, "height: 42px")
	assert.Contains(t, blok, "border-radius: 12px")
	assert.Contains(t, blok, "box-shadow: inset 0 0 0 1px var(--pn-line-2)")
	assert.Contains(t, blok, "padding: 0 13px")
	assert.Contains(t, blok, "font-size: var(--fs-base)")
	assert.NotContains(t, blok, "border: 1px solid",
		"kenarlık yerine iç çizgi kullanılmalı; border kutu ölçüsüne giriyor")

	odak := blokBul(t, ".input:focus, .select:focus, .textarea:focus")
	assert.Contains(t, odak, "inset 0 0 0 1.5px var(--accent",
		"odak halkası iç çizgi olmalı; dış çizgi alanı büyütüyordu")
}

func TestSelectOkuPacelyIleAyni(t *testing.T) {
	blok := blokBul(t, "select.input, select.select")
	assert.Contains(t, blok, "appearance: none")
	assert.Contains(t, blok, "padding-right: 34px")
	assert.Contains(t, blok, "right 17px center, right 12px center")
	assert.Contains(t, blok, "background-size: 5px 5px")
}

func TestFiltreIzgarasiHizali(t *testing.T) {
	blok := blokBul(t, ".filtre-grid")
	assert.Contains(t, blok, "grid-template-columns: repeat(6, minmax(0, 1fr))")
	assert.Contains(t, blok, "gap: 10px")

	b, err := os.ReadFile(filepath.Join("templkit", "icerikler.templ"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `class="filtre-grid"`)

	i := strings.Index(s, `class="filtre-grid"`)
	require.Greater(t, i, 0)
	j := strings.Index(s[i:], "</form>")
	require.Greater(t, j, 0)
	filtre := s[i : i+j]

	assert.NotContains(t, filtre, `style="max-width:`,
		"filtre alanlarında satır içi genişlik kalmamalı; sütunlar eşit olmalı")
	assert.Contains(t, filtre, `class="input fq-2"`,
		"arama ve sıralama iki sütun kaplamalı")
}

func TestFiltreEtiketlerindeEmojiYok(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "icerikler.templ"))
	require.NoError(t, err)
	s := string(b)

	for _, e := range []string{"🎙", "⏳", "⚠", "🔥", "🕒", "✅", "🕗", "🔤"} {
		assert.NotContainsf(t, s, e, "filtre etiketinde emoji kalmamalı: %s", e)
	}
}

func TestYikiciDugmeDoluKirmiziDegil(t *testing.T) {
	blok := blokBul(t, ".btn.danger")
	assert.Contains(t, blok, "background: var(--card)")
	assert.Contains(t, blok, "color: var(--pn-bad)",
		"yıkıcı metin token'a bağlı olmalı; sabit #b3261e karanlık temada okunmuyor")
	assert.NotContains(t, blok, "background: #b3261e",
		"yıkıcı düğme dolu kırmızı olmamalı")

	// Pacely --bad: açık #b3261e, karanlık #f2877b. İkisi de tanımlı olmalı.
	s := cssMetni(t)
	assert.Equal(t, 2, strings.Count(s, "--pn-bad:"),
		"--pn-bad hem açık hem karanlık temada tanımlanmalı")
	assert.Contains(t, s, "--pn-bad: #b3261e")
	assert.Contains(t, s, "--pn-bad: #f2877b")

	hayalet := blokBul(t, ".btn.ghost, .btn.secondary")
	assert.Contains(t, hayalet, "inset 0 0 0 1px var(--pn-line-2)",
		"hayalet düğme beyaz kart üstünde çerçevesiz düz metin gibi okunuyordu")
}

func TestBlokTutamaciKonumlanir(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "icerikform.templ"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, ".block-editor{position:relative}",
		"tutamaç editöre göre konumlanmalı, sayfaya göre değil")
	assert.Contains(t, s, ".block-editor:hover .be-drag",
		"tutamaç boşta gezerken görünmemeli")
}
