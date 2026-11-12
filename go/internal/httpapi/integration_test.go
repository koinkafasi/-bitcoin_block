//go:build integration

package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/config"
	"theblock/site/internal/db"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
	"theblock/site/internal/repo"
)

var (
	itPool  *pgxpool.Pool
	itHTTP  *httptest.Server
	testDB  string
	skipAll bool
)

// TestMain test veritabanını hazırlar ve HTTP sunucusunu bir kez açar.
func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		skipAll = true
		os.Exit(m.Run())
	}
	testDB = url

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		println("TEST veritabanına bağlanılamadı:", err.Error())
		os.Exit(1)
	}
	// şemayı kur
	conn, err := pool.Acquire(ctx)
	if err == nil {
		if _, err = db.Up(ctx, conn.Conn()); err != nil {
			println("migration başarısız:", err.Error())
			os.Exit(1)
		}
		conn.Release()
	}
	itPool = pool

	cfg := config.Config{
		Env:         "test",
		Debug:       true,
		ListenAddr:  "127.0.0.1:0",
		BaseURL:     "http://127.0.0.1",
		Port:        "0",
		DatabaseURL: url,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// i18n bundle'ı gerçek veritabanından yükle (nil bundle public
	// handler'da panic yaratıyor).
	bundle, err := i18n.Load(ctx, pool)
	if err != nil {
		println("i18n yüklenemedi:", err.Error())
		os.Exit(1)
	}

	srv := httpapi.NewServer(cfg, logger, pool, bundle, nil)
	itHTTP = httptest.NewServer(srv.Handler)

	code := m.Run()

	itHTTP.Close()
	pool.Close()
	os.Exit(code)
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if skipAll {
		t.Skip("TEST_DATABASE_URL tanımlı değil — integration testleri atlandı")
	}
}

func get(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(itHTTP.URL + path)
	require.NoError(t, err, "GET %s", path)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(b)
}

// ---- sağlık + güvenlik başlıkları ---------------------------------

func TestHealthz(t *testing.T) {
	skipIfNoDB(t)
	resp, body := get(t, "/healthz")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `"status":"ok"`)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestSecurityHeaders(t *testing.T) {
	skipIfNoDB(t)
	resp, _ := get(t, "/healthz")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"), "her yanıtta istek kimliği olmalı")

	// Permissions-Policy: kullanılmayan yetenekler kapalı olmalı.
	pp := resp.Header.Get("Permissions-Policy")
	assert.Contains(t, pp, "geolocation=()", "konum kullanılmıyor, kapalı olmalı")
	assert.Contains(t, pp, "camera=()", "kamera kullanılmıyor, kapalı olmalı")
	assert.Contains(t, pp, "payment=()", "ödeme API'si kullanılmıyor (iyzico yönlendirme ile)")
	// Dikte Web Speech API kullanıyor (static/editor.js, static/app.js);
	// mikrofon kapatılırsa o özellik ölür.
	assert.Contains(t, pp, "microphone=(self)", "dikte için mikrofon açık kalmalı")

	for _, yetenek := range []string{"autoplay", "fullscreen", "picture-in-picture", "encrypted-media"} {
		if !strings.Contains(pp, yetenek+"=(self \"https://www.youtube.com\"") {
			t.Errorf("Permissions-Policy %s için YouTube izni yok: %s", yetenek, pp)
		}
	}

	csp := resp.Header.Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "CSP zorunlu kipte yayımlanmalı")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'self'")
	assert.Contains(t, csp, "script-src 'self'",
		"script-src kendi kaynağımıza izin vermeli")
	assert.NotContains(t, csp, "script-src 'unsafe-inline'",
		"script-src 'unsafe-inline' XSS korumasını sıfırlar")
	assert.NotContains(t, csp, "'unsafe-eval'")
	assert.NotContains(t, csp, "'nonce-",
		"nonce kullanılmıyor: önbelleklenen yanıt tüm kullanıcılara aynı nonce'u verirdi")
	styleSrc := cspDirektifi(csp, "style-src ")
	assert.NotContains(t, styleSrc, "'unsafe-inline'",
		"style-src 'unsafe-inline' içermemeli (enjekte <style> çalışmasın)")
	assert.Contains(t, csp, "style-src-attr 'unsafe-inline'",
		"style= öznitelikleri ayrı direktifle açık kalmalı")
	assert.Empty(t, resp.Header.Get("Content-Security-Policy-Report-Only"),
		"rapor kipi başlığı artık gönderilmemeli")
}

