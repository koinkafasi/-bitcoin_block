package repo

import (
	"strings"
	"testing"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>theblock.com.tr</title><link>/</link><description>açıklama</description>
<item><title>Başlık Bir</title><link>/baslik-bir-1</link><description>ilk özet</description><pubDate>Mon, 14 Oct 2026 03:00:00 GMT</pubDate></item>
<item><title>Başlık İki</title><link>/baslik-iki-2</link><description>ikinci özet</description><pubDate>Tue, 15 Oct 2026 03:00:00 GMT</pubDate></item>
</channel></rss>`

func TestParseRSS(t *testing.T) {
	items, err := parseRSS(sampleRSS)
	if err != nil {
		t.Fatalf("parse hatası: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("2 öğe bekleniyor, %d", len(items))
	}
	if items[0].Title != "Başlık Bir" || items[0].Link != "/baslik-bir-1" {
		t.Fatalf("ilk öğe: %+v", items[0])
	}
	if !strings.Contains(items[0].Description, "özet") {
		t.Fatalf("açıklama yok: %q", items[0].Description)
	}
	if items[0].PubDate.IsZero() {
		t.Fatal("pubDate ayrışmadı")
	}
}

func TestParseRSSCategories(t *testing.T) {
	doc := `<rss version="2.0"><channel><item>
<title>Kategori testi</title><link>https://x.test/a-1</link>
<category>Gündem</category><category><![CDATA[Ekonomi]]></category>
</item></channel></rss>`
	items, err := parseRSS(doc)
	if err != nil || len(items) != 1 {
		t.Fatalf("parse: %v %d", err, len(items))
	}
	if len(items[0].Categories) != 2 || items[0].Categories[0] != "Gündem" || items[0].Categories[1] != "Ekonomi" {
		t.Fatalf("kategoriler: %#v", items[0].Categories)
	}
}

func TestParseAtomCategories(t *testing.T) {
	doc := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
<entry><title>Atom kategori</title><link href="https://x.test/a-2"/>
<category term="Spor" label="Spor"/></entry></feed>`
	items, err := parseRSS(doc)
	if err != nil || len(items) != 1 {
		t.Fatalf("parse: %v %d", err, len(items))
	}
	if len(items[0].Categories) != 1 || items[0].Categories[0] != "Spor" {
		t.Fatalf("kategoriler: %#v", items[0].Categories)
	}
}

func TestParseAtom(t *testing.T) {
	atom := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
<entry><title>Atom Başlık</title><link href="https://x.test/a-1"/><summary>özet metin</summary></entry></feed>`
	items, err := parseRSS(atom)
	if err != nil {
		t.Fatalf("atom hatası: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Atom Başlık" {
		t.Fatalf("atom öğe: %+v", items)
	}
}
