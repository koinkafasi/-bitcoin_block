package templkit

import (
	"strconv"
	"strings"
)

// itoaT int → string (şablon yardımcıları).
func itoaT(n int) string { return strconv.Itoa(n) }

// itoaT64 int64 → string (şablon yardımcıları).
func itoaT64(n int64) string { return strconv.FormatInt(n, 10) }

// parseT64 string → int64. Şablonlarda içerik kimliği metin olarak
// taşınıyor; sayısal alan isteyen bileşenlere geçerken bu yardımcı
// kullanılır. Çözülemeyen değer 0 döner (bileşen kendini gizler).
func parseT64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// joinComma string dilimini virgülle birleştirir.
func joinComma(v []string) string {
	out := ""
	for i, s := range v {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// yedekIpucu künye alanının yer tutucusunu üretir: yedek varsa onu gösterir,
// yoksa genel bir yönlendirme bırakır. Boş yer tutucu, alanın neden boş
// olduğunu açıklamaz.
func yedekIpucu(yedek string) string {
	if yedek != "" {
		return "boş = " + yedek
	}
	return "zorunlu"
}

// boolStr aria-* öznitelikleri için "true"/"false" üretir. templ v0.3'te
// templ.KV yalnız class içinde çalışıyor; aria-pressed metin bekliyor.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// lowerTR arama filtresi için metni küçük harfe çevirir.
//
// Türkçe'de "I" küçültünce "ı", "İ" küçültünce "i" olur; strings.ToLower
// bunu yapmaz (Unicode eşlemesi I→i). Yardım aramasında "ISLEM" yazan
// kullanıcı "işlem" maddesini bulamıyordu. Bu yüzden önce Türkçe İ/I
// karakterleri eşlenir, sonra genel küçültme uygulanır.
func lowerTR(v string) string {
	r := strings.NewReplacer("İ", "i", "I", "ı")
	return strings.ToLower(r.Replace(v))
}
