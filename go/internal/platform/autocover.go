// Otomatik kapak görseli: içerik görseli yoksa kategori `image_service` değerine göre
// (a) Pexels API (PEXELS_API_KEY varsa) ya da (b) deterministik marka kapak üretir.
// spec 06 K-D (01 §L2)
package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"hash/fnv"
	"image"
	"image/color"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ErrNoPexels Pexels anahtarı yok ya da API yanıt vermedi.
var ErrNoPexels = errors.New("pexels kullanılamıyor")

// AutoCoverSpec otomatik kapak üretimi parametreleri.
type AutoCoverSpec struct {
	Title    string
	Category string
	Slug     string
	Badge    string
	Query    string // pexels arama sorgusu
	Width    int
	Height   int
}

// pexelsSearch Pexels API'den görsel arar (API key yoksa sessizce atlanır).
func pexelsSearch(query, apiKey string) ([]byte, string, error) {
	if apiKey == "" || strings.TrimSpace(query) == "" {
		return nil, "", ErrNoPexels
	}
	url := "https://api.pexels.com/v1/search?per_page=1&orientation=landscape&size=large&query=" +
		url.QueryEscape(query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", apiKey)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", ErrNoPexels
	}
	var body struct {
		Photos []struct {
			URL string `json:"url"`
			Src struct {
				Original string `json:"original"`
				Large    string `json:"large"`
			} `json:"src"`
		} `json:"photos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || len(body.Photos) == 0 {
		return nil, "", ErrNoPexels
	}
	imgURL := body.Photos[0].Src.Large
	if imgURL == "" {
		imgURL = body.Photos[0].Src.Original
	}
	if imgURL == "" {
		return nil, "", ErrNoPexels
	}
	ir, err := client.Get(imgURL)
	if err != nil {
		return nil, "", err
	}
	defer ir.Body.Close()
	if ir.StatusCode != http.StatusOK {
		return nil, "", ErrNoPexels
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(ir.Body); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), ir.Header.Get("Content-Type"), nil
}

// AutoCover kategori görsel hizmetine göre kapak üretir.
// Dönüş: JPEG baytları + kaynak ("pexels"|"generate").
func AutoCover(spec AutoCoverSpec) ([]byte, string, error) {
	if spec.Width <= 0 {
		spec.Width = 1200
	}
	if spec.Height <= 0 {
		spec.Height = 675
	}
	if key := strings.TrimSpace(os.Getenv("PEXELS_API_KEY")); key != "" {
		if data, _, err := pexelsSearch(spec.Query, key); err == nil {
			if out, _, err := CropToRatio(data, spec.Width, 16.0/9.0); err == nil {
				return out, "pexels", nil
			}
		}
	}
	img := generateBrandCover(spec)
	out, err := encodeJPEG(img, 82)
	if err != nil {
		return nil, "", err
	}
	return out, "generate", nil
}

// generateBrandCover başlıktan türetilen deterministik marka kapak çizer.
func generateBrandCover(spec AutoCoverSpec) image.Image {
	w, h := spec.Width, spec.Height
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := seedFrom(spec.Slug + "|" + spec.Title)
	r1, r2, r3 := paletteFor(seed)
	// dikey gradyan
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		setRow(img, y, lerpRGBA(r1, r2, t))
	}
	// diyagonal accent ışıkları
	for i := 0; i < 3; i++ {
		phase := float64(seed%1000)/1000.0 + float64(i)*0.33
		cx := int(float64(w) * math.Mod(phase, 1.0))
		rad := w / 3
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				d := math.Hypot(float64(x-cx), float64(y-h/2))
				if d < float64(rad) {
					a := 0.16 * (1 - d/float64(rad))
					blendPixel(img, x, y, r3, a)
				}
			}
		}
	}
	// alt karartma (metin okunurluğu)
	overlayBottomShade(img)
	// kategori rozeti (kutu rengi marka accent'i; yazı drawBadge içinde beyaz)
	if spec.Category != "" {
		drawBadge(img, strings.ToUpper(spec.Category), r3, 34, 64, 64)
	}
	// başlık (en fazla 3 satır)
	title := strings.TrimSpace(spec.Title)
	if title != "" {
		lines := wrapText(title, 3, 30)
		size := 52
		if len(title) > 90 {
			size = 42
		}
		y := h - 96 - (len(lines)-1)*int(float64(size)*1.25)
		for _, ln := range lines {
			// gölge (okunabilirlik) + opak beyaz metin
			drawTextBlock(img, ln, 64, y+3, "left", size, 700, 1.25, color.RGBA{0, 0, 0, 255})
			drawTextBlock(img, ln, 64, y, "left", size, 700, 1.25, color.RGBA{255, 255, 255, 255})
			y += int(float64(size) * 1.25)
		}
	}
	// marka
	drawTextBlock(img, "theblock", w-230, h-70, "left", 30, 700, 1.2, color.RGBA{255, 255, 255, 255})
	return img
}

func seedFrom(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// paletteFor tohumdan koyu-okunur marka paleti üretir.
func paletteFor(seed uint64) (color.RGBA, color.RGBA, color.RGBA) {
	hue := float64(seed%360) / 360.0
	c1 := hsl(hue, 0.55, 0.16)
	c2 := hsl(math.Mod(hue+0.12, 1.0), 0.62, 0.30)
	c3 := hsl(math.Mod(hue+0.5, 1.0), 0.85, 0.55)
	return c1, c2, c3
}

func hsl(h, s, l float64) color.RGBA {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h*6, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch int(h * 6) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255), G: uint8((g + m) * 255), B: uint8((b + m) * 255), A: 255,
	}
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// setRow görüntüyü tek satır boyunca düz renkle doldurur.
func setRow(img *image.RGBA, y int, c color.RGBA) {
	b := img.Bounds()
	if y < b.Min.Y || y >= b.Max.Y {
		return
	}
	for x := b.Min.X; x < b.Max.X; x++ {
		blendPixel(img, x, y, c, 1)
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA, a float64) {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return
	}
	o := img.PixOffset(x, y)
	img.Pix[o+0] = uint8(float64(img.Pix[o+0]) + (float64(c.R)-float64(img.Pix[o+0]))*a)
	img.Pix[o+1] = uint8(float64(img.Pix[o+1]) + (float64(c.G)-float64(img.Pix[o+1]))*a)
	img.Pix[o+2] = uint8(float64(img.Pix[o+2]) + (float64(c.B)-float64(img.Pix[o+2]))*a)
	img.Pix[o+3] = 255
}

// overlayBottomShade alt yarıyı koyulaştırır (metin kontrastı).
func overlayBottomShade(img *image.RGBA) {
	b := img.Bounds()
	start := b.Dy() / 2
	for y := start; y < b.Dy(); y++ {
		t := float64(y-start) / float64(b.Dy()-start)
		blendRowAlpha(img, y, 0.55*t*t)
	}
}

func blendRowAlpha(img *image.RGBA, y int, a float64) {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		blendPixel(img, x, y, color.RGBA{0, 0, 0, 255}, a)
	}
}

// wrapText metni kelime sınırında bölerek en fazla n satıra sarar.
func wrapText(text string, maxLines, maxChars int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := ""
	for _, wd := range words {
		cand := wd
		if cur != "" {
			cand = cur + " " + wd
		}
		if len([]rune(cand)) > maxChars && cur != "" {
			lines = append(lines, cur)
			cur = wd
			if len(lines) == maxLines {
				return lines
			}
			continue
		}
		cur = cand
	}
	if cur != "" && len(lines) < maxLines {
		lines = append(lines, cur)
	}
	if len(lines) > 0 && len(lines) == maxLines {
		last := lines[len(lines)-1]
		if r := []rune(last); len(r) > maxChars {
			lines[len(lines)-1] = string(r[:maxChars-1]) + "…"
		}
	}
	return lines
}
