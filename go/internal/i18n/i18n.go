// Package i18n çok dilli (24 dil) yerelleştirme: yerel katalog + arayüz çevirileri.
package i18n

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Locale bir yerelleştirmeyi temsil eder.
type Locale struct {
	Code       string
	NativeName string
	RTL        bool
	Enabled    bool
	SortOrder  int
}

// Bundle uygulama başında DB'den yüklenen çeviri paketidir.
type Bundle struct {
	msgs    map[string]string // key: "locale\x00key"
	locales map[string]Locale
	order   []Locale
}

// Load yerelleştirmeleri ve çevirileri hafızaya yükler.
func Load(ctx context.Context, pool *pgxpool.Pool) (*Bundle, error) {
	b := &Bundle{
		msgs:    map[string]string{},
		locales: map[string]Locale{},
	}

	rows, err := pool.Query(ctx,
		`SELECT code, native_name, rtl, enabled, sort_order FROM locales ORDER BY sort_order, code`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l Locale
		if err := rows.Scan(&l.Code, &l.NativeName, &l.RTL, &l.Enabled, &l.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		b.locales[l.Code] = l
		if l.Enabled {
			b.order = append(b.order, l)
		}
	}
	rows.Close()

	trows, err := pool.Query(ctx, `SELECT locale, key, value FROM translations`)
	if err != nil {
		return nil, err
	}
	for trows.Next() {
		var loc, key, val string
		if err := trows.Scan(&loc, &key, &val); err != nil {
			trows.Close()
			return nil, err
		}
		b.msgs[loc+"\x00"+key] = val
	}
	trows.Close()

	sort.SliceStable(b.order, func(i, j int) bool { return b.order[i].SortOrder < b.order[j].SortOrder })
	return b, nil
}

// T verilen yerel için anahtarı döndürür; yoksa tr'ye, o da yoksa anahtara düşer.
func (b *Bundle) T(locale, key string) string {
	if v, ok := b.msgs[locale+"\x00"+key]; ok && v != "" {
		return v
	}
	if v, ok := b.msgs["tr\x00"+key]; ok && v != "" {
		return v
	}
	return key
}

// RTL verilen yerel için sağdan-sola düzen gerekip gerekmediğini söyler.
func (b *Bundle) RTL(locale string) bool {
	l, ok := b.locales[locale]
	return ok && l.RTL
}

// Enabled etkin yerelleri (dil menüsü için).
func (b *Bundle) Enabled() []Locale {
	out := make([]Locale, len(b.order))
	copy(out, b.order)
	return out
}

// IsSupported verilen kodun etkin bir yerel olup olmadığını söyler.
func (b *Bundle) IsSupported(code string) bool {
	l, ok := b.locales[code]
	return ok && l.Enabled
}

// Default verilen kodları sırayla deneyip geçerli ilkini, yoksa "tr" döndürür.
func (b *Bundle) Default(preferred ...string) string {
	for _, c := range preferred {
		if b.IsSupported(c) {
			return c
		}
	}
	return "tr"
}

// DetectLocale URL ilk segmentinden (ve çereze düşen default'a) göre yerel seçer.
// path ör: "/en/kategori/spor" → "en". "/kategori/spor" → "" (default).
func (b *Bundle) DetectLocale(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if i := strings.Index(trimmed, "/"); i >= 0 {
		trimmed = trimmed[:i]
	} else if trimmed != "" {
		// tek segment
	} else {
		return ""
	}
	if b.IsSupported(trimmed) {
		return trimmed
	}
	return ""
}

// StripLocale bilinen yerel önekini path'ten ayırır. dönen: (locale, kalanPath)
func (b *Bundle) StripLocale(path string) (string, string) {
	loc := b.DetectLocale(path)
	if loc == "" {
		return "", path
	}
	rest := strings.TrimPrefix(path, "/"+loc)
	if rest == "" {
		rest = "/"
	}
	return loc, rest
}
