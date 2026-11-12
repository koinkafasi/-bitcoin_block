package repo

import (
	"strings"
	"unicode"
)

func spotEsitBaslik(spot, baslik string) bool {
	a := spotNormalize(spot)
	b := spotNormalize(baslik)
	if a == "" || b == "" {
		return a == b
	}
	if a == b {
		return true
	}
	// Başlık spottan uzunsa ve spot başlığın öneki ise yine tekrar.
	if strings.HasPrefix(b, a) && len([]rune(a)) >= 40 {
		return true
	}
	return false
}

// spotNormalize karşılaştırma biçimi: küçük harf, harf/rakam dışı atılır.
func spotNormalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
