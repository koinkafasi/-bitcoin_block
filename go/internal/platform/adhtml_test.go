package platform

import (
	"strings"
	"testing"
)

// TestSanitizeAdHTML_BlocksScript XSS vektörlerinin çıktıya sızmadığını doğrular.
// Regresyon: reklam HTML'i eskiden ham basılıyordu (kalıcı XSS).
func TestSanitizeAdHTML_BlocksScript(t *testing.T) {
	cases := []struct {
		name, in string
		bad      []string
	}{
		{"script", `<div>a</div><script>alert(1)</script><p>b</p>`, []string{"script", "alert"}},
		{"script buyuk harf", `<SCRIPT SRC=//evil/x.js></SCRIPT>`, []string{"SCRIPT", "evil"}},
		{"olay ozniteligi", `<img src="https://x/y.png" onerror="alert(1)">`, []string{"onerror", "alert"}},
		{"javascript url", `<a href="javascript:alert(1)">tikla</a>`, []string{"javascript", "alert"}},
		{"javascript bosluklu", "<a href=\"java\tscript:alert(1)\">x</a>", []string{"javascript", "alert"}},
		{"data text html", `<a href="data:text/html;base64,PHNjcmlwdD4=">x</a>`, []string{"data:text"}},
		{"iframe javascript", `<iframe src="javascript:alert(1)"></iframe>`, []string{"javascript", "alert"}},
		{"style expression", `<div style="width:expression(alert(1))">x</div>`, []string{"expression"}},
		{"svg onload", `<svg onload="alert(1)"></svg>`, []string{"onload", "alert"}},
		{"nesne", `<object data="x.swf"></object>`, []string{"object"}},
	}
	for _, c := range cases {
		got := strings.ToLower(SanitizeAdHTML(c.in))
		for _, b := range c.bad {
			if strings.Contains(got, strings.ToLower(b)) {
				t.Errorf("%s: çıktıda %q kaldı → %q", c.name, b, got)
			}
		}
	}
}

// TestSanitizeAdHTML_KeepsAdsMarkup meşru reklam işaretlerinin korunduğunu
// doğrular (AdSense <ins>, görsel, bağlantı, data-* öznitelikleri).
func TestSanitizeAdHTML_KeepsAdsMarkup(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{
			"adsense ins",
			`<ins class="adsbygoogle" style="display:block" data-ad-client="ca-pub-123" data-ad-slot="456" data-ad-format="auto" data-full-width-responsive="true"></ins>`,
			`data-ad-client="ca-pub-123"`,
		},
		{
			"gorsel ve baglanti",
			`<a href="https://example.com/kampanya" rel="nofollow sponsored" target="_blank"><img src="https://cdn.example.com/a.png" alt="Kampanya" width="300" height="250" loading="lazy"></a>`,
			`src="https://cdn.example.com/a.png"`,
		},
		{
			"iframe reklam",
			`<iframe src="https://ads.example.com/slot" width="300" height="250" sandbox="allow-scripts" frameborder="0"></iframe>`,
			`src="https://ads.example.com/slot"`,
		},
		{
			"basit metin",
			`<div class="ad"><strong>Sponsorlu</strong> içerik</div>`,
			`<strong>Sponsorlu</strong>`,
		},
	}
	for _, c := range cases {
		got := SanitizeAdHTML(c.in)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: beklenen parça yok\n istenen: %s\n çıktı:   %s", c.name, c.want, got)
		}
	}
}

// TestSanitizeAdHTML_Idempotent iki kez süzmek çıktıyı bozmamalı.
func TestSanitizeAdHTML_Idempotent(t *testing.T) {
	in := `<div class="ad"><a href="https://x/y">t</a><img src="https://x/i.png" onerror="x()"></div>`
	once := SanitizeAdHTML(in)
	iki := SanitizeAdHTML(once)
	if once != iki {
		t.Errorf("idempotent değil:\n1: %s\n2: %s", once, iki)
	}
}

// TestSanitizeAdHTML_Empty boş/whitespace girdi boş dönmeli.
func TestSanitizeAdHTML_Empty(t *testing.T) {
	for _, in := range []string{"", "   ", "\x00"} {
		if got := SanitizeAdHTML(in); got != "" {
			t.Errorf("%q → %q, boş beklenirdi", in, got)
		}
	}
}
