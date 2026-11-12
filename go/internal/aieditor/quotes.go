package aieditor

import (
	"fmt"
	"strings"

	"theblock/site/internal/quality"
)

func quoteWarnings(r *Result, source string) []string {
	if len(r.Quotes) == 0 {
		return nil
	}
	rep := quality.VerifyQuotes(r.Quotes, source)
	if rep.OK() {
		return nil
	}
	bad := make([]string, 0, len(rep.Unverified))
	for _, u := range rep.Unverified {
		bad = append(bad, u.Quote)
	}
	r.UnverifiedQuotes = bad
	return []string{fmt.Sprintf(
		"%d alıntı kaynak transkriptte doğrulanamadı (%d/%d doğrulandı). "+
			"Tırnak içindeki cümleleri transkriptten birebir alın veya alıntıyı kaldırın.",
		len(rep.Unverified), rep.Verified, rep.Checked)}
}

// factWarnings iki olgu kapısını birlikte çalıştırır: alıntı doğrulama ve
// sayı doğrulama. Sayılar taslağın başlık + spot + gövdesinden çıkarılır.
func factWarnings(r *Result, source string) []string {
	warns := quoteWarnings(r, source)

	text := draftNumbersText(r)
	rep := quality.VerifyNumbers(text, source)
	if !rep.OK() {
		vals := make([]string, 0, len(rep.Unverified))
		for _, u := range rep.Unverified {
			vals = append(vals, u.Value)
		}
		r.UnverifiedNumbers = vals
		warns = append(warns, fmt.Sprintf(
			"%d sayı kaynak transkriptte bulunamadı (%d/%d doğrulandı): %s. "+
				"Rakamları transkriptten doğrulayın; uydurma sayı yayına gitmemeli.",
			len(rep.Unverified), rep.Verified, rep.Checked, strings.Join(vals, ", ")))
	}
	return warns
}

// draftNumbersText taslağın sayı denetimine girecek metnini kurar.
func draftNumbersText(r *Result) string {
	var b strings.Builder
	if t := strings.TrimSpace(r.Fields["title"]); t != "" {
		b.WriteString(t)
		b.WriteString("\n")
	}
	if s := strings.TrimSpace(r.Fields["spot"]); s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}
	if body := strings.TrimSpace(r.Fields["body"]); body != "" {
		b.WriteString(body)
	}
	return b.String()
}
