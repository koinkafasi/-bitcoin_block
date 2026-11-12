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
