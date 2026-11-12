package platform

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	SeoBaslikSertMax   = 70
	SeoAciklamaSertMax = 165
)

func SeoKirp(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || max <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	r = r[:max]
	// Sondaki yarım kelimeyi at (en az yarısı kalsın).
	for i := len(r) - 1; i > max/2; i-- {
		if unicode.IsSpace(r[i]) {
			r = r[:i]
			break
		}
	}
	return strings.TrimRight(strings.TrimRightFunc(string(r), unicode.IsSpace), ".,;:·-–—|")
}
