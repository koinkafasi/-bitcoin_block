//go:build integration

package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/db"
)

func geciciPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL yok — scheduler integration testi atlandı")
	}
	ctx := context.Background()

	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rastgele ad: %v", err)
	}
	ad := "tb_sch_" + hex.EncodeToString(b[:])

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("yönetici bağlantısı: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+ad); err != nil {
		admin.Close(ctx)
		t.Fatalf("geçici veritabanı: %v", err)
	}
	admin.Close(ctx)

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + ad

	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), base)
		if err != nil {
			return
		}
		defer c.Close(context.Background())
		_, _ = c.Exec(context.Background(),
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", ad)
		_, _ = c.Exec(context.Background(), "DROP DATABASE IF EXISTS "+ad)
	})

	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatalf("havuz: %v", err)
	}
	t.Cleanup(pool.Close)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("bağlantı: %v", err)
	}
	defer conn.Release()
	if _, err := db.Up(ctx, conn.Conn()); err != nil {
		t.Fatalf("migration: %v", err)
	}
	return pool
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&yazici{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testUser zorunlu yabancı anahtarlar için tek kullanıcı üretir.
func testUser(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (name, email, password_hash, role, status)
		VALUES ('Sched Test', $1, 'x', 'Yönetici', 'aktif') RETURNING id`,
		"sched-"+strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")+"@test.local").Scan(&id)
	if err != nil {
		t.Fatalf("kullanıcı: %v", err)
	}
	return id
}

// testicerik içerik ekler; tip anahtarı seed'den gelir.
func testicerik(t *testing.T, pool *pgxpool.Pool, author int64, durum string, publishAt *time.Time, updatedAt time.Time) int64 {
	t.Helper()
	var tip string
	if err := pool.QueryRow(context.Background(), `SELECT key FROM content_types LIMIT 1`).Scan(&tip); err != nil {
		t.Fatalf("content_types boş: %v", err)
	}
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO contents (type_key, status, title, author_id, publish_at, updated_at, version)
		VALUES ($1, $2, 'Zamanlanmış test', $3, $4, $5, 1)
		RETURNING id`, tip, durum, author, publishAt, updatedAt).Scan(&id)
	if err != nil {
		t.Fatalf("içerik: %v", err)
	}
	return id
}

func durumOku(t *testing.T, pool *pgxpool.Pool, id int64) (string, *time.Time, int) {
	t.Helper()
	var d string
	var pub *time.Time
	var v int
	if err := pool.QueryRow(context.Background(),
		`SELECT status, published_at, version FROM contents WHERE id=$1`, id).Scan(&d, &pub, &v); err != nil {
		t.Fatalf("okuma: %v", err)
	}
	return d, pub, v
}

func varMi(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) bool {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("sayım: %v", err)
	}
	return n > 0
}

// Zamanı gelmiş planlı içerik yayına geçer; gelecekteki ve plansız olan
// dokunulmadan kalır. Version artar, published_at doldurulur.
func TestTickZamanlanmisYayin(t *testing.T) {
	t.Setenv("SEARCH_SVC_URL", "")
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	gecmis := time.Now().Add(-time.Minute)
	gelecek := time.Now().Add(time.Hour)
	hazir := testicerik(t, pool, yazar, "planlanmis", &gecmis, time.Now())
	bekleyen := testicerik(t, pool, yazar, "planlanmis", &gelecek, time.Now())
	plansiz := testicerik(t, pool, yazar, "planlanmis", nil, time.Now())

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	d, pub, v := durumOku(t, pool, hazir)
	if d != "yayinda" {
		t.Errorf("zamanı gelen içerik %q; yayinda bekleniyordu", d)
	}
	if pub == nil {
		t.Error("published_at doldurulmalı")
	}
	if v != 2 {
		t.Errorf("version = %d; 2 bekleniyordu", v)
	}

	if d, _, _ := durumOku(t, pool, bekleyen); d != "planlanmis" {
		t.Errorf("gelecekteki içerik %q; planlanmis kalmalı", d)
	}
	if d, _, _ := durumOku(t, pool, plansiz); d != "planlanmis" {
		t.Errorf("publish_at NULL içerik %q; planlanmis kalmalı", d)
	}
}

