package quality

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Bu dosya alıntı doğrulamasıdır. Haber üretiminde en yüksek risk, modelin
// transkriptte geçmeyen bir cümleyi tırnak içinde "alıntı" olarak yazmasıdır.
// Google sıralamasından bağımsız olarak bu bir doğruluk/güvenilirlik sorunudur
// (E-E-A-T ve düzeltme maliyeti).
//
// Yöntem deterministiktir: alıntı, kaynak metnin normalize edilmiş hâlinde
// aranır. Küçük farklar (noktalama, büyük/küçük harf, fazladan bağlaç) tolere
// edilir; anlamca değiştirilmiş veya uydurulmuş alıntı yakalanır.

// AtifBulgu tek bir alıntının doğrulama sonucudur.
type AtifBulgu struct {
	Quote  string  `json:"quote"`
	Found  bool    `json:"found"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail"`
}

// AtifRapor alıntı doğrulama özetidir.
type AtifRapor struct {
	Checked    int         `json:"checked"`
	Verified   int         `json:"verified"`
	Unverified []AtifBulgu `json:"unverified"`
	// Score 0-100: doğrulanan alıntı oranı. Alıntı yoksa 100 döner.
	Score int `json:"score"`
}

// OK doğrulanamayan alıntı yok mu?
func (r AtifRapor) OK() bool { return len(r.Unverified) == 0 }

// minQuoteWords bu uzunluğun altındaki alıntılar ("Evet.", "Hayır.")
// doğrulanmış sayılır: kısa ünlemler transkriptte farklı yerde geçebilir ve
// kontrol edilmesi yanlış pozitif üretir.
const minQuoteWords = 4

// verifyThreshold alıntının doğrulanmış sayılması için gereken sıralı
// sözcük örtüşme oranı.
const verifyThreshold = 0.85

// VerifyQuotes alıntıları kaynak metne karşı doğrular. quotes boşsa Score 100.
func VerifyQuotes(quotes []string, source string) AtifRapor {
	src := normalizeForQuote(source)
	srcTokens := tokenize(src)

	r := AtifRapor{}
	for _, raw := range quotes {
		q := strings.TrimSpace(raw)
		if q == "" {
			continue
		}
		words := tokenize(normalizeForQuote(q))
		if len(words) < minQuoteWords {
			r.Checked++
			r.Verified++
			continue
		}
		r.Checked++
		if score := sequentialOverlap(words, srcTokens); score >= verifyThreshold {
			r.Verified++
			continue
		} else {
			r.Unverified = append(r.Unverified, AtifBulgu{
				Quote: q, Found: false, Score: round3(score),
				Detail: fmt.Sprintf("Alıntı kaynakta bulunamadı (örtüşme %%%.0f). Tırnak içindeki cümleyi transkriptten birebir alın.", score*100),
			})
		}
	}
	if r.Checked == 0 {
		r.Score = 100
		return r
	}
	r.Score = int(float64(r.Verified) / float64(r.Checked) * 100)
	sort.Slice(r.Unverified, func(i, j int) bool { return r.Unverified[i].Score < r.Unverified[j].Score })
	return r
}

// sequentialOverlap alıntı sözcüklerinin kaynakta sırayla ne kadarının
// bulunduğunu ölçer (0..1). Kaydırılmış eşleşme aranır: alıntının en uzun
// bitişik parçasının oranı kullanılır.
func sequentialOverlap(quote, src []string) float64 {
	if len(quote) == 0 || len(src) == 0 {
		return 0
	}
	best := 0
	// Her olası başlangıç için sıralı eşleşme say.
	for i := 0; i < len(src); i++ {
		j, k := 0, i
		matched := 0
		for j < len(quote) && k < len(src) {
			if quote[j] == src[k] {
				matched++
				j++
				k++
				continue
			}
			// küçük kaymalara izin ver: kaynakta bir sözcük atla
			if k+1 < len(src) && quote[j] == src[k+1] {
				k += 2
				matched++
				j++
				continue
			}
			break
		}
		if matched > best {
			best = matched
		}
		if best == len(quote) {
			return 1
		}
	}
	return float64(best) / float64(len(quote))
}

// normalizeForQuote alıntı karşılaştırması için metni sadeleştirir:
// küçük harf, Türkçe/ASCII noktalama farkları ve tırnaklar kaldırılır.
func normalizeForQuote(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case '’', '‘', '‛', '`', '´':
			continue
		case '“', '”', '„', '"':
			continue
		case 'ı':
			b.WriteRune('i')
		case 'İ':
			b.WriteRune('i')
		case '\u00a0':
			b.WriteRune(' ')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
				b.WriteRune(r)
			} else {
				b.WriteRune(' ')
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
