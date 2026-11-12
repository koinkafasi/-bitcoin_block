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

var (
	labelFor  = regexp.MustCompile(`<label\b[^>]*\bfor="([^"]+)"`)
	literalID = regexp.MustCompile(`\bid="([^"{}]+)"`)
)

func dosyalar(t *testing.T) []string {
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

// TestLabelForHedefiVar: her `<label for="X">` için aynı dosyada id="X" olmalı.
// Kırık bağ, etikete tıklamayı sessizce işlevsiz bırakır.
func TestLabelForHedefiVar(t *testing.T) {
	var kirik []string
	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		for _, m := range labelFor.FindAllStringSubmatch(s, -1) {
			if strings.Contains(s, `id="`+m[1]+`"`) {
				continue
			}
			satir := strings.Count(s[:strings.Index(s, m[0])], "\n") + 1
			kirik = append(kirik, filepath.Base(f)+":"+strconv.Itoa(satir)+" -> "+m[1])
		}
	}
	assert.Emptyf(t, kirik, "label[for] var olmayan kimliğe işaret ediyor: %v", kirik)
}

// TestLiteralIdTekil: bir şablon dosyasında aynı literal id birden fazla
// geçmemeli. Döngü içinde literal id kullanmanın en yaygın sonucu budur.
func TestLiteralIdTekil(t *testing.T) {
	var cift []string
	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		say := map[string]int{}
		for _, m := range literalID.FindAllStringSubmatch(s, -1) {
			say[m[1]]++
		}
		// site_*.html birden çok sayfa tanımı taşır ({{define}}); aynı id
		// farklı sayfalarda geçebilir, aynı belgede geçmez.
		if strings.HasPrefix(filepath.Base(f), "site_") {
			continue
		}
		for k, v := range say {
			if v > 1 {
				cift = append(cift, filepath.Base(f)+": "+k+"×"+strconv.Itoa(v))
			}
		}
	}
	assert.Emptyf(t, cift, "aynı dosyada yinelenen id: %v", cift)
}

func TestFieldEtiketiBagli(t *testing.T) {
	var eksik []string
	denetlenen := 0

	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)

		for i := 0; ; {
			bas := strings.Index(s[i:], `<div class="field">`)
			if bas < 0 {
				break
			}
			bas += i
			son := dengeliDivSonu(s, bas)
			if son < 0 {
				break
			}
			blok := s[bas:son]
			i = son

			if !strings.Contains(blok, "<label") {
				continue
			}
			// İç içe .field: dış kabuğu atla.
			if strings.Contains(blok[len(`<div class="field">`):], `<div class="field">`) {
				continue
			}
			if strings.Contains(blok, "aria-label") || strings.Contains(blok, "aria-labelledby") {
				continue
			}
			kontroller := sayKontrol(blok)
			if kontroller != 1 {
				continue // radyo/onay kutusu grupları: başlık etiketi
			}
			denetlenen++

			// Sarmalayan <label> … <input> … </label>: ilişki zaten var
			// (ekran okuyucu etiket metnini kontrolün adı olarak okur).
			lbl := strings.Index(blok, "<label")
			kapanis := strings.Index(blok, "</label>")
			kontrolYeri := -1
			for _, e := range []string{"<input", "<select", "<textarea"} {
				if j := strings.Index(blok, e); j >= 0 && (kontrolYeri < 0 || j < kontrolYeri) {
					kontrolYeri = j
				}
			}
			if lbl >= 0 && kapanis > kontrolYeri && kontrolYeri > lbl {
				continue
			}

			bagli := false
			for _, fm := range labelFor.FindAllStringSubmatch(blok, -1) {
				if strings.Contains(blok, `id="`+fm[1]+`"`) {
					bagli = true
					break
				}
			}
			if bagli {
				continue
			}
			if strings.Contains(blok, "for={") && strings.Contains(blok, " id={") {
				continue
			}
			satir := strings.Count(s[:bas], "\n") + 1
			eksik = append(eksik, filepath.Base(f)+":"+strconv.Itoa(satir))
		}
	}

	assert.GreaterOrEqualf(t, denetlenen, 40,
		"beklenenden az .field bloğu denetlendi (%d); ayrıştırıcı bozulmuş olabilir", denetlenen)
	assert.Emptyf(t, eksik, ".field alanı etiketsiz (for/id ya da aria-label gerekli): %v", eksik)
}

func dengeliDivSonu(s string, bas int) int {
	i, derinlik := bas, 0
	for i < len(s) {
		if strings.HasPrefix(s[i:], "<div") {
			derinlik++
			k := strings.IndexByte(s[i:], '>')
			if k < 0 {
				return -1
			}
			i += k + 1
			continue
		}
		if strings.HasPrefix(s[i:], "</div>") {
			derinlik--
			i += 6
			if derinlik == 0 {
				return i
			}
			continue
		}
		i++
	}
	return -1
}

func sayKontrol(blok string) int {
	n := 0
	for _, etiket := range []string{"<input", "<select", "<textarea"} {
		n += strings.Count(blok, etiket)
	}
	return n
}
