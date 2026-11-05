package repo

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/quality"
)

// Bot hattı yakın-kopya denetimi.
//
// Sorun: bot yalnızca `crawled_items.url_hash` ile mükerrer denetimi
// yapıyordu. Aynı haberi MSN, Google News, Cumhuriyet ve Habertürk farklı
// URL'lerle yayınladığı için url bazlı denetim geçiyor, aynı metin tekrar
// tekrar taslağa düşüyordu. Ölçüm (28 Ekim 2026): 1.923 başlık grubunda
// 10.442 fazladan kayıt; en çok tekrarlayan haber 122 kopya.
//
// Çözüm iki aşamalı: (1) trigram indeksiyle aday daralt (ucuz, indeksli),
// (2) `quality.CheckDuplicatesAt` ile shingle+Jaccard kararı (kesin). Karar
// mantığı SQL'den ayrı tutulur ki saf fonksiyon olarak test edilebilsin.
//
// Kaynak hiyerarşisi (B3): MSN/Google News gibi agregatörler kendi haberi
// olmayan toplayıcılardır. Onlardan gelen öğe, asıl kaynağın kopyasıysa
// atılır; tersi durumda (agregatör kopyası zaten kayıtlıyken orijinal
// gelirse) orijinal kazanır ve agregatör kopyası arşivlenir.

const (
	// dedupeLookbackDays bot dedupe'unun geriye bakış penceresi (gün).
	// Haber akışında aynı olay günlerce dolaşabilir.
	dedupeLookbackDays = 7
	// dedupeCandidateLimit trigram sorgusunun döndüreceği en fazla aday.
	dedupeCandidateLimit = 12
	// aggregatorThreshold agregatör kaynaklarda benzerlik eşiği. Orijinal
	// kaynaklarda quality.DuplicateThreshold (0.60) kullanılır.
	aggregatorThreshold = 0.45
)

// DedupeAction mükerrer bulunduğunda ne yapılacağıdır.
type DedupeAction string

const (
	// DedupeSkip yeni öğe kaydedilmez (mevcut kopya korunur).
	DedupeSkip DedupeAction = "atla"
	// DedupeReplace mevcut kopya arşivlenir, orijinal kaynak kaydedilir.
	DedupeReplace DedupeAction = "degistir"
)

// DedupeDecision iki kaynağın hiyerarşisine göre kararı verir. Saf fonksiyon.
//
// Kural tek cümle: orijinal agregatörü yener, agregatör orijinali asla yenmez.
func DedupeDecision(incomingAggregator, existingAggregator bool) DedupeAction {
	if existingAggregator && !incomingAggregator {
		return DedupeReplace
	}
	return DedupeSkip
}

// DuplicateOfCandidates yeni öğenin adaylar arasında yakın kopyası olup
// olmadığını söyler (varsayılan eşik). Saf fonksiyondur.
//
// Başlık boşsa karar verilmez (geçirilir): FetchRSS boş başlıklı öğeyi
// zaten atlar, buraya düşerse engellemek yerine geçirmek güvenlidir.
func DuplicateOfCandidates(title, spot string, cands []quality.Candidate) (quality.BenzerlikBulgu, bool) {
	return DuplicateOfCandidatesAt(title, spot, cands, quality.DuplicateThreshold)
}

// DuplicateOfCandidatesAt eşiği çağıranın belirlediği sürümdür.
func DuplicateOfCandidatesAt(title, spot string, cands []quality.Candidate, threshold float64) (quality.BenzerlikBulgu, bool) {
	title = strings.TrimSpace(title)
	if title == "" || len(cands) == 0 {
		return quality.BenzerlikBulgu{}, false
	}
	// GÜÇLÜ KURAL: başlık (normalize) aynıysa spot ne olursa olsun aynı
	// haberdir.
	//
	// ÖLÇÜM (2 Kasım 2026): son 7 günde 194 grup / 416 fazla satır, en çok
	// 6 kopya. Aynı manşetin başlık benzerliği 1.00 ama spot benzerliği
	// 0.51–0.77; birleşik metin eşiğin (0.60) altında kalıyordu. Her kaynak
	// özeti farklı yazdığı için bu, dedupe'u fiilen devre dışı bırakıyordu.
	//
	// Kural burada (saf fonksiyonda) durur: DB yolu ve testler aynı kararı
	// versin; iki yerde ayrı yazılırsa biri sessizce eskir.
	if b, ok := ayniBaslikBulgu(title, cands); ok {
		return b, true
	}
	rapor := quality.CheckDuplicatesAt(dedupeText(title, spot), cands, threshold)
	if rapor.OK() {
		return quality.BenzerlikBulgu{}, false
	}
	return rapor.Matches[0], true
}

