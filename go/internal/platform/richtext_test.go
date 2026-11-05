package platform

import (
	"strings"
	"testing"
)

func TestSanitizeRichHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		has  string // beklenen alt dize
	}{
		{
			"temiz biçimlendirme korunur",
			`<p>Merhaba <strong>dünya</strong><em>x</em></p>`,
			"",
			"<p>Merhaba <strong>dünya</strong><em>x</em></p>",
		},
		{
			"script cıkarılır",
			`<p>güvenli</p><script>alert(1)</script>`,
			"",
			`<p>güvenli</p>`,
		},
		{
			"olay öznitelikleri cıkarılır",
			`<p onclick="evil()" style="x">metin</p>`,
			"",
			`<p>metin</p>`,
		},
		{
			"iframe/iframe cıkarılır",
			`<iframe src="x"></iframe><b>kalan</b>`,
			"",
			`<b>kalan</b>`,
		},
		{
			"a href guvenli şema korunur",
			`<a href="https://site.com/x">link</a>`,
			"",
			`<a href="https://site.com/x">link</a>`,
		},
		{
			"a href javascript engellenir",
			`<a href="javascript:alert(1)">kötü</a>`,
			"",
			`<a>kötü</a>`,
		},
		{
			"a href göreli korunur",
			`<a href="/icerik/1">iç</a>`,
			"",
			`<a href="/icerik/1">iç</a>`,
		},
		{
			"kapanmamış etiket kapatilir",
			`<p>metin<strong>kalın`,
			"",
			`<p>metin<strong>kalın</strong></p>`,
		},
		{
			"div ve style cıkarılır (blok metni saf)",
			`<div class="x"><p><span style="color:red">metin</span></p></div>`,
			"",
			`<p><span>metin</span></p>`,
		},
		{
			"liste korunur",
			`<ul><li>bir</li><li>iki</li></ul>`,
			"",
			`<ul><li>bir</li><li>iki</li></ul>`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SanitizeRichHTML(c.in)
			if c.has != "" && !strings.Contains(got, c.has) {
				t.Errorf("SanitizeRichHTML(%q) = %q; beklenen içeriyor: %q", c.in, got, c.has)
			}
			if strings.Contains(got, "<script") || strings.Contains(got, "onclick") ||
				strings.Contains(strings.ToLower(got), "javascript:") {
				t.Errorf("güvenlik açığı: girdi %q → çıktı %q", c.in, got)
			}
		})
	}
}

func TestSanitizeRichHTMLEmpty(t *testing.T) {
	if got := SanitizeRichHTML(""); got != "" {
		t.Fatalf("boş girdi boş dönmeli, %q", got)
	}
	if got := SanitizeRichHTML("<script>x</script>"); got != "" {
		t.Fatalf("yalnız script boş dönmeli, %q", got)
	}
}
