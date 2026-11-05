package platform

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SEO alanlarının yazma anındaki sınırları.
//
// ÖLÇÜM (üretim, 2 Kasım 2026):
//
//	yayındaki 986 içeriğin 233'ünde seo_title 70 karakterden uzun (en uzun 138)
//	206'sında seo_title BAŞLIĞIN BİREBİR KOPYASI
//	236'sında meta açıklama 165 karakterden uzun (en uzun 180; EN'de 311)
//
// Neden yazma anında: `aieditor.SeoDogrula` üretim yolunda kırpıyor, ama
// panelden elle kaydetme (`handleContentSave`) ve çeviri kaydı
// (`ceviri.Kaydet`) HİÇ sınır uygulamıyordu. Kural tek yerde dursun.
const (
	SeoBaslikSertMax   = 70
	SeoAciklamaSertMax = 165
)

// SeoKirp metni rune sınırında, kelimeyi bölmeden ve son noktalamayı koruyarak
// kısaltır. Boş metin boş kalır.
//
// Neden kelime sınırı: Google başlığı kelime ortasında kesince tıklama
// düşüyor; biz kesiyorsak tam kelimede keselim.
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
