package youtube

import (
	"strings"
	"testing"
)

func TestRSSOkuTurlarBaglantidanBelirlenir(t *testing.T) {
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns:media="http://search.yahoo.com/mrss/">
 <title>BİTCOŞAR</title>
 <entry>
  <yt:videoId>KfLVTp9jlyU</yt:videoId>
  <title>Kısa video başlığı</title>
  <link rel="alternate" href="https://www.youtube.com/shorts/KfLVTp9jlyU"/>
  <published>2026-11-04T20:25:05+00:00</published>
  <media:group>
   <media:description>açıklama</media:description>
   <media:thumbnail url="https://i4.ytimg.com/vi/KfLVTp9jlyU/hqdefault.jpg"/>
   <media:community><media:statistics views="1270"/></media:community>
  </media:group>
 </entry>
 <entry>
  <yt:videoId>8oVvSLjkMv8</yt:videoId>
  <title>Uzun video başlığı</title>
  <link rel="alternate" href="https://www.youtube.com/watch?v=8oVvSLjkMv8"/>
  <published>2026-10-24T05:00:23+00:00</published>
  <media:group>
   <media:thumbnail url="https://i1.ytimg.com/vi/8oVvSLjkMv8/hqdefault.jpg"/>
   <media:community><media:statistics views="320"/></media:community>
   <yt:duration seconds="3725"/>
  </media:group>
 </entry>
</feed>`
	oge, err := RSSOku(rss)
	if err != nil {
		t.Fatalf("RSSOku hata: %v", err)
	}
	if len(oge) != 2 {
		t.Fatalf("2 öğe beklenirdi, %d geldi", len(oge))
	}
	if oge[0].Tur != "short" {
		t.Errorf("ilk öğe short olmalı, %q", oge[0].Tur)
	}
	if oge[1].Tur != "video" {
		t.Errorf("ikinci öğe video olmalı, %q", oge[1].Tur)
	}
	if oge[1].SureSn != 3725 {
		t.Errorf("süre 3725 olmalı, %d", oge[1].SureSn)
	}
	if oge[1].Izlenme != 320 {
		t.Errorf("izlenme 320 olmalı, %d", oge[1].Izlenme)
	}
	if oge[0].YayinAt.IsZero() {
		t.Error("yayın zamanı çözülemedi")
	}
	if !strings.Contains(oge[0].URL, "KfLVTp9jlyU") {
		t.Errorf("URL video kimliğini taşımalı: %q", oge[0].URL)
	}
}

func TestYayinlarOkuKendiBasligi(t *testing.T) {
	html := `<html><script>var ytInitialData = {"contents":{"a":[` +
		`{"lockupViewModel":{"contentId":"MctQ1WmwSaA","metadata":{"lockupMetadataViewModel":{"title":{"content":"Birinci yayın"}}}}},` +
		`{"lockupViewModel":{"contentId":"MctQ1WmwSaA","metadata":{"lockupMetadataViewModel":{"title":{"content":"Birinci yayın"}}}}},` +
		`{"lockupViewModel":{"contentId":"7NR24nMcqec","metadata":{"lockupMetadataViewModel":{"title":{"content":"İkinci yayın"}}}}}` +
		`]}};</script></html>`
	kayit := YayinlarOku(html)
	if len(kayit) != 2 {
		t.Fatalf("2 tekil yayın beklenirdi, %d geldi", len(kayit))
	}
	if kayit[0].VideoID != "MctQ1WmwSaA" || kayit[0].Baslik != "Birinci yayın" {
		t.Errorf("birinci kayıt yanlış: %+v", kayit[0])
	}
	if kayit[1].VideoID != "7NR24nMcqec" || kayit[1].Baslik != "İkinci yayın" {
		t.Errorf("ikinci kayıt yanlış: %+v", kayit[1])
	}
}

func TestCanliVideoID(t *testing.T) {
	canli := `<html><head><link rel="canonical" href="https://www.youtube.com/watch?v=abc12345678"></head><body>"isLive":true</body></html>`
	if id := CanliVideoID(canli); id != "abc12345678" {
		t.Errorf("canlı kimliği abc12345678 olmalı, %q", id)
	}

	// Canlı değil: canonical kanal adresi, sayfada isLive yok.
	degil := `<html><head><link rel="canonical" href="https://www.youtube.com/channel/UCPtM806ijpKatyIHH5w4kmg"></head></html>`
	if id := CanliVideoID(degil); id != "" {
		t.Errorf("canlı olmayan sayfada kimlik boş olmalı, %q", id)
	}

	// isLive:false taşıyan izleme sayfası canlı sayılmamalı.
	sahte := `<html><head><link rel="canonical" href="https://www.youtube.com/watch?v=abc12345678"></head><body>"isLive":false</body></html>`
	if id := CanliVideoID(sahte); id != "" {
		t.Errorf("isLive:false canlı sayılmamalı, %q", id)
	}
}

func TestGecerliKanalVeHandle(t *testing.T) {
	if !GecerliKanalID("UCPtM806ijpKatyIHH5w4kmg") {
		t.Error("gerçek kanal kimliği geçerli olmalı")
	}
	for _, kotu := range []string{"", "UC", "../../etc/passwd", "UCPtM806ijpKatyIHH5w4kmg?x=1"} {
		if GecerliKanalID(kotu) {
			t.Errorf("geçersiz kanal kimliği kabul edildi: %q", kotu)
		}
	}
	if !GecerliHandle("bitcosar") {
		t.Error("bitcosar geçerli olmalı")
	}
	for _, kotu := range []string{"", "@bitcosar", "a/b", "bit cosar"} {
		if GecerliHandle(kotu) {
			t.Errorf("geçersiz handle kabul edildi: %q", kotu)
		}
	}
}

func TestSure(t *testing.T) {
	for _, c := range []struct {
		sn   int
		isti string
	}{{0, ""}, {65, "1:05"}, {3725, "1:02:05"}, {600, "10:00"}} {
		if got := Sure(c.sn); got != c.isti {
			t.Errorf("Sure(%d) = %q, beklenen %q", c.sn, got, c.isti)
		}
	}
}
