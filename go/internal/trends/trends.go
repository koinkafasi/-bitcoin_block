package trends

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Source dış trend kaynağıdır (trend_sources satırının saf kopyası).
type Source struct {
	ID              int64
	SiteID          int64
	Name            string
	Kind            string
	URL             string
	Geo             string
	HL              string
	IntervalSeconds int
	Aktif           bool
	ETag            string
	LastModified    string
}

// News trend terimine bağlı haber öğesidir.
type News struct {
	Title     string
	URL       string
	Source    string
	Picture   string
	Snippet   string
	Published time.Time
}

// Term tek bir trend terimidir.
type Term struct {
	Term          string
	ApproxTraffic int
	Picture       string
	PictureSource string
	PublishedAt   time.Time
	News          []News
}

// Result bir çekim sonucudur.
type Result struct {
	Terms        []Term
	ETag         string
	LastModified string
	NotModified  bool
}

// Provider dış kaynak adaptörüdür.
type Provider interface {
	Kind() string
	Fetch(ctx context.Context, src Source) (Result, error)
}

func NormalizeTerm(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	space := false
	for _, r := range s {
		switch r {
		case 'İ':
			r = 'i'
		case 'I':
			r = 'ı'
		case 'Ş':
			r = 'ş'
		case 'Ğ':
			r = 'ğ'
		case 'Ü':
			r = 'ü'
		case 'Ö':
			r = 'ö'
		case 'Ç':
			r = 'ç'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			space = false
			continue
		}
		// Noktalama/sembol/boşluk: tek boşluğa indir.
		if !space && b.Len() > 0 {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

// StopWords Türkçe içerikte anlam taşımayan kelimelerdir. Başlık eşleşmesi
// ve trend terimi çıkarımında kullanılır.
var StopWords = map[string]bool{
	"acaba": true, "ama": true, "ancak": true, "artık": true, "asla": true,
	"az": true, "bazı": true, "belki": true, "ben": true, "bile": true,
	"bir": true, "birçok": true, "biz": true, "bu": true, "böyle": true,
	"da": true, "daha": true, "de": true, "defa": true, "diye": true,
	"en": true, "gibi": true, "hem": true, "hep": true, "hepsi": true,
	"her": true, "hiç": true, "için": true, "ile": true, "ise": true,
	"kez": true, "ki": true, "kim": true, "mi": true, "mı": true,
	"nasıl": true, "ne": true, "neden": true, "nerede": true, "niçin": true,
	"o": true, "olan": true, "olarak": true, "oldu": true, "olduğu": true,
	"olmak": true, "olması": true, "olsa": true, "olsun": true, "on": true,
	"şey": true, "sadece": true, "sen": true, "siz": true, "sonra": true,
	"şu": true, "tüm": true, "ve": true, "veya": true, "ya": true,
	"yani": true, "yine": true, "yoksa": true, "zaten": true,
	"iki": true, "üç": true, "dört": true, "beş": true,
	"yıl": true, "yılı": true, "yılın": true, "gün": true, "günü": true,
	"bugün": true, "dün": true, "yarın": true, "son": true, "kadar": true,
	"göre": true, "karşı": true, "arasında": true, "içinde": true,
}

// Tokens anlamlı kelimelere ayırır (stop-word ve 2 harften kısa kelimeler atılır).
func Tokens(s string) []string {
	norm := NormalizeTerm(s)
	if norm == "" {
		return nil
	}
	fields := strings.Fields(norm)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len([]rune(f)) < 2 || StopWords[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Fingerprint terimin kaynak/geo/dil bağımsız tekilliğini üretir.
func Fingerprint(termNorm, geo, hl string) string {
	h := sha1.Sum([]byte(termNorm + "|" + strings.ToUpper(strings.TrimSpace(geo)) + "|" + strings.TrimSpace(hl)))
	return hex.EncodeToString(h[:])
}

// HasLatin terimde en az bir Latin harfi var mı? Türkçe yayın için Arapça/
// Kiril vb. dil sızıntılarını ayıklamakta kullanılır.
func HasLatin(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

// ParseApproxTraffic "5000+", "1.000+", "1,5 Mn+" gibi değerleri tam sayıya
// çevirir. Bilinmeyen biçimde 0 döner.
func ParseApproxTraffic(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, "+")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	if s == "" {
		return 0
	}
	mult := 1.0
	switch {
	case strings.HasSuffix(s, "mn"), strings.HasSuffix(s, "m"):
		mult = 1_000_000
		s = strings.TrimSuffix(strings.TrimSuffix(s, "mn"), "m")
	case strings.HasSuffix(s, "bin"), strings.HasSuffix(s, "k"):
		mult = 1_000
		s = strings.TrimSuffix(strings.TrimSuffix(s, "bin"), "k")
	case strings.HasSuffix(s, "b"):
		mult = 1_000_000_000
		s = strings.TrimSuffix(s, "b")
	}
	// Binlik ayırıcı: "1.000" / "1,000"
	if strings.Count(s, ".") == 1 && strings.Count(s, ",") == 0 {
		parts := strings.SplitN(s, ".", 2)
		if len(parts[1]) == 3 {
			s = parts[0] + parts[1]
		}
	}
	s = strings.ReplaceAll(s, ",", ".")
	// "1.5" ondalık ise çarp; tam sayıysa doğrudan.
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(math.Round(f * mult))
	}
	return 0
}

// Weights skor bileşen ağırlıklarıdır (toplam 100 önerilir).
type Weights struct {
	Traffic   float64 `json:"trafik"`
	Freshness float64 `json:"tazelik"`
	News      float64 `json:"haber"`
	Relevance float64 `json:"alaka"`
	Demand    float64 `json:"talep"`
}

// DefaultWeights varsayılan skor ağırlıkları.
func DefaultWeights() Weights {
	return Weights{Traffic: 25, Freshness: 15, News: 15, Relevance: 25, Demand: 20}
}

// Normalize eksik/negatif ağırlıkları varsayılana çeker.
func (w Weights) Normalize() Weights {
	d := DefaultWeights()
	if w.Traffic < 0 {
		w.Traffic = d.Traffic
	}
	if w.Freshness < 0 {
		w.Freshness = d.Freshness
	}
	if w.News < 0 {
		w.News = d.News
	}
	if w.Relevance < 0 {
		w.Relevance = d.Relevance
	}
	if w.Demand < 0 {
		w.Demand = d.Demand
	}
	if w.Traffic+w.Freshness+w.News+w.Relevance+w.Demand == 0 {
		return d
	}
	return w
}

// ScoreInput skor hesaplama girdisidir.
type ScoreInput struct {
	ApproxTraffic int
	NewsCount     int
	FirstSeen     time.Time
	LastSeen      time.Time
	Now           time.Time
	Relevance     float64 // 0..1 site taksonomi/başlık eşleşmesi
	Demand        float64 // 0..1 site içi arama + okunma talebi
	Coverage      int     // son 72 saatte eşleşen yayında içerik sayısı
	Weights       Weights
}

// Score 0-100 arası trend skoru üretir. Saf fonksiyondur; test edilebilir.
func Score(in ScoreInput) int {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	w := in.Weights.Normalize()
	traffic := clamp(math.Log10(1+float64(in.ApproxTraffic))/4, 0, 1)
	last := in.LastSeen
	if last.IsZero() {
		last = now
	}
	hours := now.Sub(last).Hours()
	if hours < 0 {
		hours = 0
	}
	fresh := math.Exp(-hours / 12)
	news := math.Min(float64(in.NewsCount), 10) / 10
	rel := clamp(in.Relevance, 0, 1)
	demand := clamp(in.Demand, 0, 1)

	s := w.Traffic*traffic + w.Freshness*fresh + w.News*news + w.Relevance*rel + w.Demand*demand
	switch {
	case in.Coverage <= 0:
		s += 5 // içerik boşluğu bonusu
	case in.Coverage == 1:
		s -= 2
	default:
		s -= 10 // doygun kapsama
	}
	if !in.FirstSeen.IsZero() && now.Sub(in.FirstSeen) < 24*time.Hour {
		s += 5 // yükselen/yeni terim
	}
	return int(math.Round(clamp(s, 0, 100)))
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// MatchRatio bir terimin metin içindeki anlamlı kelime örtüşme oranıdır.
// Tam ifade geçiyorsa 1 döner. En az bir ortak kelime yoksa 0.
func MatchRatio(term, text string) float64 {
	tn := NormalizeTerm(term)
	if tn == "" {
		return 0
	}
	txt := NormalizeTerm(text)
	if txt == "" {
		return 0
	}
	if strings.Contains(" "+txt+" ", " "+tn+" ") {
		return 1
	}
	tt := Tokens(term)
	if len(tt) == 0 {
		return 0
	}
	textSet := map[string]bool{}
	for _, w := range Tokens(text) {
		textSet[w] = true
	}
	hit := 0
	for _, w := range tt {
		if textSet[w] {
			hit++
		}
	}
	if hit == 0 {
		return 0
	}
	return float64(hit) / float64(len(tt))
}

// Covered eşleşme eşiği: tek kelimelik terimde 1.0, çok kelimede %60.
func Covered(term, text string) bool {
	tt := Tokens(term)
	if len(tt) == 0 {
		return false
	}
	need := 1.0
	if len(tt) > 1 {
		need = 0.6
	}
	return MatchRatio(term, text) >= need
}

// CoverageState kapsama sayısını panel etiketine çevirir.
func CoverageState(n int) string {
	switch {
	case n <= 0:
		return "bosluk"
	case n == 1:
		return "zayif"
	default:
		return "kapsandi"
	}
}
