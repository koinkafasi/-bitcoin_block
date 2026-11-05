// Package quality içerik metnini editoryal kalite açısından ölçer.
//
// Tasarım ilkesi: burada "AI dedektörü" taklit edilmez. Üçüncü taraf AI
// dedektörleri güvenilmez (yanlış pozitif oranları yüksek) ve Google sıralama
// sinyali olarak kullanılmaz. Bunun yerine okunabilirliği, özgünlüğü ve
// tekrarı ölçen deterministik metrikler üretilir; sonuç editöre "neden zayıf"
// bilgisini verir, otomatik yayın kararı vermez.
//
// Ölçülenler:
//   - cümle uzunluğu ortalaması ve değişkenliği (tekdüze ritim kötü yazıdır),
//   - sözcük çeşitliliği (MATTR — hareketli ortalama tip-token oranı),
//   - dil bazlı klişe ifadeler,
//   - uzun aralıklı birebir tekrar (kendi kendini tekrar / şablon metin),
//   - paragraf yapısı.
package quality

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Seviye eşikleri. Haber gövdesi için ölçülmüş, muhafazakâr değerler.
const (
	// MinWordsBlocker altında metin haber sayılamaz (taslak/iskelet).
	MinWordsBlocker = 60
	// MinWordsWarning altında metin büyük ihtimalle eksik.
	MinWordsWarning = 120
	// LongSentenceWords bu uzunluktan uzun cümle "uzun" sayılır.
	LongSentenceWords = 32
	// mattrWindow sözcük çeşitliliği penceresi.
	mattrWindow = 50
	// repeatGram tekrar aranan n-gram uzunluğu.
	repeatGram = 7
	// repeatWarn kez tekrarlanan n-gram uyarı üretir.
	repeatWarn = 3
	// repeatBlock bu kez tekrarlanan uzun aralık yayını engeller.
	repeatBlock = 5
)

// Finding tek bir bulgudur. Code kararlıdır (testler ve UI bunu kullanır).
type Finding struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Count  int    `json:"count,omitempty"`
}

// Metrics ölçülen ham değerlerdir (panelde gösterilir).
type Metrics struct {
	Words            int     `json:"words"`
	Sentences        int     `json:"sentences"`
	Paragraphs       int     `json:"paragraphs"`
	AvgSentenceWords float64 `json:"avg_sentence_words"`
	SentenceStdDev   float64 `json:"sentence_std_dev"`
	LongSentencePct  float64 `json:"long_sentence_pct"`
	MATTR            float64 `json:"mattr"`
	Cliques          int     `json:"cliches"`
	RepeatedPhrases  int     `json:"repeated_phrases"`
}

// Report içerik kalite raporudur. Blockers yayın öncesi çözülmeli,
// Warnings editöre gösterilir.
type Report struct {
	Score    int       `json:"score"`
	Grade    string    `json:"grade"`
	Blockers []Finding `json:"blockers"`
	Warnings []Finding `json:"warnings"`
	Metrics  Metrics   `json:"metrics"`
	// Imla imla/biçim bulgularıdır; Warnings içine de eklenir ama panelde
	// ayrı bir liste olarak gösterilir.
	Imla []ImlaBulgu `json:"imla"`
}

// OK hiç engelleyici bulgu yok mu?
func (r Report) OK() bool { return len(r.Blockers) == 0 }

// cliches dil bazlı klişe ifadeler. Amaç AI'yı "insanlaştırmak" değil,
// okunabilirliği düşüren kalıplaşmış ifadeleri editöre göstermek.
var cliches = map[string][]string{
	"tr": {
		"günümüz dünyasında", "günümüzde", "unutmayalım ki", "sonuç olarak",
		"özetle", "bu bağlamda", "bu doğrultuda", "söz konusu", "adeta",
		"hiç kuşkusuz", "şüphesiz", "kısacası", "bilindiği üzere",
		"ayrıca belirtmek gerekir", "her şeyden önce", "son derece önemli",
		"büyük önem taşıyor", "nefes kesen", "tam anlamıyla", "gözler çevrildi",
		"bir yandan", "diğer yandan", "dikkat çekiyor", "ön plana çıkıyor",
	},
	"en": {
		"in today's fast-paced world", "in today's world", "it is important to note",
		"it's important to note", "in conclusion", "delve into", "tapestry",
		"landscape of", "navigate the complexities", "unlock the potential",
		"game-changer", "game changer", "cutting-edge", "seamless", "leverage",
		"in the realm of", "at the end of the day", "it goes without saying",
		"needless to say", "furthermore", "moreover", "a testament to",
	},
}

