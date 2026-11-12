package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var oluHedefler = []string{
	"/video/coindesk-daily", "/video/spotlight", "/video/shorts",
	"/newsletters/coindesk-headlines", "/newsletters/crypto-daybook-americas",
	"/index/cd20", "/index/xbx", "/index/etx",
	"/price/stock/spy", "/price/stock/coin",
	"/events",
}

// TestCDMenuOluHedefYok: menü tanımında ölü hedef kalmamalı.
func TestCDMenuOluHedefYok(t *testing.T) {
	// cdMenuSabit doğrudan incelenir (Go verisi).
	var hedefler []string
	for _, g := range cdMenuSabit {
		hedefler = append(hedefler, g.Href)
		for _, sut := range g.Columns {
			for _, it := range sut.Items {
				if it.Href != "" {
					hedefler = append(hedefler, it.Href)
				}
			}
		}
	}
	for _, olu := range oluHedefler {
		for _, h := range hedefler {
			assert.NotEqual(t, olu, h,
				"menüde ölü hedef var: %s (karşılığı yok, 404 döner)", olu)
		}
	}
}

func TestCDMenuSabitHedeflerVar(t *testing.T) {
	for _, g := range cdMenuSabit {
		assert.True(t, strings.HasPrefix(g.Href, "/"), "grup hedefi yol olmalı: %q", g.Href)
		for _, sut := range g.Columns {
			for _, it := range sut.Items {
				if it.Kind == "baslik" {
					assert.Empty(t, it.Href, "başlık satırı bağlantı taşımaz: %q", it.Label)
					continue
				}
				assert.True(t, strings.HasPrefix(it.Href, "/"),
					"menü satırı yol olmalı: %q → %q", it.Label, it.Href)
			}
		}
	}
}

func TestCDMenuVeriYoksaSutunBasilmaz(t *testing.T) {
	bos := CDShell{}
	gruplar := cdMenuGroups(bos)
	for _, g := range gruplar {
		if g.Label == "News" {
			assert.Len(t, g.Columns, 1, "etiket verisi yokken News tek sütun olmalı")
		}
		if g.Href == "/price" {
			require.Len(t, g.Columns, 2)
			var etiketler []string
			for _, it := range g.Columns[0].Items {
				etiketler = append(etiketler, it.Label)
			}
			assert.Contains(t, etiketler, "Tüm kripto fiyatlarını gör",
				"fiyat verisi yokken yine de 'tümü' satırı olmalı")
		}
	}
}

// TestCDMenuVeriVarsaEklenir: veri gelince sütun dolar.
func TestCDMenuVeriVarsaEklenir(t *testing.T) {
	dolu := CDShell{
		MenuTags: []CDMenuItem{
			{Label: "Bitcoin News", Href: "/tag/bitcoin"},
			{Label: "Ethereum News", Href: "/tag/ethereum"},
		},
		MenuCoins: []CDMenuItem{
			{Label: "Bitcoin (BTC)", Href: "/price/btc"},
		},
	}
	gruplar := cdMenuGroups(dolu)
	var news, prices *CDMenuGroup
	for i := range gruplar {
		switch gruplar[i].Href {
		case "/latest-crypto-news":
			news = &gruplar[i]
		case "/price":
			prices = &gruplar[i]
		}
	}
	require.NotNil(t, news)
	require.NotNil(t, prices)
	assert.Len(t, news.Columns, 2, "etiket sütunu eklenmeli")
	assert.Len(t, prices.Columns[0].Items, 3, "başlık + 1 coin + tümü")
}

// TestCDMenuAktif: aktif başlık işaretlenir (aria-current için).
func TestCDMenuAktif(t *testing.T) {
	nav := []CDNavLink{
		{Label: "Markets", Href: "/markets", Active: true},
		{Label: "Tech", Href: "/tech"},
	}
	assert.True(t, cdMenuAktif(nav, "/markets"))
	assert.False(t, cdMenuAktif(nav, "/tech"), "aktif olmayan başlık işaretlenmez")
	assert.False(t, cdMenuAktif(nav, ""))
	assert.False(t, cdMenuAktif(nil, "/markets"))
}