// ayniBaslikBulgu: başlığı normalize edilmiş hâliyle eşleşen adayı bulur.
func ayniBaslikBulgu(title string, cands []quality.Candidate) (quality.BenzerlikBulgu, bool) {
	t := normalizeNewsTitle(title)
	if t == "" {
		return quality.BenzerlikBulgu{}, false
	}
	for _, c := range cands {
		if normalizeNewsTitle(c.Title) == t {
			return quality.BenzerlikBulgu{
				Ref:    c.Ref,
				Title:  c.Title,
				Score:  1,
				Detail: "başlık aynı (spot farkı göz ardı edildi)",
			}, true
		}
	}
	return quality.BenzerlikBulgu{}, false
}

// dedupeText karşılaştırma metnini üretir. Başlık ve spot birlikte
// kullanılır: tek başına başlık kısa kalıp shingle üretemiyor, tek başına
// spot ise bazı kaynaklarda boş geliyor.
func dedupeText(title, spot string) string {
	return strings.TrimSpace(title + " " + strings.TrimSpace(spot))
}

// dedupeCandidate karşılaştırılacak mevcut içeriktir.
type dedupeCandidate struct {
	ID         int64
	Title      string
	Spot       string
	Aggregator bool
}

// findDuplicateCandidates trigram indeksini (ix_contents_title_trgm)
// kullanarak başlığa benzeyen son N günün içeriklerini getirir.
//
// `title % $n` operatörü GIN trigram indeksini kullanır; `similarity() >= x`
// kullanmaz. Bu yüzden daraltma indeksli operatörle, kesin karar Go
// tarafında yapılır.
//
// Agregatör bilgisi feed adı üzerinden okunur: içerik oluşturulurken
// source_name = feed adı yazılır. Feed silinmişse false (orijinal) sayılır.
func findDuplicateCandidates(ctx context.Context, pool *pgxpool.Pool, siteID int64, title string, days int) ([]dedupeCandidate, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}
	if days <= 0 {
		days = dedupeLookbackDays
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, COALESCE(c.spot,''), COALESCE(f.aggregator, false)
		FROM contents c
		LEFT JOIN feeds f ON f.name = c.source_name AND f.site_id = c.site_id
		WHERE c.site_id = $1
		  AND c.created_at >= now() - ($2::int || ' days')::interval
		  AND c.title % $3
		ORDER BY similarity(c.title, $3) DESC
		LIMIT $4`, siteID, days, title, dedupeCandidateLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dedupeCandidate
	for rows.Next() {
		var c dedupeCandidate
		if err := rows.Scan(&c.ID, &c.Title, &c.Spot, &c.Aggregator); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// duplicateDraft yeni öğenin kaydedilmeye değer olup olmadığını söyler.
// Hata durumunda "mükerrer değil" döner: dedupe bir kolaylıktır, veritabanı
// hatası yüzünden içerik akışı durmamalıdır.
//
// Dönüş: (mükerrer mi, ne yapılmalı, çakışan içeriğin id'si).
func duplicateDraft(ctx context.Context, pool *pgxpool.Pool, siteID int64, title, spot string, incomingAggregator bool) (bool, DedupeAction, int64) {
	cands, err := findDuplicateCandidates(ctx, pool, siteID, title, dedupeLookbackDays)
	if err != nil || len(cands) == 0 {
		return false, DedupeSkip, 0
	}
	byRef := make(map[string]dedupeCandidate, len(cands))
	qualityCands := make([]quality.Candidate, 0, len(cands))
	for _, c := range cands {
		ref := strconv.FormatInt(c.ID, 10)
		byRef[ref] = c
		qualityCands = append(qualityCands, quality.Candidate{
			Ref:   ref,
			Title: c.Title,
			Text:  dedupeText(c.Title, c.Spot),
		})
	}
	threshold := quality.DuplicateThreshold
	if incomingAggregator {
		threshold = aggregatorThreshold
	}
	bulgu, dup := DuplicateOfCandidatesAt(title, spot, qualityCands, threshold)
	if !dup {
		return false, DedupeSkip, 0
	}
	hit, ok := byRef[bulgu.Ref]
	if !ok {
		return true, DedupeSkip, 0
	}
	return true, DedupeDecision(incomingAggregator, hit.Aggregator), hit.ID
}
