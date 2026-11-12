//go:build integration

package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sarmalCalistir: fn'i ayrı goroutine'de başlatır, ilk koşunun bitmesini
// bekler, sonra iptal edip dönüşünü doğrular.
func sarmalCalistir(t *testing.T, ad string, pool *pgxpool.Pool, every time.Duration,
	fn func(context.Context, *pgxpool.Pool, *slog.Logger, time.Duration)) {
	t.Helper()
	ctx, iptal := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		fn(ctx, pool, testLogger(), every)
		close(done)
	}()
	// İlk koşu senkron başlar; küçük bir pencere bırakıp iptal ediyoruz.
	time.Sleep(150 * time.Millisecond)
	iptal()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("%s: iptalden sonra dönmedi (sızıntı)", ad)
	}
}

func TestRunSarmallariTemizCikar(t *testing.T) {
	pool := geciciPool(t)
	t.Setenv("SEARCH_SVC_URL", "")

	kasalar := []struct {
		ad    string
		every time.Duration
		fn    func(context.Context, *pgxpool.Pool, *slog.Logger, time.Duration)
	}{
		{"RunGSC", time.Hour, RunGSC},
		{"RunTrends", time.Hour, RunTrends},
		{"RunIcerikSkor", time.Hour, RunIcerikSkor},
		{"RunTrendScore", time.Hour, RunTrendScore},
		{"RunAISource", time.Hour, RunAISource},
	}
	for _, k := range kasalar {
		t.Run(k.ad, func(t *testing.T) { sarmalCalistir(t, k.ad, pool, k.every, k.fn) })
	}
}

// every<=0 varsayılana düşmeli (sıfır aralık ticker paniği üretmemeli).
func TestRunSarmallariSifirAralik(t *testing.T) {
	pool := geciciPool(t)
	t.Setenv("SEARCH_SVC_URL", "")
	sarmalCalistir(t, "RunIcerikSkor/0", pool, 0, RunIcerikSkor)
}

// RunPodcast: imzası farklı (aralık yok). İlk build dünkü özeti üretmeye
// çalışır; espeak/ffmpeg yoksa uyarı loglar ve döngüye girer.
func TestRunPodcastIptalIleDoner(t *testing.T) {
	pool := geciciPool(t)
	ctx, iptal := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunPodcast(ctx, pool, testLogger())
		close(done)
	}()
	time.Sleep(300 * time.Millisecond)
	iptal()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("RunPodcast iptalden sonra dönmedi")
	}
}

// --- tryGSC: kimlik yokken erken dönüş, dış istek yok ---

func TestTryGSCKimlikYokkenDoner(t *testing.T) {
	pool := geciciPool(t)
	tryGSC(context.Background(), pool, testLogger())
}

// GSC "son eşitleme" 20 saatten yeniyse hiç çalışmaz.
func TestTryGSCYakinEslestirmedeAtlar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	raw, _ := json.Marshal(time.Now().Format(time.RFC3339))
	if _, err := pool.Exec(ctx, `
		INSERT INTO settings (section, key, value) VALUES ('trend','gsc_last_sync',$1::jsonb)
		ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, string(raw)); err != nil {
		t.Skipf("ayar yazılamadı: %v", err)
	}
	tryGSC(ctx, pool, testLogger())
}

// --- SyncGSC / SyncGA4: yapılandırma eksikse açıklayıcı hata ---

func TestSyncGSCPropertyYokHata(t *testing.T) {
	pool := geciciPool(t)
	_, err := SyncGSC(context.Background(), pool, testLogger(), 1)
	if err == nil {
		t.Fatal("gsc_property yokken hata bekleniyordu")
	}
}

func TestSyncGA4KimlikYokHata(t *testing.T) {
	pool := geciciPool(t)
	_, err := SyncGA4(context.Background(), pool, testLogger())
	if err == nil {
		t.Fatal("kimlik yokken hata bekleniyordu")
	}
}
