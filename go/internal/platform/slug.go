package platform

import (
	"strings"
	"unicode"
)

// trMap Türkçe karakterleri ASCII karşılığına eşler (küçük harf bazında).
var trMap = map[rune]string{
	'ç': "c", 'ğ': "g", 'ı': "i", 'i': "i", 'ö': "o", 'ş': "s", 'ü': "u",
	'Ç': "c", 'Ğ': "g", 'İ': "i", 'Ö': "o", 'Ş': "s", 'Ü': "u", 'I': "i",
	'â': "a", 'î': "i", 'û': "u", 'Â': "a", 'Î': "i", 'Û': "u",
}

// Slugify Türkçe-uyumlu ASCII slug üretir. Örn: "Türkiye'nin Gündemi!" → "turkiyenin-gundemi"
func Slugify(s string) string {
	var b strings.Builder
	prevDash := true // baştaki ayraçları yut
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if rep, ok := trMap[r]; ok {
			for _, rr := range rep {
				writeSlugRune(&b, rr, &prevDash)
			}
			continue
		}
		writeSlugRune(&b, r, &prevDash)
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "icerik"
	}
	return out
}

func writeSlugRune(b *strings.Builder, r rune, prevDash *bool) {
	switch {
	case r == '\'':
		// kesme işareti atlanır: "Türkiye'nin" → "turkiyenin"
	case unicode.IsLetter(r) || unicode.IsDigit(r):
		b.WriteRune(r)
		*prevDash = false
	case *prevDash:
		// yut
	default:
		b.WriteByte('-')
		*prevDash = true
	}
}
