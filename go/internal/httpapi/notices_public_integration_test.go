//go:build integration

package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/config"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
)

func ilanTestSunucu(t *testing.T) *httptest.Server {
	t.Helper()
	bundle, err := i18n.Load(t.Context(), itPool)
	require.NoError(t, err)
	srv := httpapi.NewServer(config.Config{
		Env: "test", ListenAddr: "127.0.0.1:0", BaseURL: "http://127.0.0.1",
		Port: "0", DatabaseURL: testDB,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), itPool, bundle, nil)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}

// ilanEkle test ilanları yazar ve temizliği kaydeder.
func ilanEkle(t *testing.T) (bugunID, dunID int64) {
	t.Helper()
	ctx := t.Context()
	_, _ = itPool.Exec(ctx, `DELETE FROM official_notices WHERE title LIKE 'TEST İLAN%'`)
	t.Cleanup(func() {
		_, _ = itPool.Exec(context.Background(),
			`DELETE FROM official_notices WHERE title LIKE 'TEST İLAN%'`)
	})

	require.NoError(t, itPool.QueryRow(ctx, `
		INSERT INTO official_notices (site_id,title,institution,city,notice_date,url,body,sort)
		VALUES (1,'TEST İLAN Ankara Mahkeme Satışı','Ankara 3. Asliye Hukuk Mahkemesi','Ankara',
		        CURRENT_DATE,'https://example.com/a.pdf','Açık artırma ile satış metni.',1)
		RETURNING id`).Scan(&bugunID))

	require.NoError(t, itPool.QueryRow(ctx, `
		INSERT INTO official_notices (site_id,title,institution,city,notice_date,url,body,sort)
		VALUES (1,'TEST İLAN İzmir İcra Satışı','İzmir 1. İcra Müdürlüğü','İzmir',
		        CURRENT_DATE - 1,NULL,'Taşınır mal satış ilanı.',1)
		RETURNING id`).Scan(&dunID))
	return bugunID, dunID
}

func TestIlanDetaySayfasiVeKanonikYonlendirme(t *testing.T) {
	skipIfNoDB(t)
	bugunID, _ := ilanEkle(t)
	ts := ilanTestSunucu(t)

	// Liste, ilanın kalıcı adresine bağlanmalı.
	_, liste := getFrom(t, ts, "/resmi-ilanlar")
	assert.Contains(t, liste, "/resmi-ilanlar/", "liste kalıcı ilan adresine bağlanmalı")
	assert.Contains(t, liste, "TEST İLAN Ankara Mahkeme Satışı")

	// Detay sayfası açılmalı ve ilan metnini göstermeli.
	yol := "/resmi-ilanlar/test-ilan-ankara-mahkeme-satisi-" + itoa64Test(bugunID)
	resp, detay := getFrom(t, ts, yol)
	require.Equal(t, 200, resp.StatusCode, "ilan detayı açılmalı: %s", yol)
	assert.Contains(t, detay, "TEST İLAN Ankara Mahkeme Satışı")
	assert.Contains(t, detay, "Açık artırma ile satış metni.")
	assert.Contains(t, detay, "Belgeyi Aç", "belge bağlantısı gösterilmeli")

	// Başlık değişirse eski slug'la gelen istek kanonik adrese taşınmalı:
	// aynı ilan iki ayrı adreste indekslenmemeli.
	istemci := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	r2, err := istemci.Get(ts.URL + "/resmi-ilanlar/eski-baslik-" + itoa64Test(bugunID))
	require.NoError(t, err)
	defer r2.Body.Close()
	assert.Equal(t, http.StatusMovedPermanently, r2.StatusCode)
	assert.Equal(t, yol, r2.Header.Get("Location"), "kanonik adrese taşınmalı")

	// Olmayan kimlik 404 olmalı; kimliksiz adres de.
	r3, _ := getFrom(t, ts, "/resmi-ilanlar/olmayan-ilan-99999999")
	assert.Equal(t, 404, r3.StatusCode)
	r4, _ := getFrom(t, ts, "/resmi-ilanlar/kimliksiz")
	assert.Equal(t, 404, r4.StatusCode)
}

func TestIlanFiltresiTumTarihlereGenisler(t *testing.T) {
	skipIfNoDB(t)
	ilanEkle(t)
	ts := ilanTestSunucu(t)

	_, varsayilan := getFrom(t, ts, "/resmi-ilanlar")
	assert.NotContains(t, varsayilan, "TEST İLAN İzmir İcra Satışı",
		"tarih verilmeyince yalnız en son gün listelenmeli")

	_, sehirli := getFrom(t, ts, "/resmi-ilanlar?sehir=%C4%B0zmir")
	assert.Contains(t, sehirli, "TEST İLAN İzmir İcra Satışı",
		"şehir filtresi dünkü ilanı da bulmalı")
	assert.NotContains(t, sehirli, "TEST İLAN Ankara Mahkeme Satışı",
		"başka şehrin ilanı listelenmemeli")

	_, arama := getFrom(t, ts, "/resmi-ilanlar?q=Ta%C5%9F%C4%B1n%C4%B1r")
	assert.Contains(t, arama, "TEST İLAN İzmir İcra Satışı",
		"ilan metninde arama çalışmalı")

	_, bos := getFrom(t, ts, "/resmi-ilanlar?sehir=Bulunmayan")
	assert.Contains(t, bos, "Bu filtreye uyan ilan bulunamadı")
}

func TestIlanlarSiteHaritasinda(t *testing.T) {
	skipIfNoDB(t)
	bugunID, dunID := ilanEkle(t)
	ts := ilanTestSunucu(t)

	resp, xml := getFrom(t, ts, "/sitemap.xml")
	require.Equal(t, 200, resp.StatusCode)

	for _, id := range []int64{bugunID, dunID} {
		assert.Contains(t, xml, "/resmi-ilanlar/", "ilan adresleri site haritasında olmalı")
		_ = id
	}
	assert.Contains(t, xml, "test-ilan-ankara-mahkeme-satisi-"+itoa64Test(bugunID))
	assert.Contains(t, xml, "test-ilan-izmir-icra-satisi-"+itoa64Test(dunID))

	// lastmod zorunlu: tebliğ "son güncelleme zamanlarını içeren" site
	// haritası istiyor.
	i := strings.Index(xml, "test-ilan-ankara-mahkeme-satisi-")
	require.Greater(t, i, 0)
	kuyruk := xml[i:]
	son := kuyruk[:min(220, len(kuyruk))]
	assert.Contains(t, son, "<lastmod>", "ilan girdisinde lastmod bulunmalı")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func itoa64Test(v int64) string { return strconv.FormatInt(v, 10) }

// getFrom belirli bir test sunucusundan GET yapar (paketin ortak get()
// yardımcısı itHTTP'ye bağlı).
func getFrom(t *testing.T, ts *httptest.Server, path string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(ts.URL + path)
	require.NoError(t, err, "GET %s", path)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(b)
}
