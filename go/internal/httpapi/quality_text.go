package httpapi

import (
	"html"
	"regexp"
	"strings"

	"theblock/site/internal/repo"
)

// htmlTagRe kalite ölçümünden önce sökülen etiketler.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func cleanForQuality(s string) string {
	if s == "" {
		return ""
	}
	// İki kez çöz: kaynakta "&amp;nbsp;" biçiminde çift kaçış olabiliyor.
	s = html.UnescapeString(s)
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = htmlTagRe.ReplaceAllString(s, "")

	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == '\t'
		}), " ")
	}
	out := strings.Join(lines, "\n")
	// Üç ve daha fazla boş satırı ikiye indir (paragraf sınırı korunur).
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(out)
}

// qualityText ölçülecek düz metni kurar. Gövde blokları düz metne çevrilir,
// HTML varlıklarından arındırılır; başlık ve spot bağlam için eklenir.
func qualityText(c repo.Content) string {
	body := cleanForQuality(blocksToText(c.Detail))
	var b strings.Builder
	if t := cleanForQuality(c.Title); t != "" {
		b.WriteString(t)
		b.WriteString("\n\n")
	}
	if sp := cleanForQuality(c.Spot); sp != "" {
		b.WriteString(sp)
		b.WriteString("\n\n")
	}
	b.WriteString(body)
	return b.String()
}
