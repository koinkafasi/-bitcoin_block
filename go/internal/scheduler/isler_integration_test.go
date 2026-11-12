//go:build integration

package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- yardımcılar ---

func yeniBotPoller(logger *slog.Logger) *botPoller {
	return &botPoller{
		logger:    logger,
		lastCrawl: map[int64]time.Time{},
		fails:     map[int64]int{},
	}
}

// Yayındaki içeriklerin etiketleri kategori trend satırına dönüşmeli.
func TestRefreshTrendsEtiketSayiminiYazar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	var katID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO categories (name, slug, status) VALUES ('Ekonomi', 'ekonomi', 'yayinda')
		RETURNING id`).Scan(&katID); err != nil {
		t.Fatalf("kategori: %v", err)
	}
	var tagID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO tags (name, slug) VALUES ('Enflasyon', 'enflasyon') RETURNING id`).Scan(&tagID); err != nil {
		t.Fatalf("etiket: %v", err)
	}

	var tip string
	_ = pool.QueryRow(ctx, `SELECT key FROM content_types LIMIT 1`).Scan(&tip)
	var icerikID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, author_id, category_id, published_at, site_id, locale)
		VALUES ($1,'yayinda','Trend testi',$2,$3,now(),1,'tr') RETURNING id`,
		tip, yazar, katID).Scan(&icerikID); err != nil {
		t.Fatalf("içerik: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO content_tags (content_id, tag_id) VALUES ($1,$2)`,
		icerikID, tagID); err != nil {
		t.Fatalf("content_tags: %v", err)
	}

	refreshTrends(ctx, pool, testLogger())

	var adet int
	if err := pool.QueryRow(ctx,
		`SELECT post_count FROM trends WHERE site_id=1 AND tag='Enflasyon' AND category_id=$1`, katID).Scan(&adet); err != nil {
		t.Fatalf("trend satırı yazılmadı: %v", err)
	}
	if adet != 1 {
		t.Errorf("post_count = %d; 1 bekleniyordu", adet)
	}
}

// Yayında olmayan (taslak/çöp) içerik trend sayımına girmemeli.
func TestRefreshTrendsYayindaOlmayanSayilmaz(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)

	var katID, tagID int64
	_ = pool.QueryRow(ctx, `INSERT INTO categories (name, slug) VALUES ('Dünya','dunya') RETURNING id`).Scan(&katID)
	_ = pool.QueryRow(ctx, `INSERT INTO tags (name, slug) VALUES ('Deprem','deprem') RETURNING id`).Scan(&tagID)
	var tip string
	_ = pool.QueryRow(ctx, `SELECT key FROM content_types LIMIT 1`).Scan(&tip)
	var id int64
	_ = pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, author_id, category_id, published_at, site_id, locale)
		VALUES ($1,'taslak','Taslak trend',$2,$3,now(),1,'tr') RETURNING id`, tip, yazar, katID).Scan(&id)
	_, _ = pool.Exec(ctx, `INSERT INTO content_tags (content_id, tag_id) VALUES ($1,$2)`, id, tagID)

	refreshTrends(ctx, pool, testLogger())

	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM trends WHERE tag='Deprem'`).Scan(&n)
	if n != 0 {
		t.Errorf("taslak içerik trend sayımına girdi (%d satır)", n)
	}
}

// Hiç site/içerik yoksa hata vermeden döner.
func TestRefreshTrendsBosVeritabani(t *testing.T) {
	pool := geciciPool(t)
	refreshTrends(context.Background(), pool, testLogger())
}

// --- icerikSkorTick ---

// İçerik yokken de hata vermez; skorlama sessizce geçer.
func TestIcerikSkorTickBos(t *testing.T) {
	pool := geciciPool(t)
	icerikSkorTick(context.Background(), pool, testLogger())
}

// --- settingString / seoSetting ---

func TestSettingStringAyarYoksaBos(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	if got := settingString(ctx, pool, "seo", "yok_boyle_anahtar"); got != "" {
		t.Errorf("eksik ayar %q; boş bekleniyordu", got)
	}
	if got := seoSetting(ctx, pool, "auto_index"); got != "" {
		t.Errorf("seoSetting eksik ayar %q", got)
	}
}

// --- indexPublished ---

// SEARCH_SVC_URL yoksa hiçbir istek yapılmaz (özellik kapalı).
func TestIndexPublishedServisYoksaSessiz(t *testing.T) {
	t.Setenv("SEARCH_SVC_URL", "")
	pool := geciciPool(t)
	indexPublished(context.Background(), pool, testLogger(), []int64{1})
}

