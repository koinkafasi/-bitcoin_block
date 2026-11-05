package platform

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestResizeToWidthPreservesRatio(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 200))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, err := ResizeToWidth(buf.Bytes(), 200)
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeImage(out)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 100 {
		t.Fatalf("resize boyutu = %v, 200x100 beklenirdi", img.Bounds())
	}
	// Kaynaktan büyük genişlik istemek görseli büyütmemeli.
	out, err = ResizeToWidth(buf.Bytes(), 1600)
	if err != nil {
		t.Fatal(err)
	}
	img, _ = DecodeImage(out)
	if img.Bounds().Dx() != 400 {
		t.Fatalf("upscale engellenmedi: %v", img.Bounds())
	}
}

func TestTransformImageDimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 3))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, img, err := TransformImage(buf.Bytes(), ".png", "rotate-right")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 || img.Bounds().Dx() != 3 || img.Bounds().Dy() != 2 {
		t.Fatalf("döndürme boyutu = %v", img.Bounds())
	}
	if _, img, err := TransformImage(buf.Bytes(), ".png", "flip-h"); err != nil || img.Bounds().Dx() != 2 || img.Bounds().Dy() != 3 {
		t.Fatalf("çevirme başarısız: %v %v", img, err)
	}
	if _, _, err := TransformImage(buf.Bytes(), ".webp", "rotate-right"); err == nil {
		t.Fatal("webp düzenleme reddedilmeli")
	}
}
