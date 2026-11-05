package quality

import (
	"sort"
	"strings"
)

// Bu dosya alıntıların hangi transkript segmentinden geldiğini bulur.
//
// Neden: alıntı "kaynakta var" demek yeterli değil; editörün alıntının
// saniyesini görüp kaydı oradan dinleyebilmesi gerekir. Böylece doğrulama
// tıklanabilir bir kanıta dönüşür (E-E-A-T ve düzeltme maliyeti).

// SourceSegment alıntı eşleştirmesi için transkript bölümüdür.
type SourceSegment struct {
	StartMS int64
	EndMS   int64
	Speaker string
	Text    string
}

// QuoteSource alıntının kaynak segmentidir.
type QuoteSource struct {
	Quote    string  `json:"quote"`
	StartMS  int64   `json:"start_ms"`
	EndMS    int64   `json:"end_ms"`
	Speaker  string  `json:"speaker,omitempty"`
	Segment  int     `json:"segment"`
	Verified bool    `json:"verified"`
	Overlap  float64 `json:"overlap"`
}

// AttributeQuotes her alıntıyı en iyi eşleşen segmente bağlar. Eşleşme
// bulunamazsa Verified=false ve Segment=-1 döner.
//
// Eşik alıntı doğrulamasıyla aynıdır (verifyThreshold): segment metni
// alıntının >=%85'ini sırayla içermelidir.
func AttributeQuotes(quotes []string, segments []SourceSegment) []QuoteSource {
	if len(quotes) == 0 {
		return nil
	}
	normSegs := make([]string, len(segments))
	tokens := make([][]string, len(segments))
	for i, s := range segments {
		normSegs[i] = normalizeForQuote(s.Text)
		tokens[i] = tokenize(normSegs[i])
	}

	out := make([]QuoteSource, 0, len(quotes))
	for _, raw := range quotes {
		q := strings.TrimSpace(raw)
		if q == "" {
			continue
		}
		qTokens := tokenize(normalizeForQuote(q))
		src := QuoteSource{Quote: q, Segment: -1}
		if len(qTokens) == 0 {
			out = append(out, src)
			continue
		}
		best, bestScore := -1, 0.0
		for i := range segments {
			if len(tokens[i]) == 0 {
				continue
			}
			if sc := sequentialOverlap(qTokens, tokens[i]); sc > bestScore {
				best, bestScore = i, sc
			}
		}
		if best >= 0 && bestScore >= verifyThreshold {
			src.Segment = best
			src.StartMS = segments[best].StartMS
			src.EndMS = segments[best].EndMS
			src.Speaker = strings.TrimSpace(segments[best].Speaker)
			src.Verified = true
			src.Overlap = round3(bestScore)
		}
		out = append(out, src)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SortQuoteSourcesVerifiedFirst doğrulanmış alıntıları öne alır.
func SortQuoteSourcesVerifiedFirst(in []QuoteSource) []QuoteSource {
	out := make([]QuoteSource, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Verified != out[j].Verified {
			return out[i].Verified
		}
		return out[i].StartMS < out[j].StartMS
	})
	return out
}
