package quality

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// SayiBulgu tek bir sayısal ifadenin doğrulama sonucudur.
type SayiBulgu struct {
	Value  string `json:"value"`
	Found  bool   `json:"found"`
	Detail string `json:"detail"`
}

// SayiRapor sayı doğrulama özetidir.
type SayiRapor struct {
	Checked    int         `json:"checked"`
	Verified   int         `json:"verified"`
	Unverified []SayiBulgu `json:"unverified"`
	// Score 0-100 doğrulanan sayı oranı; sayı yoksa 100.
	Score int `json:"score"`
}

// OK doğrulanamayan sayı yok mu?
func (r SayiRapor) OK() bool { return len(r.Unverified) == 0 }

var (
	// Yıl (1900-2099) ve büyük sayılar özellikle risklidir; hepsi yakalanır.
	yearRe = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	// Binlik ayırıcılı, ondalıklı veya düz sayılar: 480, 1.500, 12,5, %42,3
	numRe = regexp.MustCompile(`\d+(?:[.,]\d+)*`)
)

func VerifyNumbers(text, source string) SayiRapor {
	srcNums := numberSet(source)
	var found []string
	seen := map[string]bool{}

	for _, raw := range extractNumbers(text) {
		norm := normalizeNumber(raw)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		found = append(found, norm)
	}

	r := SayiRapor{}
	for _, n := range found {
		r.Checked++
		if srcNums[n] {
			r.Verified++
			continue
		}
		r.Unverified = append(r.Unverified, SayiBulgu{
			Value: n, Found: false,
			Detail: fmt.Sprintf("“%s” kaynakta bulunamadı. Sayıyı transkriptten doğrulayın veya kaldırın.", n),
		})
	}
	if r.Checked == 0 {
		r.Score = 100
		return r
	}
	r.Score = int(float64(r.Verified) / float64(r.Checked) * 100)
	sort.Slice(r.Unverified, func(i, j int) bool { return r.Unverified[i].Value < r.Unverified[j].Value })
	return r
}

// extractNumbers metindeki sayısal ifadeleri sırayla döndürür.
func extractNumbers(text string) []string {
	// Yüzdeleri işaretten bağımsız yakala: %42,3 → 42,3
	out := make([]string, 0, 8)
	out = append(out, numRe.FindAllString(text, -1)...)
	return out
}

// numberSet kaynaktaki tüm sayıları normalize edilmiş küme olarak döndürür.
// Ek olarak yıl ve ondalık varyantları da eklenir.
func numberSet(source string) map[string]bool {
	set := map[string]bool{}
	for _, raw := range numRe.FindAllString(source, -1) {
		if n := normalizeNumber(raw); n != "" {
			set[n] = true
		}
	}
	for _, y := range yearRe.FindAllString(source, -1) {
		set[y] = true
	}
	return set
}

// normalizeNumber binlik/ondalık ayırıcı farklarını tek biçime indirir:
// "1.500" → "1500", "1.250.000" → "1250000", "12,5" → "12.5".
func normalizeNumber(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Yüzde/para işaretleri ve boşluklar sayının parçası değildir.
	s = strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '.' || r == ',' {
			return r
		}
		return -1
	}, s)
	if s == "" {
		return ""
	}

	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	switch {
	case hasDot && hasComma:
		// Sonuncu ayırıcı ondalıktır; diğerleri binliktir.
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case hasComma:
		parts := strings.Split(s, ",")
		if isThousandsGrouping(parts) {
			s = strings.Join(parts, "")
		} else {
			s = strings.ReplaceAll(s, ",", ".")
		}
	case hasDot:
		parts := strings.Split(s, ".")
		if isThousandsGrouping(parts) {
			s = strings.Join(parts, "")
		}
	}
	// Sondaki sıfırları kırp: 12.50 → 12.5
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	// Baştaki sıfırlar (tam sayıda): 007 → 7
	if !strings.Contains(s, ".") {
		s = strings.TrimLeft(s, "0")
		if s == "" {
			s = "0"
		}
	}
	return s
}

// isThousandsGrouping "1.250.000" / "1,500" gibi grupları tanır: ilk grup
// 1-3 hane, sonraki tüm gruplar tam 3 hane.
func isThousandsGrouping(parts []string) bool {
	if len(parts) < 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[0]) > 3 || parts[0] == "0" {
		return false
	}
	for _, p := range parts[1:] {
		if len(p) != 3 {
			return false
		}
	}
	return true
}
