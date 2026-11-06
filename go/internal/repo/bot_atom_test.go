package repo

import "testing"

// Atom (RFC 4287) ayrıştırma sözleşmesi.
//
// ÖLÇÜM (6 Kasım 2026): Atom beslemeleri SIFIR kullanılabilir öğe
// veriyordu ve bu sessizdi — besleme 200 dönüyor, 15 `<entry>` geliyor,
// bot hiçbirini almıyordu. İki sebep:
//
//  1. Atom'da bağlantı eleman METNİ değil `<link href="…">` ÖZNİTELİĞİ.
//     `xml:"link"` boş dönüyor, bot `Link == ""` olanı atlıyor.
//  2. Kapak görseli gövde HTML'inde aranıyordu; Atom yayıncıları
//     `media:group/media:content@url` veriyor.
//
// Bu testler düzeltmeyi sabitler. Kırılırsa Atom beslemeleri yine sessizce
// boşa düşer — en pahalı hata sınıfı.

// YouTube beslemesinin GERÇEK yapısı (kısaltılmış, alanlar birebir).
const atomYouTube = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015"
      xmlns:media="http://search.yahoo.com/mrss/"
      xmlns="http://www.w3.org/2005/Atom">
 <title>CoinDesk</title>
 <link rel="self" href="http://www.youtube.com/feeds/videos.xml?user=CoinDesk"/>
 <entry>
  <id>yt:video:98v9WPLU4nw</id>
  <yt:videoId>98v9WPLU4nw</yt:videoId>
  <title>Fireblocks CEO on the Open Transaction Layer</title>
  <link rel="alternate" href="https://www.youtube.com/watch?v=98v9WPLU4nw"/>
  <published>2026-11-05T22:00:18+00:00</published>
  <updated>2026-11-06T01:10:00+00:00</updated>
  <media:group>
   <media:title>Fireblocks CEO on the Open Transaction Layer</media:title>
   <media:content url="https://www.youtube.com/v/98v9WPLU4nw?version=3" type="application/x-shockwave-flash" width="640" height="390"/>
   <media:thumbnail url="https://i4.ytimg.com/vi/98v9WPLU4nw/hqdefault.jpg" width="480" height="360"/>
   <media:description>Michael Shaulov anlatiyor.</media:description>
  </media:group>
 </entry>
</feed>`

// Bağlantı `href` özniteliğinden okunmalı.
func TestAtomBaglantiHrefOzniteligindenOkunur(t *testing.T) {
	items, err := parseRSS(atomYouTube)
	if err != nil {
		t.Fatalf("ayrıştırılamadı: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("öğe sayısı %d, 1 bekleniyordu", len(items))
	}
	const bek = "https://www.youtube.com/watch?v=98v9WPLU4nw"
	if items[0].Link != bek {
		t.Errorf("bağlantı %q, beklenen %q", items[0].Link, bek)
	}
	if items[0].Title == "" {
		t.Error("başlık boş")
	}
}

// rel="self" ÖĞENİN bağlantısı değildir; seçilmemeli.
func TestAtomSelfBaglantisiSecilmez(t *testing.T) {
	doc := `<feed xmlns="http://www.w3.org/2005/Atom"><entry>
	  <title>X</title>
	  <link rel="self" href="https://ornek.com/feed.xml"/>
	  <link rel="alternate" href="https://ornek.com/haber/1"/>
	</entry></feed>`
	items, err := parseRSS(doc)
	if err != nil || len(items) != 1 {
		t.Fatalf("ayrıştırma: %v (öğe %d)", err, len(items))
	}
	if items[0].Link != "https://ornek.com/haber/1" {
		t.Errorf("self seçilmiş: %q", items[0].Link)
	}
}

// rel yoksa varsayılan alternate'tir; bağlantı yine alınmalı.
func TestAtomRelYoksaBaglantiAlinir(t *testing.T) {
	doc := `<feed xmlns="http://www.w3.org/2005/Atom"><entry>
	  <title>X</title><link href="https://ornek.com/haber/2"/>
	</entry></feed>`
	items, _ := parseRSS(doc)
	if len(items) != 1 || items[0].Link != "https://ornek.com/haber/2" {
		t.Errorf("rel'siz bağlantı alınmadı: %+v", items)
	}
}

// Görsel media:group/thumbnail'dan okunmalı.
//
// media:content burada flash oynatıcı (geçerli görsel değil); seçim
// gecerliGorselURL'den geçtiği için thumbnail kazanmalı.
func TestAtomGorselMediaGroupdanOkunur(t *testing.T) {
	items, err := parseRSS(atomYouTube)
	if err != nil || len(items) != 1 {
		t.Fatalf("ayrıştırma: %v", err)
	}
	const bek = "https://i4.ytimg.com/vi/98v9WPLU4nw/hqdefault.jpg"
	if items[0].Image != bek {
		t.Errorf("görsel %q, beklenen %q", items[0].Image, bek)
	}
}

// `published`, `updated`'dan önce gelmeli.
//
// NEDEN: yeniden yayımlamada `updated` değişir ve eski içerik yeni gibi
// sıralanır. Fikstürde published 05 Kasım, updated 06 Kasım.
func TestAtomPublishedUpdatedOnunde(t *testing.T) {
	items, _ := parseRSS(atomYouTube)
	if len(items) != 1 {
		t.Fatal("öğe yok")
	}
	if items[0].PubDate.IsZero() {
		t.Fatal("tarih ayrıştırılamadı")
	}
	if g := items[0].PubDate.Day(); g != 5 {
		t.Errorf("gün %d, 5 bekleniyordu (published kullanılmalı, updated değil)", g)
	}
}

// Gövde boşsa media:description kullanılmalı.
func TestAtomGovdeMediaDescriptiondanDoldurulur(t *testing.T) {
	items, _ := parseRSS(atomYouTube)
	if len(items) != 1 {
		t.Fatal("öğe yok")
	}
	if items[0].Description == "" {
		t.Error("açıklama boş; media:description kullanılmalıydı")
	}
}

// RSS 2.0 davranışı BOZULMAMALI — Atom düzeltmesi regresyon getirmesin.
func TestRSS20HalaCalisiyor(t *testing.T) {
	doc := `<rss version="2.0"><channel><item>
	  <title>Haber</title>
	  <link>https://ornek.com/h/1</link>
	  <description>Spot metni</description>
	  <pubDate>Wed, 05 Nov 2026 10:00:00 GMT</pubDate>
	</item></channel></rss>`
	items, err := parseRSS(doc)
	if err != nil || len(items) != 1 {
		t.Fatalf("RSS 2.0 ayrıştırma bozuldu: %v (öğe %d)", err, len(items))
	}
	if items[0].Link != "https://ornek.com/h/1" || items[0].Title != "Haber" {
		t.Errorf("RSS 2.0 alanları yanlış: %+v", items[0])
	}
}
