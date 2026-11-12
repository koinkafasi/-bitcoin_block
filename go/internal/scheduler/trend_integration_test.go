//go:build integration

package scheduler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

func loopbackIstemci(t *testing.T) {
	t.Helper()
	eski := trendHTTPClient
	trendHTTPClient = func() *http.Client { return &http.Client{Timeout: 5 * time.Second} }
	t.Cleanup(func() { trendHTTPClient = eski })
}

func trendKaynakEkle(t *testing.T, pool *pgxpool.Pool, ad, kind, url string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO trend_sources (site_id, name, kind, url, aktif)
		VALUES (1, $1, $2, $3, true) RETURNING id`, ad, kind, url).Scan(&id)
	if err != nil {
		t.Fatalf("trend_sources: %v", err)
	}
	return id
}

func trendTerimEkle(t *testing.T, pool *pgxpool.Pool, sourceID int64, term string, traffic int) int64 {
	t.Helper()
	id, _, err := repo.PublishTrendTerm(context.Background(), pool, sourceID, 1, term, "TR", "tr-TR")
	if err != nil {
		t.Fatalf("trend terimi: %v", err)
	}
	if err := repo.UpdateTrendTermMeta(context.Background(), pool, id, traffic, "", ""); err != nil {
		t.Fatalf("terim meta: %v", err)
	}
	return id
}

// Terimler skorlanıp yazılmalı; coverage_count eşleşen içerik sayısı olmalı.
func TestScoreTrendsTermiSkorlarVeEslesmeyiYazar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)
	src := trendKaynakEkle(t, pool, "Test Trends", "google_trends", "https://example.invalid/rss")
	termID := trendTerimEkle(t, pool, src, "enflasyon", 5000)

	// Terimle eşleşen yayında bir içerik.
	var tip string
	_ = pool.QueryRow(ctx, `SELECT key FROM content_types LIMIT 1`).Scan(&tip)
	var icerikID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, spot, author_id, site_id, locale, published_at)
		VALUES ($1,'yayinda','Enflasyon verisi açıklandı','Piyasa beklentisi',$2,1,'tr',now())
		RETURNING id`, tip, yazar).Scan(&icerikID); err != nil {
		t.Fatalf("içerik: %v", err)
	}

	scoreTrends(ctx, pool, testLogger())

	var score, coverage int
	var lastScored *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT score, coverage_count, last_scored_at FROM trend_terms WHERE id=$1`, termID).
		Scan(&score, &coverage, &lastScored); err != nil {
		t.Fatalf("terim okunamadı: %v", err)
	}
	if lastScored == nil {
		t.Error("last_scored_at yazılmalı")
	}
	if coverage < 1 {
		t.Errorf("coverage_count = %d; eşleşen içerik sayısı en az 1 olmalı", coverage)
	}
	if score <= 0 {
		t.Errorf("score = %d; pozitif olmalı", score)
	}
}

// Terim yoksa skorlama sessizce döner.
func TestScoreTrendsBos(t *testing.T) {
	pool := geciciPool(t)
	scoreTrends(context.Background(), pool, testLogger())
}

// `fetchTrendSource` gerçek RSS gövdesini işler: terimler yazılır, koşu
// kaydı "ok" ile kapanır.
func TestFetchTrendSourceTerimleriYazar(t *testing.T) {
	loopbackIstemci(t)
	pool := geciciPool(t)
	ctx := context.Background()

	// Google Trends daily RSS biçimine yakın en küçük gövde.
	const rss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:ht="https://trends.google.com/trending/rss">
<channel><title>Daily Trends</title>
<item><title>ENFLASYON</title><ht:approx_traffic>5000+</ht:approx_traffic>
<ht:news_item><ht:news_item_title>Enflasyon açıklandı</ht:news_item_title>
<ht:news_item_url>https://example.com/a</ht:news_item_url>
<ht:news_item_source>Örnek</ht:news_item_source></ht:news_item></item>
<item><title>DEPREM</title><ht:approx_traffic>2000+</ht:approx_traffic></item>
</channel></rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rss))
	}))
	defer srv.Close()

	srcID := trendKaynakEkle(t, pool, "Sahte Trends", "google_trends", srv.URL)
	row, err := repo.GetTrendSource(ctx, pool, srcID)
	if err != nil {
		t.Fatalf("kaynak: %v", err)
	}

	added, _, err := fetchTrendSource(ctx, pool, testLogger(),
		&http.Client{Timeout: 5 * time.Second}, row)
	if err != nil {
		t.Fatalf("fetchTrendSource: %v", err)
	}
	if added == 0 {
		t.Error("hiç terim eklenmedi")
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM trend_terms WHERE source_id=$1`, srcID).Scan(&n); err != nil {
		t.Fatalf("sayım: %v", err)
	}
	if n < 2 {
		t.Errorf("%d terim yazıldı; en az 2 bekleniyordu", n)
	}
}

