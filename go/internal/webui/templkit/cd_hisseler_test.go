package templkit

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HİSSE DİZİNİ SAYFASI (9 Kasım 2026).
//
// NEDEN TEST: sayfa theblock'tan kalan `site_public.html` bloğundan
// (`hsp-*` + site-header/site-footer) geliyordu. Yeni temaya geçmiş her
// sayfa `cd-*` kabuğunu kullanırken bu sayfa geride kalmıştı; kabuk
// sessizce eski şablona dönerse kullanıcı iki farklı site görür.
//
// Test iki şeyi kilitler:
//  1. sayfa cd-* kabuğunu kullanır (cd-topbar / cd-footer / cd-table),
//  2. eski kabuk sınıfları (site-header, hsp-table) BASILMAZ.
func TestCDHisselerSayfasiKabuk(t *testing.T) {
	v := CDHisselerPage{
		Shell: CDShell{
			SiteName:   "bitcosar",
			SearchHref: "/arama",
			LoginHref:  "/giris",
			SignupHref: "/kayit",
		},
		Title:     "Tüm Hisseler",
		Desc:      "BIST hisseleri",
		Canonical: "https://bitcosar.com/borsa/hisseler",
		Harfler:   []string{"A", "B", "C"},
		Adet:      633,
		Toplam:    633,
		Satir: []CDHisseSatir{
			{Code: "THYAO", Name: "TÜRK HAVA YOLLARI", Ilk: "T", Arama: "thyao turk hava yollari",
				Href: "/price/stock/THYAO.IS", Fiyat: "312,50", Yuzde: "%+1,25", Up: true, Var: true},
			{Code: "XXXXX", Name: "VERİSİZ", Ilk: "X", Href: "/borsa/hisseler?q=XXXXX"},
		},
	}

	var sb strings.Builder
	require.NoError(t, CDHisselerPageView(v).Render(context.Background(), &sb))
	h := sb.String()

	// Yeni kabuk.
	assert.Contains(t, h, `class="cd-topbar"`, "üst çubuk cd-topbar olmalı")
	assert.Contains(t, h, `class="cd-footer"`, "alt bilgi cd-footer olmalı")
	assert.Contains(t, h, `class="cd-table cd-hisse-tablo"`, "tablo cd-table olmalı")
	assert.Contains(t, h, `class="cd"`, "gövde sınıfı cd olmalı")

	// Eski kabuk KALINTISI OLMAMALI.
	for _, eski := range []string{"site-header", "site-footer", "hsp-table", "hsp-wrap", `class="page"`} {
		assert.NotContains(t, h, eski, "eski şablon kalıntısı: %s", eski)
	}

	// Veri satırları.
	assert.Contains(t, h, "THYAO")
	assert.Contains(t, h, "312,50")
	assert.Contains(t, h, "%+1,25")
	assert.Contains(t, h, "633")

	// Verisiz satır "—" basar, uydurma sayı YOK.
	assert.Contains(t, h, "—")

	// A-Z dizini: seçili harf yoksa "TÜMÜ" işaretli.
	assert.Contains(t, h, "TÜMÜ")
	assert.Contains(t, h, `aria-current="page"`)
}

// TestCDHisseHarfHref: A-Z bağlantısı arama metnini KORUR.
//
// NEDEN: kullanıcı "THY" yazıp sonra "T" harfine basarsa süzgeç
// kaybolmamalı; iki parametre birlikte taşınır.
func TestCDHisseHarfHref(t *testing.T) {
	assert.Equal(t, "/borsa/hisseler", cdHisseHarfHref("", ""))
	assert.Equal(t, "/borsa/hisseler?harf=A", cdHisseHarfHref("A", ""))
	assert.Equal(t, "/borsa/hisseler?q=THY", cdHisseHarfHref("", "THY"))
	assert.Equal(t, "/borsa/hisseler?harf=T&q=THY", cdHisseHarfHref("T", "THY"))
}
