package templkit

import "strconv"

func cdAnalizDilim(kartlar []CDAnalizKart, kolon int) []CDAnalizKart {
	if kolon < 0 || kolon > 2 {
		return nil
	}
	var out []CDAnalizKart
	for i := kolon; i < len(kartlar); i += 3 {
		out = append(out, kartlar[i])
	}
	return out
}

// cdAnalizRozet başlığın yanındaki gerçek sayı.
func cdAnalizRozet(sayi int) string {
	return "son 7 günde " + strconv.Itoa(sayi) + " analiz"
}
