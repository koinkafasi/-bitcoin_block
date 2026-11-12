package transcribe

import (
	"fmt"
	"strings"
)

const (
	subMaxLineChars = 42
	subMaxLines     = 2
	subMaxChars     = subMaxLineChars * subMaxLines
	subMaxMS        = 6000
	subMinMS        = 800
	// subGapMS çakışan kareleri ayırmak için bırakılan minik boşluk.
	subGapMS = 40
	// subMSPerChar bitiş zamanı eksikse karakter başına varsayılan süre.
	subMSPerChar = 60
)

// Cue tek altyazı karesidir.
type Cue struct {
	StartMS int64
	EndMS   int64
	Speaker string
	Text    string
}

// BuildCues segmentleri okunabilir altyazı karelerine böler. Sonuç zamana
// göre sıralı, çakışmasız ve sıfırdan uzun sürelidir.
func BuildCues(segs []Segment) []Cue {
	var out []Cue
	for _, seg := range segs {
		text := normalizeSpace(seg.Text)
		if text == "" {
			continue
		}
		start, end := seg.StartMS, seg.EndMS
		if start < 0 {
			start = 0
		}
		if end <= start {
			end = start + estimateMS(text)
		}
		if runeLen(text) <= subMaxChars && end-start <= subMaxMS {
			out = append(out, Cue{StartMS: start, EndMS: end, Speaker: seg.Speaker, Text: text})
			continue
		}
		if len(seg.Words) > 0 {
			out = append(out, splitByWords(seg, start, end)...)
		} else {
			out = append(out, splitByChars(seg, text, start, end)...)
		}
	}
	return normalizeCues(out)
}

// splitByWords kareleri kelime zamanlarına göre böler: kare sınırı hep bir
// kelimenin gerçek başlangıcı/bitişidir, zaman tahmin edilmez.
func splitByWords(seg Segment, start, end int64) []Cue {
	var out []Cue
	var cur []Word
	chars := 0

	flush := func() {
		if len(cur) == 0 {
			return
		}
		parts := make([]string, 0, len(cur))
		for _, w := range cur {
			parts = append(parts, w.Text)
		}
		text := normalizeSpace(strings.Join(parts, " "))
		s := cur[0].StartMS
		e := cur[len(cur)-1].EndMS
		if s < start {
			s = start
		}
		if e <= s {
			e = s + estimateMS(text)
		}
		if text != "" {
			out = append(out, Cue{StartMS: s, EndMS: e, Speaker: seg.Speaker, Text: text})
		}
		cur = nil
		chars = 0
	}

	for _, w := range seg.Words {
		wt := normalizeSpace(w.Text)
		if wt == "" {
			continue
		}
		wl := runeLen(wt)
		if len(cur) > 0 {
			tooLong := chars+1+wl > subMaxChars
			tooSlow := w.EndMS-cur[0].StartMS > subMaxMS
			if tooLong || tooSlow {
				flush()
			}
		}
		cur = append(cur, Word{StartMS: w.StartMS, EndMS: w.EndMS, Text: wt})
		if chars > 0 {
			chars++
		}
		chars += wl
	}
	flush()

	if len(out) == 0 {
		// Kelimelerin hepsi boş çıktı: segmentin kendisine düş.
		text := normalizeSpace(seg.Text)
		if text == "" {
			return nil
		}
		return []Cue{{StartMS: start, EndMS: end, Speaker: seg.Speaker, Text: text}}
	}
	return out
}

func splitByChars(seg Segment, text string, start, end int64) []Cue {
	words := strings.Fields(text)
	total := 0
	for _, w := range words {
		total += runeLen(w)
	}
	if total == 0 {
		return nil
	}
	span := end - start
	var out []Cue
	var cur []string
	curChars, done := 0, 0

	flush := func() {
		if len(cur) == 0 {
			return
		}
		n := 0
		for _, w := range cur {
			n += runeLen(w)
		}
		s := start + span*int64(done)/int64(total)
		e := start + span*int64(done+n)/int64(total)
		if e <= s {
			e = s + subMinMS
		}
		out = append(out, Cue{
			StartMS: s, EndMS: e, Speaker: seg.Speaker,
			Text: strings.Join(cur, " "),
		})
		done += n
		cur = nil
		curChars = 0
	}

	for _, w := range words {
		wl := runeLen(w)
		if len(cur) > 0 && curChars+1+wl > subMaxChars {
			flush()
		}
		cur = append(cur, w)
		if curChars > 0 {
			curChars++
		}
		curChars += wl
	}
	flush()
	return out
}

func normalizeCues(cues []Cue) []Cue {
	for i := range cues {
		if cues[i].EndMS <= cues[i].StartMS {
			cues[i].EndMS = cues[i].StartMS + subMinMS
		}
		if i == 0 {
			continue
		}
		if cues[i].StartMS < cues[i-1].EndMS {
			if cues[i].StartMS-subGapMS > cues[i-1].StartMS {
				cues[i-1].EndMS = cues[i].StartMS - subGapMS
			} else {
				cues[i].StartMS = cues[i-1].EndMS + subGapMS
			}
		}
		if cues[i].EndMS <= cues[i].StartMS {
			cues[i].EndMS = cues[i].StartMS + subMinMS
		}
	}
	return cues
}

func SRT(cues []Cue) string {
	var b strings.Builder
	n := 0
	for _, c := range cues {
		text := normalizeSpace(c.Text)
		if text == "" {
			continue
		}
		if sp := normalizeSpace(c.Speaker); sp != "" {
			text = sp + ": " + text
		}
		n++
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n",
			n, srtTime(c.StartMS), srtTime(c.EndMS), wrapLines(text))
	}
	return b.String()
}

// WebVTT kareleri WebVTT biçiminde yazar. Konuşmacı, oynatıcıların
// tanıdığı <v Ad> ses etiketiyle verilir.
func WebVTT(cues []Cue) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, c := range cues {
		text := normalizeSpace(c.Text)
		if text == "" {
			continue
		}
		body := wrapLines(vttEscape(text))
		if sp := normalizeSpace(c.Speaker); sp != "" {
			body = "<v " + vttEscape(sp) + ">" + body
		}
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n",
			vttTime(c.StartMS), vttTime(c.EndMS), body)
	}
	return b.String()
}

func wrapLines(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if runeLen(cur)+1+runeLen(w) <= subMaxLineChars {
			cur += " " + w
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	return strings.Join(lines, "\n")
}

func vttEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func srtTime(ms int64) string { return subTime(ms, ',') }
func vttTime(ms int64) string { return subTime(ms, '.') }

func subTime(ms int64, sep byte) string {
	if ms < 0 {
		ms = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d%c%03d",
		ms/3_600_000, (ms%3_600_000)/60_000, (ms%60_000)/1_000, sep, ms%1_000)
}

// normalizeSpace tüm boşlukları tek boşluğa indirir. Satır sonu ve "-->"
// benzeri artıklar SRT/WebVTT ayrıştırıcısını bozacağı için şarttır.
func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

func runeLen(s string) int { return len([]rune(s)) }

// estimateMS bitiş zamanı eksikse metni okumak için kaba süre üretir.
func estimateMS(text string) int64 {
	ms := int64(runeLen(text)) * subMSPerChar
	if ms < subMinMS {
		ms = subMinMS
	}
	return ms
}
