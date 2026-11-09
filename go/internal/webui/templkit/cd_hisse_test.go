package templkit

import (
	"context"
	"strings"
	"testing"
)

// Hisse detay sayfası (/price/stock/{sembol}) YAPI sözleşmesi.
//
// ÖLÇÜM (coindesk.com/price/stock/meta, spec §10.6):
//
//	içerik   : 1432 px, ızgara 392 + 976 (gap 32)
//	sol ray  : breadcrumb + ad + büyük fiyat 28/42 w500, "Key Stats"
//	sağ kolon: Basic/Advanced, 1D·5D·1M·YTD·1Y·All, OHLC okuması,
//	           976×500 grafik, 88 px performans satırı
//	<h1>     : CoinDesk'te YOK (ad küçük metin)
//
// NİYE TEST: bu şablon coin şablonundan ayrı; sekmeler/performans/OHLC
// sessizce kaybolursa sayfa "coin sayfası"na döner ve kimse fark etmez.

func hisseCiz(t *testing.T, v CDHisseSayfa) string {
	t.Helper()
	var sb strings.Builder
	if err := CDHisseSayfaView(v).Render(context.Background(), &sb); err != nil {
		t.Fatalf("hisse sayfası çizilemedi: %v", err)
	}
	return sb.String()
}

func ornekHisse() CDHisseSayfa {
	return CDHisseSayfa{
		Shell:     CDShell{SiteName: "bitcosar"},
		Title:     "Meta Platforms, Inc. (META) Hisse Fiyatı",
		Desc:      "Meta Platforms güncel hisse fiyatı.",
		Canonical: "https://ornek.test/price/stock/meta",
		Ad:        "Meta Platforms, Inc.",
		Sembol:    "META",
		Borsa:     "NasdaqGS",
		Para:      "USD",
		Fiyat:     "$610.15",
		Degisim:   "+0.84%",
		Up:        true,
		AsOf:      "9 Kas 2026 15:59 EST",
		Istatistik: []CDHisseStat{
			{Etiket: "52 hafta aralığı", Deger: "$520.26 – $779.82"},
			{Etiket: "Hacim", Deger: "15,705,472"},
		},
		Yol:        "M0 90L640 90",
		Alan:       "M0 90L640 90L640 180L0 180Z",
		GrafikUp:   true,
		GrafikVar:  true,
		GrafikMin:  "$520.26",
		GrafikMaks: "$779.82",
		EtiketSol:  "9 Kas 2025",
		EtiketSag:  "9 Kas 2026",
		Acilis:     "$606.82",
		Yuksek:     "$625.55",
		Dusuk:      "$606.82",
		Sekmeler: []CDHisseSekme{
			{Kod: "1d", Ad: "1D", Aktif: true},
			{Kod: "5d", Ad: "5D", Href: "/price/stock/meta?aralik=5d"},
			{Kod: "1m", Ad: "1M", Href: "/price/stock/meta?aralik=1m"},
			{Kod: "ytd", Ad: "YTD", Href: "/price/stock/meta?aralik=ytd"},
			{Kod: "1y", Ad: "1Y", Href: "/price/stock/meta?aralik=1y"},
			{Kod: "all", Ad: "All", Href: "/price/stock/meta?aralik=all"},
		},
		AktifKod:        "1d",
		Kip:             "basic",
		KipHrefBasic:    "/price/stock/meta",
		KipHrefAdvanced: "/price/stock/meta?kip=advanced",
		Performans: []CDHisseDonem{
			{Ad: "24 hours", Yuzde: "+0.84%", Up: true},
			{Ad: "5 days", Yuzde: "+0.56%", Up: true},
			{Ad: "1 month", Yuzde: "-15.41%", Up: false},
			{Ad: "YTD", Yuzde: "-6.20%", Up: false},
			{Ad: "1 year", Yuzde: "-1.86%", Up: false},
		},
		Kaynak: "Kaynak: Yahoo Finance · NasdaqGS · 60 saniye önbellek",
	}
}

