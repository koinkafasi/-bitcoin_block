package trends

import (
	"strings"
	"time"
)

// HazirTerim bir trend terimini önceden ayrıştırılmış hâlde tutar.
type HazirTerim struct {
	ID         int64
	Term       string
	Norm       string
	Tokens     []string
	LastSeenAt time.Time
	Traffic    int
}

// Hazirla terimi bir kez ayrıştırır. Ayrıştırılamayan terim (boş ya da
// yalnız durak sözcük) geçersiz döner.
func Hazirla(id int64, term string, lastSeen time.Time, traffic int) (HazirTerim, bool) {
	norm := NormalizeTerm(term)
	tok := Tokens(term)
	if norm == "" || len(tok) == 0 {
		return HazirTerim{}, false
	}
	return HazirTerim{
		ID: id, Term: term, Norm: norm, Tokens: tok,
		LastSeenAt: lastSeen, Traffic: traffic,
	}, true
}

// HazirBelge bir içeriği önceden ayrıştırılmış hâlde tutar.
type HazirBelge struct {
	norm   string
	tokens map[string]bool
}

// HazirlaBelge metni bir kez ayrıştırır.
func HazirlaBelge(text string) HazirBelge {
	norm := NormalizeTerm(text)
	if norm == "" {
		return HazirBelge{}
	}
	set := make(map[string]bool, 24)
	for _, w := range Tokens(text) {
		set[w] = true
	}
	return HazirBelge{norm: norm, tokens: set}
}

// oran, MatchRatio ile aynı hesabı hazır veriyle yapar.
func (b HazirBelge) oran(t HazirTerim) float64 {
	if b.norm == "" {
		return 0
	}
	// MatchRatio önce tam ifade geçişine bakar.
	if containsWord(b.norm, t.Norm) {
		return 1
	}
	hit := 0
	for _, w := range t.Tokens {
		if b.tokens[w] {
			hit++
		}
	}
	return float64(hit) / float64(len(t.Tokens))
}

// kapsiyor, Covered ile aynı eşiği uygular.
func (b HazirBelge) kapsiyor(t HazirTerim) (int, bool) {
	need := 1.0
	if len(t.Tokens) > 1 {
		need = 0.6
	}
	o := b.oran(t)
	if o < need {
		return 0, false
	}
	return int(o * 100), true
}

func EnIyiEslesme(b HazirBelge, terimler []HazirTerim) (HazirTerim, int, bool) {
	var en HazirTerim
	enOran, bulundu := 0, false
	for _, t := range terimler {
		o, ok := b.kapsiyor(t)
		if !ok {
			continue
		}
		if !bulundu || o > enOran || (o == enOran && t.LastSeenAt.After(en.LastSeenAt)) {
			en, enOran, bulundu = t, o, true
		}
	}
	return en, enOran, bulundu
}

// containsWord, MatchRatio'daki
// strings.Contains(" "+txt+" ", " "+tn+" ") kontrolünün eşdeğeridir.
func containsWord(haystackNorm, needleNorm string) bool {
	if needleNorm == "" || haystackNorm == "" {
		return false
	}
	return strings.Contains(" "+haystackNorm+" ", " "+needleNorm+" ")
}
