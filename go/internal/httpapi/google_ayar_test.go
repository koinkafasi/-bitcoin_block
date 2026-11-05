package httpapi

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Faz 206 — Google ayarlarının panelde tek kaynaktan okunup yazılması.
//
// ÖLÇÜM (3 Kasım 2026, üretim): trend.ga4_property = 16218714509 iken
// AdSense panelindeki "GA4 Property ID" alanı BOŞ görünüyordu; çünkü panel
// eski adsense/ga_property anahtarını okuyordu. Aynı değer iki ayrı yerde
// tutulunca biri doldurulup diğeri unutuluyor.
//
// Bu testler gerçek DB ister (DATABASE_URL); yoksa atlanır.

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// Bu testler AYAR YAZAR. Varsayılan kapalı olmalı: DATABASE_URL üretim
	// veritabanını gösterirken kazara koşup gerçek GA4 mülk kimliğini
	// ezebiliyor (bir kez oldu: trend.ga4_property 16218714509 → 555555555).
	if os.Getenv("GOOGLE_AYAR_TEST") != "1" {
		t.Skip("GOOGLE_AYAR_TEST=1 değil (ayar yazan test)")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	ayarlariGeriAl(t, pool)
	return pool
}

// ayarlariGeriAl testin dokunduğu anahtarların ÖNCEKİ değerini saklar ve
// test bitince geri yazar.
//
// Neden: bu testler gerçek ayar tablosuna yazar. Temizlik olmadan koşunca
// üretimdeki gerçek GA4 mülk kimliği test değeriyle ezildi (ölçüldü:
// trend.ga4_property 16218714509 → 555555555). Test, kendi çöpünü toplar.
func ayarlariGeriAl(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	type anahtar struct{ section, key string }
	anahtarlar := []anahtar{
		{"trend", "ga4_property"},
		{"adsense", "ga_property"},
	}
	onceki := map[anahtar][]byte{}
	ctx := context.Background()
	for _, a := range anahtarlar {
		if v, err := repo.GetSetting(ctx, pool, a.section, a.key); err == nil {
			onceki[a] = v
		}
	}
	t.Cleanup(func() {
		ctx := context.Background()
		for _, a := range anahtarlar {
			if v, ok := onceki[a]; ok {
				_ = repo.SetSetting(ctx, pool, a.section, a.key, v)
			}
		}
	})
}

func TestGA4MulkuTekKaynagaYazilir(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := &Server{pool: pool}

	// Temiz başlangıç: iki anahtar da farklı değerlerde.
	require.NoError(t, repo.SetTrendSetting(ctx, pool, "ga4_property", "111111111"))
	require.NoError(t, repo.SetSetting(ctx, pool, "adsense", "ga_property", []byte(`"222222222"`)))

	// API biçiminde (properties/…) gelen değer normalize edilip TEK kaynağa
	// yazılmalı, eski anahtar da aynalanmalı.
	got := s.ga4MulkuKaydet(ctx, "properties/333333333")
	assert.Equal(t, "333333333", got)

	assert.Equal(t, "333333333", repo.GetTrendSetting(ctx, pool, "ga4_property"), "kanonik anahtar güncellenmedi")

	eski, err := repo.GetSetting(ctx, pool, "adsense", "ga_property")
	require.NoError(t, err)
	// GetSetting JSON tırnaklarını soyar (settingText): düz değer karşılaştırılır.
	assert.Equal(t, "333333333", strings.Trim(string(eski), `"`), "eski anahtar aynalanmadı")
}

func TestGA4MulkuBosDegerMevcutAyarıBozmaz(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := &Server{pool: pool}

	require.NoError(t, repo.SetTrendSetting(ctx, pool, "ga4_property", "444444444"))

	// Panelde alanı temizleyip kaydetmek GA4 raporlamasını sessizce
	// kapatmamalı: boş/geçersiz değer yok sayılır.
	for _, ham := range []string{"", "   ", "abc", "properties/"} {
		assert.Equal(t, "", s.ga4MulkuKaydet(ctx, ham), "girdi %q", ham)
	}
	assert.Equal(t, "444444444", repo.GetTrendSetting(ctx, pool, "ga4_property"), "boş gönderim değeri bozdu")
}

