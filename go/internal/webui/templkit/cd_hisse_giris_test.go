package templkit

import (
	"context"
	"strings"
	"testing"
)

func hisseGirisCiz(t *testing.T, rows []CDPriceRow, tumHref string) string {
	t.Helper()
	var sb strings.Builder
	if err := cdHisseGiris(rows, tumHref, "tr").Render(context.Background(), &sb); err != nil {
		t.Fatalf("cdHisseGiris çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekHisseSatirlari() []CDPriceRow {
	return []CDPriceRow{
		{Sym: "THYAO", Name: "Türk Hava Yolları", Price: "₺306.75", Change: "+2.51%",
			Up: true, Href: "/price/stock/THYAO.IS"},
		{Sym: "TUPRS", Name: "Tüpraş", Price: "₺335.75", Change: "-0.52%",
			Up: false, Href: "/price/stock/TUPRS.IS"},
	}
}

// TestCDHisseGirisBaglantilar her satırın KENDİ detay sayfasına gittiğini
// kilitler (hepsi /markets'e ya da boş bağlantıya dönerse test düşer).
func TestCDHisseGirisBaglantilar(t *testing.T) {
	html := hisseGirisCiz(t, ornekHisseSatirlari(), "/borsa/hisseler")

	for _, istenen := range []string{
		`href="/price/stock/THYAO.IS"`,
		`href="/price/stock/TUPRS.IS"`,
		`href="/borsa/hisseler"`,
		"Hisse Senetleri",
		"₺306.75",
		"+2.51%",
		"-0.52%",
		"Türk Hava Yolları",
	} {
		if !strings.Contains(html, istenen) {
			t.Errorf("eksik: %s", istenen)
		}
	}
	// Yön sınıfı: yükselen yeşil, düşen kırmızı.
	if !strings.Contains(html, `class="cd-up">+2.51%`) {
		t.Errorf("yükseliş cd-up ile basılmamış")
	}
	if !strings.Contains(html, `class="cd-down">-0.52%`) {
		t.Errorf("düşüş cd-down ile basılmamış")
	}
	// Tablo sınıfı yeniden kullanılır → yeni CSS gerekmez.
	if !strings.Contains(html, `class="cd-table"`) {
		t.Errorf("cd-table sınıfı kullanılmamış")
	}
	// Ölü bağlantı yok: her href dolu.
	if strings.Contains(html, `href=""`) {
		t.Errorf("boş href basılmış")
	}
}

// TestCDHisseGirisBos liste boşsa bölüm HİÇ basılmaz.
func TestCDHisseGirisBos(t *testing.T) {
	for _, rows := range [][]CDPriceRow{nil, {}} {
		html := hisseGirisCiz(t, rows, "/borsa/hisseler")
		if strings.Contains(html, "cd-table") || strings.Contains(html, "Hisse Senetleri") {
			t.Errorf("boş listede bölüm basılmış: %q", html)
		}
	}
}

// TestCDHisseGirisTumHrefBoş "Tümü" hedefi boşsa bağlantı basılmamalı.
func TestCDHisseGirisTumHrefBoş(t *testing.T) {
	html := hisseGirisCiz(t, ornekHisseSatirlari(), "")
	if strings.Contains(html, "Tümü") {
		t.Errorf("boş hedefe 'Tümü' bağlantısı basılmış")
	}
	if !strings.Contains(html, "Hisse Senetleri") {
		t.Errorf("bölüm başlığı yok")
	}
}
