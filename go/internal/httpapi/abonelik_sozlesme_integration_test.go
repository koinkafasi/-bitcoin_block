//go:build integration

package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/auth"
	"theblock/site/internal/config"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
)

func TestAbonelikOdemeSozlesmeOnayiOlmadanReddedilir(t *testing.T) {
	skipIfNoDB(t)
	ctx := t.Context()

	for _, kv := range [][2]string{
		{"api_key", `"test-api-key"`},
		{"secret_key", `"test-secret-key"`},
		{"aktif", `"1"`},
	} {
		_, err := itPool.Exec(ctx,
			`INSERT INTO settings (section,key,value) VALUES ('iyzico',$1,$2::jsonb)
			 ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`, kv[0], kv[1])
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = itPool.Exec(context.Background(), `DELETE FROM settings WHERE section='iyzico'`)
	})

	// Oturum açmış bir üye gerekir: handler önce requireMember çağırıyor.
	var uid int64
	require.NoError(t, itPool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash, role, status)
		 VALUES ('Sözleşme Testi','sozlesme-testi@example.com','x','Üye','aktif')
		 ON CONFLICT (email) DO UPDATE SET status='aktif'
		 RETURNING id`).Scan(&uid))
	sid, err := auth.CreateSession(ctx, itPool, uid, "sozlesme-testi")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = itPool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, sid)
		_, _ = itPool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, uid)
	})

	// Aktif bir katman seç; yoksa test anlamsız.
	var tierSlug string
	if err := itPool.QueryRow(ctx,
		`SELECT slug FROM membership_tiers WHERE is_active LIMIT 1`).Scan(&tierSlug); err != nil {
		t.Skipf("aktif abonelik katmanı yok: %v", err)
	}

	bundle, err := i18n.Load(ctx, itPool)
	require.NoError(t, err)
	srv := httpapi.NewServer(config.Config{
		Env: "test", ListenAddr: "127.0.0.1:0", BaseURL: "http://127.0.0.1",
		Port: "0", DatabaseURL: testDB,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), itPool, bundle, nil)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	gonder := func(onay bool) string {
		form := url.Values{
			"tier":    {tierSlug},
			"cycle":   {"monthly"},
			"tckn":    {"11111111111"},
			"city":    {"İstanbul"},
			"address": {"Test adresi"},
		}
		if onay {
			form.Set("sozlesme_onay", "1")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			ts.URL+"/abonelik/odeme", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sid})
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(b)
	}

	govde := gonder(false)
	assert.Contains(t, govde, "onaylamanız gerekir",
		"onay kutusu işaretlenmeden ödeme reddedilmeli")

	assert.NotContains(t, govde, "Ödeme altyapısı etkin değil",
		"test iyzico'yu etkinleştirememiş; onay kapısı sınanmamış olur")

	// Onay yokken hiçbir kabul kaydı düşmemeli.
	var kayit int
	require.NoError(t, itPool.QueryRow(ctx,
		`SELECT count(*) FROM logs WHERE action='abonelik_sozlesme_onay'`).Scan(&kayit))
	assert.Equal(t, 0, kayit, "onaysız istek kabul kaydı oluşturmamalı")
}
