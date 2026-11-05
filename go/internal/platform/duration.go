package platform

import (
	"strconv"
	"strings"
)

// ---- Video süresi yardımcıları (Faz 17) ----
// Editör "3:42", "1:02:03" veya "222" yazabiliyor; depoda saniye saklanıyor.

// ParseDurationTR video süresini saniyeye çevirir. Geçersiz/boş girdide 0 döner.
func ParseDurationTR(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if !strings.Contains(s, ":") {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return 0
		}
		return n
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0
	}
	total := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return 0
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0
		}
		total = total*60 + n
	}
	if total < 0 {
		return 0
	}
	return total
}

// FormatDurationTR saniyeyi "3:42" / "1:02:03" biçiminde yazar.
// 0 veya altı için boş döner (süre bilinmiyor).
func FormatDurationTR(sec int) string {
	if sec <= 0 {
		return ""
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return strconv.Itoa(h) + ":" + pad2(m) + ":" + pad2(s)
	}
	return strconv.Itoa(m) + ":" + pad2(s)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
