package platform

import (
	"bytes"
	"context"
	"fmt"
	"image"
	stddraw "image/draw"
	"image/jpeg"
	"image/png"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// Targets hedef en-boy oranları (spec: 16:9 manşet, 4:3 içerik, kare/9:16 sosyal).
type Target struct {
	Name  string
	W     int // hedef pixel genişlik (orana göre yükseklik hesaplanır)
	H     int // 0 ise W/oran
	Ratio float64
	Fit   bool // true=kapak kırp (center-crop), false=fit inside
}

var (
	Target_16x9   = Target{Name: "16x9", W: 1200, Ratio: 16.0 / 9.0, Fit: false}
	Target_16x9s  = Target{Name: "16x9", W: 800, Ratio: 16.0 / 9.0, Fit: false}
	Target_4x3    = Target{Name: "4x3", W: 900, Ratio: 4.0 / 3.0, Fit: false}
	Target_Square = Target{Name: "1x1", W: 720, Ratio: 1.0, Fit: false}
)

// DecodeImage görsel verisini çözer (jpeg/png/webp).
func DecodeImage(data []byte) (image.Image, error) {
	if img, err := jpeg.Decode(bytes.NewReader(data)); err == nil {
		return img, nil
	}
	if img, err := png.Decode(bytes.NewReader(data)); err == nil {
		return img, nil
	}
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("desteklenmeyen görsel (jpg/png/webp): %w", err)
	}
	return img, nil
}

// CropToRatio görseli hedef orana center-crop yapar; 2. geçiş istenen genişliğe ölçekler.
// Kodlanan çıktı: JPEG (q=82).
func CropToRatio(data []byte, width int, ratio float64) ([]byte, image.Image, error) {
	src, err := DecodeImage(data)
	if err != nil {
		return nil, nil, err
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return nil, nil, fmt.Errorf("görsel boyutu geçersiz")
	}

	targetRatio := ratio
	if targetRatio == 0 {
		targetRatio = 16.0 / 9.0
	}

	// center-crop dikdörtgeni
	cur := float64(sw) / float64(sh)
	var crop image.Rectangle
	if cur > targetRatio {
		// fazla geniş: yanlardan kırp
		nw := int(float64(sh) * targetRatio)
		if nw > sw {
			nw = sw
		}
		x0 := (sw - nw) / 2
		crop = image.Rect(x0, 0, x0+nw, sh)
	} else {
		// fazla uzun: üstten/alttan
		nh := int(float64(sw) / targetRatio)
		if nh > sh {
			nh = sh
		}
		y0 := (sh - nh) / 2
		crop = image.Rect(0, y0, sw, y0+nh)
	}

	cropped := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	stddraw.Draw(cropped, cropped.Bounds(), src, crop.Min, stddraw.Src)

	dstW := width
	if dstW <= 0 {
		dstW = cropped.Bounds().Dx()
	}
	dstH := int(float64(dstW) / targetRatio)
	if dstH <= 0 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), xdraw.Src, nil)

	out, err := encodeJPEG(dst, 82)
	return out, dst, err
}

// CreateThumb küçük kare önizleme üretir.
//
// JPEG girdide iş image-svc'ye devredilir (kare kırpma = ratio 1.0).
// Ölçüm (8 Kasım 2026): 480 px kare küçültme yerel yolda ~40 ms, serviste
// ~18 ms. Panel ve medya uçları her yeni görselde bir kez üretir; liste
// sayfaları çok sayıda yeni görselde aynı anda tetikleyebiliyor.
func CreateThumb(data []byte, size int) ([]byte, error) {
	if size > 0 {
		if cfg, err := jpeg.DecodeConfig(bytes.NewReader(data)); err == nil && jpegMi(data) {
			yan := cfg.Width
			if cfg.Height < yan {
				yan = cfg.Height
			}
			// Küçültme değilse (istenen boy kaynaktan büyük) yerel yolda kalır.
			if size < yan {
				if out, err := imageSvcIsle(context.Background(), data, 1.0, size, 80, "jpeg"); err == nil {
					return out, nil
				}
			}
		}
	}
	src, err := DecodeImage(data)
	if err != nil {
		return nil, err
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 {
		return nil, fmt.Errorf("görsel boyutu geçersiz")
	}
	side := sw
	if sh < side {
		side = sh
	}
	x0, y0 := (sw-side)/2, (sh-side)/2

	img := image.NewRGBA(image.Rect(0, 0, side, side))
	stddraw.Draw(img, img.Bounds(), src, image.Pt(x0, y0), stddraw.Src)
	if size > 0 && size < side {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)
		img = dst
	}
	return encodeJPEG(img, 80)
}

// ResizeToWidth görselin en-boy oranını koruyarak hedef genişliğe ölçekler ve
// JPEG olarak kodlar. Responsive srcset varyantları için kullanılır.
//
// Görsel bir mikroservis adresi tanımlıysa (SetImageSvcURL) ve girdi JPEG ise
// iş image-svc'ye (Rust/libjpeg-turbo) devredilir; servis yoksa, hata verirse
// veya girdi PNG/WebP ise aşağıdaki yerel yol kullanılır.
func ResizeToWidth(data []byte, width int) ([]byte, error) {
	if out, ok := resizeViaImageSvc(context.Background(), data, width); ok {
		return out, nil
	}
	src, err := DecodeImage(data)
	if err != nil {
		return nil, err
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return nil, fmt.Errorf("görsel boyutu geçersiz")
	}
	if width <= 0 || width > sw {
		width = sw // küçük görseli büyütme
	}
	dstH := int((int64(sh)*int64(width) + int64(sw)/2) / int64(sw))
	if dstH <= 0 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, dstH))
	// Saydam PNG'ler JPEG'e dönüşürken siyah zemin oluşmasın.
	stddraw.Draw(dst, dst.Bounds(), image.White, image.Point{}, stddraw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return encodeJPEG(dst, 82)
}

// TransformImage rotate/flip uygular. JPEG/PNG girişlerinde aynı biçimde kodlar.
func TransformImage(data []byte, ext, op string) ([]byte, image.Image, error) {
	src, err := DecodeImage(data)
	if err != nil {
		return nil, nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var dst *image.RGBA
	switch op {
	case "rotate-left":
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	case "rotate-right":
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	case "flip-h":
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, y, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	case "flip-v":
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(x, h-1-y, src.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	default:
		return nil, nil, fmt.Errorf("bilinmeyen görsel işlemi")
	}
	out, err := encodeByExt(dst, ext)
	return out, dst, err
}

func encodeByExt(img image.Image, ext string) ([]byte, error) {
	switch strings.ToLower(ext) {
	case ".png":
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case ".jpg", ".jpeg":
		return encodeJPEG(img, 90)
	default:
		return nil, fmt.Errorf("bu görsel biçimi düzenlenemiyor")
	}
}

func encodeJPEG(img image.Image, q int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
