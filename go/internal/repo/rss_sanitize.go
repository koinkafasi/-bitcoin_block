package repo

import (
	"html"
	"strings"
)

type rssXMLItem struct {
	Title      string         `xml:"title"`
	Link       string         `xml:"link"`
	Desc       string         `xml:"description"`
	Content    string         `xml:"encoded"`
	PubDate    string         `xml:"pubDate"`
	DCDate     string         `xml:"date"`
	Categories []string       `xml:"category"`
	Medias     []rssMedia     `xml:"content"`
	Thumbnails []rssMedia     `xml:"thumbnail"`
	Enclosures []rssEnclosure `xml:"enclosure"`
}

// rssMedia Media RSS görsel öğesi.
type rssMedia struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Medium string `xml:"medium,attr"`
}

// rssEnclosure RSS 2.0 ek dosyası (çoğunlukla görsel).
type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

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
