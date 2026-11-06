package repo

import (
	"strings"
	"testing"
)

// Kaynak gövdesi çekici — saf ayrıştırma testleri.
//
// Ağ çağrısı YOK: okuyucu proxy'sinin biçimi 6 Kasım 2026'da gözlemlenip
// buraya sabitlendi. Biçim değişirse `govdeAyikla` boş paragraf döner,
// bot RSS özetiyle devam eder (zenginleştirme opsiyonel).

// okuyucuOrnegi r.jina.ai çıktısının kısaltılmış gerçek biçimidir.
const okuyucuOrnegi = `Title: 18 years in, what has Bitcoin truly done?

URL Source: https://www.coindesk.com/opinion/2026/10/31/18-years-in-what-has-bitcoin-truly-done

Markdown Content:
[Skip to main content](https://www.coindesk.com/opinion/2026/10/31/x#content)

Search

/

*   News [News](https://www.coindesk.com/latest-crypto-news)

## The Bitcoin white paper achieved a lot of what Satoshi Nakamoto set out to accomplish, but the cultural impact may outweigh Bitcoin itself.

![Image 5: (Photo by Janos Kummer/Getty Images)](https://www.coindesk.com/_next/image?url=https%3A%2F%2Fcdn.sanity.io%2Fimages%2Fs3y3vcno%2Fproduction.jpg)

Maybe crypto really has made it. Eighteen years after Satoshi Nakamoto published the Bitcoin white paper, it's unclear whether the oldest decentralized cryptocurrency has truly arrived.

Bitcoin [BTC$76,125.31](https://www.coindesk.com/price/bitcoin) itself is now firmly entrenched in the financial system, and that changes how the industry talks about itself.

1.   1[Over $1 billion in bullish bets liquidated](https://www.coindesk.com/daybook-us/2026/11/06/x)

Note: The views expressed in this column are those of the author.

## İlgili haberler
`

func TestGovdeAyiklaYalnizParagraflariSecer(t *testing.T) {
	g := govdeAyikla(okuyucuOrnegi)
	if len(g.Paragraflar) != 2 {
		t.Fatalf("paragraf sayısı %d, beklenen 2: %v", len(g.Paragraflar), g.Paragraflar)
	}
	// Gezinme satırı, ilgili haber listesi ve "Note:" sonrası gövdeye girmemeli.
	hepsi := g.Paragraflar[0] + " " + g.Paragraflar[1]
	for _, yasak := range []string{"Skip to main content", "latest-crypto-news", "bullish bets", "views expressed"} {
		if strings.Contains(hepsi, yasak) {
			t.Errorf("gövdeye sızma: %q", yasak)
		}
	}
	// Gömülü canlı fiyat rozeti temizlenir; cümle kalır.
	if strings.Contains(hepsi, "BTC$") {
		t.Errorf("fiyat rozeti temizlenmedi: %q", g.Paragraflar[1])
	}
	if !strings.Contains(hepsi, "Bitcoin itself is now firmly entrenched") {
		t.Errorf("gövde cümlesi kayboldu: %q", hepsi)
	}
	// Kapak görseli alınır (gerçek makale görseli, uydurma değil).
	if g.Gorsel == "" {
		t.Error("kapak görseli bulunamadı")
	}
}

func TestGovdeAyiklaBosGirdi(t *testing.T) {
	if g := govdeAyikla("Markdown Content:\nSearch\n/\n"); len(g.Paragraflar) != 0 {
		t.Errorf("paragraf üretilmemeliydi: %v", g.Paragraflar)
	}
}

func TestMarkdownTemizleBaglantiyiMetneCevirir(t *testing.T) {
	got := markdownTemizle("**Önemli** bağlantı [buraya](https://x.test) ve ETH$3,200.10 fiyatı.")
	if got != "Önemli bağlantı buraya ve fiyatı." {
		t.Errorf("beklenmeyen temizlik: %q", got)
	}
}

func TestSiteHostSorgudan(t *testing.T) {
	u := "https://www.bing.com/news/search?q=site%3Acoindesk.com%2Fopinion&format=rss"
	if got := siteHostSorgudan(u); got != "coindesk.com" {
		t.Errorf("host %q, beklenen coindesk.com", got)
	}
	if got := siteHostSorgudan("https://www.bing.com/news/search?q=kripto"); got != "" {
		t.Errorf("site: yokken boş dönmeliydi: %q", got)
	}
}

func TestApiclickHedef(t *testing.T) {
	ham := "http://www.bing.com/news/apiclick.aspx?ref=FexRss&url=https%3a%2f%2fwww.coindesk.com%2fopinion%2f2026%2f10%2f31%2fx&c=1"
	if got := apiclickHedef(ham); got != "https://www.coindesk.com/opinion/2026/10/31/x" {
		t.Errorf("hedef %q", got)
	}
	duz := "https://www.coindesk.com/opinion/2026/10/31/x"
	if got := apiclickHedef(duz); got != duz {
		t.Errorf("yönlendirme olmayan adres bozuldu: %q", got)
	}
	if got := apiclickHedef("javascript:alert(1)"); got != "" {
		t.Errorf("geçersiz şema kabul edildi: %q", got)
	}
}
