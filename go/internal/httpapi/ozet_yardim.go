package httpapi

import "strings"

// ilkCumle özetin kapalı durumda görünen tek satırını üretir.
func ilkCumle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := cumleSonu(s); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func cumlelereBol(s string, adet int) []string {
	s = strings.TrimSpace(s)
	if s == "" || adet <= 0 {
		return nil
	}
	var out []string
	kalan := s
	for len(out) < adet {
		i := cumleSonu(kalan)
		if i <= 0 {
			if t := strings.TrimSpace(kalan); t != "" {
				out = append(out, t)
			}
			break
		}
		out = append(out, strings.TrimSpace(kalan[:i]))
		kalan = strings.TrimSpace(kalan[i:])
		if kalan == "" {
			break
		}
	}
	return out
}

func cumleSonu(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', '!', '?':
			// Sonraki karakter boşluk ya da metin sonu olmalı.
			if i+1 == len(s) {
				return i + 1
			}
			if s[i+1] == ' ' || s[i+1] == '\n' || s[i+1] == '\t' {
				return i + 1
			}
		}
	}
	return -1
}
