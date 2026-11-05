package templkit

import (
	"context"
	"strings"
	"testing"
)

// Menü, COINDESK-SPEC.md §3'teki ÖLÇÜLEN yapıya birebir oturmalı.
//
// Bu testler tasarımı değil, YAPIYI korur: biri menüye "iyileştirme" diye
// kalem eklerse ya da sırayı bozarsa burada kırılır. Ölçüm belgesi
// değişmeden menü değişmemeli.

func menuCiz(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	s := CDShell{SiteName: "bitcosar", LangBase: ""}
	if err := cdMegaNav(s).Render(context.Background(), &sb); err != nil {
		t.Fatalf("menü çizilemedi: %v", err)
	}
	return sb.String()
}

// Üst gezinmedeki 7 başlık, ölçülen SIRAYLA.
func TestMenuUstBasliklarSirali(t *testing.T) {
	h := menuCiz(t)
	beklenen := []string{"News", "Video", "Prices", "Research", "Events", "Data & Indices", "Sponsored"}

	// &, şablonda &amp; olarak kaçar.
	kacirilmis := strings.ReplaceAll(h, "&amp;", "&")

	son := -1
	for _, b := range beklenen {
		i := strings.Index(kacirilmis[son+1:], ">"+b)
		if i < 0 {
			t.Fatalf("üst başlık eksik ya da sırası bozuk: %q", b)
		}
		son = son + 1 + i
	}
	if n := len(cdMenuGroups()); n != len(beklenen) {
		t.Fatalf("üst başlık sayısı %d, beklenen %d (spec §3)", n, len(beklenen))
	}
}

// Ölçümde Research ve Events'te açılır menü YOK; diğer beşinde VAR.
func TestMenuAcilirOlanlar(t *testing.T) {
	acilirYok := map[string]bool{"Research": true, "Events": true}
	for _, g := range cdMenuGroups() {
		var_ := cdHasMenu(g)
		if acilirYok[g.Label] && var_ {
			t.Errorf("%q açılır menü taşımamalı (spec §3)", g.Label)
		}
		if !acilirYok[g.Label] && !var_ {
			t.Errorf("%q açılır menü taşımalı (spec §3)", g.Label)
		}
	}
}

// News menüsündeki ölçülen satırlar eksiksiz olmalı.
func TestMenuNewsIcerigi(t *testing.T) {
	h := menuCiz(t)
	for _, s := range []string{
		"Latest Crypto News", "Markets", "Finance", "Tech", "Policy", "Focus",
		"Newsletters", "CoinDesk Headlines", "Crypto Daybook Americas",
		"State of Crypto", "Crypto Long &amp; Short", "Crypto for Advisors",
		"Tokenization Weekly", "CoinDesk Insider",
		"Bitcoin News", "Ethereum News", "XRP News", "Solana News",
	} {
		if !strings.Contains(h, s) {
			t.Errorf("News menüsünde eksik: %q", s)
		}
	}
}

// Prices menüsü: ölçülen 5 kripto + 5 hisse + iki "tümünü gör".
func TestMenuPricesIcerigi(t *testing.T) {
	h := menuCiz(t)
	for _, s := range []string{
		"Bitcoin (BTC)", "Ethereum (ETH)", "XRP (XRP)", "Solana (SOL)", "Hyperliquid (HYPE)",
		"S&amp;P 500 (SPY)", "Bullish (BLSH)", "Circle (CRCL)", "Coinbase (COIN)", "Strategy (MSTR)",
		"See all crypto prices", "See all US stock prices",
	} {
		if !strings.Contains(h, s) {
			t.Errorf("Prices menüsünde eksik: %q", s)
		}
	}
	if n := strings.Count(h, "cd-mega__tumu"); n < 2 {
		t.Errorf(`"tümünü gör" satırı %d, en az 2 bekleniyor`, n)
	}
}

// URL'ler §4'teki desene bağlı olmalı; uydurma yol olmamalı.
func TestMenuYollariSpecDesenine_Uygun(t *testing.T) {
	gecerli := []string{
		"/latest-crypto-news", "/markets", "/business", "/tech", "/policy",
		"/focus", "/opinion", "/news-analysis", "/press-release",
		"/sponsored-content", "/video", "/podcasts", "/research", "/price",
		"/index", "/tag/", "/newsletters/", "/events",
	}
	for _, g := range cdMenuGroups() {
		tum := []CDMenuItem{{Label: g.Label, Href: g.Href}}
		for _, s := range g.Columns {
			tum = append(tum, s.Items...)
		}
		for _, it := range tum {
			if it.Kind == "baslik" {
				if it.Href != "" {
					t.Errorf("grup başlığı %q bağlantı taşımamalı", it.Label)
				}
				continue
			}
			if it.Href == "" {
				t.Errorf("%q boş Href", it.Label)
				continue
			}
			uygun := false
			for _, p := range gecerli {
				if strings.HasPrefix(it.Href, p) {
					uygun = true
					break
				}
			}
			if !uygun {
				t.Errorf("%q yolu spec §4 desenlerine uymuyor: %s", it.Label, it.Href)
			}
		}
	}
}

// Spec kuralı: stil şablonda değil CSS'te. Satır içi style olmamalı.
func TestMenuSatirIciStilYok(t *testing.T) {
	h := menuCiz(t)
	if strings.Contains(h, "style=") {
		t.Error("menüde satır içi style var; ölçüler cd-menu.css'te olmalı")
	}
}

// Erişilebilirlik: açılır başlıkta aria-haspopup ve panelle eşleşen id.
func TestMenuErisilebilirlik(t *testing.T) {
	h := menuCiz(t)
	if n := strings.Count(h, `aria-haspopup="true"`); n != 5 {
		t.Errorf("aria-haspopup sayısı %d, beklenen 5 (açılır menüsü olanlar)", n)
	}
	for _, g := range cdMenuGroups() {
		if !cdHasMenu(g) {
			continue
		}
		id := cdMenuID(g.Label)
		if !strings.Contains(h, `aria-controls="`+id+`"`) {
			t.Errorf("%q için aria-controls=%q yok", g.Label, id)
		}
		if !strings.Contains(h, `id="`+id+`"`) {
			t.Errorf("%q için panel id=%q yok", g.Label, id)
		}
	}
}

// LangBase verilince tüm bağlantılar ön ekle çıkmalı (çok dillilik).
func TestMenuDilOnEki(t *testing.T) {
	var sb strings.Builder
	s := CDShell{SiteName: "bitcosar", LangBase: "/tr"}
	if err := cdMegaNav(s).Render(context.Background(), &sb); err != nil {
		t.Fatalf("çizilemedi: %v", err)
	}
	h := sb.String()
	if !strings.Contains(h, `href="/tr/markets"`) {
		t.Error(`LangBase uygulanmamış: href="/tr/markets" bekleniyordu`)
	}
	if strings.Contains(h, `href="/markets"`) {
		t.Error("ön eksiz bağlantı kalmış")
	}
}
