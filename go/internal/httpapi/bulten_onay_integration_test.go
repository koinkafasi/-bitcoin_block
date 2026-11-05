//go:build integration

// Bülten çift onayının üretim kipinde gerçekten korumalı olduğunu doğrular.
//
// Neden ayrı bir sunucu kuruluyor: paketin ortak test sunucusu Debug=true ile
// çalışıyor (integration_test.go, TestMain) ve o kipte onay bağlantısı
// bilerek yanıta konur. Korunması gereken davranış üretim kipindedir, bu
// yüzden burada Debug=false ile ikinci bir sunucu ayağa kaldırılır.
package httpapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/config"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
)

// TestBultenOnayTokeniUretimdeSizmaz: SMTP yapılandırılmamışken bile onay
// token'ı yanıta yazılmamalı.
//
// Sızarsa çift onay anlamsızlaşır: formu dolduran kişi başkasının e-posta
// adresini girip dönen bağlantıyı kendisi açarak o adresi 'aktif' aboneye
// çevirebilir. Gönderim listesi status='aktif' filtresi kullandığı için
// (repo/subscribers.go) o adres bundan sonra onayı olmadan ticari ileti alır.
func TestBultenOnayTokeniUretimdeSizmaz(t *testing.T) {
	skipIfNoDB(t)

	ctx := t.Context()
	bundle, err := i18n.Load(ctx, itPool)
	require.NoError(t, err)

	cfg := config.Config{
		Env:         "production",
		Debug:       false,
		ListenAddr:  "127.0.0.1:0",
		BaseURL:     "http://127.0.0.1",
		Port:        "0",
		DatabaseURL: testDB,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httpapi.NewServer(cfg, logger, itPool, bundle, nil)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	const email = "bulten-sizinti-testi@example.com"
	resp, err := http.PostForm(ts.URL+"/bulten", url.Values{"email": {email}})
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(raw)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "kayıt alınmalı: %s", body)
	assert.NotContains(t, body, "/bulten/onay/",
		"onay bağlantısı üretim kipinde yanıta yazılmamalı: %s", body)

	// Token'ın kendisi de hiçbir biçimde geçmemeli.
	var token, status string
	require.NoError(t, itPool.QueryRow(ctx,
		`SELECT token, status FROM subscribers WHERE email=$1`, email).Scan(&token, &status))
	require.NotEmpty(t, token, "abone kaydı oluşmalı")
	assert.NotContains(t, body, token, "token yanıtta görünmemeli")

	// Onaylanmadan 'aktif' olmamalı: gönderim listesine girmemeli.
	assert.NotEqual(t, "aktif", status, "onay alınmadan abone aktif olmamalı")

	// Temizlik: test adresi kalmasın.
	_, _ = itPool.Exec(ctx, `DELETE FROM subscribers WHERE email=$1`, email)
}

// TestBultenOnayTokeniGelistirmedeDoner: geliştirme kolaylığı korunmalı —
// aksi hâlde SMTP'siz yerel ortamda bülten akışı test edilemez hâle gelir.
func TestBultenOnayTokeniGelistirmedeDoner(t *testing.T) {
	skipIfNoDB(t)

	ctx := t.Context()
	const email = "bulten-gelistirme-testi@example.com"
	_, _ = itPool.Exec(ctx, `DELETE FROM subscribers WHERE email=$1`, email)

	// Ortak sunucu Debug=true ile kurulu (integration_test.go).
	resp, err := http.PostForm(itHTTP.URL+"/bulten", url.Values{"email": {email}})
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(raw)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, strings.Contains(body, "/bulten/onay/"),
		"geliştirme kipinde onay bağlantısı dönmeli: %s", body)

	_, _ = itPool.Exec(ctx, `DELETE FROM subscribers WHERE email=$1`, email)
}
