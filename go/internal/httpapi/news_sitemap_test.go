package httpapi

import "testing"

const gecerliSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
<url><loc>https://site.com/a-1</loc><news:news><news:publication><news:name>Site</news:name><news:language>tr</news:language></news:publication><news:publication_date>2026-10-18T10:00:00Z</news:publication_date><news:title>A Haber</news:title></news:news></url>
<url><loc>https://site.com/b-2</loc><news:news><news:publication><news:name>Site</news:name><news:language>tr</news:language></news:publication><news:publication_date>2026-10-17T08:30:00Z</news:publication_date><news:title>B Haber</news:title></news:news></url>
</urlset>`

func TestValidateNewsSitemapXMLGecerli(t *testing.T) {
	rep := validateNewsSitemapXML([]byte(gecerliSitemap))
	if rep.Durum != "ok" {
		t.Fatalf("durum ok olmalı, %s: %s", rep.Durum, rep.Mesaj)
	}
	if rep.ToplamURL != 2 {
		t.Errorf("2 kayıt bekleniyordu, %d bulundu", rep.ToplamURL)
	}
	if rep.EnYeni != "2026-10-18 10:00" {
		t.Errorf("en yeni tarih: %q", rep.EnYeni)
	}
	if rep.EnEski != "2026-10-17 08:30" {
		t.Errorf("en eski tarih: %q", rep.EnEski)
	}
}

func TestValidateNewsSitemapEksikAlan(t *testing.T) {
	// İkinci kayıtta news:title yok -> hata
	bozuk := `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">` +
		`<url><loc>https://site.com/a-1</loc><news:news><news:publication><news:name>Site</news:name><news:language>tr</news:language></news:publication><news:publication_date>2026-10-18T10:00:00Z</news:publication_date><news:title>A</news:title></news:news></url>` +
		`<url><loc>https://site.com/b-2</loc><news:news><news:publication><news:name>Site</news:name><news:language>tr</news:language></news:publication><news:publication_date>2026-10-17T08:30:00Z</news:publication_date></news:news></url>` +
		`</urlset>`
	rep := validateNewsSitemapXML([]byte(bozuk))
	if rep.Durum != "hata" {
		t.Fatalf("durum hata olmalı, %s: %s", rep.Durum, rep.Mesaj)
	}
	if rep.ToplamURL != 2 {
		t.Errorf("2 kayıt bekleniyordu, %d", rep.ToplamURL)
	}
}

func TestValidateNewsSitemapBos(t *testing.T) {
	bos := `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9"></urlset>`
	rep := validateNewsSitemapXML([]byte(bos))
	if rep.Durum != "uyari" {
		t.Fatalf("durum uyari olmalı, %s: %s", rep.Durum, rep.Mesaj)
	}
	if rep.ToplamURL != 0 {
		t.Errorf("0 kayıt bekleniyordu, %d", rep.ToplamURL)
	}
}

func TestValidateNewsSitemapNewsAdAlaniYok(t *testing.T) {
	// news ad alanı olmadan üretilmiş dosya -> Google kabul etmez
	rep := validateNewsSitemapXML([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>x</loc></url></urlset>`))
	if rep.Durum != "hata" {
		t.Fatalf("durum hata olmalı, %s: %s", rep.Durum, rep.Mesaj)
	}
}
