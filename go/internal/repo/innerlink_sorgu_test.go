package repo

import (
	"strings"
	"testing"
)

func TestLinkSuggestionsSorgusu(t *testing.T) {
	s := strings.Join(strings.Fields(linkSuggestionsSQL), " ")

	if !strings.HasPrefix(strings.ToUpper(s), "WITH ADAY AS") {
		t.Error("aday CTE'si yok: etiketler tüm havuz için toplanıyor olabilir (40,2 ms)")
	}
	if !strings.Contains(s, "LIMIT 400") {
		t.Error("CTE içinde LIMIT 400 yok: aday daraltması kaybolmuş")
	}
	if !strings.Contains(s, "ORDER BY c.updated_at DESC, c.id DESC") {
		t.Error("tie-break (c.id DESC) yok: sıralama plana bağlı hale gelir")
	}
	if !strings.Contains(s, "ORDER BY a.sira") {
		t.Error("dış sorguda ORDER BY a.sira yok: grup sırası belirsiz")
	}
	// Havuz sorgusu hafif kalmalı: `detail` (uzun gövde) SEÇİLMEMELİ.
	if strings.Contains(strings.ToLower(s), "c.detail") {
		t.Error("alaka havuzu c.detail seçiyor: gereksiz veri çekiliyor")
	}
}
