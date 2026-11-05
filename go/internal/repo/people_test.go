package repo

import "testing"

func TestPersonSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Alper Şahin", "alper-sahin"},
		{"Çağrı İğdır", "cagri-igdir"},
		{"  Ayşe  Nur  ", "ayse-nur"},
		{"3. Sayfa", "3-sayfa"},
		{"", ""},
	}
	for _, c := range cases {
		if got := PersonSlug(c.in); got != c.want {
			t.Errorf("PersonSlug(%q) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}
