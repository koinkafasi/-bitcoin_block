package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHreflangLinks, yazı sayfası hreflang etiketlerinin mutlak adresle,
// aktif dilin kendi URL'siyle ve x-default ile üretildiğini doğrular.
func TestHreflangLinks(t *testing.T) {
	links := []langLink{
		{Label: "Türkçe", Code: "tr", Active: true},
		{Label: "English", Code: "en", URL: "/en/poltava-42"},
	}

	got := hreflangLinks("https://theblock.com.tr/", "/poltava-42", "tr", links)
	assert.Equal(t, []langLink{
		{Code: "tr", URL: "https://theblock.com.tr/poltava-42"},
		{Code: "en", URL: "https://theblock.com.tr/en/poltava-42"},
		{Code: "x-default", URL: "https://theblock.com.tr/poltava-42"},
	}, got)

	// Varsayılan dilin çevirisi yoksa x-default kendi adresine düşer.
	got = hreflangLinks("https://x.test", "/en/a-1", "tr", []langLink{
		{Code: "en", Active: true},
		{Code: "de", URL: "/de/a-2"},
	})
	assert.Equal(t, "https://x.test/en/a-1", got[len(got)-1].URL)

	// Tek dilde hreflang basılmaz.
	assert.Nil(t, hreflangLinks("https://x.test", "/a-1", "tr", links[:1]))
}
