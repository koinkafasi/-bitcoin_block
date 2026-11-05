package platform

import (
	"net/url"
	"strings"
)

// ---- Reklam HTML'i süzgeci ----
//
// Reklam kaydındaki "özel HTML" alanı public sayfalarda ham olarak basılıyordu
// (ads_handlers.buildAdHTML → template.HTML). Panelde yetki denetimi olsa bile
// bir yazım hatası ya da ele geçirilmiş bir editör hesabı kalıcı XSS'e
// dönüşüyordu. Bu süzgeç yalnız reklam için gereken alt kümeyi korur:
// AdSense <ins> işaretleri, görsel/iframe, temel metin etiketleri ve
// data-*/aria-* öznitelikleri. <script>, <style> gövdesiyle birlikte,
// olay öznitelikleri (on*) ve javascript:/data:text gibi URL'ler çıkarılır.

// adAllowTags reklamda izin verilen etiketler.
var adAllowTags = map[string]bool{
	"ins": true, "div": true, "span": true, "a": true, "img": true, "iframe": true,
	"p": true, "br": true, "strong": true, "b": true, "em": true, "i": true,
	"u": true, "s": true, "small": true,
	"ul": true, "ol": true, "li": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "td": true, "th": true,
	"picture": true, "source": true, "video": true, "audio": true,
	"figure": true, "figcaption": true,
}

// adVoidTags kendiliğinden kapanan etiketler.
var adVoidTags = map[string]bool{"br": true, "img": true, "source": true}

// adGlobalAttrs her etikette kabul edilen öznitelikler.
var adGlobalAttrs = map[string]bool{
	"class": true, "id": true, "style": true, "title": true, "dir": true,
	"lang": true, "width": true, "height": true, "align": true, "role": true,
}

// adTagAttrs etikete özel kabul edilen öznitelikler.
var adTagAttrs = map[string]map[string]bool{
	"a":      {"href": true, "target": true, "rel": true, "hreflang": true},
	"img":    {"src": true, "srcset": true, "alt": true, "loading": true, "decoding": true, "referrerpolicy": true, "sizes": true, "crossorigin": true},
	"iframe": {"src": true, "frameborder": true, "allow": true, "allowfullscreen": true, "loading": true, "referrerpolicy": true, "sandbox": true, "scrolling": true, "name": true},
	"video":  {"src": true, "controls": true, "poster": true, "muted": true, "loop": true, "playsinline": true, "preload": true, "autoplay": true},
	"audio":  {"src": true, "controls": true, "muted": true, "loop": true, "preload": true},
	"source": {"src": true, "srcset": true, "type": true, "media": true, "sizes": true},
	"td":     {"colspan": true, "rowspan": true},
	"th":     {"colspan": true, "rowspan": true, "scope": true},
}

// adURLAttrs değeri URL olan öznitelikler.
var adURLAttrs = map[string]bool{"href": true, "src": true, "poster": true}

// SanitizeAdHTML reklam HTML'ini güvenli alt kümeye indirger.
func SanitizeAdHTML(in string) string {
	in = strings.ReplaceAll(in, "\x00", "")
	var out strings.Builder
	var stack []string
	i := 0
	for i < len(in) {
		if in[i] != '<' {
			out.WriteByte(in[i])
			i++
			continue
		}
		j := strings.IndexByte(in[i:], '>')
		if j < 0 {
			break
		}
		raw := in[i : i+j+1]
		i += j + 1
		closeTag := len(raw) > 1 && raw[1] == '/'
		var name string
		if closeTag {
			name = extractTagName(raw[2:])
		} else {
			name = extractTagName(raw[1:])
		}
		if !adAllowTags[name] {
			// Tehlikeli/izin dışı etiket: script ve style gibi blokların
			// gövdesi de atılmalı, aksi hâlde içerik sayfaya düz metin
			// olarak sızıp yorum/JS kalıntısı bırakır.
			if name == "script" || name == "style" || name == "title" || name == "textarea" {
				end := strings.ToLower(in[i:])
				if c := strings.Index(end, "</"+name+">"); c >= 0 {
					i += c + len("</"+name+">")
				}
			}
			continue
		}
		if closeTag {
			for k := len(stack) - 1; k >= 0; k-- {
				if stack[k] == name {
					out.WriteString("</" + name + ">")
					stack = append(stack[:k], stack[k+1:]...)
					break
				}
			}
			continue
		}
		// Kendiliğinden kapanan etiketler
		if name == "br" {
			out.WriteString("<br>")
			continue
		}
		if adVoidTags[name] {
			out.WriteString(buildAdTag(name, raw[1:len(raw)-1]))
			continue
		}
		out.WriteString(buildAdTag(name, raw[1:len(raw)-1]))
		stack = append(stack, name)
	}
	for k := len(stack) - 1; k >= 0; k-- {
		out.WriteString("</" + stack[k] + ">")
	}
	return strings.TrimSpace(out.String())
}

