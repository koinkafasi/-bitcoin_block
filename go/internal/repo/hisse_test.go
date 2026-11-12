package repo

import (
	"regexp"
	"testing"
)

func TestHisseListeBicimi(t *testing.T) {
	// BIST kodları 4-6 karakterdir ve rakam içerebilir (A1CAP, 1000?).
	kod := regexp.MustCompile(`^[A-Z0-9]{4,6}$`)
	gorulen := map[string]bool{}
	for _, h := range bistEvreni {
		if !kod.MatchString(h.Code) {
			t.Errorf("geçersiz BIST kodu: %q", h.Code)
		}
		if h.Name == "" {
			t.Errorf("%s için görünen ad boş", h.Code)
		}
		if gorulen[h.Code] {
			t.Errorf("yinelenen kod: %s", h.Code)
		}
		gorulen[h.Code] = true
	}
	if len(bistEvreni) < 400 {
		t.Errorf("tüm hisseler sayfası için geniş evren beklenir, %d var", len(bistEvreni))
	}
	// Referans düzeni alfabetik olduğu için liste de alfabetik tutulur.
	for i := 1; i < len(bistEvreni); i++ {
		if bistEvreni[i-1].Code >= bistEvreni[i].Code {
			t.Errorf("liste alfabetik değil: %s >= %s", bistEvreni[i-1].Code, bistEvreni[i].Code)
		}
	}
	if got := len(HisseSembolleri()); got != len(bistEvreni) {
		t.Errorf("HisseSembolleri %d döndürdü, %d bekleniyordu", got, len(bistEvreni))
	}
	if got := len(HisseEvreni()); got != len(bistEvreni) {
		t.Errorf("HisseEvreni %d döndürdü, %d bekleniyordu", got, len(bistEvreni))
	}
}

func TestHisseAd(t *testing.T) {
	if got := hisseAd("THYAO"); got == "THYAO" || got == "" {
		t.Errorf("THYAO için görünen ad bulunmalı, %q döndü", got)
	}
	if got := hisseAd("YOKBOYLE"); got != "YOKBOYLE" {
		t.Errorf("bilinmeyen kod kendini döndürmeli, %q döndü", got)
	}
}
