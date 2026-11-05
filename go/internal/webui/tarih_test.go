package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tarih alanları etiket taşımalı (Faz 176).
//
// NEDEN: yerel `<input type="date">` ve `<input type="datetime-local">`
// tarayıcı tarafından çizilir ve `placeholder` niteliğini HİÇ göstermez
// (HTML spec: bu tiplerde desteklenmez). Bu alanlarda placeholder tek
// etiket olarak kullanılıyordu: 10 alanın anlamı ekranda hiç görünmüyor,
// hiçbirinin de erişilebilir adı yoktu.
//
// Ölçüldü (tarayıcı): /admin/reklamlar `baslangic`/`bitis` ve
// /admin/gazete `tarih` alanlarında aria-label, title ve görünür etiket
// yoktu; yalnızca çizilmeyen bir placeholder vardı.
//
// Yerel alan KENDİ biçim ipucunu çizer (gg.aa.yyyy / mm/dd/yyyy) — o yüzden
// alan boş görünmüyor. Eksik olan semantikti.

var tarihGirdisi = regexp.MustCompile(`(?i)<input\b[^>]*\btype="(date|datetime-local)"[^>]*>`)

// TestTarihAlanlariPlaceholderKullanmaz: yerel tarih alanında placeholder
// yazılımı ölü niteliğe güvendirmemeli.
func TestTarihAlanlariPlaceholderKullanmaz(t *testing.T) {
	dosyalar := templDosyalari(t)
	var suclu []string
	denetlenen := 0
	for _, f := range dosyalar {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		for _, m := range tarihGirdisi.FindAllStringIndex(s, -1) {
			etiket := s[m[0]:m[1]]
			denetlenen++
			if strings.Contains(etiket, "placeholder=") {
				satir := strings.Count(s[:m[0]], "\n") + 1
				suclu = append(suclu, filepath.Base(f)+":"+strconv.Itoa(satir))
			}
		}
	}
	assert.GreaterOrEqualf(t, denetlenen, 15,
		"beklenenden az tarih alanı bulundu (%d); ayrıştırıcı bozulmuş olabilir", denetlenen)
	assert.Emptyf(t, suclu,
		"yerel tarih alanında placeholder çizilmez; aria-label ya da <label> kullanın: %v", suclu)
}

// TestTarihAlanlariErisilebilirAd: her tarih alanının bir adı olmalı —
// aria-label, aria-labelledby, sarmalayan <label> ya da aynı kutudaki <label for>.
func TestTarihAlanlariErisilebilirAd(t *testing.T) {
	dosyalar := templDosyalari(t)
	var adsiz []string
	denetlenen := 0

	for _, f := range dosyalar {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		for _, m := range tarihGirdisi.FindAllStringIndex(s, -1) {
			etiket := s[m[0]:m[1]]
			denetlenen++
			if strings.Contains(etiket, "aria-label=") || strings.Contains(etiket, "aria-labelledby=") {
				continue
			}
			// Aynı satırdaki id'ye bağlı <label for="..."> var mı?
			idm := regexp.MustCompile(`\bid="([^"]+)"`).FindStringSubmatch(etiket)
			if len(idm) == 2 && strings.Contains(s, `for="`+idm[1]+`"`) {
				continue
			}
			// Sarmalayan <label> … <input …> … </label>: input'un hemen
			// önündeki metin bir <label> açılışı ve kapanışı taşıyorsa
			// gerçekten sarmalanmış olur. Kardeş <label> YETMEZ (for/id
			// olmadan ekran okuyucu ilişkilendiremez).
			bas := m[0] - 120
			if bas < 0 {
				bas = 0
			}
			onceki := s[bas:m[0]]
			if i := strings.LastIndex(onceki, "<label"); i >= 0 &&
				!strings.Contains(onceki[i:], "</label>") {
				continue
			}
			satir := strings.Count(s[:m[0]], "\n") + 1
			adsiz = append(adsiz, filepath.Base(f)+":"+strconv.Itoa(satir))
		}
	}

	assert.GreaterOrEqualf(t, denetlenen, 15,
		"beklenenden az tarih alanı bulundu (%d)", denetlenen)
	assert.Emptyf(t, adsiz, "tarih alanı erişilebilir ad taşımalı: %v", adsiz)
}

// templDosyalari: şablon ve templ dosyaları (üretilen *_templ.go hariç).
func templDosyalari(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, desen := range []string{"templkit/*.templ", "templates/*.html"} {
		yol, err := filepath.Glob(desen)
		require.NoError(t, err)
		for _, f := range yol {
			if strings.HasSuffix(f, "_templ.go") {
				continue
			}
			out = append(out, f)
		}
	}
	require.NotEmpty(t, out)
	return out
}
