package httpapi

import (
	"strings"
	"testing"
)

// TestCSPReklamAglariIzinli: reklam ağlarının GERÇEKTEN ihtiyaç duyduğu
// alan adları CSP'de olmalı.
//
// NEDEN BU TEST: iki kez üst üste yanlış alan adı setiyle yayına çıkıldı.
//  1. Hiçbiri yoktu → betikler bloklandı, reklam yüklenmedi.
//  2. Yalnız betik sunucuları eklendi (coinzillatag.com) → betikler yüklendi
//     ama ASIL REKLAM `request-global.czilladx.com`'dan geldiği için yine
//     görünmedi (canlı tarayıcı konsolu, 13 Kasım 2026).
//
// Bu test hem betik hem İÇERİK hem izleyici uçlarını kilitler.
func TestCSPReklamAglariIzinli(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")

	// direktif → o direktifte bulunması ZORUNLU alan adları
	gerekli := map[string][]string{
		"script-src": {
			"coinzillatag.com",              // Coinzilla betikleri
			"appsha1.cointraffic.io",        // Cointraffic betiği
			"pagead2.googlesyndication.com", // AdSense
		},
		"connect-src": {
			"request-global.czilladx.com", // Coinzilla REKLAM İÇERİĞİ
			"apps.adsgravity.io",          // Cointraffic izleyici
			"pagead2.googlesyndication.com",
		},
		"frame-src": {
			"request-global.czilladx.com", // reklam iframe'i
			// Cointraffic izleyicisi kendi iframe'ini basiyor; eksikse canli
			// konsolda "violates frame-src" hatasi cikiyordu (olculdu).
			"apps.adsgravity.io",
			"googleads.g.doubleclick.net",
			"securepubads.g.doubleclick.net",
		},
		"img-src": {
			"https:", // reklam görselleri her https kaynaktan gelebilir
		},
	}

	for direktif, alanlar := range gerekli {
		parca := direktifParcasi(csp, direktif)
		if parca == "" {
			t.Fatalf("%s direktifi CSP'de yok", direktif)
		}
		for _, a := range alanlar {
			if !strings.Contains(parca, a) {
				t.Errorf("%s içinde %q yok — reklam bu uçtan gelemez.\n  %s", direktif, a, parca)
			}
		}
	}
}

// TestCSPGuvenlikDuraklari: reklam izinleri gevşetme adına tehlikeli
// anahtar kelimeler eklenmemeli.
func TestCSPGuvenlikDuraklari(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")

	yasak := []string{
		"'unsafe-eval'", // eval() — XSS yüzeyi
		"script-src *",  // her kaynak
		"script-src 'self' *",
		"default-src *",
	}
	for _, y := range yasak {
		if strings.Contains(csp, y) {
			t.Errorf("CSP'de tehlikeli izin var: %q", y)
		}
	}
	// script-src'de joker host olmamalı ("https:" tek başına her kaynağı
	// açar). DİKKAT: `strings.Contains(sc, " https:")` YANLIŞ olurdu —
	// " https://x" bu dizgeyi de içerir; bu yüzden token'lara ayırıp
	// TAM eşleşme aranıyor.
	for _, tok := range strings.Fields(direktifParcasi(csp, "script-src")) {
		if tok == "https:" || tok == "http:" || tok == "*" {
			t.Errorf("script-src joker kaynak içeriyor: %q", tok)
		}
	}
	// object-src none ve base-uri self korunmalı (CSP'nin temel sertleştirmesi).
	for _, z := range []string{"object-src 'none'", "base-uri 'self'", "frame-ancestors 'self'"} {
		if !strings.Contains(csp, z) {
			t.Errorf("temel sertleştirme kaybolmuş: %q", z)
		}
	}
}

// direktifParcasi "ad ...;" parçasını döndürür.
func direktifParcasi(csp, ad string) string {
	for _, p := range strings.Split(csp, ";") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, ad+" ") || p == ad {
			return p
		}
	}
	return ""
}
