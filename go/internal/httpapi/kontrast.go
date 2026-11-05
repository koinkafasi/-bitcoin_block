package httpapi

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Renk yardımcıları — erişilebilir kontrast (Faz 180, A-7).
//
// SORUN: marka rengi kullanıcı tarafından seçiliyor (Marka Merkezi) ve
// varsayılan değeri #fa0000. Bu rengin üstüne beyaz yazı konduğunda
// kontrast 4.15 çıkıyor; WCAG AA küçük metin eşiği 4.5.
//
// YANLIŞ ÇÖZÜM: marka rengini koyulaştırmak. Renk kullanıcının seçimi;
// başlıkta, bağlantıda ve grafiklerde aynı ton görünmeli.
//
// DOĞRU ÇÖZÜM: marka rengi AYNEN kalır; yalnız "üstünde beyaz yazı taşıyan
// DOLGU" için türetilmiş bir renk üretilir (`--accent-fill`). Türetme,
// beyazla 4.5'i geçen EN AÇIK tonu bulur — yani markaya en yakın olanı.
// #fa0000 için sonuç #ef0000 (4.5% koyu): 250 → 239 kırmızı kanalı,
// gözle ayırt edilemez.

const (
	// wcagNormalText küçük metin için eşik.
	wcagNormalText = 4.5
	// enFazlaKarartma: aşırı uçuk marka renklerinde (ör. sarı) beyaz yazı
	// hiç geçmeyebilir. Bu sınırda durup en koyu tonu döndürürüz; çağıran
	// dilerse yazıyı koyu yapar. Pratikte marka renkleri bu aralıkta.
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

// accentFill: `accent` renginin üstüne BEYAZ yazı konduğunda 4.5'i geçen
// en açık (markaya en yakın) tonu döndürür.
//
// Geçersiz girdide boş döner; çağıran `--accent-fill` yazmaz ve CSS'teki
// `var(--accent-fill, var(--accent))` yedeği devreye girer.
// accentTextFill: `accent` rengi METİN olarak açık zeminde 4.5'i geçsin
// diye türetilmiş ton.
//
// Ölçüt, panelin EN KOYU yüzeyi: yan menü zemini #e7e4dd. Sayfa zemini
// #f3f2ef ve kart #fdfdfb daha açık; en koyu olana göre ayarlanınca diğer
// ikisi de geçiyor. (Üretimde ölçüldü: #f3f2ef'e göre ayarlanmış ton yan
// menüde 4.0 veriyordu.)
func accentTextFill(h string) string {
	return karartEsige(h, "#e7e4dd")
}

// accentTextKoyu: `accent` rengi KOYU temada METIN olarak 4.5'i geçsin
// diye türetilmiş ton. Koyu zeminde marka rengini koyulaştırmak tersine
// çalışır; beyaza doğru açılır. Ölçüt, koyu temanın kart zemini (#18181b);
// sayfa zemini (#131316) daha da koyu olduğu için ikisi birlikte sınanır.
func accentTextKoyu(h string) string {
	return koyuTemaMetin(h)
}

// koyuTemaMetin: kullanici secimi bir metin rengi (or. #000000) koyu temada
// 4.5'i gecsin diye beyaza dogru acilir.
//
// NEDEN: "Gorunum & Marka" panelinde secilen metin rengi TEK bir deger;
// iki temaya da uygulaniyordu. Kullanici siyah sectiginde koyu temada
// metin siyah kalip okunmuyordu (olculdu: kontrast 1.2).
func koyuTemaMetin(h string) string {
	r, g, b, ok := hexRGB(h)
	if !ok {
		return ""
	}
	// Ölçüt, koyu temanın EN AÇIK yüzeyi: `--surface-2` ≈ #27272a.
	// Açık metin için zemin açıldıkça kontrast DÜŞER; bu yüzden en kötü
	// durum en açık yüzeydir. Önce kart (#1e1e22) ve sayfa (#131316)
	// sınanmıştı; üretilen ton #27272a üzerinde 4.2 veriyordu.
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
	// DİKKAT: adaylar YUVARLANMIŞ kanallarla test edilir. Sürekli değerle
	// test edip sonra hex'e yuvarlamak, eşiği kıl payı geçen tonu bir adım
	// AÇIĞA yuvarlayabiliyor (ölçüldü: #16a34a için 4.49).
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
