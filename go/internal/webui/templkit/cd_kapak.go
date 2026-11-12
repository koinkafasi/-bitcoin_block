package templkit

import (
	"strings"
	"unicode"
)

func cdKapakKategori(s CDStory) string {
	k := strings.ToLower(strings.TrimSpace(s.Kicker))
	switch {
	case strings.Contains(k, "piyasa"), strings.Contains(k, "fiyat"):
		return "piyasa"
	case strings.Contains(k, "politik"), strings.Contains(k, "regülas"), strings.Contains(k, "düzenle"):
		return "politika"
	case strings.Contains(k, "teknoloji"), strings.Contains(k, "blokzincir"):
		return "teknoloji"
	case strings.Contains(k, "analiz"), strings.Contains(k, "araştırma"), strings.Contains(k, "görüş"):
		return "analiz"
	case strings.Contains(k, "finans"), strings.Contains(k, "iş dünyası"):
		return "finans"
	default:
		return "genel"
	}
}

func cdKapakHarf(baslik string) string {
	for _, r := range baslik {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return string(unicode.ToUpper(r))
		}
	}
	return "B"
}
