package templkit

import (
	"net/url"
	"strconv"
)

// cdItoa sayıyı metne çevirir (templ içinde `strconv` çağırmak yerine).
func cdItoa(n int) string { return strconv.Itoa(n) }

// cdHisseHarfHref A-Z dizini bağlantısı.
//
// Harf boşsa "tümü" bağlantısıdır. Arama metni korunur: kullanıcı "THY"
// yazıp sonra "T" harfine basarsa süzgeç kaybolmamalı.
func cdHisseHarfHref(harf, q string) string {
	v := url.Values{}
	if harf != "" {
		v.Set("harf", harf)
	}
	if q != "" {
		v.Set("q", q)
	}
	if len(v) == 0 {
		return "/borsa/hisseler"
	}
	return "/borsa/hisseler?" + v.Encode()
}