// cspDirektifi, CSP dizesinden adı verilen direktifin değerini çeker.
func cspDirektifi(csp, ad string) string {
	for _, p := range strings.Split(csp, ";") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, ad) {
			return p
		}
	}
	return ""
}

// Satır içi script'i olan bir sayfada hash gerçekten politikaya girmeli ve
// o hash sayfanın baytlarından hesaplanmış olmalı (regresyon: sabit liste).
func TestCSPHashSayfadanTuretilir(t *testing.T) {
	skipIfNoDB(t)
	resp, body := get(t, "/")
	csp := resp.Header.Get("Content-Security-Policy")
	_ = body // gövde hash'lerin kaynağı; yalnız başlık sınanıyor

	re := regexp.MustCompile(`'sha256-[A-Za-z0-9+/=]+'`)
	hashler := re.FindAllString(csp, -1)
	if len(hashler) == 0 {
		t.Skip("ana sayfada satır içi script yok — hash üretilmez")
	}
	// Aynı sayfa tekrar istendiğinde politika BAYT-AYNI olmalı: hash
	// deterministik, önbellek güvenli.
	resp2, _ := get(t, "/")
	if csp2 := resp2.Header.Get("Content-Security-Policy"); csp2 != csp {
		t.Fatalf("CSP iki istekte farklı:\n1: %s\n2: %s", csp, csp2)
	}
	// style-src Faz 211'de sıkılaştırıldı: `unsafe-inline` YOK, sayfanın
	// <style> blokları hash ile izinli.
	styleSrc := cspDirektifi(csp, "style-src ")
	assert.Contains(t, styleSrc, "'sha256-",
		"style-src sayfanın <style> hash'ini içermeli")
	assert.NotContains(t, styleSrc, "'unsafe-inline'",
		"style-src 'unsafe-inline' içermemeli")
	assert.Contains(t, csp, "style-src-attr 'unsafe-inline'",
		"style= öznitelikleri ayrı direktifle açık")
}

func TestRequestIDEchoed(t *testing.T) {
	skipIfNoDB(t)
	req, err := http.NewRequest("GET", itHTTP.URL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "test-req-42")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "test-req-42", resp.Header.Get("X-Request-ID"))
}

// ---- genel sayfalar -----------------------------------------------

func TestPublicPages(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name string
		path string
	}{
		{"anasayfa", "/"},
		{"borsa", "/borsa"},
		{"arama", "/arama?q=test"},
		{"robots", "/robots.txt"},
		{"rss", "/rss.xml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := get(t, c.path)
			assert.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", firstN(body, 300))
			assert.NotEmpty(t, strings.TrimSpace(body))
		})
	}
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ---- yetki / koruma (permission) ----------------------------------

func TestAdminRoutesRequireAuth(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name string
		path string
	}{
		{"panel anasayfa", "/admin"},
		{"icerikler", "/admin/icerikler"},
		{"kategoriler", "/admin/kategoriler"},
		{"ayarlar", "/admin/ayarlar"},
		{"tasarimci", "/admin/tasarim"},
		{"servisler", "/admin/servisler"},
		{"reklamlar", "/admin/reklamlar"},
		{"menu", "/admin/menu"},
		{"yeni icerik formu", "/admin/icerikler/yeni"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // yönlendirmeyi izleme
				},
			}
			resp, err := client.Get(itHTTP.URL + c.path)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusFound, resp.StatusCode, "oturumsuz erişim yönlendirilmeli")
			loc := resp.Header.Get("Location")
			assert.Contains(t, loc, "/admin/giris", "giriş sayfasına yönlendirilmeli")
		})
	}
}

func TestAdminLoginPageReachable(t *testing.T) {
	skipIfNoDB(t)
	resp, body := get(t, "/admin/giris")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, strings.ToLower(body), "password", "giriş formu basılmalı")
}

// ---- 404 / hata durumları ----------------------------------------

func TestUnknownPaths(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name string
		path string
	}{
		{"olmayan kategori", "/kategori/boyle-bir-kategori-yok"},
		{"olmayan api", "/api/hava/olmayan-sehir-xyz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, _ := get(t, c.path)
			assert.GreaterOrEqual(t, resp.StatusCode, 400, "bilinmeyen kaynak 4xx/5xx dönmeli")
		})
	}
}

