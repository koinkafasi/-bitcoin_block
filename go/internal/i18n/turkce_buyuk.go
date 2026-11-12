package i18n

import "strings"

func BuyukHarf(s string) string {
	s = strings.ReplaceAll(s, "i", "İ")
	s = strings.ReplaceAll(s, "ı", "I")
	return strings.ToUpper(s)
}