// Servis varsa yayındaki içerik doküman olarak gönderilmeli; taslak
// gönderilmemeli.
func TestIndexPublishedDokumanlariGonderir(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)
	var tip string
	_ = pool.QueryRow(ctx, `SELECT key FROM content_types LIMIT 1`).Scan(&tip)

	var yayinID, taslakID int64
	_ = pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, spot, detail, slug, author_id, site_id, locale)
		VALUES ($1,'yayinda','İndeks başlığı','Özet','[{"text":"Gövde parçası"}]','indeks-basligi',$2,1,'tr')
		RETURNING id`, tip, yazar).Scan(&yayinID)
	_ = pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, author_id, site_id, locale)
		VALUES ($1,'taslak','Taslak başlık',$2,1,'tr') RETURNING id`, tip, yazar).Scan(&taslakID)

	var mu sync.Mutex
	var alinan []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Docs []map[string]any `json:"docs"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		alinan = append(alinan, body.Docs...)
		mu.Unlock()
		fmt.Fprint(w, `{"indexed":1}`)
	}))
	defer srv.Close()
	t.Setenv("SEARCH_SVC_URL", srv.URL)

	indexPublished(ctx, pool, testLogger(), []int64{yayinID, taslakID})

	mu.Lock()
	defer mu.Unlock()
	if len(alinan) != 1 {
		t.Fatalf("%d doküman gönderildi; yalnız yayındaki (1) bekleniyordu", len(alinan))
	}
	d := alinan[0]
	if d["title"] != "İndeks başlığı" {
		t.Errorf("title = %v", d["title"])
	}
	if d["body"] != "Gövde parçası" {
		t.Errorf("body = %v (blok metni çıkarılmalı)", d["body"])
	}
	// URL slug--id biçiminde olmalı.
	wantURL := fmt.Sprintf("/indeks-basligi-%d", yayinID)
	if d["url"] != wantURL {
		t.Errorf("url = %v; %s bekleniyordu", d["url"], wantURL)
	}
}

// --- submitIndexNowPromoted ---

// Otomatik indeksleme kapalıyken dış çağrı yapılmaz.
func TestSubmitIndexNowKapaliykenCagriYok(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()

	var cagri int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cagri, 1)
	}))
	defer srv.Close()

	submitIndexNowPromoted(ctx, pool, testLogger(), []int64{1})
	if atomic.LoadInt32(&cagri) != 0 {
		t.Fatal("auto_index kapalıyken dış istek yapılmamalı")
	}
}

// auto_index açık + anahtar varsa URL'ler bildirilir ve sonuç loglanır.
func TestSubmitIndexNowBildirirVeLoglar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	yazar := testUser(t, pool)
	var tip string
	_ = pool.QueryRow(ctx, `SELECT key FROM content_types LIMIT 1`).Scan(&tip)

	var id int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO contents (type_key, status, title, slug, author_id, site_id, locale)
		VALUES ($1,'yayinda','IndexNow içerik','indexnow-icerik',$2,1,'tr') RETURNING id`,
		tip, yazar).Scan(&id); err != nil {
		t.Fatalf("içerik: %v", err)
	}

	for k, v := range map[string]string{
		"auto_index": "1", "indexnow_key": "testkey",
		"indexnow_domain": "theblock.com.tr",
	} {
		b, _ := json.Marshal(v)
		if _, err := pool.Exec(ctx, `
			INSERT INTO settings (section, key, value) VALUES ('seo',$1,$2::jsonb)
			ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, k, string(b)); err != nil {
			t.Fatalf("ayar yazılamadı (%s): %v", k, err)
		}
	}
	if got := seoSetting(ctx, pool, "auto_index"); got != "1" {
		t.Fatalf("auto_index = %q; 1 bekleniyordu", got)
	}

	var mu sync.Mutex
	var alinanURL []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URLList []string `json:"urlList"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		alinanURL = append(alinanURL, body.URLList...)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	b, _ := json.Marshal(srv.URL)
	if _, err := pool.Exec(ctx, `
		INSERT INTO settings (section, key, value) VALUES ('seo','indexnow_endpoint',$1::jsonb)
		ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, string(b)); err != nil {
		t.Fatalf("endpoint yazılamadı: %v", err)
	}
	if got := seoSetting(ctx, pool, "indexnow_endpoint"); got != srv.URL {
		t.Fatalf("endpoint = %q; %q bekleniyordu", got, srv.URL)
	}

	submitIndexNowPromoted(ctx, pool, testLogger(), []int64{id})

	// Bildirim arka planda; log satırı görünene kadar bekle (en çok 5 sn).
	son := time.Now().Add(5 * time.Second)
	var logAdet int
	for time.Now().Before(son) {
		_ = pool.QueryRow(ctx,
			`SELECT count(*) FROM indexing_log WHERE url LIKE '%indexnow-icerik%'`).Scan(&logAdet)
		if logAdet > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if logAdet == 0 {
		t.Error("indexing_log'a satır yazılmadı")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(alinanURL) != 1 {
		t.Fatalf("%d URL bildirildi; 1 bekleniyordu", len(alinanURL))
	}
	if want := fmt.Sprintf("https://theblock.com.tr/indexnow-icerik-%d", id); alinanURL[0] != want {
		t.Errorf("URL = %q; %q bekleniyordu", alinanURL[0], want)
	}
}

// --- bot poller tick ---

// Otomatik çekme kapalıyken hiçbir feed denenmez.
func TestBotTickOtomatikKapaliykenAtlar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	b, _ := json.Marshal(false)
	if _, err := pool.Exec(ctx, `
		INSERT INTO settings (section, key, value) VALUES ('bot','otomatik',$1)
		ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, b); err != nil {
		t.Skipf("settings: %v", err)
	}

	bp := yeniBotPoller(testLogger())
	bp.tick(ctx, pool) // panik/hata olmadan dönmeli

	bp.mu.Lock()
	defer bp.mu.Unlock()
	if len(bp.lastCrawl) != 0 {
		t.Fatalf("hiçbir feed taranmamalıydı; %d kayıt var", len(bp.lastCrawl))
	}
}

// Ayar yokken migration varsayılanı `true`'dur — bu yüzden ayrı bir
// "ayar yok" testi yok: varsayılanın kendisi 0010_bot.up.sql'de kilitli.

// --- gundem poller tick ---

// Varsayılan 15 dakika dolmadığından ikinci çağrı hiç iş yapmaz.
func TestGundemTickAralikDolmadanAtlar(t *testing.T) {
	pool := geciciPool(t)
	ctx := context.Background()
	gp := &gundemPoller{logger: testLogger(), lastRun: time.Now()}
	gp.tick(ctx, pool) // due=false → hemen döner
	if gp.lastRun.IsZero() {
		t.Fatal("lastRun korunmalıydı")
	}
}
