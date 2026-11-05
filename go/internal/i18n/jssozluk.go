package i18n

import (
	"encoding/json"
	"html/template"
)

// JSSozluk istemci tarafı için anahtar→çeviri sözlüğünü JSON olarak verir.
//
// Neden: bazı metinler JS ile üretiliyor (ör. kart geri bildirim düğmeleri
// "Buna benzer daha fazla"). Bunlar şablon değişkeni alamıyor; sözlük
// sayfaya bir kez basılıp app.js'ten okunuyor. html/template bu değeri
// <script> içine gömerken kaçış yaptığı için template.JS döner.
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
