package repo

import "testing"

func TestAIMatchTitleCrossCategory(t *testing.T) {
	fransa := aiTitleTokens("Fransa'da hafta sonu yapılacak gösteriler yasaklandı")
	toulon := aiTitleTokens("Fransa'nın Toulon şehrinde hafta sonu yapılacak gösteriler yasaklandı")
	if score, ok := aiMatchTitle(fransa, toulon, false, false); !ok || score < 0.7 {
		t.Fatalf("aynı olayın çapraz kategori başlıkları eşleşmeli: score=%v ok=%v", score, ok)
	}

	gs := aiTitleTokens("Galatasaray maç hazırlıklarını sürdürdü")
	fb := aiTitleTokens("Fenerbahçe maç hazırlıklarını sürdürdü")
	if score, ok := aiMatchTitle(gs, fb, false, false); ok {
		t.Fatalf("farklı takımlar çapraz kategoride eşleşmemeli: score=%v", score)
	}

	mac := aiTitleTokens("Bayern Münih - Arsenal maçı ne zaman, saat kaçta?")
	spor := aiTitleTokens("Bayern Münih - Arsenal maçı ne zaman, saat kaçta, hangi kanalda?")
	if _, ok := aiMatchTitle(mac, spor, false, true); !ok {
		t.Fatal("aynı maç duyurusu çapraz kategoride eşleşmeli")
	}
}

func TestAICrossCategoryThresholds(t *testing.T) {
	a := map[string]bool{"fransa": true, "toulon": true, "gosteriler": true}
	b := map[string]bool{"fransa": true, "gosteriler": true, "paris": true, "police": true, "gozalti": true}
	if score, ok := aiMatchTitle(a, b, false, false); ok {
		t.Fatalf("düşük benzerlik yeni grupta eşleşmemeli: %v", score)
	}
	if score, ok := aiMatchTitle(a, b, false, true); ok {
		t.Fatalf("düşük benzerlik mevcut kümede eşleşmemeli: %v", score)
	}
}

func TestAINewGroupDominantCategory(t *testing.T) {
	cat1, cat2 := int64(1), int64(2)
	g := newAINewGroup(AISourceItem{CategoryID: &cat1}, map[string]bool{"a": true})
	g.add(AISourceItem{CategoryID: &cat1}, map[string]bool{"b": true})
	g.add(AISourceItem{CategoryID: &cat2}, map[string]bool{"c": true})
	if got := g.dominantCategory(); got == nil || *got != cat1 {
		t.Fatalf("baskın kategori 1 olmalı: %#v", got)
	}
	tie := newAINewGroup(AISourceItem{CategoryID: &cat2}, map[string]bool{"a": true})
	tie.add(AISourceItem{CategoryID: &cat1}, map[string]bool{"b": true})
	if got := tie.dominantCategory(); got == nil || *got != cat2 {
		t.Fatalf("eşitlikte ilk kategori korunmalı: %#v", got)
	}
}
