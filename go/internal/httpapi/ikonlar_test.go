package httpapi

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

var (
	reSembol   = regexp.MustCompile(`id="(ti-[a-z0-9-]+)"`)
	reKullanim = regexp.MustCompile(`#(ti-[a-z0-9-]+)`)
)

// spriteSembolleri tam sprite'taki sembol kimliklerini döndürür.
func spriteSembolleri(t *testing.T, dosya string) map[string]bool {
	t.Helper()
	b, err := webui.FS.ReadFile("static/vendor/" + dosya)
	require.NoErrorf(t, err, "%s okunamadı", dosya)
	out := map[string]bool{}
	for _, m := range reSembol.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = true
	}
	return out
}

func TestIkonSpriteKoddaKullanilanlariKapsar(t *testing.T) {
	sembol := spriteSembolleri(t, "tabler-icons.svg")
	require.NotEmpty(t, sembol, "sprite boş")

	kullanilan := map[string]string{} // ad -> ilk görüldüğü dosya
	err := fs.WalkDir(webui.FS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := webui.FS.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range reKullanim.FindAllStringSubmatch(string(b), -1) {
			if _, ok := kullanilan[m[1]]; !ok {
				kullanilan[m[1]] = p
			}
		}
		return nil
	})
	require.NoError(t, err)

	var eksik []string
	for ad, dosya := range kullanilan {
		// Çekirdek sprite'takiler public'te kullanılıyor; onlar ayrı dosyada.
		if !sembol[ad] {
			eksik = append(eksik, ad+" ("+dosya+")")
		}
	}
	sort.Strings(eksik)
	require.Emptyf(t, eksik,
		"şu ikonlar şablonlarda kullanılıyor ama sprite'ta yok — boş kutu "+
			"olarak basılırlar. Çözüm: adı deploy/scripts/ikon-uret.py "+
			"listesine ekleyip sprite'ı yeniden üretin:\n%s",
		strings.Join(eksik, "\n"))
}

func TestIkonCekirdekSpritePublicIcinYeterli(t *testing.T) {
	cekirdek := spriteSembolleri(t, "ikon-cekirdek.svg")
	require.NotEmpty(t, cekirdek)

	pub, err := webui.FS.ReadFile("templates/partials.html")
	require.NoError(t, err)
	s := string(pub)
	cut := strings.Index(s, `{{define "admin_head"}}`)
	require.Greater(t, cut, 0)
	pubHead := s[:cut]

	// Public head'de yalnız çekirdek sprite referans edilmeli.
	assert.NotContains(t, pubHead, "tabler-icons.svg?v=b89545a6",
		"public head tam sprite'ı yüklememeli (233 KB, 4 ikon için)")
	require.Contains(t, pubHead, "ikon-cekirdek.svg")

	// Public şablonlarda kullanılan ikonlar çekirdekte olmalı.
	var eksik []string
	for _, m := range reKullanim.FindAllStringSubmatch(pubHead, -1) {
		if !cekirdek[m[1]] {
			eksik = append(eksik, m[1])
		}
	}
	assert.Emptyf(t, eksik, "public'te kullanılan ama çekirdek sprite'ta olmayan: %v", eksik)
}

// TestIkonManifestSpriteIleTutarli: galeri manifesti (ikonlar.txt) sprite ile
// aynı listeyi göstermeli; yoksa galeride tıklanan ad boş kutu basar.
func TestIkonManifestSpriteIleTutarli(t *testing.T) {
	sembol := spriteSembolleri(t, "tabler-icons.svg")
	adlar := ikonAdlari()
	require.NotEmpty(t, adlar, "ikonlar.txt boş — deploy/scripts/ikon-uret.py çalıştırılmalı")

	var eksik []string
	for _, a := range adlar {
		if !sembol["ti-"+a] {
			eksik = append(eksik, a)
		}
	}
	assert.Emptyf(t, eksik, "manifestte olup sprite'ta olmayan: %v", eksik)
	assert.GreaterOrEqual(t, len(adlar), 400,
		"kütüphane beklenenden küçük (%d); üretici listesi kırpılmış olabilir", len(adlar))
}

// TestIkonSpinnerDosyalari: galeride listelenen her spinner gerçekten
// satır içi basılabiliyor ve rengi temadan alıyor (currentColor).
func TestIkonSpinnerDosyalari(t *testing.T) {
	for _, ad := range ikonSpinnerAdlari() {
		svg := string(ikonSpinnerSVG(ad))
		require.NotEmptyf(t, svg, "spinner basılamadı: %s", ad)
		assert.Containsf(t, svg, "currentColor",
			"spinner rengi sabit olmamalı (currentColor): %s", ad)
		assert.NotContainsf(t, svg, "#000",
			"spinner sabit siyah renk taşıyor: %s", ad)
	}
}

// TestIkonAnimasyonSiniflariCSSdeTanimli: galeride tanıtılan her animasyon
// sınıfı ikonlar.css'te gerçekten var olmalı; yoksa "çalışmıyor" der.
func TestIkonAnimasyonSiniflariCSSdeTanimli(t *testing.T) {
	b, err := webui.FS.ReadFile("static/ikonlar.css")
	require.NoError(t, err)
	css := string(b)

	for _, sinif := range []string{
		"ico-spin", "ico-spin-yavas", "ico-pulse", "ico-ring",
		"ico-bounce", "ico-draw", "ico-shake", "ico-fade-in",
	} {
		assert.Containsf(t, css, "."+sinif, "animasyon sınıfı CSS'te yok: %s", sinif)
	}
	// Hareket hassasiyeti: hepsi reduced-motion altında durmalı.
	assert.Contains(t, css, "prefers-reduced-motion",
		"animasyonlar reduced-motion altında durdurulmalı")
}

// TestIkonLisansiBelgeli: MIT lisans metni depoda kalmalı. Tabler ve
// SVG Spinners MIT; telif bildirimi korunmak zorunda.
func TestIkonLisansiBelgeli(t *testing.T) {
	b, err := webui.FS.ReadFile("static/vendor/IKON-LISANS.txt")
	require.NoError(t, err, "vendor/IKON-LISANS.txt eksik")
	s := string(b)
	for _, gerekli := range []string{"MIT", "Tabler", "SVG Spinners", "Paweł Kuna", "Utkarsh Verma"} {
		assert.Containsf(t, s, gerekli, "lisans dosyasında eksik: %s", gerekli)
	}
}
