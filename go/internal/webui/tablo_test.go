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
	tabloBlok = regexp.MustCompile(`(?s)<table class="table"[^>]*>(.*?)</table>`)
	thSayisi  = regexp.MustCompile(`<th\b`)
	bosCagri  = regexp.MustCompile(`@TabloBos(?:Metin)?\(\s*(\d+)\s*,`)
)

func TestTabloBosColspan(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	require.NotEmpty(t, dizin)

	kontrol := 0
	for _, f := range dizin {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)

		for _, m := range tabloBlok.FindAllStringSubmatchIndex(s, -1) {
			blok := s[m[2]:m[3]]
			sutun := len(thSayisi.FindAllString(blok, -1))
			if sutun == 0 {
				continue
			}
			// Bloktaki boş durum çağrıları (genelde tek).
			for _, c := range bosCagri.FindAllStringSubmatch(blok, -1) {
				kontrol++
				n, err := strconv.Atoi(c[1])
				require.NoError(t, err)
				satir := strings.Count(s[:m[2]], "\n") + 1 +
					strings.Count(blok[:strings.Index(blok, c[0])], "\n")
				assert.Equalf(t, sutun, n,
					"%s:%d: tabloda %d sütun var ama boş durum %d sütun kaplıyor",
					filepath.Base(f), satir, sutun, n)
			}
		}
	}

	// Test kendi kendini doğrulasın: hiç çağrı bulamadıysa regex bozulmuştur.
	assert.GreaterOrEqual(t, kontrol, 20,
		"boş durum çağrısı beklenenden az bulundu; ayrıştırıcı bozulmuş olabilir")
	t.Logf("denetlenen tablo boş durumu: %d", kontrol)
}

func TestTabloBosElleYazilmaz(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)

	elle := regexp.MustCompile(`<tr[^>]*><td[^>]*colspan="\d+"[^>]*class="muted`)
	var bulunan []string
	for _, f := range dizin {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		for _, m := range elle.FindAllStringIndex(s, -1) {
			satir := strings.Count(s[:m[0]], "\n") + 1
			bulunan = append(bulunan, filepath.Base(f)+":"+strconv.Itoa(satir))
		}
	}
	assert.Emptyf(t, bulunan,
		"boş durum elle yazılmış; @TabloBos / @TabloBosMetin kullanın: %v", bulunan)
}