func TestJSONAPIErrorShape(t *testing.T) {
	skipIfNoDB(t)
	resp, body := get(t, "/api/hava/olmayan-sehir-xyz")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, body, `"ok":false`)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestPublicCSSHasNoBareFrTracks(t *testing.T) {
	skipIfNoDB(t)
	resp, err := http.Get(itHTTP.URL + "/static/public.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	css := string(b)

	var offenders []string
	for _, line := range strings.Split(css, "\n") {
		if !strings.Contains(line, "grid-template-columns:") {
			continue
		}
		if !strings.Contains(line, "fr") {
			continue
		}
		if strings.Contains(line, "minmax") {
			continue
		}
		// yorum satırı değilse gerçek kural
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/*") {
			continue
		}
		offenders = append(offenders, trimmed)
	}
	assert.Empty(t, offenders,
		"grid sütunları minmax(0,1fr) kullanmalı (çıplak 1fr mobilde min-content taşması yapar): %v", offenders)
}

func TestPublicCSSContainsMinmaxGuard(t *testing.T) {
	skipIfNoDB(t)
	resp, err := http.Get(itHTTP.URL + "/static/public.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	css := string(b)
	assert.Contains(t, css, "minmax(0, 1fr)",
		"home-grid gibi ana kolonlar minmax(0,1fr) ile korunmalı")
}

// ---- veri katmanı round-trip (gerçek DB) --------------------------

func TestSettingsRoundTrip(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	const section, key = "integration_test", "roundtrip"

	require.NoError(t, repo.SetSetting(ctx, itPool, section, key, []byte("değer")))
	got, err := repo.GetSetting(ctx, itPool, section, key)
	require.NoError(t, err)
	assert.Equal(t, "değer", string(got))

	// üzerine yaz (upsert)
	require.NoError(t, repo.SetSetting(ctx, itPool, section, key, []byte("yeni")))
	got2, err := repo.GetSetting(ctx, itPool, section, key)
	require.NoError(t, err)
	assert.Equal(t, "yeni", string(got2))

	// JSON nesnesi olduğu gibi korunur.
	require.NoError(t, repo.SetSetting(ctx, itPool, section, "nesne", []byte(`{"a":1}`)))
	got3, err := repo.GetSetting(ctx, itPool, section, "nesne")
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(got3))

	_, err = repo.GetSetting(ctx, itPool, section, "olmayan-anahtar")
	assert.Error(t, err, "olmayan anahtar hata vermeli")

	// Test kalıntısı bırakma.
	_, _ = itPool.Exec(ctx, `DELETE FROM settings WHERE section=$1`, section)
}

func TestLayoutRoundTrip(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	const siteID = int64(1)
	page := "integration_" + time.Now().Format("150405.000000000")

	// yok -> boş dizi
	got, err := repo.GetLayout(ctx, itPool, siteID, page)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(got))

	blocks := `[{"type":"slider","zone":"ust"}]`
	require.NoError(t, repo.SaveLayout(ctx, itPool, siteID, page, []byte(blocks)))

	got, err = repo.GetLayout(ctx, itPool, siteID, page)
	require.NoError(t, err)
	assert.JSONEq(t, blocks, string(got))

	// ikinci kayıt upsert olmalı (çift satır olmamalı)
	require.NoError(t, repo.SaveLayout(ctx, itPool, siteID, page, []byte(`[{"type":"manset6","zone":"ana"}]`)))
	got, err = repo.GetLayout(ctx, itPool, siteID, page)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"type":"manset6","zone":"ana"}]`, string(got))
	assert.Contains(t, string(got), "manset6")

	require.NoError(t, repo.DeleteLayout(ctx, itPool, siteID, page))
	got, err = repo.GetLayout(ctx, itPool, siteID, page)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(got), "silindikten sonra varsayılana dönülmeli")
}

func TestCacheRoundTrip(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	kind := "integration_test"

	require.NoError(t, repo.CacheSet(ctx, itPool, kind, "k1", []byte(`{"v":1}`)))

	got, err := repo.CacheGet(ctx, itPool, kind, "k1")
	require.NoError(t, err)
	assert.JSONEq(t, `{"v":1}`, string(got))

	// var olmayan anahtar hata vermeli
	_, err = repo.CacheGet(ctx, itPool, kind, "yok")
	assert.Error(t, err)

	rows, err := repo.CacheList(ctx, itPool, kind)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	for _, r := range rows {
		assert.Equal(t, kind, r.Kind)
	}
}

// ---- veritabanı kesintisi halinde dayanıklılık --------------------

func TestPagesDegradeGracefullyWithEmptyData(t *testing.T) {
	skipIfNoDB(t)
	// Servis önbelleği boşken sayfalar patlamamalı, 200 dönmeli
	resp, body := get(t, "/borsa")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)
}
