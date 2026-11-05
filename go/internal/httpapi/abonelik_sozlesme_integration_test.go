//go:build integration

// Abonelik ödemesinde ön bilgilendirme / mesafeli satış sözleşmesi onayının
// SUNUCU tarafında zorunlu olduğunu doğrular.
//
// Neden sunucu tarafı: formdaki `required` yalnız tarayıcıda çalışır. İstek
// doğrudan gönderilirse (curl, bot, eski sekme) onay alınmadan sözleşme
// kurulurdu. Mesafeli Sözleşmeler Yönetmeliği uyarınca sağlayıcı, sözleşme
// kurulmadan önce ön bilgilendirme yapmak zorundadır; tüketici cayma hakkı
// konusunda gereği gibi bilgilendirilmezse 14 günlük süre işlemez ve cayma
// hakkı sürenin bittiği tarihten bir yıl sonrasına kadar kullanılabilir.
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

	// iyzico'yu test veritabanında etkinleştir: onay kapısı, ödeme altyapısı
	// kontrolünden SONRA çalışıyor. Etkin değilse handler daha erken dönerdi
	// ve test, onay kapısını hiç sınamamış olurdu.
	//
	// Anahtarlar sahte ve kasıtlı olarak geçersiz: testin NEGATİF yolu
	// (onaysız istek) iyzico'ya hiç ulaşmaz, dolayısıyla dış ağa çıkılmaz.
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
	// İddia hata MESAJINA özgü olmalı. "Mesafeli Satış Sözleşmesi" ifadesi
	// aramak yetmez: abonelikHata tüm abonelik sayfasını yeniden basıyor ve
	// o sayfadaki onay kutusunun etiketinde aynı ifade zaten geçiyor — kapı
	// kaldırılsa bile test geçerdi (bu yanlış güven bir kez yaşandı).
	assert.Contains(t, govde, "onaylamanız gerekir",
		"onay kutusu işaretlenmeden ödeme reddedilmeli")

	// Kapının gerçekten onay yokluğundan ötürü kapandığını doğrula: ödeme
	// altyapısı kapalı olsaydı aynı istek "Ödeme altyapısı etkin değil"
	// derdi ve test yanlış yere güvenirdi.
	assert.NotContains(t, govde, "Ödeme altyapısı etkin değil",
		"test iyzico'yu etkinleştirememiş; onay kapısı sınanmamış olur")

	// Onay yokken hiçbir kabul kaydı düşmemeli.
	var kayit int
	require.NoError(t, itPool.QueryRow(ctx,
		`SELECT count(*) FROM logs WHERE action='abonelik_sozlesme_onay'`).Scan(&kayit))
	assert.Equal(t, 0, kayit, "onaysız istek kabul kaydı oluşturmamalı")
}
