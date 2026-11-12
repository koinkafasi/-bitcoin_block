package i18n

import (
	"encoding/json"
	"html/template"
)

func (b *Bundle) JSSozluk(locale string, anahtarlar []string) template.JS {
	if b == nil {
		return template.JS("{}")
	}
	m := make(map[string]string, len(anahtarlar))
	for _, k := range anahtarlar {
		// T() anahtar bulunamazsa anahtarın kendisini döndürür; sözlüğe
		// yazmak ekrana "related_more" basılmasına yol açardı.
		if v := b.T(locale, k); v != k {
			m[k] = v
		}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(raw)
}
