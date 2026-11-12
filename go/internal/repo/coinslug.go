package repo

import "strings"

func CoinSlug(ad string) string {
	ad = strings.ToLower(strings.TrimSpace(ad))
	var b strings.Builder
	b.Grow(len(ad))
	for _, r := range ad {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == '/':
			b.WriteByte('-')
		// Türkçe harfler: ada düşerse harf SESSİZCE kaybolurdu
		// ("türk" → "trk"). Latin karşılığına çevir.
		case r == 'ç':
			b.WriteByte('c')
		case r == 'ğ':
			b.WriteByte('g')
		case r == 'ı':
			b.WriteByte('i')
		case r == 'ö':
			b.WriteByte('o')
		case r == 'ş':
			b.WriteByte('s')
		case r == 'ü':
			b.WriteByte('u')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}
