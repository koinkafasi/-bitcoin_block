package httpapi

import (
	"strings"
	"testing"
)

// Faz 6 koruyucu testleri — specs/13-gazete-sayfa-duzenleyici.md

// TestGazeteBaskiYaziTipiMevcut baskıda bildirilen ailenin sunucuda gerçekten
// kurulu olduğunu doğrular.
//
// Eskiden CSS Georgia istiyordu ama kurulu değildi; çıktı sessizce DejaVu
// Serif'e düşüyordu, yani gazete amaçlanan yazı tipiyle basılmıyordu ve bunu
// ancak PDF'in font tablosuna bakınca görüyorduk.
func TestGazeteBaskiYaziTipiMevcut(t *testing.T) {
	kurulu := gazeteKuruluAileler()
	if len(kurulu) == 0 {
		t.Skip("fc-list yok; kurulu aileler okunamadı")
	}
	secilen := gazeteBaskiYaziTipi()
	if secilen == "" {
		t.Fatal("baskı yazı tipi seçilmedi")
	}
	if !kurulu[secilen] {
		t.Errorf("seçilen aile %q sunucuda kurulu değil; çıktı sessizce "+
			"başka bir yazı tipine düşer", secilen)
	}
	// Seçim tercih listesinden olmalı; rastgele bir aile değil.
	var listede bool
	for _, ad := range gazeteSerifTercihi {
		if ad == secilen {
			listede = true
			break
		}
	}
	if !listede {
		t.Errorf("seçilen aile %q tercih listesinde yok", secilen)
	}
	t.Logf("baskı yazı tipi: %s", secilen)
}

// TestGazeteYaziTipiCSSGuvenli şablondan gelen aile adının CSS'e güvenle
// yazıldığını doğrular. Aile adı veridir; tırnak veya noktalı virgül taşırsa
// kural gövdesinden çıkıp başka bildirim enjekte edebilirdi.
func TestGazeteYaziTipiCSSGuvenli(t *testing.T) {
	// Geçerli ad öne alınmalı.
	got := gazeteYaziTipiCSS("Libre Baskerville")
	if !strings.HasPrefix(got, `"Libre Baskerville"`) {
		t.Errorf("bildirilen aile öne alınmadı: %q", got)
	}
	if !strings.HasSuffix(got, "serif") {
		t.Errorf("genel serif yedeği yok: %q", got)
	}

	// Kötü niyetli adlar atılmalı.
	for _, kotu := range []string{
		`x"; background:url(evil)`,
		"x; color:red",
		"x}",
		"a'b",
		strings.Repeat("A", 100),
	} {
		got := gazeteYaziTipiCSS(kotu)
		if strings.ContainsAny(got, ";}{'") {
			t.Errorf("tehlikeli aile adı CSS'e sızdı: giriş %q → %q", kotu, got)
		}
		if strings.Contains(got, "url(") {
			t.Errorf("url() enjekte edildi: %q", got)
		}
	}

	// Boş ad: yalnız seçilen aile ve serif kalmalı.
	bos := gazeteYaziTipiCSS("")
	if strings.Count(bos, ",") != 1 {
		t.Errorf("boş adda beklenmeyen biçim: %q", bos)
	}
}

// TestGazeteAileGuvenli ad denetiminin sınırlarını sabitler.
func TestGazeteAileGuvenli(t *testing.T) {
	gecerli := []string{"Georgia", "Times New Roman", "Source Serif 4",
		"Noto-Serif", "PT_Serif"}
	for _, ad := range gecerli {
		if !gazeteAileGuvenli(ad) {
			t.Errorf("geçerli ad reddedildi: %q", ad)
		}
	}
	gecersiz := []string{"", "a;b", `a"b`, "a'b", "a}b", "a(b)", "a/b",
		strings.Repeat("x", 65)}
	for _, ad := range gecersiz {
		if ad == "" {
			continue // boş ad ayrı ele alınıyor
		}
		if gazeteAileGuvenli(ad) {
			t.Errorf("geçersiz ad kabul edildi: %q", ad)
		}
	}
}

// TestGazeteMmYuzde kılavuz ölçülerinin blok koordinatlarıyla aynı birime
// çevrildiğini doğrular.
func TestGazeteMmYuzde(t *testing.T) {
	// A4 genişliği 210mm: 5mm ≈ %2.381
	got := gazeteMmYuzde(5, "210mm")
	if got < 2.37 || got > 2.39 {
		t.Errorf("5mm/210mm için %%2.381 beklenirken %.3f", got)
	}
	// A3 genişliği 297mm: 3mm ≈ %1.010
	if got := gazeteMmYuzde(3, "297mm"); got < 1.00 || got > 1.02 {
		t.Errorf("3mm/297mm için %%1.01 beklenirken %.3f", got)
	}
	// Bozuk girdi sıfır dönmeli, çökmemeli.
	for _, bozuk := range []string{"", "abc", "0mm", "-5mm"} {
		if got := gazeteMmYuzde(5, bozuk); got != 0 {
			t.Errorf("bozuk ölçü %q için 0 beklenirken %.3f", bozuk, got)
		}
	}
}

// TestGazeteSayfaDegiskenleriYaziTipi sayfa bildiriminin yazı tipini de
// taşıdığını doğrular; baskı CSS'i artık --gz-yazi'ya dayanıyor.
func TestGazeteSayfaDegiskenleriYaziTipi(t *testing.T) {
	d := gazeteSayfaDegiskenleriTam(gazeteBoyut("A4"), 1, "Libre Baskerville")
	if !strings.Contains(d, "--gz-yazi:") {
		t.Errorf("yazı tipi değişkeni yok:\n%s", d)
	}
	if !strings.Contains(d, "Libre Baskerville") {
		t.Errorf("şablonun ailesi bildirilmedi:\n%s", d)
	}
	// Baskı CSS'i değişkeni gerçekten kullanmalı.
	css := cssYorumsuz(gazeteBaskiCSS())
	if !strings.Contains(css, "var(--gz-yazi") {
		t.Error("baskı CSS'i --gz-yazi kullanmıyor; sabit aile kalmış olabilir")
	}
	// Kural gövdelerinde sabit aile kalmamalı (yorumlar ayıklandı: metinde
	// gerekçe olarak geçmesi sorun değil).
	for _, sabit := range []string{"Georgia", "Times New Roman"} {
		if strings.Contains(css, sabit) {
			t.Errorf("baskı CSS'inde sabit %q kalmış; aile sunucuda seçilmeli", sabit)
		}
	}
}
