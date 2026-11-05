// Package db migration çalıştırıcısı + (ileride) veritabanı yardımcıları.
// Migration SAHİBİ Go'dur (spec 03 §3): şema go/migrations/*.sql, sürüm tablosu schema_migrations.
package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"theblock/site/migrations"
)

// Migration tek bir sürümlü göçü temsil eder.
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
	HasDown bool
}

// LoadMigrations embedded *.sql dosyalarından sıralı migration listesi üretir.
// Dosya adı kalıbı: {sürüm}_{ad}.{up|down}.sql
func LoadMigrations() ([]Migration, error) { return loadFrom(migrations.FS) }

// loadFrom aynı işi verilen dosya sisteminde yapar.
//
// Ayrıldı çünkü hata yolları (bozuk ad, bilinmeyen yön, okunamayan dosya)
// embed edilmiş gerçek dizinle tetiklenemez. `fstest.MapFS` ile her kural
// ayrı ayrı sınanabiliyor (Faz 208).
func loadFrom(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("migrations okunamadı: %w", err)
	}

	byVersion := map[int]*Migration{}
	var order []int

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		base := strings.TrimSuffix(name, ".sql") // 0001_p0_core.up
		idx := strings.LastIndex(base, ".")
		if idx < 0 {
			return nil, fmt.Errorf("geçersiz migration adı (up/down yok): %s", name)
		}
		dir := base[idx+1:] // up | down
		stem := base[:idx]  // 0001_p0_core
		sep := strings.Index(stem, "_")
		if sep <= 0 {
			return nil, fmt.Errorf("geçersiz migration adı (sürüm yok): %s", name)
		}
		v, err := strconv.Atoi(stem[:sep])
		if err != nil {
			return nil, fmt.Errorf("geçersiz sürüm: %s: %w", name, err)
		}

		m, ok := byVersion[v]
		if !ok {
			m = &Migration{Version: v, Name: stem[sep+1:]}
			byVersion[v] = m
			order = append(order, v)
		}

		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("dosya okunamadı: %s: %w", name, err)
		}
		switch dir {
		case "up":
			m.Up = string(b)
		case "down":
			m.Down = string(b)
			m.HasDown = true
		default:
			return nil, fmt.Errorf("bilinmeyen yön: %s", name)
		}
	}

	sort.Ints(order)
	out := make([]Migration, 0, len(order))
	for _, v := range order {
		out = append(out, *byVersion[v])
	}
	return out, nil
}

func ensureSchemaTable(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INT PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	return err
}

// Applied sürümleri döndürür (var olmayan tabloya karşı dayanıklı).
func Applied(ctx context.Context, conn *pgx.Conn) (map[int]struct{}, error) {
	applied := map[int]struct{}{}
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return applied, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = struct{}{}
	}
	return applied, rows.Err()
}

// Up bekleyen migration'ları sırayla uygular; her biri kendi işleminde.
func Up(ctx context.Context, conn *pgx.Conn) ([]Migration, error) {
	all, err := LoadMigrations()
	if err != nil {
		return nil, err
	}
	applied, err := Applied(ctx, conn)
	if err != nil {
		return nil, err
	}

	var done []Migration
	for _, m := range all {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		if m.Up == "" {
			return nil, fmt.Errorf("%d_%s için up dosyası yok", m.Version, m.Name)
		}
		if err := apply(ctx, conn, m.Version, m.Name, m.Up); err != nil {
			return nil, fmt.Errorf("migration %d_%s BAŞARISIZ: %w", m.Version, m.Name, err)
		}
		done = append(done, m)
	}
	return done, nil
}

// Down en sondaki migration'ı (all=true ise tümünü) geri alır.
func Down(ctx context.Context, conn *pgx.Conn, all bool) ([]Migration, error) {
	ls, err := LoadMigrations()
	if err != nil {
		return nil, err
	}
	byVersion := map[int]Migration{}
	for _, m := range ls {
		byVersion[m.Version] = m
	}
	applied, err := Applied(ctx, conn)
	if err != nil {
		return nil, err
	}

	versions := make([]int, 0, len(applied))
	for v := range applied {
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(versions)))
	if len(versions) == 0 {
		return nil, nil
	}
	if !all {
		versions = versions[:1]
	}

	var rolled []Migration
	for _, v := range versions {
		m, ok := byVersion[v]
		if !ok {
			return nil, fmt.Errorf("sürüm %d için migration bulunamadı", v)
		}
		if !m.HasDown {
			return nil, fmt.Errorf("%d_%s için down dosyası yok — elle çözülmeli", m.Version, m.Name)
		}
		if err := rollback(ctx, conn, m); err != nil {
			return nil, fmt.Errorf("migration %d_%s GERİ ALINAMADI: %w", m.Version, m.Name, err)
		}
		rolled = append(rolled, m)
	}
	return rolled, nil
}

func apply(ctx context.Context, conn *pgx.Conn, version int, name, sql string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := ensureSchemaTable(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, version, name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func rollback(ctx context.Context, conn *pgx.Conn, m Migration) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.Down); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, m.Version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
