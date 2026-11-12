package platform

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

// benchJPEG 1600×900 gerçekçi bir JPEG üretir (srcset ölçümü için).
func benchJPEG(b *testing.B) []byte {
	b.Helper()
	const w, h = 1600, 900
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: uint8((x * y) % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
	return buf.Bytes()
}

// BenchmarkResizeToWidthYerel saf Go yolunu ölçer (image-svc kapalı).
func BenchmarkResizeToWidthYerel(b *testing.B) {
	SetImageSvcURL("")
	data := benchJPEG(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ResizeToWidth(data, 1200); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResizeToWidthImageSvc(b *testing.B) {
	u := os.Getenv("IMAGE_SVC_URL")
	if u == "" {
		b.Skip("IMAGE_SVC_URL tanımlı değil")
	}
	SetImageSvcURL(u)
	defer SetImageSvcURL("")
	data := benchJPEG(b)

	// Isınma: ilk çağrıda bağlantı kurulumu ölçüme girmesin.
	if _, err := ResizeToWidth(data, 1200); err != nil {
		b.Fatalf("ısınma çağrısı: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := ResizeToWidth(data, 1200)
		if err != nil {
			b.Fatal(err)
		}
		if len(out) == 0 {
			b.Fatal("boş çıktı")
		}
	}
}
