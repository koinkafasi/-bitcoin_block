package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Kaynak görsel ucu (/kg/{hash}) güvenlik kapısı.
//
// NEDEN TEST: bu uç dışarıdan gelen bir adresi sunucuya indirtiyor.
// Yanlış yazılmış bir hash doğrulaması yol aşımına, SVG kabulü ise
// görselin içine gömülü script'e kapı açar.

// TestGecerliKaynakGorselHash: uzunluk VE karakter birlikte doğrulanır.
//
// Saldırı denemeleri: "../" (yol aşımı), kodlanmış "..", büyük harf
// (dosya sistemi büyük/küçük harf duyarsız olabilir → iki hash aynı
// dosyaya düşer), boşluk, uzunluk ihlali.
func TestGecerliKaynakGorselHash(t *testing.T) {
	kabul := []string{
		"80fe8a843afddb33661ff0f019b8ac11",
		"0123456789abcdef0123456789abcdef",
	}
	for _, s := range kabul {
		assert.True(t, gecerliKaynakGorselHash(s), "kabul edilmeli: %q", s)
	}
	ret := []string{
		"",
		"80FE8A843AFDDB33661FF0F019B8AC11", // büyük harf
		"../../etc/passwd",
		"80fe8a843afddb33661ff0f019b8ac1",   // 31 karakter
		"80fe8a843afddb33661ff0f019b8ac111", // 33 karakter
		"80fe8a843afddb33661ff0f019b8ac1z",  // hex dışı
		"80fe8a84/3afddb33661ff0f019b8ac1",
		"80fe8a843afddb33661ff0f019b8ac1 ",
	}
	for _, s := range ret {
		assert.False(t, gecerliKaynakGorselHash(s), "reddedilmeli: %q", s)
	}
}

// TestKaynakGorselYol: web yolu hash'ten üretilir, uzantı taşımaz.
func TestKaynakGorselYol(t *testing.T) {
	assert.Equal(t, "/kg/abc123", repo.KaynakGorselYol("abc123"))
	assert.Equal(t, "", repo.KaynakGorselYol(""), "boş hash yol üretmez")
}

// TestKaynakGorselDiskYoluDagilimi: ilk 2 karakter alt dizin olur.
//
// NEDEN: tek dizinde on binlerce dosya bazı dosya sistemlerinde yavaşlar;
// ayrıca yedekleme/rsync dağıtımı tek dizinde kilitlenir.
//
// Yol artık KÖKTEN GÖRELİ üretilir (os.Root hapishanesi): mutlak yol
// kurulmaz, böylece depolama dizini dışına çıkmak yapısal olarak imkânsız.
func TestKaynakGorselDiskYoluDagilimi(t *testing.T) {
	yol := kaynakGorselGoreli("80fe8a843afddb33661ff0f019b8ac11", "")
	assert.Equal(t, filepath.Join("80", "80fe8a843afddb33661ff0f019b8ac11"), yol)
	assert.False(t, filepath.IsAbs(yol), "göreli yol kök dışına çıkamaz")
}

// ---- Ölçekli uç: /kg/{hash}/{genislik} ----

// TestKaynakGorselGenislikKapaliListe: genişlik serbest OLMAMALI.
//
// NEDEN TEST (güvenlik, kaynak tüketimi): istemci istediği genişliği
// isteyebilseydi /kg/<hash>/1, /kg/<hash>/2, … ile sınırsız dosya
// üretilebilirdi — disk dolar ve her istek ölçekleme için CPU yakar.
// Liste kapalı; bu test kapının açılmasını engeller.
func TestKaynakGorselGenislikKapaliListe(t *testing.T) {
	for _, g := range []int{320, 480, 768, 960, 1200, 1600} {
		assert.True(t, kaynakGorselGenislikGecerli(g), "kabul edilmeli: %d", g)
	}
	for _, g := range []int{0, -1, 1, 2, 319, 321, 481, 1601, 99999} {
		assert.False(t, kaynakGorselGenislikGecerli(g), "reddedilmeli: %d", g)
	}
}

