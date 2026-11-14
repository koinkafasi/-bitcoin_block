package httpapi

import "testing"

// TestSiteAdiCoz: ayar kaydı iki biçimde gelebilir ve boş değer siteyi
// adsız bırakmamalı. 14 Kasım 2026'da canlıda <title> "theblock.com.tr"
// kalmıştı; kök neden adın açılışta donmasıydı (bkz. siteAdi).
func TestSiteAdiCoz(t *testing.T) {
	ornekler := []struct {
		ad    string
		ham   string
		bekle string
	}{
		{"tek dil", `{"tr":"BİTCOŞAR"}`, "BİTCOŞAR"},
		{"boşluklu", `{"tr":"  BİTCOŞAR  "}`, "BİTCOŞAR"},
		{"tr boş, en dolu", `{"tr":"","en":"TheBlok"}`, "TheBlok"},
		{"yalnız başka dil", `{"de":"TheBlok"}`, "TheBlok"},
		{"düz metin", `"bitcosar"`, "bitcosar"},
		{"boş nesne", `{}`, ""},
		{"boş dize", `""`, ""},
		{"bozuk json", `{`, ""},
		{"hiç yok", ``, ""},
	}
	for _, o := range ornekler {
		if got := siteAdiCoz([]byte(o.ham)); got != o.bekle {
			t.Errorf("%s: siteAdiCoz(%s) = %q, beklenen %q", o.ad, o.ham, got, o.bekle)
		}
	}
}

// TestSiteAdiVarsayilan: ayar okunamadığında (havuz yok) site eski markaya
// ("theblock.com.tr") DÜŞMEMELİ.
func TestSiteAdiVarsayilan(t *testing.T) {
	s := &Server{}
	if got := s.siteAdi(); got != siteAdiVarsayilan {
		t.Fatalf("siteAdi() = %q, beklenen %q", got, siteAdiVarsayilan)
	}
	if siteAdiVarsayilan == "theblock.com.tr" {
		t.Fatal("varsayılan ad eski marka olmamalı")
	}
}
