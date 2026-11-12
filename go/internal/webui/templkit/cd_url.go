package templkit

import "strings"

func cdMutlak(u, base string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return u
	}
	return base + "/" + strings.TrimPrefix(u, "/")
}
