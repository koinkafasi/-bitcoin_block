package aieditor

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

	"theblock/site/internal/ai"
	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// TestServiceResearchIntegration gerçek Brave araması + DB kaynak kaydını
// doğrular. AI_EDITOR_INTEGRATION=1 ve BRAVE_SEARCH_API_KEY gerekir.
func TestServiceResearchIntegration(t *testing.T) {
	if os.Getenv("AI_EDITOR_INTEGRATION") != "1" || os.Getenv("BRAVE_SEARCH_API_KEY") == "" {
		t.Skip("AI_EDITOR_INTEGRATION=1 + BRAVE_SEARCH_API_KEY gerekli")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	var contentID, authorID int64
	err = pool.QueryRow(ctx, `SELECT id, author_id FROM contents WHERE author_id IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&contentID, &authorID)
	if err != nil {
		t.Skipf("uygun içerik yok: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"test-model","choices":[{"message":{"content":"{\"title\":\"Kaynaklı Başlık\",\"spot\":\"Spot\",\"intro\":\"Giriş\",\"body\":\"Gövde [S1]\",\"source_refs\":[\"S1\"]}"}}],"usage":{"prompt_tokens":50,"completion_tokens":30}}`)
	}))
	defer srv.Close()
	t.Setenv("OMNIROUTE_API_KEY", "test-key")
	t.Setenv("OMNIROUTE_BASE_URL", srv.URL+"/v1")
	t.Setenv("OMNIROUTE_MODEL", "test-model")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, ai.NewClient(logger), logger)
	res, jobID, err := svc.Run(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "research_news",
		Selection: "Türkiye ekonomi son gelişmeler",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Sources)
	assert.Equal(t, "Kaynaklı Başlık", res.Fields["title"])

	sources, err := repo.ListAIJobSources(ctx, pool, jobID)
	require.NoError(t, err)
	assert.NotEmpty(t, sources)

	_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, jobID)
}