// Analyze metni ölçer. lang "tr"/"en"; tanınmayan dilde iki liste birlikte
// kullanılır (çok dilli sitelerde güvenli varsayılan).
func Analyze(text, lang string) Report {
	body := strings.TrimSpace(text)
	sentences := splitSentences(body)
	paragraphs := splitParagraphs(body)
	words := tokenize(body)

	m := Metrics{
		Words:      len(words),
		Sentences:  len(sentences),
		Paragraphs: len(paragraphs),
	}
	if len(sentences) > 0 {
		lens := make([]float64, 0, len(sentences))
		long := 0
		for _, s := range sentences {
			n := len(tokenize(s))
			lens = append(lens, float64(n))
			if n > LongSentenceWords {
				long++
			}
		}
		m.AvgSentenceWords = round1(mean(lens))
		m.SentenceStdDev = round1(stdDev(lens))
		m.LongSentencePct = round1(float64(long) / float64(len(lens)) * 100)
	}
	m.MATTR = round3(mattr(words, mattrWindow))

	var blockers, warnings []Finding
	imla := CheckImla(body, lang)

	lower := strings.ToLower(body)
	for _, phrase := range clicheList(lang) {
		if n := strings.Count(lower, phrase); n > 0 {
			m.Cliques += n
		}
	}
	if m.Cliques > 0 {
		warnings = append(warnings, Finding{
			Code: "cliche", Count: m.Cliques,
			Title:  "Kalıplaşmış ifadeler",
			Detail: fmt.Sprintf("%d kalıplaşmış ifade bulundu; bunları somut bilgiyle değiştirin.", m.Cliques),
		})
	}

	if m.Words == 0 {
		blockers = append(blockers, Finding{
			Code: "empty", Title: "Metin boş",
			Detail: "Kalite ölçümü için gövde metni gerekli.",
		})
		return finish(m, blockers, warnings, imla)
	}
	if m.Words < MinWordsBlocker {
		blockers = append(blockers, Finding{
			Code: "too_short", Count: m.Words,
			Title:  "Metin çok kısa",
			Detail: fmt.Sprintf("%d sözcük var; haber gövdesi için en az %d sözcük beklenir.", m.Words, MinWordsWarning),
		})
	} else if m.Words < MinWordsWarning {
		warnings = append(warnings, Finding{
			Code: "short", Count: m.Words,
			Title:  "Metin kısa",
			Detail: fmt.Sprintf("%d sözcük; okuyucuya yeterli bağlam vermek için %d+ sözcük önerilir.", m.Words, MinWordsWarning),
		})
	}

	// Tekdüze cümle ritmi: her cümle neredeyse aynı uzunlukta.
	if m.Sentences >= 6 && m.SentenceStdDev < 4 {
		warnings = append(warnings, Finding{
			Code:   "uniform_rhythm",
			Title:  "Cümle ritmi tekdüze",
			Detail: fmt.Sprintf("Cümle uzunlukları birbirine çok yakın (sapma %.1f). Kısa ve uzun cümleleri karıştırın.", m.SentenceStdDev),
		})
	}
	if m.LongSentencePct > 25 {
		warnings = append(warnings, Finding{
			Code:   "long_sentences",
			Title:  "Uzun cümle oranı yüksek",
			Detail: fmt.Sprintf("Cümlelerin %%%.0f'i %d sözcükten uzun; bölerek okunabilirliği artırın.", m.LongSentencePct, LongSentenceWords),
		})
	}
	if m.Words >= 200 && m.MATTR > 0 && m.MATTR < 0.60 {
		warnings = append(warnings, Finding{
			Code:   "low_variety",
			Title:  "Sözcük çeşitliliği düşük",
			Detail: fmt.Sprintf("MATTR %.2f; aynı sözcükler sık yineleniyor, eş anlamlı ve somut ifadeler kullanın.", m.MATTR),
		})
	}

	repeats := repeatedPhrases(words, repeatGram)
	m.RepeatedPhrases = len(repeats)
	if len(repeats) > 0 {
		sort.Slice(repeats, func(i, j int) bool { return repeats[i].n > repeats[j].n })
		top := repeats[0]
		f := Finding{
			Code: "repeated_phrase", Count: top.n,
			Title:  "Birebir tekrar",
			Detail: fmt.Sprintf("%d kez yinelenen ifade: “%s”", top.n, top.text),
		}
		if top.n >= repeatBlock {
			blockers = append(blockers, f)
		} else if top.n >= repeatWarn {
			warnings = append(warnings, f)
		}
	}

	if m.Words >= 250 && m.Paragraphs < 3 {
		warnings = append(warnings, Finding{
			Code:   "wall_of_text",
			Title:  "Paragraf yapısı zayıf",
			Detail: fmt.Sprintf("%d paragraf; uzun metni en az 3 paragrafa bölün.", m.Paragraphs),
		})
	}

	return finish(m, blockers, warnings, imla)
}

