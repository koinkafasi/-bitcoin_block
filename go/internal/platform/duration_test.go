package platform

import "testing"

func TestParseDurationTR(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"3:42", 222},
		{"1:02:03", 3723},
		{"222", 222},
		{"  03:05 ", 185},
		{"0:30", 30},
		{"", 0},
		{"abc", 0},
		{"3:xx", 0},
		{"1:2:3:4", 0}, // fazla parça
		{"-5", 0},
		{"3:", 0},
	}
	for _, c := range cases {
		if got := ParseDurationTR(c.in); got != c.want {
			t.Errorf("ParseDurationTR(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}

func TestFormatDurationTR(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{222, "3:42"},
		{3723, "1:02:03"},
		{59, "0:59"},
		{60, "1:00"},
		{0, ""},
		{-1, ""},
	}
	for _, c := range cases {
		if got := FormatDurationTR(c.in); got != c.want {
			t.Errorf("FormatDurationTR(%d) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestDurationRoundTrip(t *testing.T) {
	// Editörün yazdığı biçim tekrar okunduğunda aynı saniyeye dönmeli.
	for _, s := range []string{"3:42", "1:02:03", "0:59", "10:00"} {
		sec := ParseDurationTR(s)
		if got := FormatDurationTR(sec); got != s {
			t.Errorf("gidiş-dönüş %q -> %d -> %q", s, sec, got)
		}
	}
}
