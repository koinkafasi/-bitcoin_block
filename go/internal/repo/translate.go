// Package repo — arayüz çevirileri yönetimi (Faz 45).
package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ListTranslationKeys tüm çeviri anahtarlarını döndürür.
func ListTranslationKeys(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT key FROM translations ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ListTranslations bir dilin anahtar→değer eşlemesini döndürür.
func ListTranslations(ctx context.Context, pool *pgxpool.Pool, locale string) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT key, value FROM translations WHERE locale=$1`, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// UpsertTranslation bir dil+anahtar için çeviriyi yazar/güncelller.
func UpsertTranslation(ctx context.Context, pool *pgxpool.Pool, locale, key, value string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO translations (locale, key, value) VALUES ($1,$2,$3)
		ON CONFLICT (locale, key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`,
		locale, key, value)
	return err
}

// EnsureTranslationKey yeni anahtarı tüm etkin dillere boş değerle ekler.
func EnsureTranslationKey(ctx context.Context, pool *pgxpool.Pool, key string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO translations (locale, key, value)
		SELECT code, $1, '' FROM locales WHERE enabled = true
		ON CONFLICT (locale, key) DO NOTHING`, key)
	return err
}

// DeleteTranslationKey anahtarı tüm dillerden siler.
func DeleteTranslationKey(ctx context.Context, pool *pgxpool.Pool, key string) error {
	_, err := pool.Exec(ctx, `DELETE FROM translations WHERE key=$1`, key)
	return err
}
