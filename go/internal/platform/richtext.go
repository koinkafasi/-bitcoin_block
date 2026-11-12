package platform

import (
	"strings"
	"unicode"
)

// richAllowTags izin verilen gömülü metin etiketleri (kendi kendini kapatmaz).
var richAllowTags = map[string]bool{
	"p": true, "br": true, "strong": true, "b": true, "em": true, "i": true,
	"u": true, "s": true, "a": true, "ul": true, "ol": true, "li": true,
	"blockquote": true, "h2": true, "h3": true, "h4": true, "h5": true,
	"code": true, "pre": true, "table": true, "thead": true, "tbody": true,
	"tr": true, "td": true, "th": true, "span": true, "sub": true, "sup": true,
}

// richHrefOnly yalnızca güvenli URL şemalarına izin verir.
func safeHref(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "#" {
		return v
	}
	low := strings.ToLower(v)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") ||
		strings.HasPrefix(low, "mailto:") || strings.HasPrefix(v, "/") ||
		strings.HasPrefix(low, "tel:") {
		return v
	}
	return ""
}

// SanitizeRichHTML zengin metni güvenli HTML alt kümesine indirger.
func SanitizeRichHTML(in string) string {
	in = strings.ReplaceAll(in, "\x00", "")
	var out strings.Builder
	var stack []string
	i := 0
	for i < len(in) {
		if in[i] == '<' {
			// etiket başlangıcı
			j := strings.IndexAny(in[i:], ">")
			if j < 0 {
				break
			}
			raw := in[i : i+j+1]
			i += j + 1
			close := len(raw) > 1 && raw[1] == '/'
			var name string
			if close {
				name = extractTagName(raw[2:])
			} else {
				name = extractTagName(raw[1:])
			}
			if !richAllowTags[name] {
				// tehlikeli/etkisiz etiket: script & style gibi etiketlerin
				// GÖVDESİ de çıkarılır (JS/Style kalıntısı render'a düşmesin).
				if name == "script" || name == "style" || name == "iframe" {
					end := strings.ToLower(in[i:])
					if c := strings.Index(end, "</"+name+">"); c >= 0 {
						i += c + len("</"+name+">")
					}
				}
				continue
			}
			if close {
				// eşleşen açılış varsa kapat
				for k := len(stack) - 1; k >= 0; k-- {
					if stack[k] == name {
						out.WriteString("</" + name + ">")
						stack = append(stack[:k], stack[k+1:]...)
						break
					}
				}
				continue
			}
			if name == "br" {
				out.WriteString("<br>")
				continue
			}
			// açılış: öznitelik süzgeci
			out.WriteString("<" + name)
			attrs := parseAttrs(raw[1 : len(raw)-1])
			for a, v := range attrs {
				if name == "a" && a == "href" {
					if u := safeHref(v); u != "" {
						out.WriteString(` href="` + htmlEscapeAttr(u) + `"`)
					}
					continue
				}
				if (a == "class" && (name == "pre" || name == "code")) || a == "dir" || a == "align" {
					out.WriteString(` ` + a + `="` + htmlEscapeAttr(v) + `"`)
					continue
				}
				// diğer öznitelikler (onclick, style vb.) reddedilir
			}
			out.WriteString(">")
			stack = append(stack, name)
			continue
		}
		// düz metin
		out.WriteByte(in[i])
		i++
	}
	// kapanmamış etiketleri kapat
	for k := len(stack) - 1; k >= 0; k-- {
		out.WriteString("</" + stack[k] + ">")
	}
	return strings.TrimSpace(out.String())
}

// extractTagName etiket gövdesinden adı çıkarır.
func extractTagName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || (r >= '0' && r <= '9') || r == '-' || r == ':' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return strings.ToLower(b.String())
}

// parseAttrs "name=val" / bare nitelikleri ayrıştırır.
func parseAttrs(s string) map[string]string {
	out := map[string]string{}
	for {
		s = strings.TrimLeftFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
		if s == "" {
			break
		}
		var name strings.Builder
		for s != "" && s[0] != '=' && s[0] != ' ' {
			name.WriteByte(s[0])
			s = s[1:]
		}
		n := strings.ToLower(name.String())
		s = strings.TrimLeftFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
		if s == "" {
			if n != "" {
				out[n] = ""
			}
			break
		}
		if s[0] == '=' {
			s = s[1:]
			s = strings.TrimLeftFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
			quote := byte(0)
			if s != "" && (s[0] == '"' || s[0] == '\'') {
				quote = s[0]
				s = s[1:]
			}
			var val strings.Builder
			for s != "" {
				if quote != 0 {
					if s[0] == quote {
						s = s[1:]
						break
					}
				} else if s[0] == ' ' || s[0] == '\t' || s[0] == '\n' {
					break
				}
				val.WriteByte(s[0])
				s = s[1:]
			}
			if n != "" {
				out[n] = val.String()
			}
			continue
		}
		if n != "" {
			out[n] = ""
		}
	}
	return out
}

// htmlEscapeAttr öznitelik değerini güvenli hale getirir.
func htmlEscapeAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;", "'", "&#39;")
	return r.Replace(s)
}