// Panel, kurulu GA4 mülkünü tek kaynaktan okumalı: eski adsense anahtarı
// boş olsa bile trend anahtarından gelir.
func TestGA4MulkPaneliKanonikKaynaktanOkur(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := &Server{pool: pool}

	require.NoError(t, repo.SetTrendSetting(ctx, pool, "ga4_property", "555555555"))
	require.NoError(t, repo.SetSetting(ctx, pool, "adsense", "ga_property", []byte(`""`)))

	assert.Equal(t, "555555555", s.ga4PropertyID(ctx),
		"AdSense paneli eski anahtara bakıp boş göstermemeli")
}

// Google Site Doğrulama: kod public <head>'e basılmalı, serbest metin
// (HTML enjeksiyonu) kabul edilmemeli.
//
// ÖLÇÜM (3 Kasım 2026): panelde alan vardı, değer kaydediliyordu ama hiçbir
// sayfada basılmıyordu; yönetici kodu girip Search Console'da doğrulamayı
// yine başarısız görüyordu.
func TestValidGVerification(t *testing.T) {
	kabul := map[string]string{
		"abc123DEF456ghi789":     "abc123DEF456ghi789",
		"  abc123DEF456ghi789  ": "abc123DEF456ghi789",
		`<meta name="google-site-verification" content="abc123DEF456ghi789" />`: "abc123DEF456ghi789",
		`content="abc123DEF456ghi789"`:                                          "abc123DEF456ghi789",
	}
	for in, want := range kabul {
		assert.Equal(t, want, validGVerification(in), "girdi: %q", in)
	}
	for _, in := range []string{
		"", "   ", "kisa", `"><script>alert(1)</script>`,
		`abc<script>`, "abc def ghi jkl", strings.Repeat("a", 130),
	} {
		assert.Equal(t, "", validGVerification(in), "reddedilmeliydi: %q", in)
	}
}

// Doğrulama kodu public sayfanın <head>'inde meta etiketi olarak çıkmalı;
// boşsa hiç basılmamalı.
func TestGVerificationHeadteBasilir(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	assert.NotContains(t, buf.String(), "google-site-verification",
		"kod yokken meta etiketi basılmamalı")

	d.GVerification = "abc123DEF456ghi789"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	assert.Contains(t, buf.String(),
		`<meta name="google-site-verification" content="abc123DEF456ghi789">`)
}

// AdSense yayıncı kimliği: panel alanı kaydediliyordu ama hiçbir sayfada
// kullanılmıyordu. Google site doğrulaması için meta etiketi olarak basılır.
func TestValidAdsensePublisher(t *testing.T) {
	kabul := map[string]string{
		"ca-pub-1234567890123456":     "ca-pub-1234567890123456",
		"  ca-pub-1234567890123456  ": "ca-pub-1234567890123456",
		`"ca-pub-1234567890123456"`:   "ca-pub-1234567890123456",
		`<meta name="google-adsense-account" content="ca-pub-1234567890123456">`: "ca-pub-1234567890123456",
	}
	for in, want := range kabul {
		assert.Equal(t, want, validAdsensePublisher(in), "girdi: %q", in)
	}
	for _, in := range []string{
		"", "ca-pub-", "ca-pub-123", "pub-1234567890123456",
		"ca-pub-123456789012345678901234", `"><script>alert(1)</script>`,
	} {
		assert.Equal(t, "", validAdsensePublisher(in), "reddedilmeliydi: %q", in)
	}
}

func TestAdsPublisherHeadteBasilir(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	assert.NotContains(t, buf.String(), "google-adsense-account")

	d.AdsPublisher = "ca-pub-1234567890123456"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "public_head", d))
	assert.Contains(t, buf.String(),
		`<meta name="google-adsense-account" content="ca-pub-1234567890123456">`)
}
