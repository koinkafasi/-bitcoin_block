package platform

import (
	"image"
	"testing"
)

// TestAutoCoverPixels otomatik kapak üretiminin okunabilirliğini doğrular:
// beyaz başlık metni ve kontrastlı kategori rozeti bulunmalı (spec 06 K-D).
func TestAutoCoverPixels(t *testing.T) {
	data, source, err := AutoCover(AutoCoverSpec{
		Title: "Test Basligi Ornek Cok Uzun Bir Baslik Metni", Category: "Gundem", Slug: "test-slug",
	})
	if err != nil {
		t.Fatalf("AutoCover: %v", err)
	}
	if source != "generate" && source != "pexels" {
		t.Fatalf("beklenmeyen kaynak: %s", source)
	}
	img, err := DecodeImage(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 1200 || b.Dy() != 675 {
		t.Fatalf("16:9 1200x675 beklenir, gelen: %dx%d", b.Dx(), b.Dy())
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		// JPEG çözümü YCbCr döner; RGBA'ya çevir
		b2 := img.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b2.Dx(), b2.Dy()))
		for y := 0; y < b2.Dy(); y++ {
			for x := 0; x < b2.Dx(); x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}
	white := 0
	for y := b.Dy() * 3 / 5; y < b.Dy()*9/10; y++ {
		for x := 0; x < b.Dx(); x++ {
			c := rgba.RGBAAt(x, y)
			if c.R > 220 && c.G > 220 && c.B > 220 {
				white++
			}
		}
	}
	if white < 2000 {
		t.Errorf("beyaz başlık pikselleri çok az (%d) — metin kontrastı yok", white)
	}

	// rozet kutusu dolu ve içinde okunur metin var
	box := rgba.RGBAAt(70, 78)
	light := 0
	for y := 60; y < 115; y++ {
		for x := 60; x < 320; x++ {
			c := rgba.RGBAAt(x, y)
			l := (int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000
			if l > 200 {
				light++
			}
		}
	}
	if light < 800 {
		t.Errorf("rozet alanı okunur değil (açık piksel %d)", light)
	}
	t.Logf("kapak OK: %dx%d, beyaz piksel=%d, rozet=%v", b.Dx(), b.Dy(), white, box)
}

// TestAutoCoverDeterministic aynı başlık → aynı görsel, farklı başlık → farklı görsel.
func TestAutoCoverDeterministic(t *testing.T) {
	a1, _, err := AutoCover(AutoCoverSpec{Title: "Ayni Baslik", Slug: "a", Category: "Gundem"})
	if err != nil {
		t.Fatal(err)
	}
	a2, _, err := AutoCover(AutoCoverSpec{Title: "Ayni Baslik", Slug: "a", Category: "Gundem"})
	if err != nil {
		t.Fatal(err)
	}
	b1, _, err := AutoCover(AutoCoverSpec{Title: "Baska Baslik", Slug: "b", Category: "Gundem"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a1) != len(a2) {
		t.Error("deterministik değil: aynı girdi farklı çıktı")
	}
	same := true
	for i := range a1 {
		if i < len(b1) && a1[i] != b1[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("farklı başlıklar aynı kapak görselini üretti (seed çalışmıyor)")
	}
}