// buildAdTag süzülmüş açılış etiketini üretir.
func buildAdTag(name, rawAttrs string) string {
	// Kendiliğinden kapanan "/>" kalıntısını at.
	rawAttrs = strings.TrimRight(strings.TrimSpace(rawAttrs), "/")
	var b strings.Builder
	b.WriteString("<" + name)
	for a, v := range parseAttrs(rawAttrs) {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" || !adAttrAllowed(name, a) {
			continue
		}
		v = strings.TrimSpace(v)
		if adURLAttrs[a] {
			u, ok := adSafeURL(v, a == "src" && name == "img")
			if !ok {
				continue
			}
			v = u
		} else if a == "srcset" {
			if !adSafeSrcset(v) {
				continue
			}
		} else if a == "style" {
			// Eski IE'nin expression() kaçışı ve şema gizleme denemeleri.
			if adUnsafeStyle(v) {
				continue
			}
		}
		if v == "" {
			// boolean öznitelik (controls, allowfullscreen, muted…)
			b.WriteString(" " + a)
			continue
		}
		b.WriteString(" " + a + `="` + htmlEscapeAttr(v) + `"`)
	}
	b.WriteString(">")
	return b.String()
}

// adUnsafeStyle style değerinde kod çalıştırma denemesi var mı?
func adUnsafeStyle(v string) bool {
	low := strings.ToLower(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, v))
	return strings.Contains(low, "expression(") || strings.Contains(low, "javascript:") ||
		strings.Contains(low, "vbscript:") || strings.Contains(low, "behavior:")
}

// adAttrAllowed özniteliğin bu etikette kullanılmasına izin var mı?
func adAttrAllowed(tag, attr string) bool {
	// Olay öznitelikleri (onclick, onerror…) hiçbir zaman.
	if strings.HasPrefix(attr, "on") {
		return false
	}
	// AdSense ve erişilebilirlik verileri serbest.
	if strings.HasPrefix(attr, "data-") || strings.HasPrefix(attr, "aria-") {
		return true
	}
	if adGlobalAttrs[attr] {
		return true
	}
	return adTagAttrs[tag][attr]
}

// adSafeURL yalnız güvenli şemaları kabul eder.
func adSafeURL(v string, allowDataImage bool) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	// Kontrol karakterleriyle şema gizleme denemesi (java\tscript:).
	v = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, v)
	low := strings.ToLower(v)
	if strings.HasPrefix(low, "data:") {
		if allowDataImage && strings.HasPrefix(low, "data:image/") {
			return v, true
		}
		return "", false
	}
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "#") {
		return v, true
	}
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto", "tel":
		return v, true
	}
	return "", false
}

// adSafeSrcset virgülle ayrılmış aday listesini doğrular.
func adSafeSrcset(v string) bool {
	low := strings.ToLower(v)
	if strings.Contains(low, "javascript:") || strings.Contains(low, "vbscript:") ||
		strings.Contains(low, "data:text") {
		return false
	}
	for _, part := range strings.Split(v, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		if _, ok := adSafeURL(fields[0], true); !ok {
			return false
		}
	}
	return true
}
