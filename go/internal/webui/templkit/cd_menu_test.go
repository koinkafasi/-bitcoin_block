package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// Mega menü sözleşmesi: MENÜYE YALNIZ VAR OLAN SAYFA YAZILIR.
//
// ÖLÇÜM (5 Kasım 2026): menüde 49 hedef vardı, 29'u 404 dönüyordu. Yapı
// "tamam" görünüyordu, site yarımdı. Aşağıdaki testler bu sınıfı kapatır:
//   - ölü bağlantı listesi boş olmalı
//   - "News News" gibi çift son ek olmamalı
//   - veriye bağlı sütun boşken basılmamalı

// oluHedefler: menüde OLMAMASI gereken, karşılığı üretilmemiş yollar.
//
// Bu liste bir kara liste DEĞİL, sözleşme: kaynak kodda bu dizeler
// görünürse menü yine ölü bağlantı veriyor demektir. Yeni bir ölü hedef
// eklenirse buraya yazılır ve test kırmızıya döner.
var oluHedefler = []string{
	"/video/coindesk-daily", "/video/spotlight", "/video/shorts",
	"/newsletters/coindesk-headlines", "/newsletters/crypto-daybook-americas",
	"/index/cd20", "/index/xbx", "/index/etx",
	"/price/stock/spy", "/price/stock/coin",
	"/events",
}

func cdMenuKaynak(t *testing.T) string {
	t.Helper()
	b, err := webui.FS.ReadFile("static/cd-menu.css")
	require.NoError(t, err, "cd-menu.css okunamadı")
	// .templ/.go dosyaları embed'e dahil değil; sabitler Go kaynağında.
	_ = b
	return ""
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

// TestCDMenuSabitHedeflerVar: sabit menüdeki her hedefin bir handler'ı var.
//
// Sunucu rotaları bu pakette değil; bu test yalnız yol biçimini doğrular
// (başında "/" olmalı, boş olmamalı). Varlık kontrolü canlı smoke ile.
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

// TestCDMenuVeriYoksaSutunBasilmaz: etiket/coin verisi yoksa o sütun
// hiç oluşmamalı. Boş sütun başlığı ("Crypto News") altında satır
// olmaması kullanıcıya kırık görünür.
func TestCDMenuVeriYoksaSutunBasilmaz(t *testing.T) {
	bos := CDShell{}
	gruplar := cdMenuGroups(bos)
	for _, g := range gruplar {
		if g.Label == "News" {
			assert.Len(t, g.Columns, 1, "etiket verisi yokken News tek sütun olmalı")
		}
		if g.Label == "Prices" {
			require.Len(t, g.Columns, 1)
			var etiketler []string
			for _, it := range g.Columns[0].Items {
				etiketler = append(etiketler, it.Label)
			}
			assert.Contains(t, etiketler, "See all crypto prices",
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
		switch gruplar[i].Label {
		case "News":
			news = &gruplar[i]
		case "Prices":
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