// finish skoru ve notu hesaplar. Ağırlıklar bilinçli olarak muhafazakâr:
// uyarılar bilgilendirir, engelleyiciler yayın öncesi çözülmelidir. Bir
// engelleyici varsa skor "yayına hazır değil" bandına (<= 50) çekilir;
// böylece kısa/boş metin yüksek not alamaz.
func finish(m Metrics, blockers, warnings []Finding, imla []ImlaBulgu) Report {
	score := 100
	if len(blockers) > 0 {
		score -= 30 * len(blockers)
		if score > 50 {
			score = 50
		}
	}
	// İmla bulguları uyarı sayılır: yayını engellemez ama puanı düşürür.
	imlaPenalty := 3 * len(imla)
	if imlaPenalty > 18 {
		imlaPenalty = 18
	}
	score -= imlaPenalty
	penalty := 5 * len(warnings)
	if penalty > 30 {
		penalty = 30
	}
	score -= penalty
	if len(blockers) > 0 && score > 50 {
		score = 50
	}
	// A notu yalnız hiç uyarı/imla bulgusu yokken verilir; aksi hâlde en iyi
	// not B'dir.
	if len(blockers) == 0 && (len(warnings) > 0 || len(imla) > 0) && score >= 90 {
		score = 89
	}
	// Ölçülecek metin yoksa not verilmez: boş iskelet "yayına hazır"
	// görünmemeli.
	for _, b := range blockers {
		if b.Code == "empty" {
			score = 0
		}
	}
	if score < 0 {
		score = 0
	}
	r := Report{Score: score, Grade: grade(score), Blockers: blockers, Warnings: warnings, Metrics: m, Imla: imla}
	if r.Blockers == nil {
		r.Blockers = []Finding{}
	}
	if r.Warnings == nil {
		r.Warnings = []Finding{}
	}
	if r.Imla == nil {
		r.Imla = []ImlaBulgu{}
	}
	return r
}

func grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	}
	return "E"
}

func clicheList(lang string) []string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "tr":
		return cliches["tr"]
	case "en":
		return cliches["en"]
	}
	out := make([]string, 0, len(cliches["tr"])+len(cliches["en"]))
	out = append(out, cliches["tr"]...)
	out = append(out, cliches["en"]...)
	return out
}

// tokenize unicode harf/rakamlardan oluşan sözcükleri küçük harfle döndürür.
// Türkçe ekler ayrılmaz; ölçümler ek-duyarlı olmak zorunda değil.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// splitSentences cümle sonu noktalama ve satır sonlarından böler.
func splitSentences(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		t := strings.TrimSpace(b.String())
		if t != "" {
			out = append(out, t)
		}
		b.Reset()
	}
	for _, r := range s {
		switch r {
		case '.', '!', '?', '…', '\n':
			b.WriteRune(r)
			flush()
		case ';':
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}

// splitParagraphs boş satırla ayrılmış blokları döndürür.
func splitParagraphs(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 && strings.TrimSpace(s) != "" {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}

// mattr hareketli ortalama tip-token oranıdır. Metin pencereden kısaysa
// düz TTR'e düşer; böylece kısa metinler cezalandırılmaz.
func mattr(words []string, window int) float64 {
	if len(words) == 0 {
		return 0
	}
	if window <= 0 || len(words) <= window {
		return float64(len(unique(words))) / float64(len(words))
	}
	var sum float64
	var n int
	for i := 0; i+window <= len(words); i++ {
		chunk := words[i : i+window]
		sum += float64(len(unique(chunk))) / float64(window)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func unique(words []string) map[string]struct{} {
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

type phraseCount struct {
	text string
	n    int
}

// repeatedPhrases n-gram'ları sayar; yalnız tekrar edenleri döndürür.
// Aynı metnin kaydırılmış kopyaları (a b c d e f g / b c d e f g h) tek
// sayılsın diye ardışık tekrarlar birleştirilir.
func repeatedPhrases(words []string, n int) []phraseCount {
	if n <= 0 || len(words) < n {
		return nil
	}
	counts := make(map[string]int, len(words))
	for i := 0; i+n <= len(words); i++ {
		counts[strings.Join(words[i:i+n], " ")]++
	}
	var out []phraseCount
	for text, c := range counts {
		if c >= repeatWarn {
			out = append(out, phraseCount{text: text, n: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].text < out[j].text
	})
	// Aynı bölgenin kayan kopyalarını ele: metinleri birbirinin içeren
	// olanlardan yalnız en çok tekrarlananı bırak.
	filtered := out[:0]
	for _, p := range out {
		skip := false
		for _, kept := range filtered {
			if strings.Contains(kept.text, p.text) || strings.Contains(p.text, kept.text) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func stdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	var s float64
	for _, x := range xs {
		d := x - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(xs)))
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
