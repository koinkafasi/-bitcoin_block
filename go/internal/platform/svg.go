package platform

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxSVGBytes = 2 << 20

var izinliSVGOgeleri = map[string]bool{
	"svg": true, "g": true, "defs": true, "symbol": true,
	"path": true, "rect": true, "circle": true, "ellipse": true,
	"line": true, "polyline": true, "polygon": true,
	"text": true, "tspan": true, "textPath": true,
	"linearGradient": true, "radialGradient": true, "stop": true,
	"clipPath": true, "mask": true, "pattern": true,
	"filter": true, "feGaussianBlur": true, "feOffset": true,
	"feBlend": true, "feColorMatrix": true, "feFlood": true,
	"feComposite": true, "feMerge": true, "feMergeNode": true,
	"feDropShadow": true, "feMorphology": true,
	"title": true, "desc": true, "style": true, "metadata": true,
}

// izinliSVGOnekleri kabul edilen ad alanı önekleri.
var izinliSVGOnekleri = map[string]bool{"xlink": true, "xml": true, "svg": true}

func SanitizeSVG(data []byte) ([]byte, error) {
	temiz, _, _, err := SanitizeSVGBoyut(data)
	return temiz, err
}

func SanitizeSVGBoyut(data []byte) (temiz []byte, genislik, yukseklik int, err error) {
	if len(data) == 0 {
		return nil, 0, 0, errors.New("svg: boş dosya")
	}
	if len(data) > maxSVGBytes {
		return nil, 0, 0, fmt.Errorf("svg: dosya çok büyük (%d bayt, en fazla %d)", len(data), maxSVGBytes)
	}
	// Kök öğe svg olmalı (HTML kamuflajı reddedilir).
	if kok := svgKokOge(data); kok != "svg" {
		if kok == "" {
			return nil, 0, 0, errors.New("svg: kök öğe okunamadı")
		}
		return nil, 0, 0, fmt.Errorf("svg: kök öğe %q; svg olmalı", kok)
	}

	coz := xml.NewDecoder(bytes.NewReader(data))
	coz.Strict = false          // HTML varlıkları (&nbsp;) logolarda yaygın
	coz.Entity = xml.HTMLEntity // &nbsp; gibi adlandırılmış varlıklar

	var out bytes.Buffer
	enc := xml.NewEncoder(&out)

	var (
		derinlik        int
		atlananDerinlik int // >0 ise bu derinlikteki her şey atılıyor
	)
	for {
		tok, e := coz.Token()
		if e != nil {
			if errors.Is(e, io.EOF) {
				break
			}
			return nil, 0, 0, fmt.Errorf("svg: XML çözümlenemedi: %w", e)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			derinlik++
			if atlananDerinlik > 0 {
				continue
			}
			ad := t.Name.Local
			if !izinliSVGOgeleri[ad] {
				// script / foreignObject / image / use …: İÇERİĞİYLE atla.
				atlananDerinlik = derinlik
				continue
			}
			if ad == "svg" && derinlik == 1 {
				genislik, yukseklik = svgOlcu(t.Attr)
			}
			// `style` ÖĞESİNİN içeriği CharData olarak gelir; aşağıda
			// denetlenir. Öznitelikler tek tek süzülür.
			saf := xml.StartElement{Name: t.Name}
			for _, a := range t.Attr {
				if svgOznitelikGecerli(a) {
					saf.Attr = append(saf.Attr, a)
				}
			}
			if e := enc.EncodeToken(saf); e != nil {
				return nil, 0, 0, fmt.Errorf("svg: yazılamadı: %w", e)
			}

		case xml.EndElement:
			if atlananDerinlik > 0 {
				if derinlik == atlananDerinlik {
					atlananDerinlik = 0
				}
				derinlik--
				continue
			}
			derinlik--
			if e := enc.EncodeToken(t); e != nil {
				return nil, 0, 0, fmt.Errorf("svg: yazılamadı: %w", e)
			}

		case xml.CharData:
			if atlananDerinlik > 0 {
				continue
			}
			// style ÖĞESİ metni de dış kaynak çekebilir (@import, url()).
			if svgCSSMetniTehlikeli(string(t)) {
				continue
			}
			if e := enc.EncodeToken(t); e != nil {
				return nil, 0, 0, fmt.Errorf("svg: yazılamadı: %w", e)
			}

		case xml.Comment, xml.Directive, xml.ProcInst:
			continue
		}
	}
	if e := enc.Flush(); e != nil {
		return nil, 0, 0, fmt.Errorf("svg: yazılamadı: %w", e)
	}
	if derinlik != 0 {
		return nil, 0, 0, errors.New("svg: dengesiz XML")
	}
	govde := out.Bytes()
	// Son savunma (savunma derinliği): temizlik sonrası hâlâ tehlikeli
	// desen var mı? Varsa dosya REDDEDİLİR.
	if svgTehlikeli(govde) {
		return nil, 0, 0, errors.New("svg: temizlemeden sonra tehlikeli içerik kaldı")
	}
	return govde, genislik, yukseklik, nil
}

