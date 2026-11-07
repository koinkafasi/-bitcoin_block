package i18n

import "strings"

// BuyukHarf metni Türkçe kurallarıyla büyük harfe çevirir.
//
// NEDEN: Go'nun strings.ToUpper'ı yerel ayar bilmez ve Türkçe'de YANLIŞ
// sonuç verir — "politika" → "POLITIKA" (olması gereken "POLİTİKA"),
// "piyasalar" → "PIYASALAR" ("PİYASALAR"). Bölüm kicker'ı büyük harfle
// basıldığı için hata doğrudan sayfada görünüyordu (ölçüm 7 Kasım 2026:
// makale üstünde "POLITIKA" yazıyordu).
//
// Kurallar (TDK): i → İ, ı → I; diğer harfler standart büyütmeyle.
// Noktalı/noktasız i ayrımı yalnız KÜÇÜK harfte anlamlı olduğundan, önce
// iki özel harf değiştirilip sonra genel büyütme uygulanır — "İ" ve "I"
// büyütmeden etkilenmez.
func BuyukHarf(s string) string {
	s = strings.ReplaceAll(s, "i", "İ")
	s = strings.ReplaceAll(s, "ı", "I")
	return strings.ToUpper(s)
}
