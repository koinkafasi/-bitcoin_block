package platform

import (
	"strings"
	"testing"
)

func TestSeoKirpSinirAltinda(t *testing.T) {
	s := "Kısa bir SEO başlığı"
	if got := SeoKirp(s, 70); got != s {
		t.Errorf("sınır altındaki metin değişti: %q", got)
	}
}

func TestSeoKirpUzunMetniKisaltir(t *testing.T) {
	s := strings.Repeat("kelime ", 30) // 210 karakter
	got := SeoKirp(s, 70)
	if len([]rune(got)) > 70 {
		t.Errorf("sınır aşıldı: %d", len([]rune(got)))
	}
	if strings.HasSuffix(got, " ") || strings.HasSuffix(got, "ke") {
		t.Errorf("kelime ortasında bitti: %q", got)
	}
}

func TestSeoKirpSondakiNoktalamayiTemizler(t *testing.T) {
	// Tam sınırda kalan "|" veya "," tıklama düşürüyor.
	got := SeoKirp("Başlık metni burada | ", 12)
	if strings.HasSuffix(got, "|") || strings.HasSuffix(got, ",") {
		t.Errorf("sondaki ayırıcı kalmış: %q", got)
	}
}

func TestSeoKirpTurkceKarakteriBozmaz(t *testing.T) {
	// Rune sınırı: ı/ş/ğ iki bayt; byte kesme bunları bozar.
	s := strings.Repeat("şğıöç ", 20)
	got := SeoKirp(s, 21)
	for _, r := range got {
		if r == '\uFFFD' {
			t.Fatalf("bozuk karakter: %q", got)
		}
	}
}

func TestSeoKirpBosVeSifir(t *testing.T) {
	if got := SeoKirp("", 70); got != "" {
		t.Errorf("boş metin değişti: %q", got)
	}
	if got := SeoKirp("metin", 0); got != "metin" {
		t.Errorf("max=0 metni bozdu: %q", got)
	}
}

func TestSeoKirpBosluklariTekler(t *testing.T) {
	if got := SeoKirp("çok    fazla\nboşluk", 70); got != "çok fazla boşluk" {
		t.Errorf("boşluk tekleşmedi: %q", got)
	}
}
