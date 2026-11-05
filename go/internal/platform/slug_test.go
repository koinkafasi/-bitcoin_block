package platform

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Türkiye'nin Gündemi!", "turkiyenin-gundemi"},
		{"Yapay Zeka ve Habercilik", "yapay-zeka-ve-habercilik"},
		{"  çok -- boşluklu  ", "cok-bosluklu"},
		{"İstanbul İzmir", "istanbul-izmir"},
		{"!!!", "icerik"},
		{"", "icerik"},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestSlugifyLowerTurkishI(t *testing.T) {
	// hem 'I' hem 'İ' → 'i' olmalı
	if got := Slugify("Aİ BİR"); got != "ai-bir" {
		t.Fatalf("I/İ dönüşümü hatalı: %q", got)
	}
}
