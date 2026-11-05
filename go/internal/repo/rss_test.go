package repo

import (
	"strings"
	"testing"
)

// E-2 testleri: bot çekimlerinin yarısı ayrıştırma hatası veriyordu.
// Üç kök neden ayrı ayrı korunuyor.

func TestSanitizeXMLNoktaliVirgulsuzVarlik(t *testing.T) {
	// "&l" gibi noktalı virgülsüz yazımlar XML ayrıştırmasını tamamen
	// durduruyordu; bilinen varlık (&nbsp;) ise gerçek karaktere çevrilir.
	in := `<item><title>A&nbsp; B &l bir şey</title></item>`
	out := sanitizeXML(in)
	if strings.Contains(out, "&l ") {
		t.Fatalf("çıplak & kaçırılmadı: %s", out)
	}
	if !strings.Contains(out, "&amp;l ") {
		t.Fatalf("beklenen kaçırma yok: %s", out)
	}
	if !strings.Contains(out, "\u00a0") {
		t.Fatalf("&nbsp; çözülmedi: %q", out)
	}
}

func TestSanitizeXMLGecerliVarliklarKorunur(t *testing.T) {
	in := `<title>a &amp; b &lt;c&gt; &#8217;</title>`
	out := sanitizeXML(in)
	if out != `<title>a &amp; b &lt;c&gt; ’</title>` {
		t.Fatalf("beklenmeyen çıktı: %s", out)
	}
}

func TestSanitizeXMLCDataKorunur(t *testing.T) {
	in := `<description><![CDATA[Fiyat 5 & 10 <b>arttı</b> &nbsp;]]></description>`
	out := sanitizeXML(in)
	if out != in {
		t.Fatalf("CDATA değiştirilmemeliydi:\n gelen: %s\n çıkan: %s", in, out)
	}
}

func TestParseRSSNoktaliVirgulsuzVarliklaAyristirir(t *testing.T) {
	doc := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0"><channel>
<item><title>Türkiye&rsquo;de &nbsp;gündem</title><link>https://ornek/1</link>
<description>Fiyat 5 & 10 arttı &l devam</description><category>Ekonomi</category></item>
</channel></rss>`
	items, err := parseRSS(doc)
	if err != nil {
		t.Fatalf("ayrıştırma hatası: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("1 öğe beklenirdi, %d geldi", len(items))
	}
	if items[0].Title == "" || items[0].Link != "https://ornek/1" {
		t.Fatalf("öğe alanları boş: %+v", items[0])
	}
	if len(items[0].Categories) != 1 || items[0].Categories[0] != "Ekonomi" {
		t.Fatalf("kategori taşınmadı: %+v", items[0].Categories)
	}
}

func TestParseRSSRDFBicimi(t *testing.T) {
	// DW Türkçe RSS 1.0 (RDF) yayımlıyor: öğeler kökün altında, tarih dc:date.
	doc := `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns="http://purl.org/rss/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel rdf:about="https://rss.dw.com/rdf/rss-tur-all"><title>DW</title></channel>
  <item rdf:about="https://dw.com/1">
    <title>Avrupa'da enerji fiyatları</title>
    <link>https://dw.com/1</link>
    <description>Doğal gaz fiyatları yükseldi.</description>
    <dc:date>2026-10-27T10:00:00Z</dc:date>
  </item>
</rdf:RDF>`
	items, err := parseRSS(doc)
	if err != nil {
		t.Fatalf("RDF ayrıştırma hatası: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("1 öğe beklenirdi, %d geldi", len(items))
	}
	if items[0].PubDate.IsZero() {
		t.Fatal("dc:date okunmadı")
	}
	if !strings.Contains(items[0].Description, "Doğal gaz") {
		t.Fatalf("açıklama taşınmadı: %q", items[0].Description)
	}
}

func TestLooksLikeHTML(t *testing.T) {
	if !looksLikeHTML("text/html; charset=utf-8", []byte("x")) {
		t.Fatal("content-type html tanınmadı")
	}
	if !looksLikeHTML("", []byte("<!DOCTYPE html><html>")) {
		t.Fatal("doctype tanınmadı")
	}
	if looksLikeHTML("application/rss+xml", []byte(`<?xml version="1.0"?><rss>`)) {
		t.Fatal("RSS yanlışlıkla HTML sayıldı")
	}
}

func TestDecodeCharsetWindows1254(t *testing.T) {
	// "Türkiye" windows-1254'te UTF-8 değildir; çözücü düzeltmeli.
	raw := []byte{0x54, 0xfc, 0x72, 0x6b, 0x69, 0x79, 0x65} // Türkiye
	got, err := decodeCharset(raw, "application/rss+xml; charset=windows-1254")
	if err != nil {
		t.Fatalf("hata: %v", err)
	}
	if string(got) != "Türkiye" {
		t.Fatalf("çözülen: %q", string(got))
	}
}

func TestDecodeCharsetBildirimYanlissaWindows1254eDuser(t *testing.T) {
	// Başlık utf-8 diyor ama gövde değil (Türkçe sitelerde yaygın).
	raw := []byte{0x47, 0xfc, 0x6e, 0x64, 0x65, 0x6d} // Gündem
	got, _ := decodeCharset(raw, "text/xml; charset=utf-8")
	if string(got) != "Gündem" {
		t.Fatalf("çözülen: %q", string(got))
	}
}
