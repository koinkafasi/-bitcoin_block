// Package platform — görsel manşet tasarım belgesini (02 §8 JSON) sunucuda raster'lar.
package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	"math"
	"os"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// DesignReq 02 §8 örnek JSON şemasıyla uyumlu tasarım belgesidir.
type DesignReq struct {
	Version int `json:"version"`
	Canvas  struct {
		W int `json:"w"`
		H int `json:"h"`
	} `json:"canvas"`
	Background struct {
		MediaID *int64 `json:"media_id"`
		Color   string `json:"color"`
		Overlay struct {
			Active    bool   `json:"active"`
			Mode      string `json:"mode"` // gradient | solid | vignette
			Color     string `json:"color"`
			Direction string `json:"direction"` // left|right|top|bottom|tl|tr|bl|br
			Spread    int    `json:"spread"`
			Opacity   int    `json:"opacity"`
		} `json:"overlay"`
	} `json:"background"`
	Badge struct {
		Active    bool   `json:"active"`
		Text      string `json:"text"`
		Color     string `json:"color"`
		TextColor string `json:"textColor"`
		FontSize  int    `json:"fontSize"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
		Radius    int    `json:"radius"`
		Padding   int    `json:"padding"`
		Uppercase bool   `json:"uppercase"`
	} `json:"badge"`
	Layers []LayerReq `json:"layers"`
}

// LayerReq metin katmanıdır.
type LayerReq struct {
	ID          string  `json:"id"`
	Role        string  `json:"role"`
	Text        string  `json:"text"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	FontSize    int     `json:"fontSize"`
	Weight      int     `json:"weight"`
	Color       string  `json:"color"`
	Align       string  `json:"align"`
	LineHeight  float64 `json:"lineHeight"`
	MaxWidth    int     `json:"maxWidth"` // 0 → otomatik (kanvas - x - kenar)
	Uppercase   bool    `json:"uppercase"`
	Shadow      bool    `json:"shadow"`
	ShadowColor string  `json:"shadowColor"`
	Visible     bool    `json:"visible"`
	Locked      bool    `json:"locked"`
}

var (
	fontMu    sync.Mutex
	regFace   *opentype.Font
	boldFace  *opentype.Font
	faceCache = map[int]font.Face{}
)

// loadFonts sistemden DejaVu yükler (yoksa basicfont).
func loadFonts() {
	if regFace != nil {
		return
	}
	parse := func(p string) *opentype.Font {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		f, err := opentype.Parse(b)
		if err != nil {
			return nil
		}
		return f
	}
	regFace = parse("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf")
	boldFace = parse("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf")
}

func faceAt(size, weight int) font.Face {
	loadFonts()
	key := size
	if weight >= 600 {
		key |= 1 << 24
	}
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := faceCache[key]; ok {
		return f
	}
	src := regFace
	if weight >= 600 && boldFace != nil {
		src = boldFace
	}
	var f font.Face = basicfont.Face7x13
	if src != nil {
		if ff, err := opentype.NewFace(src, &opentype.FaceOptions{Size: float64(size), DPI: 72}); err == nil {
			f = ff
		}
	}
	faceCache[key] = f
	return f
}

func parseColor(s string, def color.RGBA) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return def
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return def
	}
	return color.RGBA{r, g, b, 255}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

var errEmptyDesign = errors.New("tasarım boş")

// NormalizeDesign tasarım baytlarını normalize eder: çift-encode edilmiş
// (JSON string) girdiyi iç nesneye indirger (geriye uyumluluk).
func NormalizeDesign(b []byte) []byte {
	t := bytes.TrimSpace(b)
	if len(t) > 0 && t[0] == '"' {
		var s string
		if json.Unmarshal(t, &s) == nil {
			return []byte(s)
		}
	}
	return b
}

