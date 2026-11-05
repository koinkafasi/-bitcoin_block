package httpapi

import (
	"strings"
	"testing"

	"theblock/site/internal/repo"
)

func TestCleanTrendTitle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  elektrikli araç  ", "elektrikli araç"},
		{"elektrikli araç\nHaber başlığı — Mock başlık A", "elektrikli araç"},
		{"resmi gazete\r\nkarar", "resmi gazete"},
		{"çok   fazla    boşluk", "çok fazla boşluk"},
	}
	for _, tc := range cases {
		if got := cleanTrendTitle(tc.in); got != tc.want {
			t.Errorf("cleanTrendTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	long := strings.Repeat("a", 300)
	if got := cleanTrendTitle(long); len([]rune(got)) != 180 {
		t.Errorf("uzun başlık %d rune, want 180", len([]rune(got)))
	}
}

func TestCleanTrendSpot(t *testing.T) {
	got := cleanTrendSpot("elektrikli araç\nHaber başlığı\n\n  detay")
	if got != "elektrikli araç Haber başlığı detay" {
		t.Errorf("cleanTrendSpot = %q", got)
	}
	if got := cleanTrendSpot(strings.Repeat("b", 300)); len([]rune(got)) != 240 {
		t.Errorf("uzun spot %d rune, want 240", len([]rune(got)))
	}
}

func TestTrendSparkline(t *testing.T) {
	// Sıra 1 en üstte olmalı: 5 → 3 → 1 yukarı yönlü bir çizgi üretir.
	path, title, x, y := trendSparkline([]repo.TrendSparkPoint{
		{Rank: 5}, {Rank: 3}, {Rank: 1},
	})
	if want := "M3.0,25.0 L50.0,14.0 L97.0,3.0"; path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if x != 97 || y != 3 {
		t.Errorf("son nokta = (%v,%v), want (97,3)", x, y)
	}
	if !strings.Contains(title, "en iyi sıra 1") || !strings.Contains(title, "son sıra 1") {
		t.Errorf("title = %q", title)
	}

	// Aynı sıra: düz çizgi, orta yükseklik.
	path, _, _, y = trendSparkline([]repo.TrendSparkPoint{{Rank: 2}, {Rank: 2}})
	if want := "M3.0,14.0 L97.0,14.0"; path != want || y != 14 {
		t.Errorf("düz seri = %q y=%v, want %q y=14", path, y, want)
	}

	// İki noktadan az: çizgi yok.
	for _, pts := range [][]repo.TrendSparkPoint{nil, {{Rank: 1}}} {
		if path, _, _, _ := trendSparkline(pts); path != "" {
			t.Errorf("%d nokta için path = %q, want boş", len(pts), path)
		}
	}
}

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"normal":   "normal",
		"Türkçe":   "Türkçe",
		"=SUM(A1)": "'=SUM(A1)",
		"+90":      "'+90",
		"-5":       "'-5",
		"@formul":  "'@formul",
		"\tbaşlık": "'\tbaşlık",
	}
	for in, want := range cases {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, want %q", in, got, want)
		}
	}
}
