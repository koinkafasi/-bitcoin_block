package repo

import (
	"strings"
	"unicode"
)

// spotEsitBaslik spot'un başlığın tekrarı olup olmadığını söyler.
//
// NEDEN: bazı kaynakların `<description>` alanı `<title>` ile birebir aynı
// (Fed, ECB basın bültenleri). Kartta başlık iki kez basılıyordu.
//
// Karşılaştırma NORMALİZE edilir: küçük/büyük harf, noktalama ve fazla
// boşluk farkı "tekrar" sayılır. Sondaki "..."/"…" da yok sayılır
// (feed'ler başlığı kesip spot olarak verebiliyor).
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
