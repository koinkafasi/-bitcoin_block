package httpapi

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
