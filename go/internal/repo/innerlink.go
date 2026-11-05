package repo

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- P5-14: İç link önerisi + evergreen güncelleme (spec 03 P5-B) ----

// LinkSug bir iç link önerisi (hedef içerik + gerekçe + puan).
type LinkSug struct {
	ID       int64
	Title    string
	Slug     string
	Locale   string
	Category string
	Score    int
	Reasons  string
}

// EvergreenRow güncellenmesi önerilen (bayat) içerik.
type EvergreenRow struct {
	Content
	YasGun int
}

// LinkSuggestions hedef içerik için site içi link önerileri üretir.
// Puanlama: aynı konu kümesi (+6), ortak etiket (+3/her, en çok 3),
// aynı kategori (+2), başlık kelime örtüşmesi (+1/kelime, en çok 4).
func LinkSuggestions(ctx context.Context, pool *pgxpool.Pool, siteID, contentID int64, limit int) ([]LinkSug, error) {
	if limit <= 0 {
		limit = 8
	}
	tc, err := GetContent(ctx, pool, contentID)
	if err != nil {
		return nil, err
	}
	tgtCluster, _ := ContentClusterID(ctx, pool, contentID)
	tgtTags, _ := TagsOf(ctx, pool, contentID)
	tgtCat := int64(0)
	if tc.CategoryID != nil {
		tgtCat = *tc.CategoryID
	}
	tgtTokens := titleTokens(tc.Title)

	rows, err := pool.Query(ctx, `
		SELECT c.id, c.title, COALESCE(c.slug,''), c.locale, COALESCE(cat.name,''), COALESCE(cat.id,0),
		       COALESCE(cc.cluster_id,0),
		       COALESCE(array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL), '{}')
		FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN content_clusters cc ON cc.content_id = c.id
		LEFT JOIN content_tags ct ON ct.content_id = c.id
		LEFT JOIN tags t ON t.id = ct.tag_id
		WHERE c.site_id=$1 AND c.status='yayinda' AND c.id<>$2
		GROUP BY c.id, c.title, c.slug, c.locale, cat.name, cat.id, cc.cluster_id
		ORDER BY c.updated_at DESC
		LIMIT 400`, siteID, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LinkSug
	for rows.Next() {
		var (
			id      int64
			title   string
			slug    string
			locale  string
			catName string
			catID   int64
			clID    int64
			tags    []string
		)
		if err := rows.Scan(&id, &title, &slug, &locale, &catName, &catID, &clID, &tags); err != nil {
			return nil, err
		}
		score := 0
		var reasons []string
		if clID != 0 && clID == tgtCluster {
			score += 6
			reasons = append(reasons, "aynı küme")
		}
		if shared := intersectTags(tags, tgtTags); len(shared) > 0 {
			n := len(shared)
			if n > 3 {
				n = 3
			}
			score += n * 3
			reasons = append(reasons, "ortak etiket: "+strings.Join(shared, ", "))
		}
		if catID != 0 && catID == tgtCat {
			score += 2
			reasons = append(reasons, "aynı kategori")
		}
		if ov := tokenOverlap(tgtTokens, titleTokens(title)); ov > 0 {
			if ov > 4 {
				ov = 4
			}
			score += ov
			reasons = append(reasons, "başlık benzerliği")
		}
		if score <= 0 {
			continue
		}
		out = append(out, LinkSug{ID: id, Title: title, Slug: slug, Locale: locale, Category: catName, Score: score, Reasons: strings.Join(reasons, " · ")})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// EvergreenCandidates son N günden beri güncellenmeyen yayındaki içerikleri döndürür (en bayat önce).
func EvergreenCandidates(ctx context.Context, pool *pgxpool.Pool, siteID int64, staleDays, limit int) ([]EvergreenRow, error) {
	if staleDays <= 0 {
		staleDays = 180
	}
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT ` + contentCols + ` FROM contents c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.author_id
		LEFT JOIN mahrets m ON m.id = c.mahrep_id
		WHERE c.site_id=$1 AND c.status='yayinda' AND c.updated_at < now() - make_interval(days => $2)
		ORDER BY c.updated_at ASC
		LIMIT $3`
	rows, err := pool.Query(ctx, q, siteID, staleDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EvergreenRow
	now := time.Now()
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, EvergreenRow{Content: c, YasGun: int(now.Sub(c.UpdatedAt).Hours() / 24)})
	}
	return out, rows.Err()
}

// TouchContentUpdate içeriği "güncellendi" olarak işaretler (updated_at + sürüm).
func TouchContentUpdate(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `UPDATE contents SET updated_at=now(), version=version+1 WHERE id=$1`, id)
	return err
}

// ---- yardımcılar ----

var linkStopWords = map[string]bool{
	"ve": true, "ile": true, "için": true, "bir": true, "bu": true, "da": true, "de": true,
	"the": true, "and": true, "for": true, "with": true, "that": true, "son": true, "yeni": true,
}

func titleTokens(s string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(foldTR(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len([]rune(w)) >= 4 && !linkStopWords[w] {
			out = append(out, w)
		}
	}
	return out
}

func tokenOverlap(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, w := range a {
		set[w] = true
	}
	n := 0
	seen := map[string]bool{}
	for _, w := range b {
		if set[w] && !seen[w] {
			n++
			seen[w] = true
		}
	}
	return n
}

func intersectTags(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := map[string]bool{}
	for _, x := range a {
		set[strings.ToLower(strings.TrimSpace(x))] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, y := range b {
		k := strings.ToLower(strings.TrimSpace(y))
		if set[k] && !seen[k] {
			out = append(out, y)
			seen[k] = true
		}
	}
	return out
}

// foldTR services ile aynı katlamayı kullanır (clusters için kopya tampon).
func foldTR(s string) string {
	r := strings.NewReplacer(
		"I", "i", "İ", "i", "ı", "i", "Ş", "s", "ş", "s",
		"Ğ", "g", "ğ", "g", "Ü", "u", "ü", "u", "Ö", "o", "ö", "o",
		"Ç", "c", "ç", "c",
	)
	return strings.ToLower(r.Replace(s))
}
