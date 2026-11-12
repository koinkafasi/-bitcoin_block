package templkit

import (
	"net/url"
	"strconv"
)

// cdItoa sayıyı metne çevirir (templ içinde `strconv` çağırmak yerine).
func cdItoa(n int) string { return strconv.Itoa(n) }

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
