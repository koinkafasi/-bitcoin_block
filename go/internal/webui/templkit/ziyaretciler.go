package templkit

import (
	"math"
	"strconv"
)

// Ziyaretçi paneli yardımcıları.

// sayiMetni64 int64 sayıyı binlik ayraçla yazar (12.483).
//
// NEDEN: panelde 5 haneli sayı ayraçsız okunmuyor ("12483" vs "12.483").
// Var olan sayiMetni(int) dokunulmadı — başka sayfalar onu kullanıyor.
func sayiMetni64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, '.')
		}
		b = append(b, c)
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// yuzdeMetni oranı CSS genişliğine çevirir: 0..100 arası, iki ondalık.
//
// TAŞMA KIRPILIR: yuvarlama 100.4 üretebilir; genişlik 100%'ü aşarsa
// çubuk karttan taşar.
func yuzdeMetni(p float64) string {
	if p < 0 || math.IsNaN(p) {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return strconv.FormatFloat(p, 'f', 2, 64) + "%"
}
