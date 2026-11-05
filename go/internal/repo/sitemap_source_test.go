package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNewsSitemap(t *testing.T) {
	body := `<?xml version="1.0" encoding="utf-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url>
    <loc>https://example.com/haber/eski-haber</loc>
    <news:news>
      <news:title>Eski haber başlığı</news:title>
      <news:publication_date>2026-10-20T09:00:00+03:00</news:publication_date>
    </news:news>
  </url>
  <url>
    <loc>https://example.com/haber/yeni-haber</loc>
    <news:news>
      <news:title>Yeni haber başlığı &amp; ayrıntıları</news:title>
      <news:publication_date>2026-10-21T21:38:00+03:00</news:publication_date>
    </news:news>
  </url>
</urlset>`
	items := parseNewsSitemap([]byte(body))
	assert.Len(t, items, 2)
	assert.Equal(t, "Yeni haber başlığı & ayrıntıları", items[0].Title)
	assert.Equal(t, "https://example.com/haber/yeni-haber", items[0].Link)
	assert.False(t, items[0].PubDate.IsZero())
	assert.Equal(t, "Eski haber başlığı", items[1].Title)
}

func TestParseNewsSitemapSkipsEmptyAndDuplicates(t *testing.T) {
	body := `<urlset xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url><loc>https://example.com/a</loc><news:news><news:title>Aynı başlık</news:title></news:news></url>
  <url><loc>https://example.com/a</loc><news:news><news:title>Aynı başlık</news:title></news:news></url>
  <url><loc>https://example.com/b</loc><news:news><news:title></news:title></news:news></url>
</urlset>`
	items := parseNewsSitemap([]byte(body))
	assert.Len(t, items, 1)
	assert.Equal(t, "https://example.com/a", items[0].Link)
}
