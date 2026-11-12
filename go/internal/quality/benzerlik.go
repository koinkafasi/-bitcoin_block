package quality

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// ShingleSize karşılaştırma için n-gram uzunluğu.
const ShingleSize = 5

const DuplicateThreshold = 0.60

func similarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	jac := 0.0
	if union > 0 {
		jac = float64(inter) / float64(union)
	}
	small := len(a)
	if len(b) < small {
		small = len(b)
	}
	cont := 0.0
	if small > 0 {
		cont = float64(inter) / float64(small)
	}
	if cont > jac {
		return cont
	}
	return jac
}

// BenzerlikBulgu en benzer kaynakla karşılaştırma sonucudur.
type BenzerlikBulgu struct {
	Ref    string  `json:"ref,omitempty"`
	Title  string  `json:"title,omitempty"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail"`
}

// BenzerlikRapor en yüksek benzerliği ve varsa bulguyu taşır.
type BenzerlikRapor struct {
	Checked  int              `json:"checked"`
	MaxScore float64          `json:"max_score"`
	Matches  []BenzerlikBulgu `json:"matches"`
	Score    int              `json:"score"`
}

// OK yakın kopya yok mu?
func (r BenzerlikRapor) OK() bool { return len(r.Matches) == 0 }

// Shingles metnin n-gram sözcük kümesini döndürür.
func Shingles(text string, n int) map[string]struct{} {
	if n <= 0 {
		n = ShingleSize
	}
	words := tokenize(text)
	set := map[string]struct{}{}
	if len(words) < n {
		if len(words) > 0 {
			set[strings.Join(words, " ")] = struct{}{}
		}
		return set
	}
	for i := 0; i+n <= len(words); i++ {
		set[strings.Join(words[i:i+n], " ")] = struct{}{}
	}
	return set
}

// Jaccard iki shingle kümesinin Jaccard benzerliğini (0..1) verir.
func Jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// Candidate karşılaştırılacak diğer içeriktir.
type Candidate struct {
	Ref   string
	Title string
	Text  string
}

// CheckDuplicates metni adaylarla karşılaştırır. Eşik üstü benzerlikler
// "match" olarak döner; en yüksek benzerlik MaxScore'dur.
func CheckDuplicates(text string, candidates []Candidate) BenzerlikRapor {
	return CheckDuplicatesAt(text, candidates, DuplicateThreshold)
}

func CheckDuplicatesAt(text string, candidates []Candidate, threshold float64) BenzerlikRapor {
	if threshold <= 0 || threshold > 1 {
		threshold = DuplicateThreshold
	}
	base := Shingles(text, ShingleSize)
	r := BenzerlikRapor{Checked: len(candidates)}
	if len(base) == 0 {
		r.Score = 100
		return r
	}
	for _, c := range candidates {
		other := Shingles(c.Text, ShingleSize)
		if len(other) == 0 {
			continue
		}
		s := similarity(base, other)
		if s > r.MaxScore {
			r.MaxScore = s
		}
		if s >= threshold {
			r.Matches = append(r.Matches, BenzerlikBulgu{
				Ref: c.Ref, Title: c.Title, Score: round3(s),
				Detail: fmt.Sprintf("“%s” ile %%%.0f benzer. Aynı bilgiyi tekrar yayınlamak yerine özgün ayrıntı ekleyin.",
					shortTitle(c.Title, c.Ref), s*100),
			})
		}
	}
	sort.Slice(r.Matches, func(i, j int) bool { return r.Matches[i].Score > r.Matches[j].Score })
	r.MaxScore = round3(r.MaxScore)
	// Skor: 100 = hiç benzerlik yok; her eşik üstü eşleşme düşürür.
	r.Score = int((1 - r.MaxScore) * 100)
	if r.Score < 0 {
		r.Score = 0
	}
	return r
}

func shortTitle(title, ref string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		t = strings.TrimSpace(ref)
	}
	r := []rune(t)
	if len(r) > 60 {
		return string(r[:60]) + "…"
	}
	return t
}

func normalizeTextForCompare(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
