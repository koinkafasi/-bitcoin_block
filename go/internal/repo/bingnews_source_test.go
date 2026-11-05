package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMSNArticleURL(t *testing.T) {
	raw := "https://www.bing.com/news/apiclick.aspx?ref=FexRss&url=https%3a%2f%2fwww.msn.com%2ftr-tr%2fspor%2fother%2fmac-sonucu%2far-ABC123&mkt=tr-tr"
	assert.Equal(t, "https://www.msn.com/tr-tr/spor/other/mac-sonucu/ar-ABC123", msnArticleURL(raw))
	assert.Equal(t, "", msnArticleURL("https://www.bing.com/news/apiclick.aspx?url=https%3a%2f%2fexample.com%2fhaber"))
	assert.Equal(t, "https://www.msn.com/tr-tr/haber/other/konu/ar-ABC123", msnArticleURL("https://www.msn.com/tr-tr/haber/other/konu/ar-ABC123"))
}

func TestMSNSection(t *testing.T) {
	assert.Equal(t, "spor", msnSection("https://www.msn.com/tr-tr/spor/other/mac-sonucu/ar-ABC123"))
	assert.Equal(t, "finans", msnSection("https://www.msn.com/tr-tr/finans/finansal-duzenleme/konu/ar-ABC123"))
	assert.Equal(t, "haber", msnSection("https://www.msn.com/tr-tr/haber/other/konu/ar-ABC123"))
	assert.Equal(t, "", msnSection("https://www.msn.com/tr-tr"))
}
