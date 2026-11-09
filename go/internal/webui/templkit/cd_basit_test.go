package templkit

import (
	"context"
	"strings"
	"testing"
)

// CDBasitSayfaView — yeni kabuk iskeleti sözleşmesi.
//
// NEDEN: /hava-durumu, /namaz-vakitleri, /imsakiye, /astroloji, /fikstur,
// /resmi-ilanlar, /yazarlar theblock kabuğunu kullanıyordu (site_header /
// site_footer). Bu iskelet onların yeni kabukta basılmasını sağlıyor.
//
// EN KRİTİK NOKTASI: gövde KAÇIŞLANMAMALI. `SafeHTML` (= template.HTML)
// templ tarafından otomatik kaçışlanıyordu; tablo HTML'i sayfada DÜZ METİN
// olarak görünüyordu (ölçüldü, 9 Kasım 2026). Bu test onu kilitler.

func basitCiz(t *testing.T, v CDBasitSayfa) string {
	t.Helper()
	if v.Shell.SiteName == "" {
		v.Shell = CDShell{SiteName: "bitcosar"}
	}
	var sb strings.Builder
	if err := CDBasitSayfaView(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("basit sayfa çizilemedi: %v", err)
	}
	return sb.String()
}

// TestCDBasitSayfaKabuk kabuğun yeni (cd) olduğunu kilitler.
func TestCDBasitSayfaKabuk(t *testing.T) {
	html := basitCiz(t, CDBasitSayfa{
		Baslik:   "Hava Durumu",
		Aciklama: "81 ilin sıcaklığı.",
		Govde:    SafeHTML(`<p>içerik</p>`),
		Serit:    true,
	})

	for _, istenen := range []string{
		`class="cd-topbar"`,
		`class="cd-ticker"`,
		`class="cd-footer"`,
		"cd-basit",
		"Hava Durumu",
		"81 ilin sıcaklığı.",
	} {
		if !strings.Contains(html, istenen) {
			t.Errorf("eksik: %s", istenen)
		}
	}
	// ESKİ KABUK izleri OLMAMALI.
	for _, yasak := range []string{"site-header", "site-footer", "hp-drawer"} {
		if strings.Contains(html, yasak) {
			t.Errorf("eski kabuk izi kaldı: %s", yasak)
		}
	}
}

// TestCDBasitSayfaGovdeKacislanmaz gövde HTML'i DÜZ METİN olarak
// basılmamalı — tablo gerçekten tablo olmalı.
func TestCDBasitSayfaGovdeKacislanmaz(t *testing.T) {
	govde := `<table class="cd-table"><tr><td>Adana</td></tr></table>`
	html := basitCiz(t, CDBasitSayfa{Baslik: "Namaz", Govde: SafeHTML(govde)})

	if !strings.Contains(html, `<table class="cd-table">`) {
		t.Errorf("tablo kaçışlanmış (düz metin olarak basılmış)")
	}
	if strings.Contains(html, "&lt;table") {
		t.Errorf("gövde HTML kaçışlanmış: &lt;table görünüyor")
	}
}

// TestCDBasitSayfaSeritKapali Serit=false iken şerit basılmamalı.
func TestCDBasitSayfaSeritKapali(t *testing.T) {
	html := basitCiz(t, CDBasitSayfa{Baslik: "X", Govde: SafeHTML("<p>x</p>")})
	if strings.Contains(html, "cd-ticker") {
		t.Errorf("Serit=false iken şerit basılmış")
	}
}

// TestCDBasitSayfaAciklamaBosBoş aciklama boşsa spot basılmamalı.
func TestCDBasitSayfaAciklamaBos(t *testing.T) {
	html := basitCiz(t, CDBasitSayfa{Baslik: "X", Govde: SafeHTML("<p>x</p>")})
	if strings.Contains(html, "cd-card__spot") {
		t.Errorf("boş açıklama için spot basılmış")
	}
}
