package httpapi

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// wcagNormalText küçük metin için eşik.
	wcagNormalText  = 4.5
	enFazlaKarartma = 0.65
)

// hexRGB "#rrggbb" (ya da "#rgb") çözer.
func hexRGB(h string) (r, g, b float64, ok bool) {
	h = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(h), "#"))
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 0, 0, 0, false
	}
	kanal := func(s string) (float64, bool) {
		v, err := strconv.ParseUint(s, 16, 8)
		return float64(v), err == nil
	}
	var ok2, ok3 bool
	r, ok = kanal(h[0:2])
	g, ok2 = kanal(h[2:4])
	b, ok3 = kanal(h[4:6])
	return r, g, b, ok && ok2 && ok3
}

func rgbHex(r, g, b float64) string {
	k := func(v float64) int {
		return int(math.Round(math.Max(0, math.Min(255, v))))
	}
	return fmt.Sprintf("#%02x%02x%02x", k(r), k(g), k(b))
}

// bagilParlaklik WCAG tanımı.
func bagilParlaklik(r, g, b float64) float64 {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// kontrastOrani iki rengin kontrast oranı (1–21).
func kontrastOrani(r1, g1, b1, r2, g2, b2 float64) float64 {
	l1 := bagilParlaklik(r1, g1, b1)
	l2 := bagilParlaklik(r2, g2, b2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func accentTextFill(h string) string {
	return karartEsige(h, "#e7e4dd")
}

func accentTextKoyu(h string) string {
	return koyuTemaMetin(h)
}

func koyuTemaMetin(h string) string {
	r, g, b, ok := hexRGB(h)
	if !ok {
		return ""
	}
	zemin := [3]float64{39, 39, 42}
	sayfa := [3]float64{19, 19, 22}
	uygun := func(rr, gg, bb float64) bool {
		return kontrastOrani(rr, gg, bb, zemin[0], zemin[1], zemin[2]) >= wcagNormalText &&
			kontrastOrani(rr, gg, bb, sayfa[0], sayfa[1], sayfa[2]) >= wcagNormalText
	}
	if uygun(r, g, b) {
		return h
	}
	for i := 1; i <= 200; i++ {
		f := float64(i) / 200
		rr := math.Round(r + (255-r)*f)
		gg := math.Round(g + (255-g)*f)
		bb := math.Round(b + (255-b)*f)
		if uygun(rr, gg, bb) {
			return rgbHex(rr, gg, bb)
		}
	}
	return "#ffffff"
}

// karartEsige: `hedef` rengin üstünde 4.5 kontrast veren en açık tonu bulur.
func karartEsige(h, hedef string) string {
	r, g, b, ok := hexRGB(h)
	if !ok {
		return ""
	}
	zr, zg, zb, ok2 := hexRGB(hedef)
	if !ok2 {
		return ""
	}
	if kontrastOrani(r, g, b, zr, zg, zb) >= wcagNormalText {
		return h
	}
	for i := 1; i <= 200; i++ {
		k := 1 - (enFazlaKarartma*float64(i))/200
		rr := math.Round(r * k)
		gg := math.Round(g * k)
		bb := math.Round(b * k)
		if kontrastOrani(rr, gg, bb, zr, zg, zb) >= wcagNormalText {
			return rgbHex(rr, gg, bb)
		}
	}
	k := 1 - enFazlaKarartma
	return rgbHex(r*k, g*k, b*k)
}

func accentFill(h string) string {
	r, g, b, ok := hexRGB(h)
	if !ok {
		return ""
	}
	if kontrastOrani(r, g, b, 255, 255, 255) >= wcagNormalText {
		return h // zaten geçiyor: marka rengi aynen kullanılır
	}
	for i := 1; i <= 200; i++ {
		k := 1 - (enFazlaKarartma*float64(i))/200
		rr := math.Round(r * k)
		gg := math.Round(g * k)
		bb := math.Round(b * k)
		if kontrastOrani(rr, gg, bb, 255, 255, 255) >= wcagNormalText {
			return rgbHex(rr, gg, bb)
		}
	}
	k := 1 - enFazlaKarartma
	return rgbHex(r*k, g*k, b*k)
}