// RenderDesign JSON tasarımı JPEG baytlarına basar (metin netliği için JPEG).
func RenderDesign(designBytes []byte, resolveMedia func(int64) ([]byte, error)) ([]byte, error) {
	if len(bytes.TrimSpace(designBytes)) == 0 || bytes.Equal(bytes.TrimSpace(designBytes), []byte("{}")) {
		return nil, errEmptyDesign
	}
	var d DesignReq
	if err := json.Unmarshal(designBytes, &d); err != nil {
		return nil, fmt.Errorf("tasarım JSON hatalı: %w", err)
	}
	w, h := d.Canvas.W, d.Canvas.H
	if w <= 0 {
		w = 1200
	}
	if h <= 0 {
		h = 675
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	fill := parseColor(d.Background.Color, color.RGBA{0x0b, 0x0e, 0x14, 255})
	stddraw.Draw(img, img.Bounds(), &image.Uniform{C: fill}, image.Point{}, stddraw.Src)

	if d.Background.MediaID != nil && resolveMedia != nil {
		if raw, err := resolveMedia(*d.Background.MediaID); err == nil {
			if src, derr := DecodeImage(raw); derr == nil {
				drawCover(img, src)
			}
		}
	}

	if d.Background.Overlay.Active {
		applyOverlay(img, d.Background.Overlay.Mode, d.Background.Overlay.Direction,
			parseColor(d.Background.Overlay.Color, color.RGBA{0, 0, 0, 255}),
			d.Background.Overlay.Spread, d.Background.Overlay.Opacity)
	}

	if d.Badge.Active && d.Badge.Text != "" {
		bc := parseColor(d.Badge.Color, color.RGBA{0xe1, 0x1d, 0x48, 255})
		tc := parseColor(d.Badge.TextColor, color.RGBA{255, 255, 255, 255})
		bs := d.Badge.FontSize
		if bs <= 0 {
			bs = int(float64(w) * 0.016)
		}
		txt := d.Badge.Text
		if d.Badge.Uppercase {
			txt = strings.ToUpper(txt)
		}
		renderBadge(img, txt, bc, tc, bs, d.Badge.X, d.Badge.Y, d.Badge.Radius, d.Badge.Padding)
	}

	for i := len(d.Layers) - 1; i >= 0; i-- {
		l := d.Layers[i]
		if !l.Visible || l.Text == "" {
			continue
		}
		fs := l.FontSize
		if fs <= 0 {
			fs = int(float64(w) * 0.03)
		}
		tc := parseColor(l.Color, color.RGBA{255, 255, 255, 255})
		sc := parseColor(l.ShadowColor, color.RGBA{0, 0, 0, 255})
		renderTextLayer(img, l, w, h, fs, tc, sc)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ContentDesignInput blok başına kapak üretimi için içerik değerleridir (Faz 3).
type ContentDesignInput struct {
	Title      string // ana_baslik katmanına yazılır
	Spot       string // alt_baslik katmanına yazılır (varsa)
	Badge      string // rozet yazısı (kategori) — varsa
	Category   string // ust_baslik katmanına yazılır (varsa)
	CoverMedia *int64 // arka plan medyası (içerik kapağı) — nil ise tasarımın kendi medyası kalır
}

func ApplyContentDesign(designBytes []byte, in ContentDesignInput) ([]byte, error) {
	base := NormalizeDesign(designBytes)
	var m map[string]any
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, fmt.Errorf("tasarım JSON hatalı: %w", err)
	}
	if layers, ok := m["layers"].([]any); ok {
		for _, li := range layers {
			lm, ok := li.(map[string]any)
			if !ok {
				continue
			}
			role, _ := lm["role"].(string)
			switch role {
			case "ana_baslik":
				if in.Title != "" {
					lm["text"] = in.Title
				}
			case "alt_baslik":
				if in.Spot != "" {
					lm["text"] = in.Spot
				}
			case "ust_baslik":
				if in.Category != "" {
					lm["text"] = in.Category
				}
			}
		}
	}
	if in.Badge != "" {
		if bg, ok := m["badge"].(map[string]any); ok {
			bg["text"] = in.Badge
			if act, ok := bg["active"].(bool); ok {
				_ = act
			} else {
				bg["active"] = true
			}
		}
	}
	if in.CoverMedia != nil {
		if bm, ok := m["background"].(map[string]any); ok {
			bm["media_id"] = *in.CoverMedia
		} else {
			m["background"] = map[string]any{"media_id": *in.CoverMedia}
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// applyOverlay karartma modlarını uygular: gradient | solid | vignette.
func applyOverlay(img *image.RGBA, mode, dir string, c color.RGBA, spread, opacity int) {
	if mode == "" {
		mode = "gradient"
	}
	if opacity <= 0 {
		opacity = 100
	}
	maxA := clampF(float64(opacity)/100.0, 0, 1)
	if spread <= 0 {
		spread = 70
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var a float64
			switch mode {
			case "solid":
				a = maxA
			case "vignette":
				dx := (float64(x) - float64(w)/2) / (float64(w) / 2)
				dy := (float64(y) - float64(h)/2) / (float64(h) / 2)
				dist := clampF(math.Sqrt(dx*dx+dy*dy)/1.414, 0, 1)
				inner := 1 - float64(spread)/100.0
				a = maxA * clampF((dist-inner)/math.Max(0.001, 1-inner), 0, 1)
			default: // gradient
				t := gradientT(x, y, w, h, dir) // 0 (koyu) → 1 (açık)
				a = maxA * clampF(1-t*100.0/float64(spread), 0, 1)
			}
			if a <= 0.002 {
				continue
			}
			img.Set(x, y, lerp(img.At(x, y), c, a))
		}
	}
}

// gradientT verilen yön boyunca 0..1 oransal konum (koyu uç = 0).
func gradientT(x, y, w, h int, dir string) float64 {
	fw, fh := float64(w), float64(h)
	fx, fy := float64(x), float64(y)
	switch dir {
	case "right":
		return 1 - fx/fw
	case "top":
		return fy / fh
	case "bottom":
		return 1 - fy/fh
	case "tl":
		return (fx + fy) / (fw + fh)
	case "tr":
		return ((fw - fx) + fy) / (fw + fh)
	case "bl":
		return (fx + (fh - fy)) / (fw + fh)
	case "br":
		return ((fw - fx) + (fh - fy)) / (fw + fh)
	default: // left
		return fx / fw
	}
}

func lerp(p color.Color, c color.Color, a float64) color.Color {
	pr, pg, pb, _ := p.RGBA()
	cr, cg, cb, _ := c.RGBA()
	f := func(v1, v2 uint32) uint8 {
		v := float64(v1>>8)*(1-a) + float64(v2>>8)*a
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v)
	}
	return color.RGBA{f(pr, cr), f(pg, cg), f(pb, cb), 255}
}

// drawCover görseli cover (kırpma+ölçekleme) olarak doldurur.
func drawCover(dst *image.RGBA, src image.Image) {
	db := dst.Bounds()
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	scale := float64(db.Dx()) / float64(sw)
	if float64(db.Dy())/float64(sh) > scale {
		scale = float64(db.Dy()) / float64(sh)
	}
	nw, nh := int(float64(sw)*scale+0.5), int(float64(sh)*scale+0.5)
	tmp := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(tmp, tmp.Bounds(), src, sb, xdraw.Src, nil)
	sx := (nw - db.Dx()) / 2
	sy := (nh - db.Dy()) / 2
	stddraw.Draw(dst, db, tmp, image.Pt(sx, sy), stddraw.Src)
}

func measureW(face font.Face, s string) float64 {
	return float64(font.MeasureString(face, s)) / 64.0
}

// wrapDesignText açık satırları maxW genişliğine kelime kaydırmayla böler.
func wrapDesignText(face font.Face, text string, maxW float64) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, wd := range words[1:] {
			if measureW(face, line+" "+wd) <= maxW || line == "" {
				line = strings.TrimSpace(line + " " + wd)
			} else {
				out = append(out, line)
				line = wd
			}
		}
		out = append(out, line)
	}
	return out
}

// renderTextLayer katmanı (kaydırma + gölge + hizalama) çizer.
func renderTextLayer(img *image.RGBA, l LayerReq, w, h, size int, c, shadow color.RGBA) {
	face := faceAt(size, l.Weight)
	lh := l.LineHeight
	if lh <= 0 {
		lh = 1.15
	}
	maxW := float64(l.MaxWidth)
	if maxW <= 0 {
		maxW = float64(w - l.X - 40)
	}
	if maxW < 40 {
		maxW = 40
	}
	text := l.Text
	if l.Uppercase {
		text = strings.ToUpper(text)
	}
	lines := wrapDesignText(face, text, maxW)
	lineH := float64(size) * lh
	if l.Shadow {
		drawLines(img, face, lines, l.X+2, l.Y+2, l.Align, lineH, shadow)
	}
	drawLines(img, face, lines, l.X, l.Y, l.Align, lineH, c)
}

func drawLines(img *image.RGBA, face font.Face, lines []string, x, y int, align string, lineH float64, c color.RGBA) {
	d := &font.Drawer{Dst: img, Src: &image.Uniform{C: c}, Face: face}
	for i, line := range lines {
		wpx := measureW(face, line)
		startX := float64(x)
		switch align {
		case "center":
			startX = float64(x) - wpx/2
		case "right":
			startX = float64(x) - wpx
		}
		d.Dot = fixedPoint(startX, float64(y)+float64(i)*lineH)
		d.DrawString(line)
	}
}

func fixedPoint(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
}

// renderBadge gelişmiş rozet: yuvarlatılmış kutu + renkli metin.
func renderBadge(img *image.RGBA, text string, c, tc color.RGBA, size, x, y, radius, padding int) {
	face := faceAt(size, 700)
	wpx := measureW(face, text)
	if padding <= 0 {
		padding = size / 2
	}
	if radius <= 0 {
		radius = size / 3
	}
	bw := int(wpx) + padding*2
	bh := size + padding
	if x <= 0 {
		x = size / 2
	}
	if y <= 0 {
		y = size / 2
	}
	fillRoundRect(img, x, y, x+bw, y+bh, radius, c)
	d := &font.Drawer{Dst: img, Src: &image.Uniform{C: tc}, Face: face}
	d.Dot = fixedPoint(float64(x+padding), float64(y+bh/2+size*2/5))
	d.DrawString(text)
}

func fillRoundRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	if r < 0 {
		r = 0
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if r > 0 {
				// köşe yuvarlatma: köşe merkezine uzaklık
				cx, cy := x, y
				if x < x0+r {
					cx = x0 + r
				} else if x >= x1-r {
					cx = x1 - r - 1
				}
				if y < y0+r {
					cy = y0 + r
				} else if y >= y1-r {
					cy = y1 - r - 1
				}
				if (x < x0+r || x >= x1-r) && (y < y0+r || y >= y1-r) {
					dx, dy := float64(x-cx), float64(y-cy)
					if dx*dx+dy*dy > float64(r*r) {
						continue
					}
				}
			}
			img.Set(x, y, c)
		}
	}
}

