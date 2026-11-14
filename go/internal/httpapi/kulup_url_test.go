package httpapi

import "testing"

// validKulupURL — "Milyonerler Kulübü" kartının hedef doğrulaması.
//
// Kartın hedefi iki farklı dünyadan gelebiliyor: site içi satış sayfası
// (/funnel) ve dış üye paneli (https://traders…). Doğrulama bu iki biçimi
// kabul edip geri kalanı reddetmeli — özellikle `javascript:` ve
// protokol-göreli `//` varyantlarını, çünkü bunlar ziyaretçiyi başka bir
// siteye ya da kod çalıştırmaya götürür.
func TestValidKulupURL(t *testing.T) {
	for _, c := range []struct {
		ad       string
		girdi    string
		beklenen string
	}{
		{"site ici yol", "/funnel", "/funnel"},
		{"site ici kok", "/", "/"},
		{"sorgu ile", "/funnel?utm=x", "/funnel?utm=x"},
		{"bastaki bosluk kirpilir", "  /funnel  ", "/funnel"},
		{"dis https", "https://traders.bitcosar.com/", "https://traders.bitcosar.com/"},
		{"dis https alt yol", "https://example.com/satis?x=1", "https://example.com/satis?x=1"},
		{"port ile", "https://example.com:8443/a", "https://example.com:8443/a"},

		{"bos", "", ""},
		{"yalniz bosluk", "   ", ""},
		// Protokol-göreli adres: tarayıcı bunu DIŞ site sayar.
		{"protokol goreli", "//evil.example/x", ""},
		{"ters egik", "/\\evil.example", ""},
		// Şema saldırıları.
		{"javascript", "javascript:alert(1)", ""},
		{"data", "data:text/html,<script>x</script>", ""},
		{"http duz", "http://example.com/", ""},
		// Yol içinde dışarı çıkma denemesi.
		{"tirnak", "/funnel\"onmouseover=alert(1)", ""},
		{"aci parantez", "/funnel<script>", ""},
		// Alan adı olmayan dış adres.
		{"sema disi", "traders.bitcosar.com", ""},
	} {
		if got := validKulupURL(c.girdi); got != c.beklenen {
			t.Errorf("%s: validKulupURL(%q) = %q, beklenen %q", c.ad, c.girdi, got, c.beklenen)
		}
	}
}
