package httpapi

// Hisse detayı için yapısal veri (JSON-LD).
//
// NEDEN: hisse sayfaları sitemap'e girdi ama sayfa yalnız BreadcrumbList
// basıyordu. CoinDesk varlık sayfalarında WebPage + FinancialProduct
// kullanıyor (spec §10.5); aynı deseni hisseye uyarlıyoruz.
//
// UYDURMA YOK: yalnız elimizdeki gerçek alanlar yazılır (ad, sembol,
// borsa, para birimi, fiyat, günlük değişim). "Market Cap" gibi uçtan
// gelmeyen alan JSON-LD'ye de girmez.

import (
	"html/template"
	"math"
	"strings"
)

// cdHisseJSONLD hisse sayfasının WebPage + FinancialProduct bloğu.
func cdHisseJSONLD(siteName, canonical, ad, sembol, borsa, para, fiyat, degisim string) template.HTML {
	props := []any{
		map[string]any{"@type": "PropertyValue", "name": "Ticker Symbol", "value": sembol},
		map[string]any{"@type": "PropertyValue", "name": "Exchange", "value": borsa},
	}
	// Fiyat: yapısal veride SAYI beklenir ("$610.15" metni değişken
	// biçimde yorumlanır). Fiyat metni tek biçimlendiriciden geliyor
	// (repo.FiyatBicimP): virgül binlik, nokta ondalık.
	if v, ok := cdFiyatSayi(fiyat); ok {
		props = append(props, map[string]any{
			"@type": "PropertyValue", "name": "Price", "value": v, "unitText": para,
		})
	}
	if v, ok := cdYuzdeSayi(degisim); ok {
		props = append(props, map[string]any{
			"@type": "PropertyValue", "name": "Price Change (24 hours)",
			// YUVARLA: yüzde metni "+0.84" ama float aritmetiği
			// 0.8400000000000001 üretebiliyor; yapısal veride çirkin ve
			// gereksiz. İki ondalık, kaynağın kendi hassasiyeti.
			"value":    math.Round(v*100) / 100,
			"unitText": "%",
		})
	}
	return jsonldScript(map[string]any{
		"@context":           "http://schema.org",
		"@type":              []any{"WebPage", "FinancialProduct"},
		"url":                canonical,
		"name":               ad,
		"tickerSymbol":       sembol,
		"additionalProperty": props,
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  siteName,
			"logo":  "",
			"url":   canonical[:cdKokSonu(canonical)],
		},
	})
}

// cdFiyatSayi "$610.15" / "₺1,234.56" → 610.15 / 1234.56.
//
// TEK BİÇİM VARSAYIMI: fiyat metni `repo.FiyatBicimP`ten gelir (binlik
// virgül, ondalık nokta). Bu yüzden para simgesi ve virgüller atılır;
// kalan metin `cdYuzdeSayi` ile sayıya çevrilir (negatif işaret + ondalık).
func cdFiyatSayi(s string) (float64, bool) {
	t := strings.Map(func(r rune) rune {
		switch r {
		case '$', '₺', '€', '£', '¥', ' ':
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	t = strings.ReplaceAll(t, ",", "")
	if t == "" || t == "—" {
		return 0, false
	}
	v, ok := cdYuzdeSayi(t)
	if !ok || v <= 0 {
		return 0, false
	}
	return v, true
}
