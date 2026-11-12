package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/auth"
	"theblock/site/internal/config"
)

func cikisSunucu(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	s := &Server{
		mux:    http.NewServeMux(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    config.Config{Env: "test", BaseURL: "https://theblock.com.tr"},
	}
	s.routes()
	return s, s.mux
}

// GET /admin/cikis rota olarak KAYITLI olmalı — asıl hata buydu.
func TestCikisGETRotasiVar(t *testing.T) {
	_, mux := cikisSunucu(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/cikis", nil)
	req.Header.Set("Referer", "https://theblock.com.tr/admin/icerikler")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusNotFound, rec.Code,
		"GET /admin/cikis 404 dönmemeli (sidebar bu adrese GET ile gidiyor)")
}

// POST rotası korunmalı (form / htmx yolu).
func TestCikisPOSTRotasiVar(t *testing.T) {
	_, mux := cikisSunucu(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/cikis", nil)
	req.Header.Set("Origin", "https://theblock.com.tr")
	req.Host = "theblock.com.tr"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusNotFound, rec.Code, "POST /admin/cikis kayıtlı olmalı")
}

func TestCikisCaprazKaynakGETReddedilir(t *testing.T) {
	s := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		pool:   nil, // guard çalışmazsa panic olur → test yakalar
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/cikis", nil)
	req.Host = "theblock.com.tr"
	req.Header.Set("Referer", "https://kotu-site.example/sayfa")
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() { s.handleLogout(rec, req) },
		"çapraz kaynak istekte DB'ye gidilmemeli")
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/admin", rec.Header().Get("Location"),
		"çapraz kaynak istek /admin/giris'e değil, panele dönmeli")
	for _, c := range rec.Result().Cookies() {
		assert.NotEqual(t, auth.CookieName, c.Name,
			"çapraz kaynak istek çerez silmemeli")
	}
}

// Aynı kaynaktan gelen GET (sidebar tıklaması / adres çubuğu) çıkışı
// tamamlamalı: oturum silinir, İKİ çerez de temizlenir.
func TestCikisIkiCereziDeTemizler(t *testing.T) {
	pool := cikisTestPool(t)
	ctx := context.Background()
	s := &Server{pool: pool, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	sid, err := auth.CreateSessionScoped(ctx, pool, 1, "cikis-testi", auth.ScopePanel)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, sid)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/cikis", nil)
	req.Host = "theblock.com.tr"
	req.Header.Set("Referer", "https://theblock.com.tr/admin/yeniden-yazim")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sid})
	req.AddCookie(&http.Cookie{Name: memberCookieName, Value: "uye@ornek.com.imza"})
	rec := httptest.NewRecorder()

	s.handleLogout(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/admin/giris", rec.Header().Get("Location"))

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE id=$1`, sid).Scan(&n))
	assert.Equal(t, 0, n, "oturum veritabanından silinmeli")

	silinen := map[string]bool{}
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 || c.Value == "" {
			silinen[c.Name] = true
		}
	}
	assert.True(t, silinen[auth.CookieName], "tb_session çerezi silinmeli")
	assert.True(t, silinen[memberCookieName],
		"tb_uye çerezi de silinmeli — yoksa site kullanıcıyı hâlâ üye sayar (ikinci çıkış hatası)")
}

// htmx ile gelen POST'ta tam sayfa gezinme istenir (HX-Redirect).
func TestCikisHTMXteTamSayfaYonlendirir(t *testing.T) {
	pool := cikisTestPool(t)
	ctx := context.Background()
	s := &Server{pool: pool, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	sid, err := auth.CreateSessionScoped(ctx, pool, 1, "cikis-htmx", auth.ScopePanel)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, sid)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/cikis", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Origin", "https://theblock.com.tr")
	req.Host = "theblock.com.tr"
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sid})
	rec := httptest.NewRecorder()

	s.handleLogout(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "/admin/giris", rec.Header().Get("HX-Redirect"),
		"htmx yanıtı tam sayfa geçiş istemeli")
}

// cikisTestPool gerçek DB ister; DB'ye YAZAR. Varsayılan kapalı.
func cikisTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("CIKIS_TEST") != "1" {
		t.Skip("CIKIS_TEST=1 değil (oturum yazan test)")
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
	return pool
}
