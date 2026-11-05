package trends

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const googleTrendsFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:ht="https://trends.google.com/trending/rss" version="2.0">
<channel>
<title>Daily Search Trends</title>
<link>https://trends.google.com/trending/rss?geo=TR</link>
<item>
  <title>puan durumu</title>
  <ht:approx_traffic>5000+</ht:approx_traffic>
  <description/>
  <link>https://trends.google.com/trending/rss?geo=TR</link>
  <pubDate>Mon, 19 Oct 2026 11:40:00 -0700</pubDate>
  <ht:picture>https://example.test/p1.jpg</ht:picture>
  <ht:picture_source>Eurohoops</ht:picture_source>
  <ht:news_item>
    <ht:news_item_title>Basketbol Süper Ligi Puan Durumu: Bahçeşehir Koleji Namağlup Lider</ht:news_item_title>
    <ht:news_item_snippet>Ligin 4. haftası geride kaldı.</ht:news_item_snippet>
    <ht:news_item_url>https://example.test/haber/1</ht:news_item_url>
    <ht:news_item_picture>https://example.test/n1.jpg</ht:news_item_picture>
    <ht:news_item_source>Eurohoops</ht:news_item_source>
  </ht:news_item>
  <ht:news_item>
    <ht:news_item_title>Puan durumu güncellendi</ht:news_item_title>
    <ht:news_item_url>https://example.test/haber/2</ht:news_item_url>
  </ht:news_item>
</item>
<item>
  <title>İstanbul trafik</title>
  <ht:approx_traffic>1.000+</ht:approx_traffic>
  <pubDate>Mon, 19 Oct 2026 10:00:00 -0700</pubDate>
</item>
</channel>
</rss>`

func TestParseGoogleTrendsRSS(t *testing.T) {
	terms, err := ParseGoogleTrendsRSS([]byte(googleTrendsFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(terms) != 2 {
		t.Fatalf("terim sayısı = %d, beklenen 2", len(terms))
	}
	if terms[0].Term != "puan durumu" {
		t.Errorf("ilk terim = %q", terms[0].Term)
	}
	if terms[0].ApproxTraffic != 5000 {
		t.Errorf("trafik = %d, beklenen 5000", terms[0].ApproxTraffic)
	}
	if len(terms[0].News) != 2 {
		t.Fatalf("haber sayısı = %d, beklenen 2", len(terms[0].News))
	}
	if terms[0].News[0].Source != "Eurohoops" {
		t.Errorf("haber kaynağı = %q", terms[0].News[0].Source)
	}
	if terms[0].News[0].Snippet == "" {
		t.Error("haber snippet boş")
	}
	if terms[0].PublishedAt.IsZero() {
		t.Error("pubDate çözülemedi")
	}
	if terms[1].ApproxTraffic != 1000 {
		t.Errorf("ikinci trafik = %d, beklenen 1000", terms[1].ApproxTraffic)
	}
}

func TestParseApproxTraffic(t *testing.T) {
	cases := map[string]int{
		"5000+": 5000, "1.000+": 1000, "200+": 200, "": 0,
		"1,5 Mn+": 1500000, "2 Mn+": 2000000, "10B+": 10000000000,
	}
	for in, want := range cases {
		if got := ParseApproxTraffic(in); got != want {
			t.Errorf("ParseApproxTraffic(%q) = %d, beklenen %d", in, got, want)
		}
	}
}

func TestNormalizeTermTurkish(t *testing.T) {
	cases := map[string]string{
		"İstanbul'da Trafik":    "istanbul da trafik",
		"IĞDIR":                 "ığdır",
		"  ÇOK   GÜZEL! ":       "çok güzel",
		"Türkiye-AB İlişkileri": "türkiye ab ilişkileri",
	}
	for in, want := range cases {
		if got := NormalizeTerm(in); got != want {
			t.Errorf("NormalizeTerm(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint(NormalizeTerm("Puan Durumu"), "TR", "tr-TR")
	b := Fingerprint(NormalizeTerm("puan durumu"), "tr", "tr-TR")
	if a != b {
		t.Errorf("fingerprint kararsız: %s != %s", a, b)
	}
	if a == Fingerprint(NormalizeTerm("puan durumu"), "US", "en-US") {
		t.Error("geo farkı fingerprint'e yansımadı")
	}
}

func TestCovered(t *testing.T) {
	if !Covered("puan durumu", "Basketbol Süper Ligi Puan Durumu güncellendi") {
		t.Error("tam ifade eşleşmedi")
	}
	if !Covered("puan durumu", "Puan durumu ve tablo değerlendirmesi") {
		t.Error("kelime örtüşmesi eşleşmedi")
	}
	if Covered("puan durumu", "Tamamen ilgisiz bir haber") {
		t.Error("ilgisiz metin eşleşti")
	}
	if !Covered("murat ağırel", "Murat Ağırel kimdir? İşte hayatı") {
		t.Error("Türkçe büyük harf/İ eşleşmesi başarısız")
	}
}

func TestScoreGapAndCoverage(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	base := ScoreInput{
		ApproxTraffic: 5000,
		NewsCount:     3,
		FirstSeen:     now.Add(-2 * time.Hour),
		LastSeen:      now,
		Now:           now,
		Relevance:     0.8,
		Demand:        0.5,
		Coverage:      0,
	}
	gap := Score(base)
	base.Coverage = 3
	covered := Score(base)
	if gap <= covered {
		t.Errorf("boşluk skoru (%d) kapsanan skordan (%d) yüksek olmalı", gap, covered)
	}
	if gap < 0 || gap > 100 {
		t.Errorf("skor aralık dışı: %d", gap)
	}

	old := base
	old.FirstSeen = now.Add(-72 * time.Hour)
	old.LastSeen = now.Add(-48 * time.Hour)
	if Score(old) >= Score(base) {
		t.Error("eski terim daha düşük skor almalı")
	}
}

func TestHasLatin(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"puan durumu", true},
		{"Beşiktaş", true},
		{"خيتافي ضد رايو فاليكانو", false},
		{"123 !!!", false},
		{"", false},
	}
	for _, c := range cases {
		if got := HasLatin(c.in); got != c.want {
			t.Errorf("HasLatin(%q)=%v, want %v", c.in, got, c.want)
		}
	}
}

func TestGoogleTrendsProviderETag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(googleTrendsFixture))
	}))
	defer srv.Close()

	p := &GoogleTrendsProvider{}
	res, err := p.Fetch(context.Background(), Source{URL: srv.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(res.Terms) != 2 || res.ETag != `"v1"` {
		t.Fatalf("beklenmeyen sonuç: %+v", res)
	}
	res2, err := p.Fetch(context.Background(), Source{URL: srv.URL, ETag: res.ETag})
	if err != nil {
		t.Fatalf("fetch 304: %v", err)
	}
	if !res2.NotModified {
		t.Error("304 NotModified olarak işaretlenmedi")
	}
}