// İkinci tick aynı içeriği ikinci kez yayına almaz (version iki kez artmaz).
func TestTickIdempotentYayin(t *testing.T) {
	t.Setenv("SEARCH_SVC_URL", "")
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)
	gecmis := time.Now().Add(-time.Minute)
	id := testicerik(t, pool, yazar, "planlanmis", &gecmis, time.Now())

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("1. tick: %v", err)
	}
	_, _, v1 := durumOku(t, pool, id)
	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("2. tick: %v", err)
	}
	_, _, v2 := durumOku(t, pool, id)
	if v1 != v2 {
		t.Fatalf("version %d → %d; ikinci tick değiştirmemeli", v1, v2)
	}
}

// Çöp 30 günü geçmişse silinir; taze çöp ve diğer durumlar kalır.
func TestTickCopTemizligi(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	eski := time.Now().AddDate(0, 0, -31)
	taze := time.Now().AddDate(0, 0, -1)
	eskiCop := testicerik(t, pool, yazar, "cop", nil, eski)
	tazeCop := testicerik(t, pool, yazar, "cop", nil, taze)
	eskiYayin := testicerik(t, pool, yazar, "yayinda", nil, eski)

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if varMi(t, pool, `SELECT count(*) FROM contents WHERE id=$1`, eskiCop) {
		t.Error("31 günlük çöp silinmeliydi")
	}
	if !varMi(t, pool, `SELECT count(*) FROM contents WHERE id=$1`, tazeCop) {
		t.Error("taze çöp korunmalıydı")
	}
	if !varMi(t, pool, `SELECT count(*) FROM contents WHERE id=$1`, eskiYayin) {
		t.Error("eski ama yayında olan içerik silinmemeli")
	}
}

// Süresi geçmiş oturum silinir; geçerli oturum kalır.
func TestTickBayatOturumTemizligi(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	var bayat, gecerli string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, expires_at) VALUES ($1, now() - interval '1 day')
		RETURNING id::text`, yazar).Scan(&bayat); err != nil {
		t.Fatalf("bayat oturum: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, expires_at) VALUES ($1, now() + interval '1 day')
		RETURNING id::text`, yazar).Scan(&gecerli); err != nil {
		t.Fatalf("geçerli oturum: %v", err)
	}

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if varMi(t, pool, `SELECT count(*) FROM sessions WHERE id=$1`, bayat) {
		t.Error("süresi geçmiş oturum silinmeliydi")
	}
	if !varMi(t, pool, `SELECT count(*) FROM sessions WHERE id=$1`, gecerli) {
		t.Error("geçerli oturum korunmalıydı")
	}
}

// Şifre sıfırlama kayıtları 7 günden eskiyse temizlenir.
func TestTickSifreSifirlamaTemizligi(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	ekle := func(token string, expires time.Time) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO password_resets (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)`, yazar, token, expires); err != nil {
			t.Fatalf("password_resets: %v", err)
		}
	}
	ekle("eski-token", time.Now().AddDate(0, 0, -10))
	ekle("yeni-token", time.Now().Add(time.Hour))

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if varMi(t, pool, `SELECT count(*) FROM password_resets WHERE token_hash='eski-token'`) {
		t.Error("10 gün önce süresi dolan kayıt silinmeliydi")
	}
	if !varMi(t, pool, `SELECT count(*) FROM password_resets WHERE token_hash='yeni-token'`) {
		t.Error("geçerli kayıt korunmalıydı")
	}
}

// Hizmet önbelleğinde 7 günden eski satırlar silinir.
func TestTickServisOnbellegiTemizligi(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()

	var kolonlar string
	if err := pool.QueryRow(ctx, `
		SELECT string_agg(column_name, ',') FROM information_schema.columns
		WHERE table_name='services_cache'`).Scan(&kolonlar); err != nil || kolonlar == "" {
		t.Skipf("services_cache tablosu yok (%v) — bu kontrol atlandı", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO services_cache (key, value, updated_at)
		VALUES ('eski|ana', '{}'::jsonb, now() - interval '30 days'),
		       ('yeni|ana', '{}'::jsonb, now())`); err != nil {
		t.Skipf("services_cache şeması beklenenden farklı: %v", err)
	}

	if err := tick(ctx, pool, testLogger()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if varMi(t, pool, `SELECT count(*) FROM services_cache WHERE key='eski|ana'`) {
		t.Error("30 günlük önbellek satırı silinmeliydi")
	}
	if !varMi(t, pool, `SELECT count(*) FROM services_cache WHERE key='yeni|ana'`) {
		t.Error("taze önbellek satırı korunmalıydı")
	}
}
