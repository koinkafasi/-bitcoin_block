//go:build integration

package db_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"theblock/site/internal/db"
)

func geciciDB(t *testing.T) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL yok — migration integration testi atlandı")
	}
	ctx := context.Background()

	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rastgele ad: %v", err)
	}
	ad := "tb_mig_" + hex.EncodeToString(b[:])

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("yönetici bağlantısı: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, "CREATE DATABASE "+ad); err != nil {
		t.Fatalf("geçici veritabanı oluşturulamadı: %v", err)
	}

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL ayrıştırılamadı: %v", err)
	}
	u.Path = "/" + ad

	t.Cleanup(func() {
		// Açık bağlantı kalırsa DROP başarısız olur; bağlantıları zorla kes.
		c, err := pgx.Connect(context.Background(), base)
		if err != nil {
			return
		}
		defer c.Close(context.Background())
		_, _ = c.Exec(context.Background(),
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", ad)
		_, _ = c.Exec(context.Background(), "DROP DATABASE IF EXISTS "+ad)
	})
	return u.String()
}

func bagla(t *testing.T, url string) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("bağlantı: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// Up boş bir veritabanında tüm migration'ları uygular; ikinci çağrı hiçbir
// şey yapmaz (idempotent) — deploy'da her başlangıçta çalışıyor.
func TestUpIdempotent(t *testing.T) {
	conn := bagla(t, geciciDB(t))
	ctx, iptal := context.WithTimeout(context.Background(), 3*time.Minute)
	defer iptal()

	ilk, err := db.Up(ctx, conn)
	if err != nil {
		t.Fatalf("ilk Up: %v", err)
	}
	if len(ilk) == 0 {
		t.Fatal("ilk Up hiç migration uygulamadı")
	}

	ikinci, err := db.Up(ctx, conn)
	if err != nil {
		t.Fatalf("ikinci Up: %v", err)
	}
	if len(ikinci) != 0 {
		t.Fatalf("ikinci Up %d migration uyguladı; 0 bekleniyordu", len(ikinci))
	}

	// Uygulanan sürümler tabloda olmalı.
	applied, err := db.Applied(ctx, conn)
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	hepsi, _ := db.LoadMigrations()
	if len(applied) != len(hepsi) {
		t.Errorf("uygulanan %d, migration sayısı %d", len(applied), len(hepsi))
	}
	for _, m := range hepsi {
		if _, ok := applied[m.Version]; !ok {
			t.Errorf("sürüm %d uygulanmamış", m.Version)
		}
	}
}

// Applied, schema_migrations tablosu YOKKEN hata değil boş küme döner.
func TestAppliedTabloYokkenBos(t *testing.T) {
	conn := bagla(t, geciciDB(t))
	applied, err := db.Applied(context.Background(), conn)
	if err != nil {
		t.Fatalf("tablo yokken hata: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("%d sürüm; 0 bekleniyordu", len(applied))
	}
}

// Down en son sürümü geri alır ve kaydı siler; tekrar Up onu yeniden uygular.
func TestDownSonSurumuGeriAlir(t *testing.T) {
	conn := bagla(t, geciciDB(t))
	ctx, iptal := context.WithTimeout(context.Background(), 3*time.Minute)
	defer iptal()

	if _, err := db.Up(ctx, conn); err != nil {
		t.Fatalf("Up: %v", err)
	}
	hepsi, err := db.LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	son := hepsi[len(hepsi)-1]
	if !son.HasDown {
		t.Skipf("son migration'ın down dosyası yok (%d_%s)", son.Version, son.Name)
	}

	rolled, err := db.Down(ctx, conn, false)
	if err != nil {
		t.Fatalf("Down: %v", err)
	}
	if len(rolled) != 1 || rolled[0].Version != son.Version {
		t.Fatalf("geri alınan = %+v; sürüm %d bekleniyordu", rolled, son.Version)
	}

	applied, err := db.Applied(ctx, conn)
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	if _, ok := applied[son.Version]; ok {
		t.Errorf("sürüm %d kaydı silinmemiş", son.Version)
	}

	// Yeniden uygulanabilmeli.
	yeniden, err := db.Up(ctx, conn)
	if err != nil {
		t.Fatalf("yeniden Up: %v", err)
	}
	if len(yeniden) != 1 || yeniden[0].Version != son.Version {
		t.Fatalf("yeniden uygulanan = %+v", yeniden)
	}
}

// all=true bütün migration'ları ters sırayla geri alır.
func TestDownHepsi(t *testing.T) {
	conn := bagla(t, geciciDB(t))
	ctx, iptal := context.WithTimeout(context.Background(), 5*time.Minute)
	defer iptal()

	hepsi, err := db.LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	for _, m := range hepsi {
		if !m.HasDown {
			t.Skipf("%d_%s için down yok — tümünü geri alma sınanamaz", m.Version, m.Name)
		}
	}
	if _, err := db.Up(ctx, conn); err != nil {
		t.Fatalf("Up: %v", err)
	}
	rolled, err := db.Down(ctx, conn, true)
	if err != nil {
		t.Fatalf("Down(all): %v", err)
	}
	if len(rolled) != len(hepsi) {
		t.Fatalf("%d geri alındı; %d bekleniyordu", len(rolled), len(hepsi))
	}
	for i := 1; i < len(rolled); i++ {
		if rolled[i-1].Version <= rolled[i].Version {
			t.Fatalf("geri alma sırası ters değil: %d sonra %d", rolled[i-1].Version, rolled[i].Version)
		}
	}
	applied, err := db.Applied(ctx, conn)
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("%d sürüm kaldı; 0 bekleniyordu", len(applied))
	}
}

// Boş veritabanında Down hiçbir şey yapmaz, hata vermez.
func TestDownBosVeritabani(t *testing.T) {
	conn := bagla(t, geciciDB(t))
	rolled, err := db.Down(context.Background(), conn, false)
	if err != nil {
		t.Fatalf("boş veritabanında Down: %v", err)
	}
	if len(rolled) != 0 {
		t.Fatalf("%d geri alındı; 0 bekleniyordu", len(rolled))
	}
}
