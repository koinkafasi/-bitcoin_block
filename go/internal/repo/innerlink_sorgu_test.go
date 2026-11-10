package repo

import (
	"strings"
	"testing"
)

// TestLinkSuggestionsSorgusu: alaka havuzu sorgusunun iki PERFORMANS/
// TUTARLILIK sözleşmesini kilitler.
//
//  1. ADAY CTE'Sİ: etiketler yalnız 400 aday için toplanmalı. CTE
//     kaldırılıp eski tek bloklu biçime dönülürse sorgu tüm yayın
//     havuzunu (3 492 satır) + tüm etiket bağlarını (25 290 satır)
//     gruplar: ölçülen maliyet 40,2 ms → 9,9 ms.
//
//  2. TIE-BREAK: `id DESC` olmadan sıra PLAN'A bağlıdır. Üretimde binlerce
//     içerik aynı `updated_at` değerini taşıyor; tie-break yokken aynı sorgu
//     farklı planla farklı "İlgili Haberler" listesi döndürüyordu (ölçüldü:
//     {7128,8443,…} vs {6117,6119,6116}). Bu, sayfanın kendiliğinden
//     değişmesi demek.
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
