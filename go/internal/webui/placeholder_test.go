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

func kontrolEtiketleri(s string) []string {
	adlar := []string{"input", "select", "textarea"}
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != '<' {
			continue
		}
		j := i + 1
		for j < len(s) && isLetterByte(s[j]) {
			j++
		}
		ad := s[i+1 : j]
		eslesti := false
		for _, a := range adlar {
			if ad == a {
				eslesti = true
				break
			}
		}
		if !eslesti {
			continue
		}
		// Tırnak farkında tarama.
		k := i
		var tirnak byte
		for ; k < len(s); k++ {
			c := s[k]
			if tirnak != 0 {
				if c == tirnak {
					tirnak = 0
				}
				continue
			}
			if c == '"' || c == '\'' {
				tirnak = c
				continue
			}
			if c == '>' {
				break
			}
		}
		if k < len(s) && s[k] == '>' {
			out = append(out, s[i:k+1])
		}
		i = k
	}
	return out
}

func isLetterByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

var kimlik = regexp.MustCompile(`\bid="([^"{}]+)"`)

// TestPlaceholderTekBasinaEtiketDegil: placeholder taşıyan ama hiçbir ad
// taşımayan kontrol kalmamalı.
func TestPlaceholderTekBasinaEtiketDegil(t *testing.T) {
	var suclu []string
	placeholderSayisi := 0

	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)

		idler := map[string]bool{}
		for _, m := range kimlik.FindAllStringSubmatch(s, -1) {
			idler[m[1]] = true
		}

		// Etiketlerin konumlarını da topla (sarmalama testi için).
		arama := 0
		for _, tag := range kontrolEtiketleri(s) {
			if !strings.Contains(tag, "placeholder=") {
				continue
			}
			placeholderSayisi++
			if strings.Contains(tag, `type="hidden"`) {
				continue
			}
			// Etiketin dosyadaki yeri.
			i := strings.Index(s[arama:], tag)
			if i < 0 {
				continue
			}
			i += arama
			arama = i + len(tag)

			if strings.Contains(tag, "aria-label") || strings.Contains(tag, "aria-labelledby") {
				continue
			}
			kim := kimlik.FindStringSubmatch(tag)
			if len(kim) == 2 && idler[kim[1]] {
				continue
			}
			// Şablon ifadeli kimlik: for={ "x" + a.Key } + id={ "x" + a.Key }
			if strings.Contains(tag, "id={") && strings.Contains(s, "for={") {
				continue
			}
			onceki := s[maxInt(0, i-320):i]
			if sarmalanmis(onceki) {
				continue
			}
			satir := strings.Count(s[:i], "\n") + 1
			suclu = append(suclu, filepath.Base(f)+":"+strconv.Itoa(satir))
		}
	}

	assert.Greater(t, placeholderSayisi, 30, "placeholder taraması bozulmuş olabilir")
	assert.Emptyf(t, suclu,
		"placeholder tek başına etiket değil; aria-label ya da <label> gerekli: %v", suclu)
	t.Logf("placeholder'lı kontrol: %d", placeholderSayisi)
}

// sarmalanmis: kontrolun hemen öncesinde kapanmamis bir <label> var mı.
func sarmalanmis(onceki string) bool {
	i := strings.LastIndex(onceki, "<label")
	if i < 0 {
		return false
	}
	return !strings.Contains(onceki[i:], "</label>")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestAramaKutulariEtiketli: tablo araç çubuğu JS ile üretiliyor; şablonda
// görünmediği için ayrıca kilitleniyor.
func TestAramaKutulariEtiketli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-ui.js"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, "setAttribute('aria-label', 'Tabloda ara — '",
		"tablo arama kutusu aria-label taşımalı")
	assert.Contains(t, s, "function tabloAdi(",
		"aynı sayfada birden çok tablo olabildiği için ad tabloya göre ayrışmalı")
}
