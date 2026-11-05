package repo

import "testing"

func TestAITitleTokensTurkishNormalization(t *testing.T) {
	toks := aiTitleTokens("Ekonomide enflasyon kararları açıklandı")
	for _, want := range []string{"ekonomide", "enflasyon", "kararlari", "aciklandi"} {
		if !toks[want] {
			t.Fatalf("token %q eksik: %#v", want, toks)
		}
	}
	if toks["ve"] || toks["yeni"] {
		t.Fatalf("stopword token'a eklenmemeli: %#v", toks)
	}
}

func TestChooseRSSSection(t *testing.T) {
	cats := map[string]int64{"gundem": 1, "ekonomi": 2, "spor": 3, "teknoloji": 4}
	gundem := int64(1)
	ekonomi := int64(2)
	teknoloji := int64(4)

	if got := chooseRSSSection([]string{"Gündem", "Ekonomi"}, cats, &ekonomi); got != "Ekonomi" {
		t.Fatalf("feed kategorisi eşleşmesi öncelikli olmalı: %q", got)
	}
	if got := chooseRSSSection([]string{"Gündem", "Spor"}, cats, &gundem); got != "Spor" {
		t.Fatalf("Gündem dışı özel kategori seçilmeli: %q", got)
	}
	if got := chooseRSSSection([]string{"Gündem"}, cats, &teknoloji); got != "" {
		t.Fatalf("genel Gündem etiketi özel feed kategorisini ezmemeli: %q", got)
	}
	if got := chooseRSSSection([]string{"Gündem"}, cats, nil); got != "Gündem" {
		t.Fatalf("feed kategorisi yoksa Gündem kullanılmalı: %q", got)
	}
}

func TestOverlapScore(t *testing.T) {
	a := aiTitleTokens("Merkez Bankası faiz kararını açıkladı")
	b := aiTitleTokens("Merkez Bankası faiz kararı sonrası piyasa")
	if got := overlapScore(a, b); got < 0.5 {
		t.Fatalf("benzer başlık skoru düşük: %v", got)
	}
	c := aiTitleTokens("Galatasaray maçı hazırlıklarını sürdürüyor")
	if got := overlapScore(a, c); got != 0 {
		t.Fatalf("ilgisiz başlık skoru sıfır olmalı: %v", got)
	}
}
