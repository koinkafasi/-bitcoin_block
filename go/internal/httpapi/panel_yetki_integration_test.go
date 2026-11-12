//go:build integration

package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"theblock/site/internal/auth"
	"theblock/site/internal/config"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
)

// panelIstemcisi verilen kullanıcı için oturum açar ve çerezli istemci döndürür.
func panelIstemcisi(t *testing.T, base string, uid int64, scope string) *http.Client {
	t.Helper()
	ctx := t.Context()
	sid, err := auth.CreateSessionScoped(ctx, itPool, uid, "yetki-testi", scope)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = itPool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, sid)
	})
	jar := &tekCerez{sid: sid}
	return &http.Client{
		Transport: jar,
		// Yönlendirmeleri izleme: kapı 302 ile /admin/giris'e atarsa da
		// bunu "erişim reddedildi" sayarız.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// tekCerez tb_session çerezini her isteğe ekler.
type tekCerez struct {
	sid  string
	base http.RoundTripper
}

func (c *tekCerez) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: c.sid})
	rt := c.base
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(r)
}

func TestPanelYetkiMatrisi(t *testing.T) {
	skipIfNoDB(t)
	ctx := t.Context()

	kullanici := func(email, rol string) int64 {
		var id int64
		require.NoError(t, itPool.QueryRow(ctx,
			`INSERT INTO users (name, email, password_hash, role, status)
			 VALUES ('Yetki Testi',$1,'x',$2,'aktif')
			 ON CONFLICT (email) DO UPDATE SET role=EXCLUDED.role, status='aktif'
			 RETURNING id`, email, rol).Scan(&id))
		t.Cleanup(func() {
			_, _ = itPool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
		})
		return id
	}

	uye := kullanici("yetki-uye@example.com", "Üye")
	yazar := kullanici("yetki-yazar@example.com", "Yazar")
	editor := kullanici("yetki-editor@example.com", "Editör")
	yonetici := kullanici("yetki-yonetici@example.com", "Yönetici")

	bundle, err := i18n.Load(ctx, itPool)
	require.NoError(t, err)
	srv := httpapi.NewServer(config.Config{
		Env: "test", ListenAddr: "127.0.0.1:0", BaseURL: "http://127.0.0.1",
		Port: "0", DatabaseURL: testDB,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), itPool, bundle, nil)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	istemci := func(uid int64, scope string) *http.Client {
		return panelIstemcisi(t, ts.URL, uid, scope)
	}

	type vaka struct {
		ad    string
		path  string
		rol   string // "" = oturumsuz
		uid   int64
		scope string
		red   bool
	}
	vakalar := []vaka{
		// Oturumsuz: girişe yönlendirilir (403 değil, 302).
		{ad: "oturumsuz /admin", path: "/admin", red: true},

		// K1'in kalbi: Üye hiçbir panele giremez.
		{ad: "Üye /admin", path: "/admin", rol: "Üye", uid: uye, scope: auth.ScopePanel, red: true},
		{ad: "Üye /admin/icerikler", path: "/admin/icerikler", rol: "Üye", uid: uye, scope: auth.ScopePanel, red: true},
		{ad: "Üye /admin/ayarlar", path: "/admin/ayarlar", rol: "Üye", uid: uye, scope: auth.ScopePanel, red: true},

		// Üye çerezle (okuyucu kapsamı) de aynı.
		{ad: "Üye (uye kapsamı) /admin", path: "/admin", rol: "Üye", uid: uye, scope: auth.ScopeUye, red: true},

		// Yazar: içerik alanı açık, site ayarı kapalı.
		{ad: "Yazar /admin/icerikler", path: "/admin/icerikler", rol: "Yazar", uid: yazar, scope: auth.ScopePanel},
		{ad: "Yazar /admin/ayarlar", path: "/admin/ayarlar", rol: "Yazar", uid: yazar, scope: auth.ScopePanel, red: true},

		// Editör: düzen alanı açık, ödeme/site ayarı kapalı.
		{ad: "Editör /admin/tasarim", path: "/admin/tasarim", rol: "Editör", uid: editor, scope: auth.ScopePanel},
		{ad: "Editör /admin/ayarlar", path: "/admin/ayarlar", rol: "Editör", uid: editor, scope: auth.ScopePanel, red: true},
		{ad: "Editör /admin/members", path: "/admin/members", rol: "Editör", uid: editor, scope: auth.ScopePanel, red: true},

		// Yönetici: her şey açık.
		{ad: "Yönetici /admin/ayarlar", path: "/admin/ayarlar", rol: "Yönetici", uid: yonetici, scope: auth.ScopePanel},

		// KAPSAM KAPISI: rolü yönetici olsa bile okuyucu kapısından açılmış
		// oturum panele giremez (yeniden kimlik doğrulama gerekir).
		{ad: "Yönetici (uye kapsamı) /admin", path: "/admin", rol: "Yönetici", uid: yonetici, scope: auth.ScopeUye, red: true},
		{ad: "Yönetici (uye kapsamı) /admin/ayarlar", path: "/admin/ayarlar", rol: "Yönetici", uid: yonetici, scope: auth.ScopeUye, red: true},
	}

	for _, v := range vakalar {
		t.Run(v.ad, func(t *testing.T) {
			var c *http.Client
			if v.uid != 0 {
				c = istemci(v.uid, v.scope)
			} else {
				// Oturumsuz: yönlendirmeyi izleme, 302'yi doğrudan gör.
				c = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
					return http.ErrUseLastResponse
				}}
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+v.path, nil)
			require.NoError(t, err)
			// CSRF ara katmanı POST'larda Origin arar; GET etkilenmez.
			resp, err := c.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			switch {
			case v.uid == 0:
				// Oturumsuz istek girişe yönlendirilir.
				require.Equal(t, http.StatusFound, resp.StatusCode, "oturumsuz istek girişe yönlenmeli")
			case v.red:
				require.Equal(t, http.StatusForbidden, resp.StatusCode,
					"%s: %s erişimi 403 olmalı (rol=%s kapsam=%s)", v.ad, v.path, v.rol, v.scope)
			default:
				require.NotEqual(t, http.StatusForbidden, resp.StatusCode,
					"%s: %s erişimi reddedilmemeli (rol=%s)", v.ad, v.path, v.rol)
			}
		})
	}
}
