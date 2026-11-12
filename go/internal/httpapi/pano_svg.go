package httpapi

import (
	"fmt"
	"html/template"
	"math"
	"strings"
)

// panoSeri grafik için normalleştirilmiş seri.
type panoSeri struct {
	Values []int
	Labels []string
}

// panoMax serideki en büyük değer (en az 1: sıfıra bölme olmasın).
func panoMax(vals []int) int {
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	if m == 0 {
		return 1
	}
	return m
}

// panoNokta seriyi W×H kutusuna yerleştirir (dikeyde ters: SVG y aşağı).
func panoNokta(vals []int, w, h float64, pad float64) [][2]float64 {
	n := len(vals)
	if n == 0 {
		return nil
	}
	maxv := float64(panoMax(vals))
	icW, icH := w-2*pad, h-2*pad
	pts := make([][2]float64, n)
	for i, v := range vals {
		x := pad
		if n > 1 {
			x = pad + icW*float64(i)/float64(n-1)
		}
		y := pad + icH - icH*float64(v)/maxv
		pts[i] = [2]float64{math.Round(x*10) / 10, math.Round(y*10) / 10}
	}
	return pts
}

func panoEgri(pts [][2]float64) string {
	if len(pts) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "M%.1f,%.1f", pts[0][0], pts[0][1])
	for i := 1; i < len(pts); i++ {
		p0, p1 := pts[i-1], pts[i]
		dx := (p1[0] - p0[0]) / 3
		fmt.Fprintf(&b, " C%.1f,%.1f %.1f,%.1f %.1f,%.1f",
			p0[0]+dx, p0[1], p1[0]-dx, p1[1], p1[0], p1[1])
	}
	return b.String()
}

// panoAlan çizgiyi tabana kapatır (dolgu için).
func panoAlan(pts [][2]float64, h, pad float64) string {
	if len(pts) < 2 {
		return ""
	}
	egri := panoEgri(pts)
	taban := math.Round((h-pad)*10) / 10
	return egri + fmt.Sprintf(" L%.1f,%.1f L%.1f,%.1f Z", pts[len(pts)-1][0], taban, pts[0][0], taban)
}

func panoAlanChart(s panoSeri, w, h int, kim string) template.HTML {
	if len(s.Values) < 2 {
		return template.HTML(`<div class="pn-bos">Henüz veri yok</div>`)
	}
	grad := "pnAlanGrad_" + kim
	pad := 6.0
	pts := panoNokta(s.Values, float64(w), float64(h), pad)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="pn-chart pn-chart--alan" viewBox="0 0 %d %d" preserveAspectRatio="none" role="img" aria-label="Günlük okunma grafiği">`, w, h)
	b.WriteString(`<defs><linearGradient id="` + grad + `" x1="0" y1="0" x2="0" y2="1">`)
	b.WriteString(`<stop offset="0%" stop-color="currentColor" stop-opacity=".28"/>`)
	b.WriteString(`<stop offset="100%" stop-color="currentColor" stop-opacity="0"/></linearGradient></defs>`)
	b.WriteString(`<path class="pn-alan" d="` + panoAlan(pts, float64(h), pad) + `" fill="url(#` + grad + `)"/>`)
	b.WriteString(`<path class="pn-cizgi" d="` + panoEgri(pts) + `" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>`)
	for i, p := range pts {
		lbl := ""
		if i < len(s.Labels) {
			lbl = s.Labels[i]
		}
		fmt.Fprintf(&b, `<circle class="pn-nokta" cx="%.1f" cy="%.1f" r="3.2" data-i="%d" data-v="%d" data-l="%s"/>`,
			p[0], p[1], i, s.Values[i], template.HTMLEscapeString(lbl))
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

// panoSpark KPI döşemesi için küçük alan grafiği.
func panoSpark(vals []int, w, h int) template.HTML {
	if len(vals) < 2 {
		return ""
	}
	pad := 2.0
	pts := panoNokta(vals, float64(w), float64(h), pad)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="pn-spark" viewBox="0 0 %d %d" preserveAspectRatio="none" aria-hidden="true">`, w, h)
	b.WriteString(`<path class="pn-spark__alan" d="` + panoAlan(pts, float64(h), pad) + `" fill="currentColor" opacity=".16"/>`)
	b.WriteString(`<path d="` + panoEgri(pts) + `" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>`)
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

// PanoDilim halka grafiğinin bir dilimi.
type PanoDilim struct {
	Name  string
	Count int
	Pct   int
}

// panoDilimler yüzdeleri hesaplar. Yuvarlama kalanı en büyük dilime eklenir
// ki toplam %100 görünsün (küçük yuvarlama hataları göze batıyor).
func panoDilimler(in []CatSliceRef) []PanoDilim {
	toplam := 0
	for _, s := range in {
		toplam += s.Count
	}
	if toplam == 0 {
		return nil
	}
	out := make([]PanoDilim, len(in))
	acc := 0
	enBuyuk := 0
	for i, s := range in {
		pct := int(math.Round(float64(s.Count) * 100 / float64(toplam)))
		out[i] = PanoDilim{Name: s.Name, Count: s.Count, Pct: pct}
		acc += pct
		if s.Count > in[enBuyuk].Count {
			enBuyuk = i
		}
	}
	out[enBuyuk].Pct += 100 - acc
	return out
}

