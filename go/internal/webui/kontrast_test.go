package webui

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linKanal(v float64) float64 {
	v /= 255
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func parlaklik(r, g, b float64) float64 {
	return 0.2126*linKanal(r) + 0.7152*linKanal(g) + 0.0722*linKanal(b)
}

func oran(a, b [3]float64) float64 {
	l1 := parlaklik(a[0], a[1], a[2])
	l2 := parlaklik(b[0], b[1], b[2])
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func renkCoz(t *testing.T, h string) [3]float64 {
	t.Helper()
	h = regexp.MustCompile(`[^0-9a-fA-F]`).ReplaceAllString(h, "")
	require.Len(t, h, 6, "renk çözülemedi")
	var c [3]float64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		require.NoError(t, err)
		c[i] = float64(v)
	}
	return c
}

// tokenDegeri: CSS dosyasındaki `--ad: değer;` (ilk eşleşme).
func tokenDegeriDosya(t *testing.T, dosya, ad string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", dosya))
	require.NoError(t, err)
	m := regexp.MustCompile(regexp.QuoteMeta(ad) + `\s*:\s*([^;]+);`).FindStringSubmatch(string(b))
	require.NotNilf(t, m, "%s içinde %s bulunamadı", dosya, ad)
	return regexp.MustCompile(`var\([^)]*\)`).ReplaceAllString(m[1], "#ffffff")
}

var (
	beyaz    = [3]float64{255, 255, 255}
	sayfaBg  = [3]float64{243, 242, 239} // --bg #f3f2ef
	kenarBg  = [3]float64{231, 228, 221} // --bg-soft #e7e4dd (en koyu)
	kartBg   = [3]float64{253, 253, 251} // --surface #fdfdfb
	rozetBgU = [3]float64{224, 235, 226} // 14% --pn-good over kart
)

// TestKucukMetinKontrasti: türetilmiş her ton, kullanıldığı zeminde 4.5.
func TestKucukMetinKontrasti(t *testing.T) {
	durumlar := []struct {
		ad     string
		dosya  string
		deger  string
		zemin  [3]float64
		acikla string
	}{
		{"--ok-fill", "admin-shadcn.css", "--ok-fill", beyaz, "rozet dolgusu + beyaz yazı"},
		{"--warn-fill", "admin-shadcn.css", "--warn-fill", beyaz, "rozet dolgusu + beyaz yazı"},
		{"--accent-text", "app.css", "--accent-text", sayfaBg, "bağlantı metni / sayfa zemini"},
		{"--accent-text", "app.css", "--accent-text", kenarBg, "bağlantı metni / yan menü"},
		{"--accent-text", "app.css", "--accent-text", kartBg, "bağlantı metni / kart"},
		{"--pn-sig-fg", "pano.css", "--pn-sig-fg", kartBg, "12.5px pano bağlantısı"},
		{"--pn-good-fg", "pano.css", "--pn-good-fg", rozetBgU, "delta rozeti metni"},
	}

	for _, d := range durumlar {
		ham := tokenDegeriDosya(t, d.dosya, d.deger)
		o := oran(renkCoz(t, ham), d.zemin)
		assert.GreaterOrEqualf(t, o, 4.5,
			"%s (%s) = %s → kontrast %.2f (eşik 4.5)", d.ad, d.acikla, ham, o)
	}
}

func TestMarkaTokenlariKorundu(t *testing.T) {
	// Marka rengi kullanıcı seçimi; varsayılanı app.css'te.
	assert.Equal(t, "#e11d48", tokenDegeriDosya(t, "app.css", "--accent"),
		"marka rengi değişmemeli (A-7 türetilmiş tonlarla çözüldü)")

	// Pano sinyal rengi Pacely'den ölçülmüş değer; grafiklerde aynı kalmalı.
	assert.Equal(t, "#d9532a", tokenDegeriDosya(t, "pano.css", "--pn-sig"))

	// --ok ve --warn kendileri (ikon/metin olarak) korunur; yalnız dolgu
	// tonları türetildi.
	assert.Equal(t, "#16a34a", tokenDegeriDosya(t, "app.css", "--ok"))
	assert.Equal(t, "#d97706", tokenDegeriDosya(t, "app.css", "--warn"))
}

// TestAccentFillYedegi: marka kiti yoksa CSS yedeği devreye girmeli.
func TestAccentFillYedegi(t *testing.T) {
	for _, ikili := range [][2]string{
		{"admin-shadcn.css", "var(--accent-fill, var(--accent))"},
		{"admin-theme.css", "var(--accent-fill, var(--accent))"},
	} {
		b, err := os.ReadFile(filepath.Join("static", ikili[0]))
		require.NoError(t, err)
		assert.Containsf(t, string(b), ikili[1],
			"%s marka dolgusu için yedekli ton kullanmalı", ikili[0])
	}

	b, err := os.ReadFile(filepath.Join("static", "app.css"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "var(--accent-text, var(--accent))",
		"bağlantı rengi yedekli metin tonu kullanmalı")
}
