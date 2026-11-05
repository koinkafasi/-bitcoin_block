package httpapi

import (
	"strings"
	"testing"
)

// TestCSPIhlalleriniCoz rapor biçimlerinin ortak yapıya çevrilmesi.
func TestCSPIhlalleriniCoz(t *testing.T) {
	// report-uri (eski biçim)
	eski := []byte(`{"csp-report":{
		"document-uri":"https://theblock.com.tr/haber/test?utm=abc&kisisel=1",
		"blocked-uri":"https://evil.example/yol?gizli=2",
		"violated-directive":"script-src 'self'",
		"effective-directive":"script-src-elem",
		"disposition":"enforce"}}`)
	got := cspIhlalleriniCoz("application/csp-report", eski)
	if len(got) != 1 {
		t.Fatalf("1 ihlal beklenirdi, alınan %d", len(got))
	}
	g := got[0]
	if g.Direktif != "script-src-elem" {
		t.Errorf("direktif = %q", g.Direktif)
	}
	// GİZLİLİK: yol ve sorgu atılmalı; yalnız şema+host kalmalı.
	if g.Engellenen != "https://evil.example" {
		t.Errorf("engellenen sadeleşmedi: %q", g.Engellenen)
	}
	if strings.Contains(g.Engellenen, "gizli") {
		t.Error("sorgu dizesi sızdı")
	}
	if g.Belge != "/haber/test" {
		t.Errorf("belge yolu = %q (sorgu atılmalı)", g.Belge)
	}
	if g.Kip != "enforce" {
		t.Errorf("kip = %q", g.Kip)
	}

	// report-to (yeni biçim)
	yeni := []byte(`[{"type":"csp-violation","body":{
		"documentURL":"https://theblock.com.tr/admin",
		"blockedURL":"inline",
		"effectiveDirective":"script-src",
		"disposition":"report"}}]`)
	got2 := cspIhlalleriniCoz("application/reports+json", yeni)
	if len(got2) != 1 {
		t.Fatalf("1 ihlal beklenirdi, alınan %d", len(got2))
	}
	if got2[0].Engellenen != "inline" {
		t.Errorf("satır içi ihlali 'inline' olmalı: %q", got2[0].Engellenen)
	}
	if got2[0].Kip != "report" {
		t.Errorf("kip = %q", got2[0].Kip)
	}

	// İlgisiz rapor türü atlanır.
	diger := []byte(`[{"type":"deprecation","body":{}}]`)
	if got3 := cspIhlalleriniCoz("application/reports+json", diger); len(got3) != 0 {
		t.Errorf("ilgisiz rapor atlanmalıydı: %v", got3)
	}

	// Bozuk JSON panik üretmemeli.
	if got4 := cspIhlalleriniCoz("application/csp-report", []byte("{bozuk")); len(got4) != 0 {
		t.Errorf("bozuk JSON boş dönmeliydi: %v", got4)
	}
}

// TestTemizDirektif yalnız bilinen direktiflerin geçmesi.
func TestTemizDirektif(t *testing.T) {
	cases := map[string]string{
		"script-src":               "script-src",
		"SCRIPT-SRC-ELEM":          "script-src-elem",
		"script-src 'self' https:": "script-src",
		"":                         "bilinmeyen",
		"uydurma-direktif":         "diger",
	}
	for in, want := range cases {
		if got := temizDirektif(in); got != want {
			t.Errorf("temizDirektif(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

// TestSadelestirURL PII (yol/sorgu) sızmamalı.
func TestSadelestirURL(t *testing.T) {
	cases := map[string]string{
		"https://a.example/yol?x=1": "https://a.example",
		"HTTP://A.EXAMPLE/X":        "http://a.example",
		"inline":                    "inline",
		"eval":                      "eval",
		"data":                      "data",
		"":                          "",
		"bozuk":                     "diger",
	}
	for in, want := range cases {
		if got := sadelestirURL(in); got != want {
			t.Errorf("sadelestirURL(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

// TestReportDirektifi rapor adresi politikaya ekleniyor mu.
func TestReportDirektifi(t *testing.T) {
	csp := buildCSP(nil, nil, "", "https://theblock.com.tr/csp-rapor")
	if !strings.Contains(csp, "report-uri https://theblock.com.tr/csp-rapor") {
		t.Errorf("report-uri eksik: %s", csp)
	}
	if !strings.Contains(csp, "report-to csp") {
		t.Errorf("report-to grubu eksik: %s", csp)
	}
	// Adres yoksa direktif de eklenmemeli.
	bos := buildCSP(nil, nil, "", "")
	if strings.Contains(bos, "report-uri") || strings.Contains(bos, "report-to") {
		t.Errorf("rapor adresi yokken direktif olmamalı: %s", bos)
	}
}
