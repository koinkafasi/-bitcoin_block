package httpapi

import "testing"

// TestSameSitePathOpenRedirect, açık yönlendirme kapısını kilitler.
// Semgrep'in işaretlediği asıl hata auth_handlers.go'daydı: yalnız
// next[0] == '/' kontrolü "//evil.example" ve "/\evil.example" değerlerini
// geçiriyordu. Tarayıcı bunları protokol-göreli adres sayar.
func TestSameSitePathOpenRedirect(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/admin", "/admin"},
		{"/admin/icerikler?x=1", "/admin/icerikler?x=1"},
		{"  /admin  ", "/admin"},
		{"//evil.example", ""},
		{"/\\evil.example", ""},
		{"\\\\evil.example", ""},
		{"https://evil.example", ""},
		{"javascript:alert(1)", ""},
		{"/a\r\nSet-Cookie: x=1", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := sameSitePath(c.in); got != c.want {
			t.Errorf("sameSitePath(%q) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}

// TestReaderRedirectWhitelist, üye girişi dönüş hedeflerinin yalnız
// bilinen anahtarlar olduğunu doğrular.
func TestReaderRedirectWhitelist(t *testing.T) {
	for _, q := range []string{"abonelik", "hesabim"} {
		if got := readerRedirect(q); got != "/"+q {
			t.Errorf("readerRedirect(%q) = %q, beklenen /%s", q, got, q)
		}
	}
	for _, q := range []string{"", "https://evil.example", "//evil.example", "gizli"} {
		if got := readerRedirect(q); got != "/" {
			t.Errorf("readerRedirect(%q) = %q, beklenen /", q, got)
		}
	}
}