// CatSliceRef repo.CatSlice'ın bağımsız kopyası: httpapi paketi grafik
// yardımcılarını repo tiplerine bağlamadan test edebilsin.
type CatSliceRef struct {
	Name  string
	Count int
}

// panoHalka halka (donut) grafiği. Dilimler stroke-dasharray ile çizilir:
// path üretmekten kısa ve tarayıcıda net.
func panoHalka(d []PanoDilim, size int) template.HTML {
	if len(d) == 0 {
		return template.HTML(`<div class="pn-bos">Veri yok</div>`)
	}
	r := float64(size)/2 - 10
	cevre := 2 * math.Pi * r
	merkez := float64(size) / 2
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="pn-chart pn-halka" viewBox="0 0 %d %d" role="img" aria-label="Kategori dağılımı">`, size, size)
	fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="var(--pn-line)" stroke-width="14"/>`, merkez, merkez, r)
	bas := 0.0
	for i, s := range d {
		uzunluk := cevre * float64(s.Pct) / 100
		fmt.Fprintf(&b, `<circle class="pn-dilim" data-i="%d" cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="var(--pn-c%d)" stroke-width="14" stroke-linecap="butt" stroke-dasharray="%.2f %.2f" stroke-dashoffset="%.2f" transform="rotate(-90 %.1f %.1f)"><title>%s: %d (%d%%)</title></circle>`,
			i+1, merkez, merkez, r, (i%5)+1, uzunluk, cevre-uzunluk, -bas, merkez, merkez,
			template.HTMLEscapeString(s.Name), s.Count, s.Pct)
		bas += uzunluk
	}
	fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" class="pn-halka__orta" text-anchor="middle" dominant-baseline="middle">%d</text>`, merkez, merkez-2, toplamDilim(d))
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

func toplamDilim(d []PanoDilim) int {
	n := 0
	for _, s := range d {
		n += s.Count
	}
	return n
}

// panoSutun dikey sütun grafiği (yayın ritmi).
func panoSutun(s panoSeri, w, h int) template.HTML {
	if len(s.Values) == 0 {
		return template.HTML(`<div class="pn-bos">Veri yok</div>`)
	}
	maxv := float64(panoMax(s.Values))
	n := len(s.Values)
	gap := 10.0
	bw := (float64(w) - gap*float64(n-1)) / float64(n)
	if bw < 2 {
		bw, gap = 2, 2
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="pn-chart pn-sutun" viewBox="0 0 %d %d" role="img" aria-label="Aylık yayın grafiği">`, w, h)
	for i, v := range s.Values {
		bh := (float64(h) - 18) * float64(v) / maxv
		if bh < 2 {
			bh = 2 // sıfır çubuk tamamen kaybolmasın: taban çizgisi görünsün
		}
		x := float64(i) * (bw + gap)
		y := float64(h-18) - bh
		lbl := ""
		if i < len(s.Labels) {
			lbl = s.Labels[i]
		}
		fmt.Fprintf(&b, `<rect class="pn-cubuk" x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" data-i="%d" data-v="%d" data-l="%s"></rect>`,
			x, y, bw, bh, i, v, template.HTMLEscapeString(lbl))
		fmt.Fprintf(&b, `<text class="pn-eksen" x="%.1f" y="%d" text-anchor="middle">%s</text>`,
			x+bw/2, h-2, template.HTMLEscapeString(lbl))
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

// PanoCubuk yatay çubuk satırı (en çok okunanlar).
type PanoCubuk struct {
	Label string
	Value int
	Pct   int
	Href  string
}

// panoYatay yatay çubuk listesi. Genişlik yüzdesi en büyük değere göre.
func panoYatay(rows []PanoCubuk) template.HTML {
	if len(rows) == 0 {
		return template.HTML(`<div class="pn-bos">Henüz okunma yok</div>`)
	}
	var b strings.Builder
	b.WriteString(`<ol class="pn-yatay">`)
	for i, r := range rows {
		b.WriteString(`<li class="pn-yatay__satir">`)
		fmt.Fprintf(&b, `<span class="pn-yatay__sira">%d</span>`, i+1)
		if r.Href != "" {
			fmt.Fprintf(&b, `<a class="pn-yatay__ad" href="%s">%s</a>`, template.HTMLEscapeString(r.Href), template.HTMLEscapeString(r.Label))
		} else {
			fmt.Fprintf(&b, `<span class="pn-yatay__ad">%s</span>`, template.HTMLEscapeString(r.Label))
		}
		fmt.Fprintf(&b, `<span class="pn-yatay__bar"><i style="width:%d%%"></i></span>`, r.Pct)
		fmt.Fprintf(&b, `<span class="pn-yatay__deger">%s</span>`, itoa(r.Value))
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ol>`)
	return template.HTML(b.String())
}
