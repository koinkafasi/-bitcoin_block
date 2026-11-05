package httpapi

import (
	"strings"
)

// Puan Durumu Şeridi — Habertürk spor yazısı düzeninin birebir kopyası.
//
// Referans (haberturk.com/spor/... makale sayfası, 28 Ekim 2026):
//
//	<section data-type="standingsSummary">
//	  <h2 class="font-black text-xl">PUAN DURUMU</h2>  + yatay çizgi
//	  <div id="standings-container" class="flex space-x-2 overflow-auto">
//	    <div class="flex-none md:flex-1 w-10 border">   ← her takım dikey kart
//	      <a class="flex flex-col">
//	        <img logo72>                                 ← kare logo
//	        <span class="text-xxs">GS</span>             ← kısa kod
//	        <div class="border-t"><span>22</span></div>  ← puan
//	      </a>
//	    </div>
//	    … 18 takım, yatay kaydırmalı
//	  </div>
//	</section>
//
// Yani tablo DEĞİL: takım başına dikey kart, yatay şerit. Mevcut
// `puan_durumu` bloğu klasik tablodur; bu ayrı bir modüldür.

// takimKisaRef referanstaki kısa kodlar (Habertürk'ün 18 takımı).
//
// Neden sabit tablo: kodlar kısaltma kuralıyla üretilemiyor
// ("Galatasaray" → GS, "Beşiktaş" → BJK). Bilinmeyen takım için
// takimKisa() genel kurala düşer, böylece yeni takımda da bir kod çıkar.
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

// takimKisa takım adından kısa kod üretir (rozet metni).
//
// Sıra: bilinen tablo → kelime baş harfleri → ilk üç harf.
// Türkçe sadeleştirme yapılır ki anahtar eşleşmesi büyük/küçük harf ve
// aksan farkından etkilenmesin ("Beşiktaş" = "besiktas").
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
