package httpapi

import (
	"strings"
	"testing"
)

func TestGoogleSourceURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://theblock.com.tr", "https://www.google.com/preferences/source?q=theblock.com.tr"},
		{"http://theblock.com.tr", "https://www.google.com/preferences/source?q=theblock.com.tr"},
		{"https://theblock.com.tr/", "https://www.google.com/preferences/source?q=theblock.com.tr"},
		{"https://www.example.com", "https://www.google.com/preferences/source?q=www.example.com"},
		// Geliştirme: port host'un parçası, korunmalı.
		{"http://127.0.0.1:8070", "https://www.google.com/preferences/source?q=127.0.0.1:8070"},
		// Şemasız/yollu bozuk girdi: yine de domain çıkmalı.
		{"theblock.com.tr/akis", "https://www.google.com/preferences/source?q=theblock.com.tr"},
		{"", "https://www.google.com/preferences/source?q="},
	}
	for _, c := range cases {
		if got := googleSourceURL(c.in); got != c.want {
			t.Errorf("googleSourceURL(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestGoogleSourceURLHasNoSchemeInQuery(t *testing.T) {
	// Google ?q= değerinde şema olmamalı; "://" sorgu kısmında geçmemeli.
	got := googleSourceURL("https://theblock.com.tr")
	q := got[len("https://www.google.com/preferences/source?q="):]
	if q == "" {
		t.Fatalf("sorgu değeri boş: %q", got)
	}
	for _, yasak := range []string{"://", "/", "?"} {
		if strings.Contains(q, yasak) {
			t.Errorf("sorgu değerinde %q olmamalı: %q", yasak, q)
		}
	}
	if !strings.Contains(q, ".") {
		t.Errorf("sorgu değeri domain gibi görünmeli: %q", q)
	}
}
