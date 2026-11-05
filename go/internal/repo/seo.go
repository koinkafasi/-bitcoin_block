package repo

// ---- SEO puanı (spec 02 §2 / SERP) ----
// İçerik formunda gösterilen 10 kriterli SEO skoru (0-100).

// SEOPart tek bir SEO kriteri ve kazandığı ağırlık (0-10).
type SEOPart struct {
	Name   string
	Weight int
}

// SEOResult SEO puanı sonucu (0-100) ve kriter dökümü.
type SEOResult struct {
	Score int
	Parts []SEOPart
}

// ComputeSEO içeriğin SEO puanını hesaplar (10 kriter, her biri 0-10).
func ComputeSEO(title, spot string, words, h2 int, cover bool, imgs, tags, bold, italic int) SEOResult {
	parts := []SEOPart{
		{"Başlık uzunluğu (40-70)", bandWeight(runeLen(title), 40, 70, 30, 80)},
		{"Spot uzunluğu (120-160)", bandWeight(runeLen(spot), 120, 160, 90, 200)},
		{"Kelime sayısı (300+)", tierWeight(words, 300, 150)},
		{"Ara başlık (2-4)", h2Weight(h2)},
		{"Kapak görseli", boolWeight(cover)},
		{"İçerik görseli (1+)", tierWeight(imgs, 1, 1)},
		{"Etiket (3+)", tierWeight(tags, 3, 1)},
		{"Kalın vurgu (1+)", boolWeight(bold > 0)},
		{"İtalik vurgu (1+)", boolWeight(italic > 0)},
		{"Yeterli içerik (kelime ≥4)", boolWeight(words >= 4)},
	}
	score := 0
	for _, p := range parts {
		score += p.Weight
	}
	if score > 100 {
		score = 100
	}
	return SEOResult{Score: score, Parts: parts}
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// bandWeight: [idealMin, idealMax] → 10; [okMin, okMax] → 5; aksi 0.
func bandWeight(v, idealMin, idealMax, okMin, okMax int) int {
	if v >= idealMin && v <= idealMax {
		return 10
	}
	if v >= okMin && v <= okMax {
		return 5
	}
	return 0
}

// tierWeight: v ≥ min → 10; v ≥ partial → 5; aksi 0.
func tierWeight(v, min, partial int) int {
	if v >= min {
		return 10
	}
	if partial > 0 && v >= partial {
		return 5
	}
	return 0
}

func h2Weight(h2 int) int {
	if h2 >= 2 && h2 <= 4 {
		return 10
	}
	if h2 == 1 || h2 == 5 || h2 == 6 {
		return 5
	}
	return 0
}

func boolWeight(b bool) int {
	if b {
		return 10
	}
	return 0
}
