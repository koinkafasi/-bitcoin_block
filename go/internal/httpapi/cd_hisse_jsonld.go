package httpapi

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
	if v, ok := cdFiyatSayi(fiyat); ok {
		props = append(props, map[string]any{
			"@type": "PropertyValue", "name": "Price", "value": v, "unitText": para,
		})
	}
	if v, ok := cdYuzdeSayi(degisim); ok {
		props = append(props, map[string]any{
			"@type": "PropertyValue", "name": "Price Change (24 hours)",
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