// jpeg/log bağımlılık onayları
var _ = image.Point{}

// ---- Otomatik kapak (autocover) için temel çizim yardımcıları ----

// drawTextBlock çok satırlı metni hizalayarak çizer (otomatik kapak).
func drawTextBlock(img image.Image, text string, x, y int, align string, size, weight int, lineHeight float64, c color.RGBA) {
	lines := strings.Split(text, "\n")
	face := faceAt(size, weight)
	if lineHeight <= 0 {
		lineHeight = 1.15
	}
	lh := int(float64(size) * lineHeight)
	dst, ok := img.(*image.RGBA)
	if !ok {
		return
	}
	d := &font.Drawer{Dst: dst, Src: &image.Uniform{C: c}}
	for i, line := range lines {
		m := font.MeasureString(face, line)
		wpx := float64(m) / 64.0
		startX := float64(x)
		switch align {
		case "center":
			startX = float64(x) - wpx/2
		case "right":
			startX = float64(x) - wpx
		}
		d.Face = face
		d.Dot = fixedPoint(startX, float64(y)+float64(i)*float64(lh))
		d.DrawString(line)
	}
}

// drawBadge rozet rozeti: dolu yuvarlatılmış kutu içinde beyaz kalın metin (otomatik kapak).
func drawBadge(img *image.RGBA, text string, c color.RGBA, size, x, y int) {
	face := faceAt(size, 700)
	wpx := font.MeasureString(face, text)
	pad := size / 2
	bw := wpx.Round() + pad*2
	bh := size + pad
	if x <= 0 {
		x = size / 2
	}
	if y <= 0 {
		y = size / 2
	}
	fillRoundRect(img, x, y, x+bw, y+bh, size/3, c)
	fc := color.RGBA{255, 255, 255, 255}
	d := &font.Drawer{Dst: img, Src: &image.Uniform{C: fc}, Face: face}
	d.Dot = fixedPoint(float64(x+pad), float64(y+bh/2+size*2/5))
	d.DrawString(text)
}
