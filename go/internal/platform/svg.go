package platform

import (
	"fmt"
	"regexp"
	"strings"
)

// SVG yüklemesinde etkin içerik temizliği için desenler. SVG, arka plan
// görseli olarak CSS url()/resim bağlamında kullanıldığı için tarayıcı
// script çalıştırmaz; bu temizlik derinlemesine savunma (defense in depth)
// amaçlıdır ve SVG'nin ayrı bir sekmede doğrudan açılması olasılığını da
// zararsızlaştırır.
var (
	svgScriptRe    = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`)
	svgScriptSelfR = regexp.MustCompile(`(?is)<script\b[^>]*?/?>`)
	svgForeignRe   = regexp.MustCompile(`(?is)<foreignObject\b[^>]*>.*?</foreignObject\s*>`)
	svgMetaRe      = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	svgEventRe     = regexp.MustCompile(`(?i)\son[a-z]{3,}\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	svgJSHrefRe    = regexp.MustCompile(`(?i)(javascript|data:text/html)[:\s]*`)
)

// maxSVGBytes yüklenebilecek en büyük SVG boyutu.
const maxSVGBytes = 2 << 20 // 2 MB

// SanitizeSVG yüklenen SVG içeriğini doğrular ve etkin içeriği temizler.
// script, foreignObject, meta, on* olay öznitelikleri ve javascript:
// URL'leri kaldırılır. Geçersiz (kökü <svg> olmayan) veya çok büyük
// içerik hata döndürür.
func SanitizeSVG(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("boş SVG")
	}
	if len(data) > maxSVGBytes {
		return nil, fmt.Errorf("SVG çok büyük (en fazla 2 MB)")
	}
	s := string(data)
	if !strings.Contains(strings.ToLower(s), "<svg") {
		return nil, fmt.Errorf("geçersiz SVG (kök <svg> bulunamadı)")
	}
	s = svgScriptRe.ReplaceAllString(s, "")
	s = svgScriptSelfR.ReplaceAllString(s, "")
	s = svgForeignRe.ReplaceAllString(s, "")
	s = svgMetaRe.ReplaceAllString(s, "")
	s = svgEventRe.ReplaceAllString(s, "")
	s = svgJSHrefRe.ReplaceAllString(s, "")
	return []byte(s), nil
}