// TestKaynakGorselGenislikMediaSrcsetIleAyni: iki görsel sistemi aynı
// genişlik kümesini kullanmalı.
//
// NEDEN: ayrı listelere düşerlerse biri güncellenir diğeri kalır; şablon
// 480 ister, uç 404 döner ve sayfada boş kare çıkar.
func TestKaynakGorselGenislikMediaSrcsetIleAyni(t *testing.T) {
	// mediaSrcset'in bastığı genişlikler (bkz. util.go).
	for _, g := range []int{320, 480, 768, 960, 1200, 1600} {
		assert.True(t, kaynakGorselGenislikGecerli(g),
			"mediaSrcset %d basıyor ama /kg/ reddediyor", g)
	}
}

// TestKaynakGorselVaryantYolu: ölçekli dosya orijinalin YANINDA durur ve
// adı onunla ÇAKIŞMAZ.
//
// NEDEN: çakışsaydı ölçekli dosya orijinalin üzerine yazılır, orijinal
// bir daha elde edilemez ve her genişlik aynı dosyayı okur.
func TestKaynakGorselVaryantYolu(t *testing.T) {
	const h = "80fe8a843afddb33661ff0f019b8ac11"
	asil := kaynakGorselGoreli(h, "")
	v480 := kaynakGorselGoreli(h, "-480")
	v768 := kaynakGorselGoreli(h, "-768")

	assert.NotEqual(t, asil, v480, "varyant orijinalin üzerine yazmamalı")
	assert.NotEqual(t, v480, v768, "her genişlik ayrı dosya olmalı")
	assert.Contains(t, v480, filepath.Join("80", ""), "aynı shard dizininde kalmalı")
	assert.Equal(t, asil+"-480", v480)
}

// TestKaynakGorselOlceklemeKucultuyor — değişikliğin ASIL İDDİASI.
//
// Canlı ölçüm (5 Kasım 2026): ana sayfadaki beş /kg/ görseli 1,39 MB
// (en büyüğü 708 KB PNG), en geniş yuva 474x266. Bu test, ölçeklemenin
// gerçekten küçülttüğünü ve en-boy oranını koruduğunu sabitler; biri
// ResizeToWidth'i "iyileştirir" de kırarsa burada görünür.
func TestKaynakGorselOlceklemeKucultuyor(t *testing.T) {
	// 1600x900 PNG — kaynaklardan gelen büyük kapakları temsil eder.
	// Düz renk değil: düz renk sıkışıp testi anlamsız kılar.
	img := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1600; x++ {
			img.Set(x, y, color.RGBA{uint8(x * y % 251), uint8(x % 253), uint8(y % 247), 255})
		}
	}
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, img))
	asil := buf.Bytes()

	kucuk, err := platform.ResizeToWidth(asil, 480)
	assert.NoError(t, err)
	assert.NotEmpty(t, kucuk)
	assert.Less(t, len(kucuk), len(asil), "ölçekli dosya orijinalden küçük olmalı")

	// En-boy oranı korunmalı: 1600x900 → 480x270.
	ol, _, err := image.Decode(bytes.NewReader(kucuk))
	assert.NoError(t, err)
	assert.Equal(t, 480, ol.Bounds().Dx())
	assert.Equal(t, 270, ol.Bounds().Dy())
}

// TestKaynakGorselOlceklemeBuyutmuyor: küçük görsel şişirilmemeli.
//
// NEDEN: 200px'lik bir kaynak görseli 1600'e çıkarmak dosyayı büyütür,
// kaliteyi artırmaz. ResizeToWidth kaynak genişliğinde durur.
func TestKaynakGorselOlceklemeBuyutmuyor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 251), uint8(y % 253), 128, 255})
		}
	}
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, img))

	kucuk, err := platform.ResizeToWidth(buf.Bytes(), 1600)
	assert.NoError(t, err)
	ol, _, err := image.Decode(bytes.NewReader(kucuk))
	assert.NoError(t, err)
	assert.Equal(t, 200, ol.Bounds().Dx(), "kaynak genişliğinde durmalı")
}
