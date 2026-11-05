package httpapi

import (
	"strings"
	"testing"
)

// TestWebsocketOrigin collab adresinin CSP connect-src girdisine çevrilmesi.
// Eksik olursa editörün ortak düzenleme katmanı sessizce düşer.
func TestWebsocketOrigin(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"ws://127.0.0.1:1234", "ws://127.0.0.1:1234"},
		{"wss://collab.theblock.com.tr", "wss://collab.theblock.com.tr"},
		{"http://10.0.0.5:1234", "ws://10.0.0.5:1234"},
		{"https://collab.example.com/x", "wss://collab.example.com"},
		{"bozuk-adres", ""},
	}
	for _, c := range cases {
		if got := websocketOrigin(c.in); got != c.want {
			t.Errorf("websocketOrigin(%q) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}

// TestBuildCSPIncludesCollab collab adresi verildiğinde connect-src'ye
// eklendiğini doğrular.
func TestBuildCSPIncludesCollab(t *testing.T) {
	csp := buildCSP(nil, nil, "wss://collab.theblock.com.tr", "")
	if !strings.Contains(csp, "connect-src") || !strings.Contains(csp, "wss://collab.theblock.com.tr") {
		t.Errorf("collab adresi connect-src'de yok: %s", csp)
	}
}