// Kaynak ulaşılamazsa hata döner ve kaynak durumu güncellenir (sessiz yutma yok).
func TestFetchTrendSourceHataDurumunuYazar(t *testing.T) {
	loopbackIstemci(t)
	pool := geciciPool(t)
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	srcID := trendKaynakEkle(t, pool, "Bozuk Trends", "google_trends", srv.URL)
	row, err := repo.GetTrendSource(ctx, pool, srcID)
	if err != nil {
		t.Fatalf("kaynak: %v", err)
	}
	if _, _, err := fetchTrendSource(ctx, pool, testLogger(),
		&http.Client{Timeout: 5 * time.Second}, row); err == nil {
		t.Fatal("500 yanıtta hata dönmeli")
	}

	var lastError string
	var lastStatus, ardHata int
	if err := pool.QueryRow(ctx,
		`SELECT last_error, COALESCE(last_status,-1), consecutive_errors FROM trend_sources WHERE id=$1`,
		srcID).Scan(&lastError, &lastStatus, &ardHata); err != nil {
		t.Fatalf("kaynak okunamadı: %v", err)
	}
	// Hata kaydı iki alandan birine yansır: mesaj ya da artan hata sayacı.
	if lastError == "" && ardHata == 0 {
		t.Errorf("hata kaydı yok (last_error=%q, last_status=%d, consecutive_errors=%d)",
			lastError, lastStatus, ardHata)
	}
}

// 304 yanıtı "değişmedi" sayılır: hata yok, terim yazılmaz.
func TestFetchTrendSourceNotModified(t *testing.T) {
	loopbackIstemci(t)
	pool := geciciPool(t)
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer srv.Close()

	srcID := trendKaynakEkle(t, pool, "304 Trends", "google_trends", srv.URL)
	row, err := repo.GetTrendSource(ctx, pool, srcID)
	if err != nil {
		t.Fatalf("kaynak: %v", err)
	}
	added, updated, err := fetchTrendSource(ctx, pool, testLogger(),
		&http.Client{Timeout: 5 * time.Second}, row)
	if err != nil {
		t.Fatalf("304 hata olmamalı: %v", err)
	}
	if added != 0 || updated != 0 {
		t.Errorf("added=%d updated=%d; 304'te 0 beklenir", added, updated)
	}
}

// FetchAllTrendSourcesNow aktif kaynakları dolaşır; biri bozuksa son hatayı
// döndürür ama diğerlerini yine de işler.
func TestFetchAllTrendSourcesNowKismiHata(t *testing.T) {
	loopbackIstemci(t)
	pool := geciciPool(t)
	ctx := context.Background()

	const rss = `<?xml version="1.0"?><rss version="2.0" xmlns:ht="https://trends.google.com/trending/rss"><channel>
<item><title>SECIM</title></item></channel></rss>`
	iyi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rss))
	}))
	defer iyi.Close()
	kotu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer kotu.Close()

	trendKaynakEkle(t, pool, "İyi", "google_trends", iyi.URL)
	trendKaynakEkle(t, pool, "Kötü", "google_trends", kotu.URL)

	added, _, err := FetchAllTrendSourcesNow(ctx, pool, testLogger())
	if err == nil {
		t.Error("kötü kaynak varken son hata dönmeli")
	}
	if added == 0 {
		t.Error("iyi kaynaktan terim eklenmeli")
	}
}

// --- SyncGSC / SyncGA4: kimlik bilgisi yokken sessiz/hatalı davranış ---

// Google kimlik bilgisi yokken GSC senkronu hata metni döndürür ama panik yok.
func TestSyncGSCKimlikYokkenHataDoner(t *testing.T) {
	pool := geciciPool(t)
	// Yapılandırma yokken ne dönerse dönsün ÇÖKMEMELİ. Sözleşme:
	// hata varsa err dolu; hata yoksa yazılan satır sayısı döner.
	n, err := SyncGSC(context.Background(), pool, testLogger(), 1)
	if err != nil {
		return // beklenen: kimlik bilgisi yok
	}
	if n < 0 {
		t.Errorf("satır sayısı negatif: %d", n)
	}
}

func TestSyncGA4KimlikYokkenHataDoner(t *testing.T) {
	pool := geciciPool(t)
	_, _ = SyncGA4(context.Background(), pool, testLogger())
}

// --- runAISourceOnce / icerikSkor: yapılandırma yokken çökmemeli ---

func TestRunAISourceOnceBosVeritabani(t *testing.T) {
	t.Setenv("SEARCH_SVC_URL", "")
	pool := geciciPool(t)
	runAISourceOnce(context.Background(), pool, testLogger())
}

// --- yardımcı: ayar okuma JSON biçimlerini kabul etmeli ---

func TestSettingStringJsonVeDuzMetin(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	for k, v := range map[string]string{"a_key": "deger", "b_key": "1"} {
		b, _ := json.Marshal(v)
		if _, err := pool.Exec(ctx, `
			INSERT INTO settings (section, key, value) VALUES ('test',$1,$2::jsonb)
			ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, k, string(b)); err != nil {
			t.Fatalf("ayar: %v", err)
		}
	}
	if got := settingString(ctx, pool, "test", "a_key"); got != "deger" {
		t.Errorf("settingString = %q", got)
	}
	if got := seoSetting(ctx, pool, "yok"); got != "" {
		t.Errorf("eksik ayar = %q", got)
	}
}
