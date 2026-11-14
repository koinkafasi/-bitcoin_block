package httpapi

import "testing"

// validChromeEklenti yalnız Chrome Web Store eklenti sayfalarını geçirmeli:
// değer ana sayfada bağlantı olarak basılıyor, başka alan adı kabul edilirse
// panelden keyfi yönlendirme konabilir.
func TestValidChromeEklenti(t *testing.T) {
	const kimlik = "abcdefghijklmnopabcdefghijklmnop"
	const tam = "https://chromewebstore.google.com/detail/bitcosar/" + kimlik

	cases := []struct {
		ad, girdi, beklenen string
	}{
		{"adli adres", tam, tam},
		{"adsiz adres", "https://chromewebstore.google.com/detail/" + kimlik, "https://chromewebstore.google.com/detail/" + kimlik},
		{"sorgu dizesi atilir", tam + "?hl=tr&utm_source=x", tam},
		{"sondaki bolu ve bosluk atilir", "  " + tam + "/ ", tam},
		{"bos", "", ""},
		{"http reddedilir", "http://chromewebstore.google.com/detail/bitcosar/" + kimlik, ""},
		{"baska alan adi", "https://example.com/detail/bitcosar/" + kimlik, ""},
		{"alan adi son eki", "https://chromewebstore.google.com.example.com/detail/bitcosar/" + kimlik, ""},
		{"kisa kimlik", "https://chromewebstore.google.com/detail/bitcosar/abc", ""},
		{"kimlikte a-p disi harf", "https://chromewebstore.google.com/detail/bitcosar/zzzzagchadeembaojpdlfligoencpiih", ""},
		{"javascript semasi", "javascript:alert(1)", ""},
	}
	for _, c := range cases {
		if got := validChromeEklenti(c.girdi); got != c.beklenen {
			t.Errorf("%s: validChromeEklenti(%q) = %q, beklenen %q", c.ad, c.girdi, got, c.beklenen)
		}
	}
}
