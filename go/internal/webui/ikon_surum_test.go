package webui

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ikon sprite sürümü (Faz 181).
//
// NEDEN: sprite URL'si hiç değişmiyordu. Sunucu sürümsüz istekleri
// "max-age=300, must-revalidate" ile sunuyordu, ama Cloudflare'de 23 Ekim'den
// kalma bir kopya **immutable** başlığıyla duruyordu. Ölçüm:
//
//	alan adı : 11.412 bayt, 41 sembol, age=693705 (8 gün), cf-cache-status=HIT
//	sunucu   : 246.211 bayt, 641 sembol
//
// Eksik sembol HATA VERMEZ: tarayıcı `<use>` hedefini bulamaz ve sessizce
// boş kutu basar. Kullanıcı "ikonlar kayboldu" der, günlükte iz olmaz.
//
// Çözüm: URL'e içerik özeti (`?v=<sha1/8>`). İçerik değişince URL değişir,
// önbellek kendiliğinden tazelenir ve immutable önbellek doğru olur.
//
// Bu test sürümün unutulmasını engeller: sprite güncellenip
// `deploy/scripts/ikon-surum.sh` çalıştırılmazsa kırmızı olur.

// ikonSurumu sprite içeriğinin kısa özeti.
func ikonSurumu(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "vendor", "tabler-icons.svg"))
	require.NoError(t, err)
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])[:8]
}

// TestIkonSurumuGuncel: her sprite referansı güncel sürümü taşımalı.
func TestIkonSurumuGuncel(t *testing.T) {
	surum := ikonSurumu(t)

	var eski, surumsuz []string
	denetlenen := 0

	for _, f := range dosyalar(t) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for i, satir := range strings.Split(string(b), "\n") {
			if !strings.Contains(satir, "tabler-icons.svg") {
				continue
			}
			// Yorum satırları serbest (yol açıklaması).
			k := strings.TrimSpace(satir)
			if strings.HasPrefix(k, "//") || strings.HasPrefix(k, "*") || strings.HasPrefix(k, "/*") {
				continue
			}
			denetlenen++
			yer := filepath.Base(f) + ":" + strconv.Itoa(i+1)
			if m := regexp.MustCompile(`tabler-icons\.svg\?v=([0-9a-f]+)`).FindStringSubmatch(satir); m != nil {
				if m[1] != surum {
					eski = append(eski, yer+" (v="+m[1]+", beklenen "+surum+")")
				}
				continue
			}
			surumsuz = append(surumsuz, yer)
		}
	}

	assert.Greaterf(t, denetlenen, 300, "beklenenden az sprite referansı bulundu (%d)", denetlenen)
	assert.Emptyf(t, surumsuz,
		"sprite referansı sürümsüz — önbellek bayatlar: %v\n"+
			"düzeltmek için: bash deploy/scripts/ikon-surum.sh", surumsuz[:minInt(8, len(surumsuz))])
	assert.Emptyf(t, eski,
		"sprite sürümü eski — güncellemek için: bash deploy/scripts/ikon-surum.sh: %v",
		eski[:minInt(8, len(eski))])
}

// TestIkonSpriteSaglam: sprite'ın kendisi paketlenmiş ve sembol dolu olmalı.
// Yayına eksik sprite gitmesi, tüm ikonların sessizce kaybolması demek.
func TestIkonSpriteSaglam(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "vendor", "tabler-icons.svg"))
	require.NoError(t, err)

	n := strings.Count(string(b), "<symbol")
	assert.GreaterOrEqualf(t, n, 600,
		"sprite %d sembol içeriyor; 641 bekleniyordu (eksik sprite = boş ikonlar)", n)

	// Şablonların kullandığı semboller sprite'ta var mı?
	semboller := map[string]bool{}
	for _, m := range regexp.MustCompile(`<symbol[^>]*\bid="([^"]+)"`).FindAllStringSubmatch(string(b), -1) {
		semboller[m[1]] = true
	}

	var eksik []string
	for _, f := range dosyalar(t) {
		s, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, m := range regexp.MustCompile(`tabler-icons\.svg(?:\?v=[0-9a-f]+)?#(ti-[a-z0-9-]+)`).FindAllStringSubmatch(string(s), -1) {
			// Dinamik hedefler (`#ti-` + ifade) statik çözülemez: yalnız
			// "ti-" gövdesi kalır, onları atla.
			if len(m[1]) <= 3 {
				continue
			}
			if !semboller[m[1]] {
				eksik = append(eksik, filepath.Base(f)+": "+m[1])
			}
		}
	}
	assert.Emptyf(t, eksik, "şablonun istediği sembol sprite'ta yok (boş kutu çizer): %v",
		eksik[:minInt(10, len(eksik))])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
