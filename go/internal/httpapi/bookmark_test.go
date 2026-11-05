package httpapi

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBookmarkHTML(t *testing.T) {
	base, _ := url.Parse("https://example.com/haber/1")
	body := []byte(`<!doctype html><html><head>
		<title>Yedek başlık</title>
		<meta property="og:title" content="Öne çıkan başlık">
		<meta name="description" content="Kısa &amp; açıklama">
		<meta property="og:site_name" content="Örnek Site">
		<meta property="og:image" content="/img/kapak.jpg">
	</head><body></body></html>`)
	out := parseBookmarkHTML(body, base)
	assert.Equal(t, "Öne çıkan başlık", out.Title)
	assert.Equal(t, "Kısa & açıklama", out.Description)
	assert.Equal(t, "Örnek Site", out.Site)
	assert.Equal(t, "https://example.com/img/kapak.jpg", out.Image)
	assert.Equal(t, "https://example.com/haber/1", out.URL)
}

func TestValidateBookmarkURL(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/", "http://localhost/", "file:///etc/passwd", "http://10.0.0.1/"} {
		_, err := validateBookmarkURL(raw)
		assert.Error(t, err, raw)
	}
	u, err := validateBookmarkURL("https://1.1.1.1/haber")
	assert.NoError(t, err)
	assert.Equal(t, "1.1.1.1", u.Hostname())
}
