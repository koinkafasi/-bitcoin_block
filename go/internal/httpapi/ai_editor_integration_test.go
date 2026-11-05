package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/ai"
	"theblock/site/internal/auth"
	"theblock/site/internal/config"
	"theblock/site/internal/i18n"
)

// TestAIEditorHTTPIntegration gerçek şablon + oturum + DB üzerinden AI
// editörünün sonucu habere yazmadığını doğrular. Varsayılan atlanır.
func TestAIEditorHTTPIntegration(t *testing.T) {
	if os.Getenv("AI_EDITOR_INTEGRATION") != "1" {
		t.Skip("AI_EDITOR_INTEGRATION=1 değil")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	var contentID, authorID int64
	var role string
	err = pool.QueryRow(ctx, `
		SELECT c.id, c.author_id, u.role
		FROM contents c JOIN users u ON u.id=c.author_id
		WHERE u.role IN ('Yönetici','Editör','Muhabir','Yazar')
		ORDER BY c.id DESC LIMIT 1`).Scan(&contentID, &authorID, &role)
	if err != nil {
		t.Skipf("uygun içerik/kullanıcı yok: %v", err)
	}

	fakeAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"test-model","choices":[{"message":{"content":"{\"title\":\"AI Test Başlığı\",\"spot\":\"AI spot\",\"intro\":\"AI giriş\",\"body\":\"AI gövde\"}"}}],"usage":{"prompt_tokens":10,"completion_tokens":20}}`)
	}))
	defer fakeAI.Close()
	t.Setenv("OMNIROUTE_API_KEY", "test-key")
	t.Setenv("OMNIROUTE_BASE_URL", fakeAI.URL+"/v1")
	t.Setenv("OMNIROUTE_MODEL", "test-model")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bundle, err := i18n.Load(ctx, pool)
	require.NoError(t, err)
	srv := NewServer(config.Config{Env: "test", BaseURL: "http://example.test"}, logger, pool, bundle, ai.NewClient(logger))
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	sid, err := auth.CreateSession(ctx, pool, authorID, "ai-editor-test")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, sid) })
	cookie := &http.Cookie{Name: auth.CookieName, Value: sid}

	adminURL := ts.URL + "/admin/icerikler/" + itoa64(contentID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, adminURL, nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `id="aiEditor"`)
	assert.Contains(t, string(body), `data-ai-open="news"`)

	beforeTitle, beforeVersion := contentState(t, ctx, pool, contentID)

	form := url.Values{"action": {"news"}, "selection": {"seçili metin"}}
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, adminURL+"/ai/eylem", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(cookie)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "AI Test Başlığı")
	assert.Contains(t, string(body), "Seçili metnin yerine koy")

	afterTitle, afterVersion := contentState(t, ctx, pool, contentID)
	assert.Equal(t, beforeTitle, afterTitle)
	assert.Equal(t, beforeVersion, afterVersion)

	// Farklı bir Yazar/Muhabir aynı içerikte AI işlemi yapamamalı.
	var otherID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM users WHERE role IN ('Yazar','Muhabir') AND id<>$1 LIMIT 1`, authorID).Scan(&otherID)
	if otherID > 0 {
		osid, serr := auth.CreateSession(ctx, pool, otherID, "ai-editor-forbidden-test")
		require.NoError(t, serr)
		defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id=$1`, osid) }()
		req, _ = http.NewRequestWithContext(ctx, http.MethodPost, adminURL+"/ai/eylem", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: osid})
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Contains(t, string(body), "yetkiniz yok")
	}

	var jobID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM ai_jobs WHERE content_id=$1 ORDER BY id DESC LIMIT 1`, contentID).Scan(&jobID)
	if jobID > 0 {
		// Test havuzu defer ile kapanmadan önce temizle.
		_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, jobID)
		_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, jobID)
	}
}

func contentState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id int64) (string, int) {
	t.Helper()
	var title string
	var version int
	require.NoError(t, pool.QueryRow(ctx, `SELECT title, version FROM contents WHERE id=$1`, id).Scan(&title, &version))
	return title, version
}
