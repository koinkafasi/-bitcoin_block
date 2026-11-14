package httpapi

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"theblock/site/migrations"
)

// 0132 göçündeki yasal metinler geçerli TipTap belgesi olmalı ve sitenin
// işleyicisiyle tablo/bağlantı olarak basılabilmeli. Bozuk bir belge göçte
// hata vermez (jsonb geçerlidir), sayfada sessizce boş çıkar.
func TestYasalMetinlerBasilir(t *testing.T) {
	ham, err := migrations.FS.ReadFile("0132_yasal_metinler.up.sql")
	if err != nil {
		t.Fatalf("göç okunamadı: %v", err)
	}
	belgeler := regexp.MustCompile(`(?s)VALUES \('kurumsal', '[^\n]*', '([a-z-]+)', .*?\n\$tt\$(.*?)\$tt\$::jsonb`).
		FindAllStringSubmatch(string(ham), -1)
	if len(belgeler) != 4 {
		t.Fatalf("4 sayfa bekleniyordu, bulunan %d", len(belgeler))
	}

	s := &Server{}
	gorulen := map[string]string{}
	for _, b := range belgeler {
		slug, govde := b[1], b[2]
		if !json.Valid([]byte(govde)) {
			t.Fatalf("%s: geçersiz JSON", slug)
		}
		html := string(s.blocksToHTML([]byte(govde), nil))
		gorulen[slug] = html
		// Gizlilik politikası düz metin; diğer üçü tablo taşır.
		if slug != "gizlilik-politikasi" && (!strings.Contains(html, "<table") || !strings.Contains(html, "<th")) {
			t.Errorf("%s: tablo basılmadı", slug)
		}
		if !strings.Contains(html, `href="/`) {
			t.Errorf("%s: site içi bağlantı basılmadı", slug)
		}
		// İşaretleme artığı kalmamalı: çevrilmemiş bağlantı ya da tablo satırı.
		if strings.Contains(html, "](") || strings.Contains(html, "| ") {
			t.Errorf("%s: çevrilmemiş işaretleme kaldı", slug)
		}
		if !strings.Contains(html, "{eposta_kvkk}") {
			t.Errorf("%s: başvuru adresi yer tutucusu yok", slug)
		}
	}
	for _, slug := range []string{"kvkk", "gizlilik-politikasi", "cerez-politikasi", "gdpr"} {
		if gorulen[slug] == "" {
			t.Errorf("%s sayfası göçte yok", slug)
		}
	}

	// Metindeki süreler kodla eşleşmeli: konum kaydı 12 ay = konumSaklamaGun.
	if konumSaklamaGun != 365 || !strings.Contains(gorulen["kvkk"], "12 ay") {
		t.Errorf("konum saklama süresi metinle uyuşmuyor (kod: %d gün)", konumSaklamaGun)
	}
	// Çerez tablosu gerçek çerez adlarını taşımalı.
	for _, ad := range []string{vidCookie, geoCookie, konumCookie, "tb_session", "tb_riza"} {
		if !strings.Contains(gorulen["cerez-politikasi"], ad) {
			t.Errorf("çerez politikasında %s yok", ad)
		}
	}
}

// Ayar formu {"tr": "..."} biçimindeki başlığı okuyabilmeli; okuyamazsa alan
// boş gelir ve "Kaydet" başlığı siler.
func TestJSONYerelStr(t *testing.T) {
	cases := []struct{ girdi, beklenen string }{
		{`{"tr": "BİTCOŞAR"}`, "BİTCOŞAR"},
		{`{"tr": "", "en": "bitcosar"}`, "bitcosar"},
		{`"duz metin"`, "duz metin"},
		{`{"tr": ""}`, ""},
		{``, ""},
		{`[1,2]`, ""},
	}
	for _, c := range cases {
		if got := jsonYerelStr([]byte(c.girdi)); got != c.beklenen {
			t.Errorf("jsonYerelStr(%s) = %q, beklenen %q", c.girdi, got, c.beklenen)
		}
	}
}
