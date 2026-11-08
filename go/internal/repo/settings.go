package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetSetting JSON değerini ham byte olarak döndürür.
func GetSetting(ctx context.Context, pool DB, section, key string) ([]byte, error) {
	var v []byte
	err := pool.QueryRow(ctx,
		`SELECT value FROM settings WHERE section = $1 AND key = $2`, section, key,
	).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return settingText(v), err
}

// SetSetting JSON değerini yazar/upsert eder. Verilen baytlar geçerli JSON
// DEĞİLSE otomatik olarak JSON string'e sarılır; aksi halde jsonb yazımı
// sessizce başarısız olurdu (ör. saat "09:41", API anahtarları, kaynak
// metinleri kaydedilemiyordu).
func SetSetting(ctx context.Context, pool DB, section, key string, value []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO settings (section, key, value) VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (section, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		section, key, settingJSON(value))
	return err
}

// settingJSON değeri jsonb'ye yazılabilir biçime getirir: geçerli JSON ise
// olduğu gibi, değilse JSON string olarak sarar.
func settingJSON(value []byte) []byte {
	if json.Valid(value) {
		return value
	}
	b, err := json.Marshal(string(value))
	if err != nil {
		return []byte(`""`)
	}
	return b
}

// settingText GetSetting okumasını sarmalayıcıdan kurtarır: saklanan değer
// JSON string ise ham metni, değilse olduğu gibi döner.
func settingText(raw []byte) []byte {
	t := bytes.TrimSpace(raw)
	if len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' {
		var s string
		if err := json.Unmarshal(t, &s); err == nil {
			return []byte(s)
		}
	}
	return raw
}

// Setting ayar kaydıdır (JSON ham değer).
type Setting struct {
	Section string
	Key     string
	Value   []byte
}

// GetSettings bir bölümün tüm anahtarlarını döndürür.
func GetSettings(ctx context.Context, pool DB, section string) (map[string][]byte, error) {
	rows, err := pool.Query(ctx, `SELECT key, value FROM settings WHERE section = $1`, section)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var k string
		var v []byte
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// GetSettingsText bir bölümün tüm anahtarlarını, saklanan JSON sarmalı
// çözülmüş METİN olarak döndürür.
//
// NEDEN: GetSettings ham jsonb değerini verir (kaydederken json.Marshal ile
// sarmalandığı için `"bitcosar"` gibi tırnaklı gelir). Tek tek GetSetting
// çağıran kod settingText ile bunu çözüyordu; tek sorguya geçen çağıranlar
// aynı çözümü tekrar yazmasın diye burada yapılır.
func GetSettingsText(ctx context.Context, pool DB, section string) (map[string]string, error) {
	raw, err := GetSettings(ctx, pool, section)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = string(settingText(v))
	}
	return out, nil
}

// ---- audit log ----

// LogRow panel işlem kaydıdır.
type LogRow struct {
	ID        int64
	UserName  string
	Action    string
	Detail    *string
	CreatedAt time.Time
}

// AddAuditLog panel işlemini kaydeder.
func AddAuditLog(ctx context.Context, pool *pgxpool.Pool, userID *int64, action, detail string) error {
	_, err := pool.Exec(ctx, `INSERT INTO logs (user_id, action, detail) VALUES ($1,$2,$3)`,
		nullableInt64P(userID), action, nullableText(&detail))
	return err
}

// ListLogs son işlemleri döndürür.
func ListLogs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]LogRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT l.id, COALESCE(u.email,'system'), l.action, l.detail, l.created_at
		FROM logs l LEFT JOIN users u ON u.id = l.user_id
		ORDER BY l.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LogRow
	for rows.Next() {
		var r LogRow
		if err := rows.Scan(&r.ID, &r.UserName, &r.Action, &r.Detail, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClearLogs işlem kayıtlarını temizler.
func ClearLogs(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DELETE FROM logs`)
	return err
}

// DefaultLanguage varsayılan yayın dilini döndürür (genel/dil; yoksa "tr").
func DefaultLanguage(ctx context.Context, pool *pgxpool.Pool) string {
	b, err := GetSetting(ctx, pool, "genel", "dil")
	if err != nil {
		return "tr"
	}
	s := string(b)
	s = trimJSONString(s)
	if s == "" {
		return "tr"
	}
	return s
}

func trimJSONString(s string) string {
	// en dıştaki tırnakları temizle
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
