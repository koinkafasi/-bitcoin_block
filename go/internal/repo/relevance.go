package repo

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Alaka (relevance) motoru ----
//
// "İlginizi Çekebilir" ve "İlgili Haberler" listeleri tek bir seed içeriğe
// göre puanlanır. Puan bileşenleri bilinçli olarak ağırlıklandırılır ve
// test edilebilir saf fonksiyonda hesaplanır:
//
//	etiket örtüşmesi   2.5 * paylaşılan etiket (en fazla 5)
//	başlık örtüşmesi   2.0 * Jaccard benzerliği
//	kategori           3.0 aynı kategori
//	popülerlik         0.8 * ln(1+okuma)
//	tazelik            1.6 * exp(-gün/4)
//	yazar               0.4 aynı yazar
//	görsel tazelik      0.8 son 24 saat
//
// Etiketler henüz doldurulmadıysa bile başlık + kategori + popülerlik
// bileşenleri anlamlı sonuç verir.

// Alaka bileşen ağırlıkları.
const (
	wCategory = 3.0
	wTag      = 2.5
	wTitle    = 2.0
	wPopular  = 0.8
	wFresh    = 1.6
	wAuthor   = 0.4
	wHot      = 0.8

	maxTagBonus = 5
	tagWindow   = 14 // popülerlik penceresi (gün)
)

// trStopWords başlık benzerliğinde anlam taşımayan kelimeler.
var trStopWords = map[string]bool{
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
}

// normalizeWord küçük harfe indirir. Türkçe "İ" harfi ToLower ile
// "i" + birleşen nokta (U+0307) üretir; bu, "LİG" ile "lig" eşleşmesini
// bozduğu için birleşen nokta atılır. "ı" de "i"ye katlanır.
func normalizeWord(w string) string {
	s := strings.ToLower(strings.TrimSpace(w))
	if strings.ContainsRune(s, '\u0307') {
		s = strings.ReplaceAll(s, "\u0307", "")
	}
	return strings.ReplaceAll(s, "ı", "i")
}

// contentWords başlıktan anlamlı kelimeleri çıkarır (Türkçe stop-word filtresi).
func contentWords(title string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		w := normalizeWord(f)
		if len([]rune(w)) < 4 || trStopWords[w] {
			continue
		}
		out[w] = true
	}
	return out
}

// jaccard iki kelime kümesinin kesişim/birleşim oranıdır.
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// sharedTags iki etiket kümesinin kesişim sayısıdır.
func sharedTags(a, b map[int64]bool) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := 0
	for k := range a {
		if b[k] {
			n++
		}
	}
	return n
}

// relevanceScore tek bir aday içerik için alaka puanını ve bileşenlerini üretir.
//
// seedTags/candTags verilmezse etiket bileşeni atlanır (etiketsiz çalışma
// durumunda başlık benzerliği devreye girer).
func relevanceScore(seed, cand Content, seedTags, candTags map[int64]bool, hits int64, now time.Time) (float64, map[string]float64) {
	p := map[string]float64{}

	if seed.CategoryID != nil && cand.CategoryID != nil && *seed.CategoryID == *cand.CategoryID {
		p["kategori"] = wCategory
	}
	if seed.AuthorID == cand.AuthorID {
		p["yazar"] = wAuthor
	}
	if seedTags != nil && candTags != nil {
		if n := sharedTags(seedTags, candTags); n > 0 {
			if n > maxTagBonus {
				n = maxTagBonus
			}
			p["etiket"] = wTag * float64(n)
		}
	}
	if j := jaccard(contentWords(seed.Title), contentWords(cand.Title)); j > 0 {
		p["baslik"] = wTitle * j
	}
	if hits > 0 {
		p["populerlik"] = wPopular * math.Log1p(float64(hits))
	}
	if cand.PublishedAt != nil {
		age := now.Sub(cand.PublishedAt.UTC()).Hours() / 24
		if age < 0 {
			age = 0
		}
		p["tazelik"] = wFresh * math.Exp(-age/4)
		if age <= 1 {
			p["sicak"] = wHot
		}
	}

	total := 0.0
	for _, v := range p {
		total += v
	}
	return total, p
}

// SimilarTo yayındaki içerikler arasında seed'e en yakın limit adayı döndürür.
// Havuz son N gün + popülerlikle sınırlanır; adaylar Go tarafında puanlanır.
func SimilarTo(ctx context.Context, pool *pgxpool.Pool, seedID, siteID int64, locale string, limit int) ([]Content, error) {
	if limit <= 0 {
		limit = 6
	}
	seed, err := GetContent(ctx, pool, seedID)
	if err != nil {
		return nil, err
	}
	seedTags, err := tagIDSet(ctx, pool, seedID)
	if err != nil {
		return nil, err
	}

	const poolSize = 400
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id = $1 AND c.status = 'yayinda'
		  AND c.type_key NOT IN ('foto_galeri','video')
		  AND c.id <> $2
		  AND ($3 = '' OR c.locale = $3)
		  AND c.published_at > now() - ($4 || ' days')::interval
		ORDER BY c.published_at DESC NULLS LAST
		LIMIT $5`
	rows, err := pool.Query(ctx, q, siteID, seedID, locale, tagWindow, poolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cands []Content
	ids := make([]int64, 0, poolSize)
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		cands = append(cands, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return []Content{}, nil
	}

	hits := HitsOf(ctx, pool, ids)
	tags := tagIDsOf(ctx, pool, ids)
	now := time.Now()

	type scored struct {
		c Content
		s float64
	}
	ranked := make([]scored, 0, len(cands))
	for _, c := range cands {
		s, _ := relevanceScore(seed, c, seedTags, tags[c.ID], hits[c.ID], now)
		ranked = append(ranked, scored{c, s})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].s != ranked[j].s {
			return ranked[i].s > ranked[j].s
		}
		return ranked[i].c.ID > ranked[j].c.ID
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Content, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.c)
	}
	return out, nil
}

// tagIDSet bir içeriğin etiket kimliklerini döndürür (bulunamazsa nil).
func tagIDSet(ctx context.Context, pool *pgxpool.Pool, contentID int64) (map[int64]bool, error) {
	rows, err := pool.Query(ctx, `SELECT tag_id FROM content_tags WHERE content_id = $1`, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		m[id] = true
	}
	return m, rows.Err()
}

// tagIDsOf birden çok içeriğin etiketlerini tek sorguda döndürür.
func tagIDsOf(ctx context.Context, pool *pgxpool.Pool, ids []int64) map[int64]map[int64]bool {
	out := map[int64]map[int64]bool{}
	if len(ids) == 0 {
		return out
	}
	rows, err := pool.Query(ctx, `SELECT content_id, tag_id FROM content_tags WHERE content_id = ANY($1)`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var cid, tid int64
		if err := rows.Scan(&cid, &tid); err != nil {
			continue
		}
		if out[cid] == nil {
			out[cid] = map[int64]bool{}
		}
		out[cid][tid] = true
	}
	return out
}
