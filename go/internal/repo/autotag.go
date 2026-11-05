package repo

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/platform"
)

// AutoTagFromTitle başlıktan anahtar kelimeler çıkarır ve içerikle ilişkilendirir.
// Alaka motorunun "etiket örtüşmesi" bileşeni bu etiketlerle beslenir.
// Özel adlar (büyük harfle başlayan) öncelikli seçilir.
func AutoTagFromTitle(ctx context.Context, pool *pgxpool.Pool, contentID int64, title string, maxTags int) ([]string, error) {
	if maxTags <= 0 {
		maxTags = 5
	}
	words := titleKeywords(title, maxTags)
	linked := make([]string, 0, len(words))
	for _, w := range words {
		tagID, err := EnsureTag(ctx, pool, w, platform.Slugify(w))
		if err != nil || tagID == 0 {
			continue
		}
		if err := LinkContentTag(ctx, pool, contentID, tagID); err != nil {
			continue
		}
		linked = append(linked, w)
	}
	return linked, nil
}

// titleKeywords başlıktan etiket adaylarını önem sırasına göre üretir.
func titleKeywords(title string, maxTags int) []string {
	type cand struct {
		word  string
		upper bool
		pos   int
	}
	var cands []cand
	for pos, f := range strings.FieldsFunc(title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		raw := strings.TrimSpace(f)
		if raw == "" {
			continue
		}
		upper := unicode.IsUpper([]rune(raw)[0])
		w := normalizeWord(raw)
		if len([]rune(w)) < 4 || trStopWords[w] || strings.Contains(w, "http") {
			continue
		}
		cands = append(cands, cand{word: w, upper: upper, pos: pos})
	}
	// aynı kelimeyi tekrarlama (ilk konum korunur)
	seen := map[string]bool{}
	uniq := cands[:0]
	for _, c := range cands {
		if seen[c.word] {
			continue
		}
		seen[c.word] = true
		uniq = append(uniq, c)
	}
	// özel adlar önce, sonra uzunluk, sonra konum
	sort.SliceStable(uniq, func(i, j int) bool {
		if uniq[i].upper != uniq[j].upper {
			return uniq[i].upper
		}
		if len(uniq[i].word) != len(uniq[j].word) {
			return len(uniq[i].word) > len(uniq[j].word)
		}
		return uniq[i].pos < uniq[j].pos
	})
	if len(uniq) > maxTags {
		uniq = uniq[:maxTags]
	}
	out := make([]string, 0, len(uniq))
	for _, c := range uniq {
		out = append(out, c.word)
	}
	return out
}
