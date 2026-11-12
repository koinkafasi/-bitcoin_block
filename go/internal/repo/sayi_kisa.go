package repo

// SayiBicimKisa 1_635_401_465_856 → "$1.64T" · 423_349_420_032 → "₺423.35B".
func SayiBicimKisa(f float64, para string) string {
	if f <= 0 {
		return "—"
	}
	simge := "$"
	switch para {
	case "TRY":
		simge = "₺"
	case "EUR":
		simge = "€"
	case "GBP":
		simge = "£"
	}
	switch {
	case f >= 1e12:
		return simge + binlikAyrac(f/1e12, 2) + "T"
	case f >= 1e9:
		return simge + binlikAyrac(f/1e9, 2) + "B"
	case f >= 1e6:
		return simge + binlikAyrac(f/1e6, 2) + "M"
	case f >= 1e3:
		return simge + binlikAyrac(f/1e3, 2) + "K"
	default:
		return simge + binlikAyrac(f, 2)
	}
}