// svgKokOge belgenin ilk öğe adını (ad alanı öneki olmadan) döndürür.
func svgKokOge(data []byte) string {
	coz := xml.NewDecoder(bytes.NewReader(data))
	coz.Strict = false
	coz.Entity = xml.HTMLEntity
	for {
		tok, err := coz.Token()
		if err != nil {
			return ""
		}
		if s, ok := tok.(xml.StartElement); ok {
			return s.Name.Local
		}
	}
}

// svgOznitelikGecerli özniteliğin güvenli olup olmadığını söyler.
func svgOznitelikGecerli(a xml.Attr) bool {
	alt := strings.ToLower(a.Name.Local)
	if alt == "xmlns" || strings.HasPrefix(alt, "xmlns:") {
		return false
	}
	// OLAY İŞLEYİCİLERİ: onload, onclick, onbegin… hepsi reddedilir.
	if strings.HasPrefix(alt, "on") {
		return false
	}
	// Bilinmeyen ad alanı önekleri (ör. xmlns:evil) reddedilir.
	if a.Name.Space != "" && !strings.Contains(a.Name.Space, "w3.org") &&
		!izinliSVGOnekleri[strings.ToLower(a.Name.Space)] {
		return false
	}
	deger := strings.TrimSpace(a.Value)
	dusuk := strings.ToLower(deger)
	// javascript:, vbscript:, data:text/html → kod yürütme yolları.
	for _, kotu := range []string{"javascript:", "vbscript:", "data:text/html", "data:image/svg"} {
		if strings.Contains(dusuk, kotu) {
			return false
		}
	}
	// href/xlink:href: yalnız BELGE İÇİ (#id) bağlantı kabul edilir.
	// Dış adres = IP sızıntısı + izleme pikseli.
	if alt == "href" || strings.HasSuffix(alt, ":href") {
		if deger != "" && !strings.HasPrefix(deger, "#") {
			return false
		}
	}
	// style özniteliğinde url()/@import/expression() → dış kaynak çekme.
	if alt == "style" && svgCSSMetniTehlikeli(dusuk) {
		return false
	}
	return true
}

// svgCSSMetniTehlikeli CSS metninde dış kaynak/kod yürütme deseni arar.
func svgCSSMetniTehlikeli(s string) bool {
	d := strings.ToLower(s)
	for _, desen := range []string{"url(", "@import", "expression(", "javascript:", "behavior:"} {
		if strings.Contains(d, desen) {
			return true
		}
	}
	return false
}

func svgOlcu(attrs []xml.Attr) (int, int) {
	var w, h, vbW, vbH string
	for _, a := range attrs {
		switch strings.ToLower(a.Name.Local) {
		case "width":
			w = a.Value
		case "height":
			h = a.Value
		case "viewbox":
			f := strings.Fields(a.Value)
			if len(f) == 4 {
				vbW, vbH = f[2], f[3]
			}
		}
	}
	if gw, gh := svgPx(w), svgPx(h); gw > 0 && gh > 0 {
		return gw, gh
	}
	if gw, gh := svgPx(vbW), svgPx(vbH); gw > 0 && gh > 0 {
		return gw, gh
	}
	return 0, 0
}

// svgPx "240", "240px", "180pt" gibi değerleri piksele çevirir.
// Yüzde/em gibi göreli birimler 0 döner (ölçü bilinmiyor sayılır).
func svgPx(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0
	}
	olcek := 1.0
	switch {
	case strings.HasSuffix(s, "px"):
		s = strings.TrimSuffix(s, "px")
	case strings.HasSuffix(s, "pt"):
		s = strings.TrimSuffix(s, "pt")
		olcek = 96.0 / 72.0 // 1 pt = 4/3 px
	default:
		// Birimsiz = kullanıcı birimi; SVG'de px kabul edilir.
		// %, em, rem, vw… → bilinmiyor.
		for _, r := range s {
			if (r < '0' || r > '9') && r != '.' && r != '-' && r != '+' && r != 'e' {
				return 0
			}
		}
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v <= 0 {
		return 0
	}
	return int(v*olcek + 0.5)
}

// svgTehlikeli temizleme sonrası son kontrol (savunma derinliği).
func svgTehlikeli(b []byte) bool {
	l := strings.ToLower(string(b))
	for _, desen := range []string{"<script", "javascript:", "<foreignobject", "<!entity", "<!doctype"} {
		if strings.Contains(l, desen) {
			return true
		}
	}
	return false
}
