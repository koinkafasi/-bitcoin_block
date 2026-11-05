package aieditor

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/ai"
	"theblock/site/internal/auth"
	"theblock/site/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestServiceIntegration gerçek PostgreSQL şemasını ve iş kaydını doğrular.
// Varsayılan olarak atlanır: AI_EDITOR_INTEGRATION=1 + DATABASE_URL gerekir.
func TestServiceIntegration(t *testing.T) {
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
	err = pool.QueryRow(ctx, `SELECT id, author_id FROM contents WHERE author_id IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&contentID, &authorID)
	if err != nil {
		t.Skipf("uygun içerik yok: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"test-model","choices":[{"message":{"content":"{\"title\":\"Yeni Başlık\",\"spot\":\"Spot\",\"intro\":\"Giriş\",\"body\":\"Gövde\"}"}}],"usage":{"prompt_tokens":10,"completion_tokens":20}}`)
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
		Action:    "news",
		Selection: "seçili metin",
	})
	require.NoError(t, err)
	require.NotZero(t, jobID)
	assert.Equal(t, "Yeni Başlık", res.Fields["title"])
	assert.Equal(t, "test-model", res.Model)

	job, err := repo.GetAIJob(ctx, pool, jobID)
	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status)
	assert.Equal(t, PromptVersion, job.PromptVersion)
	assert.NotEmpty(t, job.Result)

	var usage int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage_logs WHERE job_id=$1`, jobID).Scan(&usage))
	assert.Equal(t, 1, usage)

	// Test kaydını temizle; içerik değişmedi.
	_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, jobID)
}

func TestDecodeResultRoundTrip(t *testing.T) {
	in := Result{Action: "news", Fields: map[string]string{"title": "T"}}
	b, _ := json.Marshal(in)
	out, err := DecodeResult(b)
	require.NoError(t, err)
	assert.Equal(t, "T", out.Fields["title"])
}