// TestCDHisseYapi ana blokların (sol ray, istatistik, sekmeler, OHLC,
// grafik, performans) basıldığını kilitler.
func TestCDHisseYapi(t *testing.T) {
	html := hisseCiz(t, ornekHisse())

	for _, istenen := range []string{
		`class="cd-hisse"`,
		`class="cd-hisse__sol"`,
		`class="cd-hisse__sag"`,
		`cd-hisse__stats`,
		`cd-hisse__sekmeler`,
		`cd-hisse__ohlc`,
		`cd-hisse__grafik-kutu`,
		`cd-hisse__performans`,
		`cd-hisse__kip-dugme--aktif`,
		"$610.15",
		"52 hafta aralığı",
		"NasdaqGS",
	} {
		if !strings.Contains(html, istenen) {
			t.Errorf("eksik: %s", istenen)
		}
	}

	// Altı zaman sekmesi: 1D 5D 1M YTD 1Y All (CoinDesk sırası).
	for _, ad := range []string{"1D", "5D", "1M", "YTD", "1Y", "All"} {
		if !strings.Contains(html, ">"+ad+"<") {
			t.Errorf("sekme yok: %s", ad)
		}
	}
	// Aktif sekme bağlantı DEĞİL (tıklanınca aynı sayfaya dönmesin).
	if strings.Contains(html, "/price/stock/meta?aralik=1d") {
		t.Errorf("aktif sekme kendine bağlantı veriyor")
	}
	// Performans satırı dönemleri.
	for _, ad := range []string{"24 hours", "5 days", "1 month", "YTD", "1 year"} {
		if !strings.Contains(html, ad) {
			t.Errorf("performans dönemi yok: %s", ad)
		}
	}
}

// TestCDHisseH1Gizli CoinDesk'te görünür h1 yok; erişilebilirlik için
// gizli h1 basılır (ekranda görünmez, a11y/SEO görür).
func TestCDHisseH1Gizli(t *testing.T) {
	html := hisseCiz(t, ornekHisse())
	if !strings.Contains(html, `<h1 class="cd-visually-hidden">Meta Platforms, Inc. fiyatı</h1>`) {
		t.Errorf("gizli h1 beklenen biçimde değil")
	}
	// Görünür (sınıfsız) h1 OLMAMALI.
	if strings.Contains(html, "<h1>Meta") {
		t.Errorf("görünür h1 basılmış")
	}
}

// TestCDHisseGrafikYok veri yoksa svg HİÇ basılmamalı (uydurma çizgi yok),
// sekmeler de anlamsız olduğu için basılmamalı.
func TestCDHisseGrafikYok(t *testing.T) {
	v := ornekHisse()
	v.GrafikVar = false
	v.Yol, v.Alan = "", ""
	v.Sekmeler = nil
	v.Performans = nil
	html := hisseCiz(t, v)
	if strings.Contains(html, "cd-hisse__grafik") {
		t.Errorf("veri yokken grafik basılmış")
	}
	if !strings.Contains(html, "cd-hisse__bos") {
		t.Errorf("boş durum metni yok")
	}
}

// TestCDHisseOlculer CSS'in CoinDesk §10.6 ölçülerini taşıdığını kilitler.
// (Tarayıcı ölçümü ayrı adım; bu test sessiz sapmayı yakalar.)
func TestCDHisseOlculer(t *testing.T) {
	css := cdCSS(t)
	for _, k := range []struct{ ad, deger string }{
		{"ızgara 392 + esnek", "grid-template-columns: 392px minmax(0, 1fr)"},
		{"grafik 500 px", "height: 500px"},
		{"performans 88 px", "min-height: 88px"},
		{"içerik bandı 1432", "max-width: var(--cd-max-section)"},
		{"tek kolon ≤1279", "@media (max-width: 1279px)"},
	} {
		if !strings.Contains(css, k.deger) {
			t.Errorf("%s eksik: %q", k.ad, k.deger)
		}
	}
	// Sağ ray ayracı gibi dikey çizgi YOK (hisse sayfası iki kolon + boşluk).
	if strings.Contains(css, ".cd-hisse__sol { border-right") {
		t.Errorf("sol raya dikey ayraç eklenmiş")
	}
}

// TestCDHisseHariciJSYok CSP `script-src 'self'`: sayfa HARİCİ script
// eklemez; grafik sunucuda üretiliyor. (Kabuk zaten site geneli CSP
// betiğini basıyor — onun dışında ek yok.)
func TestCDHisseHariciJSYok(t *testing.T) {
	html := hisseCiz(t, ornekHisse())
	if strings.Contains(html, "<script src=\"http") {
		t.Errorf("harici script basılmış")
	}
	if strings.Contains(html, "cd-hisse.js") || strings.Contains(html, "chart.js") {
		t.Errorf("hisse sayfası grafik kütüphanesi yüklemiş")
	}
}
