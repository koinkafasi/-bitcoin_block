package httpapi

import (
	"net/http/httptest"
	"testing"
)

// TestSameOriginRequest CSRF ara katmanının karar tablosu.
func TestSameOriginRequest(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		host    string
		origin  string
		referer string
		want    bool
	}{
		{"origin yok referer yok", "POST", "", "", "", true},
		{"ayni origin", "POST", "", "https://theblock.com.tr", "", true},
		{"portlu ayni origin", "POST", "127.0.0.1:8070", "http://127.0.0.1:8070", "", true},
		{"farkli origin", "POST", "", "https://evil.example", "", false},
		{"www toleransi", "POST", "www.theblock.com.tr", "https://theblock.com.tr", "", true},
		{"alt alan adi farkli", "POST", "admin.theblock.com.tr", "https://theblock.com.tr", "", false},
		{"null origin", "POST", "", "null", "", false},
		{"bozuk origin", "POST", "", "://bozuk", "", false},
		{"ayni referer", "POST", "", "", "https://theblock.com.tr/admin/icerikler", true},
		{"farkli referer", "POST", "", "", "https://evil.example/x", false},
		{"origin referer'dan oncelikli", "POST", "", "https://evil.example", "https://theblock.com.tr/x", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, "https://theblock.com.tr/admin/icerikler", nil)
		if c.host != "" {
			r.Host = c.host
		}
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.referer != "" {
			r.Header.Set("Referer", c.referer)
		}
		if got := sameOriginRequest(r); got != c.want {
			t.Errorf("%s: sameOriginRequest = %v, beklenen %v", c.name, got, c.want)
		}
	}
}

// TestSameOriginRequest_GETBypass GET/HEAD istekleri denetlenmez
// (ara katman yalnız durum değiştiren metotlara uygulanır).
func TestSameOriginRequest_GETBypass(t *testing.T) {
	r := httptest.NewRequest("GET", "https://theblock.com.tr/", nil)
	r.Header.Set("Origin", "https://evil.example")
	// sameOriginRequest teknik olarak false döner; asıl kapı withSameOrigin'de.
	// Bu test yalnız davranışın bilinçli olduğunu belgeler.
	if sameOriginRequest(r) {
		t.Log("GET istekleri de denetleniyor (beklenen değil)")
	}
}

// TestAyniSiteYolu açık yönlendirme koruması.
func TestAyniSiteYolu(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest("POST", "https://theblock.com.tr/yorum/12", nil)
	cases := []struct{ in, want string }{
		{"https://theblock.com.tr/haber/abc", "/haber/abc"},
		{"https://theblock.com.tr/haber/abc?x=1", "/haber/abc?x=1"},
		{"https://evil.example/phish", ""},
		{"//evil.example/x", ""},
		{"javascript:alert(1)", ""},
		{"/haber/abc", "/haber/abc"},
		{"", ""},
	}
	for _, c := range cases {
		if got := s.ayniSiteYolu(r, c.in); got != c.want {
			t.Errorf("ayniSiteYolu(%q) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}

// TestGenID middleware_test.go içinde (mevcut test güncellendi).
