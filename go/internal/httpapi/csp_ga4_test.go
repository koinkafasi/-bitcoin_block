package httpapi

import (
	"strings"
	"testing"
)

// TestCSPGA4ToplamaIzinli — GA4 ölçüm istekleri CSP'den geçmeli.
//
// NEDEN: 14 Kasım 2026'da canlıda ölçüldü: `gtag.js` yükleniyordu ama
// her `page_view` isteği şu hatayla bloklanıyordu:
//
//	Connecting to 'https://analytics.google.com/g/collect?...' violates
//	the following Content Security Policy directive: "connect-src ..."
//
// Yani kimlik doğru basılıyor, betik yükleniyor, ama HİÇ veri
// gitmiyordu. Bu test, connect-src'nin GA4 v2 toplama uçlarını
// içerdiğini kilitler; biri listeyi "sadeleştirirse" analitik yine
// sessizce ölür.
func TestCSPGA4ToplamaIzinli(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")

	connect := ""
	for _, d := range strings.Split(csp, ";") {
		if strings.HasPrefix(strings.TrimSpace(d), "connect-src") {
			connect = d
			break
		}
	}
	if connect == "" {
		t.Fatal("connect-src bulunamadı")
	}

	for _, gerekli := range []string{
		"https://analytics.google.com",     // GA4 v2 /g/collect
		"https://*.analytics.google.com",   // bölgesel (region1…)
		"https://www.googletagmanager.com", // gtag yapılandırma
		"https://www.google-analytics.com", // eski uç, hâlâ kullanılıyor
	} {
		if !strings.Contains(connect, gerekli) {
			t.Errorf("connect-src %q içermiyor: %s", gerekli, connect)
		}
	}
}

// TestCSPReklamAglariKorunuyor — CSP daraltılırken reklam uçları
// düşmesin. 13 Kasım 2026'da bu alan adları eksik olduğu için reklamlar
// hiç yüklenmiyordu; GA4 düzeltmesi onları geri almamalı.
func TestCSPReklamAglariKorunuyor(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")
	for _, gerekli := range []string{
		"https://request-global.czilladx.com",
		"https://coinzillatag.com",
		"https://appsha1.cointraffic.io",
		"https://apps.adsgravity.io",
		"https://securepubads.g.doubleclick.net",
	} {
		if !strings.Contains(csp, gerekli) {
			t.Errorf("CSP %q içermiyor", gerekli)
		}
	}
}

// TestCSPTehlikeliJokerYok — geniş joker eklenmemeli.
//
// GA4 için `*.analytics.google.com` eklendi; bunun gibi bir joker
// tüm alan adlarına açılırsa CSP anlamsızlaşır. En azından şemayı
// serbest bırakan `connect-src *` / `https:` biçimleri kilitli.
func TestCSPTehlikeliJokerYok(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")
	connect := ""
	for _, d := range strings.Split(csp, ";") {
		if strings.HasPrefix(strings.TrimSpace(d), "connect-src") {
			connect = d
		}
	}
	if strings.Contains(connect, "connect-src *") {
		t.Errorf("connect-src joker içeriyor: %s", connect)
	}
	// Şema-kaynağı (`https:`) tek başına token olarak yasak: her adresi
	// açar. " https:" aramak yanıltıcıydı — HER https adresi o dizgeyi
	// içeriyor. Bu yüzden boşlukla ayrılmış token'lara bakıyoruz.
	for _, tok := range strings.Fields(connect) {
		if tok == "https:" || tok == "*" || tok == "http:" {
			t.Errorf("connect-src şema jokeri içeriyor (%q): %s", tok, connect)
		}
	}
}
