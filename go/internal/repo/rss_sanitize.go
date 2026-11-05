package repo

import (
	"html"
	"strings"
)

// rssXMLItem RSS 2.0 ve RDF (RSS 1.0) öğelerinin ortak alanları.
//
// `xml:"encoded"` eşleşmesi ad alanını yok sayar, bu yüzden content:encoded
// alanını da yakalar; `xml:"date"` RDF'teki dc:date alanını yakalar.
type rssXMLItem struct {
	Title      string   `xml:"title"`
	Link       string   `xml:"link"`
	Desc       string   `xml:"description"`
	Content    string   `xml:"encoded"`
	PubDate    string   `xml:"pubDate"`
	DCDate     string   `xml:"date"`
	Categories []string `xml:"category"`
}

// sanitizeXML gövdeyi encoding/xml'in katlanabildiği hale getirir.
//
// Neden: Türkçe haber feed'leri sık sık XML'de tanımsız HTML varlıkları
// kullanıyor ("&nbsp;", "&l…" gibi noktalı virgülsüz yazımlar) ve çıplak "&"
// karakteri bırakıyor. XML yalnızca beş varlığı tanır (amp, lt, gt, quot,
// apos); diğerleri ayrıştırmayı tamamen durduruyordu — ölçümde 815 hata.
//
// Kurallar:
//   - CDATA bölümlerine dokunulmaz (orada "&" zaten serbesttir).
//   - Tanınan HTML varlıkları gerçek karaktere çevrilir; çevrilen karakter
//     XML'de özel anlam taşıyorsa (& <) yeniden kaçırılır.
//   - Tanınmayan veya noktalı virgülsüz "&" → "&amp;".
func sanitizeXML(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/64)
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "<![CDATA[") {
			end := strings.Index(s[i:], "]]>")
			if end < 0 {
				b.WriteString(s[i:])
				break
			}
			b.WriteString(s[i : i+end+3])
			i += end + 3
			continue
		}
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i + 1
		for j < len(s) && (isEntityByte(s[j])) {
			j++
		}
		if j == i+1 {
			// Çıplak "&"
			b.WriteString("&amp;")
			i++
			continue
		}
		name := s[i+1 : j]
		if j >= len(s) || s[j] != ';' {
			// "&nbsp" gibi noktalı virgülsüz yazım: "&" kaçırılır, ad metin kalır.
			b.WriteString("&amp;")
			i++
			continue
		}
		switch name {
		case "amp", "lt", "gt", "quot", "apos":
			b.WriteString(s[i : j+1])
			i = j + 1
			continue
		}
		if dec := html.UnescapeString(s[i : j+1]); dec != s[i:j+1] {
			dec = strings.ReplaceAll(dec, "&", "&amp;")
			dec = strings.ReplaceAll(dec, "<", "&lt;")
			b.WriteString(dec)
			i = j + 1
			continue
		}
		b.WriteString("&amp;")
		i++
	}
	return b.String()
}

func isEntityByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '#'
}
