package trends

import "time"

// CoverageDoc kapsama analizinde karşılaştırılan yayın özetidir. Text,
// başlık + spot + etiketlerin birleşimidir (çağıran birleştirir).
type CoverageDoc struct {
	ID          int64
	Text        string
	PublishedAt time.Time
	Hits        int64
}

// CoverageResult kapsama analizinin sonucudur.
//
// İki ayrı sayı taşımasının nedeni, bu ikisinin farklı soruların cevabı
// olmasıdır:
//
//   - Total  → "bu konuda sitede içeriğimiz var mı?" (arşiv dahil). Panelde,
//     `/api/trends` çıktısında ve `trend_terms.coverage_count` alanında
//     gösterilen **gerçek** sayı budur.
//   - Recent → "bu haber döngüsünde işledik mi?" (yalnız taze pencere).
//     Skorlamadaki "içerik boşluğu" sinyali budur.
//
// Eski kod tek bir 72 saatlik pencere kullanıyordu ve bu iki soruyu aynı
// sayıyla cevaplıyordu. Site 72 saat yayın yapmadığında eşleşme hiç
// bulunamıyor; `coverage_count` tüm terimlerde 0 kalıyor, editöre arşivde
// içerik varken "boşluk" gösteriliyor ve skorda her terime boşluk bonusu
// veriliyordu.
type CoverageResult struct {
	// Matches eşleşen içerik id → yüzde (0..100) eşleşme oranı.
	Matches map[int64]int
	// Total tüm pencerede eşleşen içerik sayısı.
	Total int
	// Recent yalnız recentWindow içinde yayınlanmış eşleşme sayısı.
	Recent int
	// HitSum eşleşen içeriklerin toplam okunması (talep sinyali).
	HitSum int64
}

// Coverage terimi belgelerle eşleştirir. Saf fonksiyondur; `Covered` ve
// `MatchRatio` eşiklerini kullanır.
//
// recentWindow 0 verilirse taze sayım yapılmaz (Recent = 0). PublishedAt'i
// boş olan belge taze sayılmaz: boş tarihi "şimdi" varsaymak, tarihi
// eksik eski içe aktarımlarda yanlış pozitif üretir.
func Coverage(term string, docs []CoverageDoc, now time.Time, recentWindow time.Duration) CoverageResult {
	res := CoverageResult{Matches: make(map[int64]int, 8)}
	if NormalizeTerm(term) == "" || len(docs) == 0 {
		return res
	}
	if now.IsZero() {
		now = time.Now()
	}
	for _, d := range docs {
		if !Covered(term, d.Text) {
			continue
		}
		res.Matches[d.ID] = int(MatchRatio(term, d.Text) * 100)
		res.Total++
		res.HitSum += d.Hits
		if recentWindow > 0 && !d.PublishedAt.IsZero() && now.Sub(d.PublishedAt) <= recentWindow {
			res.Recent++
		}
	}
	return res
}
