package repo

import (
	"context"
	"strings"
)

// PageText, Sayfa Tasarımcısı'ndan düzenlenebilen şablon metninin bir
// override kaydıdır (Faz 142). Kayıt yoksa koddaki varsayılan kullanılır.
type PageText struct {
	PageKey string
	TextKey string
	Locale  string
	Value   string
}

func ListPageTexts(ctx context.Context, pool DB, siteID int64, pageKey string) ([]PageText, error) {
	rows, err := pool.Query(ctx, `
		SELECT page_key, text_key, locale, value FROM page_texts
		WHERE site_id=$1 AND page_key=$2`, siteID, pageKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PageText
	for rows.Next() {
		var pt PageText
		if err := rows.Scan(&pt.PageKey, &pt.TextKey, &pt.Locale, &pt.Value); err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, rows.Err()
}

// SavePageText metin override'ını yazar/günceller. Boş değer "override yok"
// demektir; kayıt silinir ki varsayılana dönülsün.
func SavePageText(ctx context.Context, pool DB, siteID int64, pageKey, textKey, locale, value string, updatedBy int64) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return DeletePageText(ctx, pool, siteID, pageKey, textKey, locale)
	}
	if locale == "" {
		locale = "tr"
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO page_texts (site_id, page_key, text_key, locale, value, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (site_id, page_key, text_key, locale)
		DO UPDATE SET value=EXCLUDED.value, updated_by=EXCLUDED.updated_by, updated_at=now()`,
		siteID, pageKey, textKey, locale, value, int64OrNil(updatedBy))
	return err
}

// DeletePageText override'ı kaldırır; metin yeniden kod varsayılanına döner.
func DeletePageText(ctx context.Context, pool DB, siteID int64, pageKey, textKey, locale string) error {
	if locale == "" {
		locale = "tr"
	}
	_, err := pool.Exec(ctx, `
		DELETE FROM page_texts
		WHERE site_id=$1 AND page_key=$2 AND text_key=$3 AND locale=$4`,
		siteID, pageKey, textKey, locale)
	return err
}