// TestCDMenuIDBenzersiz: aria-controls kimlikleri çakışmamalı.
func TestCDMenuIDBenzersiz(t *testing.T) {
	gorulen := map[string]string{}
	for _, g := range cdMenuSabit {
		id := cdMenuID(g.Label)
		if onceki, varMi := gorulen[id]; varMi {
			t.Errorf("menü kimliği çakışıyor: %q ve %q → %s", onceki, g.Label, id)
		}
		gorulen[id] = g.Label
	}
}

func TestCDMenuBosBolumlerElenir(t *testing.T) {
	s := CDShell{MenuBolumler: map[string]bool{
		"markets": true, "policy": true, "tech": true, "business": true,
		"opinion": true, "research": true,
	}}
	gruplar := cdMenuGroups(s)

	var linkler []string
	for _, g := range gruplar {
		for _, sut := range g.Columns {
			for _, it := range sut.Items {
				linkler = append(linkler, it.Href)
			}
		}
	}
	icinde := func(href string) bool {
		for _, l := range linkler {
			if l == href {
				return true
			}
		}
		return false
	}
	for _, bos := range []string{"/videos", "/podcasts", "/focus", "/news-analysis", "/press-release"} {
		if icinde(bos) {
			t.Errorf("içeriği olmayan bölüm menüde: %s", bos)
		}
	}
	// İçeriği olan bölümler KALMALI (filtre fazla eleme yapmamalı).
	for _, dolu := range []string{"/markets", "/policy", "/tech", "/business", "/opinion"} {
		if !icinde(dolu) {
			t.Errorf("içeriği olan bölüm menüden düştü: %s", dolu)
		}
	}
	// Tüm sütunları boşalan "Video" grubu (başlığı da boş sayfaya gider)
	// hiç basılmamalı.
	for _, g := range gruplar {
		if g.Label == "Video" {
			t.Error("boş Video grubu menüde kaldı")
		}
	}
}

// TestCDMenuBolumVerisiYoksaFiltreYok: veri yüklenmemişse menü boşaltılmaz.
func TestCDMenuBolumVerisiYoksaFiltreYok(t *testing.T) {
	gruplar := cdMenuGroups(CDShell{})
	var videoVar bool
	for _, g := range gruplar {
		if g.Label == "Video" {
			videoVar = true
		}
	}
	if !videoVar {
		t.Error("MenuBolumler boşken menü süzülmemeli (panel ve test sayfaları)")
	}
}

// TestCDBolumSlug: yalnız tek parçalı yollar bölüm sayılır.
func TestCDBolumSlug(t *testing.T) {
	for giris, beklenen := range map[string]string{
		"/videos":             "videos",
		"/focus":              "focus",
		"/tag/bitcoin":        "",
		"/latest-crypto-news": "latest-crypto-news",
		"/":                   "",
		"":                    "",
		"/price/bitcoin":      "",
	} {
		if got := cdBolumSlug(giris); got != beklenen {
			t.Errorf("cdBolumSlug(%q) = %q, beklenen %q", giris, got, beklenen)
		}
	}
}

func TestCDMenuHisseSutunu(t *testing.T) {
	for _, kabuk := range []CDShell{
		{}, // fiyat servisi yok
		{MenuCoins: []CDMenuItem{{Label: "Bitcoin (BTC)", Href: "/price/bitcoin"}}},
	} {
		var prices *CDMenuGroup
		gruplar := cdMenuGroups(kabuk)
		for i := range gruplar {
			if gruplar[i].Href == "/price" {
				prices = &gruplar[i]
			}
		}
		require.NotNil(t, prices, "Fiyatlar grubu yok")
		require.Len(t, prices.Columns, 2, "kripto + hisse sütunu beklenir")

		hisse := prices.Columns[1].Items
		require.NotEmpty(t, hisse)
		var hedefler []string
		for _, it := range hisse {
			hedefler = append(hedefler, it.Href)
		}
		assert.Contains(t, hedefler, "/borsa/hisseler", "hisse hedefi yok")
	}
}
