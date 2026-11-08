package templkit

// Bağımlılıksız inline SVG grafik.
//
// NEDEN SVG + KENDİ KODUMUZ: site CSP'si `script-src 'self'`; harici grafik
// kütüphanesi (Chart.js vb.) yüklenemez. Ayrıca grafik sunucuda üretilir:
// JS kapalıyken de görünür, ilk boyamada hazırdır, istemci maliyeti sıfırdır.
//
// Koordinatlar `viewBox` içinde 0..W / 0..H normalize edilir; SVG
// `preserveAspectRatio="none"` ile kart genişliğine yayılır (ölçüler CSS'te).

// CDGrafikKart tek bir grafik paneli (fiyat ya da zincir-üstü metrik).
type CDGrafikKart struct {
	Baslik    string
	Deger     string // büyük değer (biçimlenmiş)
	Degisim   string // "+%1.24" gibi; boşsa basılmaz
	Up        bool
	Alt       string // değer altı açıklama ("son 30 gün" gibi)
	Yol       string // SVG path d (çizgi)
	Alan      string // SVG path d (dolgu)
	Min       string
	Maks      string
	EtiketSol string
	EtiketSag string
	Kucuk     bool // mini kart (zincir-üstü metrik şeridi)
}

// GrafikW / GrafikH SVG viewBox ölçüleri.
const (
	GrafikW = 640.0
	GrafikH = 180.0
)

// CDGrafikYolu değerleri SVG yoluna çevirir.
//
// Dönen: çizgi yolu, dolgu yolu, en küçük, en büyük, geçerli mi.
// Tek değer ya da tamamı eşitse düz çizgi üretilir (sıfıra bölme yok).
func CDGrafikYolu(deger []float64) (yol, alan string, min, maks float64, ok bool) {
	if len(deger) < 2 {
		return "", "", 0, 0, false
	}
	min, maks = deger[0], deger[0]
	for _, v := range deger {
		if v < min {
			min = v
		}
		if v > maks {
			maks = v
		}
	}
	// Dikey nefes payı: çizgi kart kenarına yapışmasın.
	const pad = 6.0
	aralik := maks - min
	if aralik <= 0 {
		aralik = 1
	}
	n := len(deger)
	adim := GrafikW / float64(n-1)
	nokta := make([][2]float64, n)
	for i, v := range deger {
		x := float64(i) * adim
		y := pad + (1-(v-min)/aralik)*(GrafikH-2*pad)
		nokta[i] = [2]float64{x, y}
	}

	b := make([]byte, 0, n*16)
	b = append(b, 'M')
	for i, p := range nokta {
		if i > 0 {
			b = append(b, 'L')
		}
		b = appendSayi(b, p[0])
		b = append(b, ' ')
		b = appendSayi(b, p[1])
	}
	yol = string(b)

	// Dolgu: çizgi + sağ alt + sol alt + kapat.
	ab := make([]byte, 0, len(yol)+48)
	ab = append(ab, yol...)
	ab = append(ab, 'L')
	ab = appendSayi(ab, GrafikW)
	ab = append(ab, ' ')
	ab = appendSayi(ab, GrafikH)
	ab = append(ab, 'L')
	ab = appendSayi(ab, 0)
	ab = append(ab, ' ')
	ab = appendSayi(ab, GrafikH)
	ab = append(ab, 'Z')
	alan = string(ab)
	return yol, alan, min, maks, true
}

// appendSayi float'ı kısa ondalıkla yazar (SVG boyutu küçük kalsın).
func appendSayi(b []byte, v float64) []byte {
	// İki ondalık yeterli: 640 px genişlikte 0,01 px fark görünmez.
	tam := int64(v * 100)
	if tam < 0 {
		b = append(b, '-')
		tam = -tam
	}
	tamsayi := tam / 100
	kesir := tam % 100
	b = appendInt(b, tamsayi)
	if kesir != 0 {
		b = append(b, '.')
		b = append(b, byte('0'+kesir/10))
		if kesir%10 != 0 {
			b = append(b, byte('0'+kesir%10))
		}
	}
	return b
}

func appendInt(b []byte, v int64) []byte {
	if v == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(b, tmp[i:]...)
}
