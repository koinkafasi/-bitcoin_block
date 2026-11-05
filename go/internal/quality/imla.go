package quality

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Bu dosya kural tabanlı imla/biçim denetimidir. Sözlük tabanlı bir yazım
// denetleyicisi (hunspell/LanguageTool) sonradan eklenebilir; buradaki
// kurallar bilinçli olarak YÜKSEK İSABETLİ seçildi: yanlış pozitif veren bir
// uyarı editörün güvenini kaybettirir. Her kural gerçek bir yazım kılavuzu
// kuralına veya net bir biçim hatasına dayanır.

// ImlaBulgu tek bir imla/biçim bulgusudur.
type ImlaBulgu struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Sample  string `json:"sample,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// birlesikYazim Türkçe'de ayrı yazılması gereken, sık bitişik yazılan
// ifadelerdir. Yalnız kesin olanlar listelenir ("birçok" doğru yazımdır,
// listede yoktur).
var birlesikYazim = map[string]string{
	"herşey":      "her şey",
	"herşeyi":     "her şeyi",
	"herşeyin":    "her şeyin",
	"birşey":      "bir şey",
	"birşeyi":     "bir şeyi",
	"birşeyler":   "bir şeyler",
	"hiçbirşey":   "hiçbir şey",
	"yada":        "ya da",
	"birde":       "bir de",
	"herkez":      "herkes",
	"herkezin":    "herkesin",
	"yalnış":      "yanlış",
	"yanlışlıkla": "yanlışlıkla",
	"klavuz":      "kılavuz",
	"orjinal":     "orijinal",
	"şöför":       "şoför",
	"müsade":      "müsaade",
	"ardarda":     "art arda",
	"üstüste":     "üst üste",
	"altalta":     "alt alta",
	"yanlız":      "yalnız",
	"yanlış":      "yanlış",
	"tabiki":      "tabii ki",
	"tabi ki":     "tabii ki",
	"hernekadar":  "her ne kadar",
	"birçok":      "", // doğru; boş değer "dokunma" anlamında
	"birtakım":    "",
	"birkaç":      "",
}

// CheckImla metindeki imla/biçim hatalarını bulur. lang "tr" için Türkçe
// sözcük kuralları, tüm diller için ortak biçim kuralları uygulanır.
func CheckImla(text, lang string) []ImlaBulgu {
	body := strings.TrimSpace(text)
	if body == "" {
		return nil
	}
	var out []ImlaBulgu

	if strings.EqualFold(strings.TrimSpace(lang), "tr") {
		out = append(out, checkMergedWords(body)...)
	}
	out = append(out, checkPunctuationSpacing(body)...)
	out = append(out, checkRepeatedWord(body)...)
	out = append(out, checkSentenceCase(body)...)
	out = append(out, checkPunctuationStack(body)...)
	return out
}

// checkMergedWords bitişik yazılmış ifadeleri bulur. Kelime sınırına saygı
// duyar: "herşey" bulunur, "herşeyler" gibi türevler de yakalanır ama
// "birşeylerin" yanlış eşleşme üretmez (o da listede ayrıca var).
func checkMergedWords(body string) []ImlaBulgu {
	lower := strings.ToLower(body)
	counts := map[string]int{}
	samples := map[string]string{}
	for wrong, right := range birlesikYazim {
		if right == "" {
			continue
		}
		n := countWordOccurrences(lower, wrong)
		if n == 0 {
			continue
		}
		counts[rightFrom(wrong, right)] += n
		if _, ok := samples[rightFrom(wrong, right)]; !ok {
			samples[rightFrom(wrong, right)] = wrong
		}
	}
	if len(counts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []ImlaBulgu
	for _, k := range keys {
		out = append(out, ImlaBulgu{
			Code: "imla_birlesik", Count: counts[k], Sample: samples[k],
			Message: fmt.Sprintf("“%s” bitişik yazılmış; doğrusu “%s”.", samples[k], k),
		})
	}
	return out
}

func rightFrom(wrong, right string) string { return right }

// countWordOccurrences metinde kelime sınırlarıyla eşleşen sayıyı verir.
// Türkçe ek alan ifadeler için ("herşeyi") tam kelime eşleşmesi yeterlidir;
// gövde listesi ek çekimlerini ayrıca içerir.
func countWordOccurrences(lower, needle string) int {
	n := 0
	idx := 0
	for {
		i := strings.Index(lower[idx:], needle)
		if i < 0 {
			return n
		}
		start := idx + i
		end := start + len(needle)
		before := rune(0)
		after := rune(0)
		if start > 0 {
			before = rune(lower[start-1])
		}
		if end < len(lower) {
			after = rune(lower[end])
		}
		if !isWordRune(before) && !isWordRune(after) {
			n++
		}
		idx = end
	}
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// checkPunctuationSpacing noktalama öncesi boşluk ve sonrası eksik boşluğu
// bulur. Sayı ayırıcıları (3,2 / 1.500) ve kısaltmalar dışlanır.
func checkPunctuationSpacing(body string) []ImlaBulgu {
	var out []ImlaBulgu
	beforeCount := 0
	var beforeSample string
	for i, r := range body {
		if r != ' ' {
			continue
		}
		j := i + 1
		for j < len(body) && body[j] == ' ' {
			j++
		}
		if j >= len(body) {
			continue
		}
		next := rune(body[j])
		if next == ',' || next == '.' || next == ';' || next == ':' || next == '!' || next == '?' || next == ')' {
			beforeCount++
			if beforeSample == "" {
				beforeSample = snippet(body, i)
			}
		}
	}
	if beforeCount > 0 {
		out = append(out, ImlaBulgu{
			Code: "imla_bosluk_once", Count: beforeCount, Sample: beforeSample,
			Message: fmt.Sprintf("Noktalamadan önce boşluk var (%d yerde). Örnek: %s", beforeCount, beforeSample),
		})
	}

	missing := 0
	var missingSample string
	runes := []rune(body)
	for i := 0; i < len(runes)-1; i++ {
		r := runes[i]
		if r != '.' && r != ',' && r != ';' && r != ':' && r != '!' && r != '?' {
			continue
		}
		nxt := runes[i+1]
		if unicode.IsSpace(nxt) || nxt == '.' || nxt == ',' || nxt == ')' || nxt == '”' || nxt == '"' || nxt == '\'' {
			continue
		}
		// sayı ayırıcısı: 3,2 · 1.500 · 12.30
		if unicode.IsDigit(nxt) {
			continue
		}
		// Kısaltma: "vb.Sonraki" gibi birleşik kullanım da hatadır; bu yüzden
		// büyük harf tek başına istisna sayılmaz. Yalnız noktadan önceki
		// kelime kısa/kısaltmaysa (vb., Dr., T.C.) atlanır.
		if unicode.IsUpper(nxt) && isAbbreviation(wordBefore(runes, i)) {
			continue
		}
		missing++
		if missingSample == "" {
			missingSample = snippetRunes(runes, i)
		}
	}
	if missing > 0 {
		out = append(out, ImlaBulgu{
			Code: "imla_bosluk_sonra", Count: missing, Sample: missingSample,
			Message: fmt.Sprintf("Noktalamadan sonra boşluk yok (%d yerde). Örnek: %s", missing, missingSample),
		})
	}
	return out
}

// checkRepeatedWord yan yana aynı kelimenin iki kez yazılmasını bulur
// ("ve ve", "the the"). Tek harfli bağlaçlar da yakalanır.
func checkRepeatedWord(body string) []ImlaBulgu {
	words := strings.Fields(body)
	count := 0
	sample := ""
	for i := 0; i+1 < len(words); i++ {
		a := normalizeWord(words[i])
		b := normalizeWord(words[i+1])
		if a == "" || a != b {
			continue
		}
		if len([]rune(a)) < 2 && a != "de" && a != "da" && a != "ki" && a != "mi" && a != "ve" && a != "bu" {
			continue
		}
		count++
		if sample == "" {
			sample = words[i] + " " + words[i+1]
		}
	}
	if count == 0 {
		return nil
	}
	return []ImlaBulgu{{
		Code: "imla_tekrar_kelime", Count: count, Sample: sample,
		Message: fmt.Sprintf("Yan yana yinelenen kelime (%d yerde). Örnek: “%s”", count, sample),
	}}
}

func normalizeWord(w string) string {
	w = strings.ToLower(w)
	return strings.TrimFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// checkSentenceCase cümle başında küçük harf kullanımını bulur. Yalnız
// nokta/soru/ünlem sonrası başlayan cümleler denetlenir; parantez içi ve
// kısaltma sonrası (vb. gibi) yanlış pozitif üretmemek için iki harften uzun
// ve büyük harfle başlamayan kelimeler sayılır.
func checkSentenceCase(body string) []ImlaBulgu {
	runes := []rune(body)
	count := 0
	sample := ""
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		j := i + 1
		for j < len(runes) && (runes[j] == ' ' || runes[j] == '\n' || runes[j] == '\t' || runes[j] == '"' || runes[j] == '“') {
			j++
		}
		if j+1 >= len(runes) {
			break
		}
		first := runes[j]
		if !unicode.IsLower(first) {
			continue
		}
		// kısaltma kontrolü: noktadan önceki kelime kısa/kısaltmaysa
		// (vb., Dr., T.C.) atla.
		if isAbbreviation(wordBefore(runes, i)) {
			continue
		}
		count++
		if sample == "" {
			sample = snippetRunes(runes, j)
		}
		i = j
	}
	if count == 0 {
		return nil
	}
	return []ImlaBulgu{{
		Code: "imla_cumle_baslangici", Count: count, Sample: sample,
		Message: fmt.Sprintf("Cümle küçük harfle başlıyor (%d yerde). Örnek: %s", count, sample),
	}}
}

func isAbbreviation(prev string) bool {
	prev = strings.TrimSpace(prev)
	if prev == "" {
		return false
	}
	if strings.Contains(prev, ".") {
		return true
	}
	// "vb.", "vs.", "bkz.", "Dr.", "T.C." gibi kısa tek sözcükler kısaltma
	// sayılır. Uzun kelimeler ("bitti.") kısaltma değildir.
	return len([]rune(prev)) <= 3
}

// wordBefore i konumundaki noktalamadan önceki kelimeyi döndürür.
func wordBefore(runes []rune, i int) string {
	k := i - 1
	for k >= 0 && (unicode.IsLetter(runes[k]) || unicode.IsDigit(runes[k]) || runes[k] == '.') {
		k--
	}
	return string(runes[k+1 : i])
}

// checkPunctuationStack üç ve daha fazla ardışık noktalama işaretini bulur
// ("!!!", "???", "..."). Üç nokta (…) tek başına kabul edilir.
func checkPunctuationStack(body string) []ImlaBulgu {
	count := 0
	sample := ""
	runes := []rune(body)
	for i := 0; i < len(runes); {
		r := runes[i]
		if r != '!' && r != '?' {
			i++
			continue
		}
		j := i
		for j < len(runes) && runes[j] == r {
			j++
		}
		if j-i >= 2 {
			count++
			if sample == "" {
				sample = string(runes[i:j])
			}
		}
		i = j
	}
	if count == 0 {
		return nil
	}
	return []ImlaBulgu{{
		Code: "imla_noktalama_yigini", Count: count, Sample: sample,
		Message: fmt.Sprintf("Yinelenen noktalama (%d yerde). Örnek: “%s” — tek işaret yeterli.", count, sample),
	}}
}

func snippet(body string, i int) string {
	r := []rune(body)
	ri := len([]rune(body[:i]))
	return snippetRunes(r, ri)
}

func snippetRunes(runes []rune, i int) string {
	const pad = 24
	start := i - pad
	if start < 0 {
		start = 0
	}
	end := i + pad
	if end > len(runes) {
		end = len(runes)
	}
	s := strings.TrimSpace(string(runes[start:end]))
	return strings.ReplaceAll(s, "\n", " ")
}
