package httpapi

import (
	"strings"
)

var takimKisaRef = map[string]string{
	"galatasaray":              "GS",
	"fenerbahce":               "FB",
	"besiktas":                 "BJK",
	"trabzonspor":              "TS",
	"kasimpasa":                "KSM",
	"goztepe":                  "GÖZ",
	"konyaspor":                "KON",
	"samsunspor":               "SAMS",
	"eyupspor":                 "EYP",
	"alanyaspor":               "ALNY",
	"istanbul basaksehir":      "İBFK",
	"basaksehir":               "İBFK",
	"caykur rizespor":          "ÇRZ",
	"rizespor":                 "ÇRZ",
	"gaziantep":                "GFK",
	"gaziantep fk":             "GFK",
	"kocaelispor":              "KOC",
	"amed sportif":             "ASF",
	"amed sportif faaliyetler": "ASF",
	"erzurumspor":              "EFK",
	"erzurumspor fk":           "EFK",
	"genclerbirligi":           "GB",
	"corum fk":                 "ÇFK",
	"karagumruk":               "KRM",
	"antalyaspor":              "ANT",
	"kayserispor":              "KAY",
	"sivasspor":                "SİV",
	"hatayspor":                "HAT",
	"bodrum":                   "BOD",
}

func takimKisa(name string) string {
	ad := strings.TrimSpace(name)
	if ad == "" {
		return ""
	}
	key := kisaAnahtar(ad)
	if k, ok := takimKisaRef[key]; ok {
		return k
	}
	// "Amed Sportif Faaliyetler" gibi çok kelimeli adlar: baş harfler.
	words := strings.Fields(ad)
	if len(words) > 1 {
		var b strings.Builder
		for _, w := range words {
			r := []rune(strings.TrimSpace(w))
			if len(r) == 0 {
				continue
			}
			// "FK", "AS" gibi kısa ekler de harf olarak sayılır.
			b.WriteString(strings.ToUpper(string(r[0])))
			if b.Len() >= 4 {
				break
			}
		}
		if b.Len() >= 2 {
			return b.String()
		}
	}
	// Tek kelime: ilk üç harf büyük ("Trabzonspor" → TRA).
	r := []rune(ad)
	if len(r) > 3 {
		r = r[:3]
	}
	return strings.ToUpper(string(r))
}

// kisaAnahtar karşılaştırma anahtarını üretir: küçük harf, Türkçe
// sadeleştirme, noktalama atılmış.
func kisaAnahtar(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case 'ı', 'i', 'î':
			r = 'i'
		case 'ş':
			r = 's'
		case 'ğ':
			r = 'g'
		case 'ü', 'û':
			r = 'u'
		case 'ö':
			r = 'o'
		case 'ç':
			r = 'c'
		case 'â':
			r = 'a'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
