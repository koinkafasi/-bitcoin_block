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

// TestServiceQueueIntegration queued -> worker -> completed akışını ve iptali
// gerçek PostgreSQL üzerinde doğrular. AI_EDITOR_INTEGRATION=1 gerekir.
func TestServiceQueueIntegration(t *testing.T) {
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
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"test-model","choices":[{"message":{"content":"{\"title\":\"Kuyruk Başlığı\",\"spot\":\"Spot\",\"intro\":\"Giriş\",\"body\":\"Gövde\"}"}}],"usage":{"prompt_tokens":10,"completion_tokens":20}}`)
	}))
	defer srv.Close()
	t.Setenv("OMNIROUTE_API_KEY", "test-key")
	t.Setenv("OMNIROUTE_BASE_URL", srv.URL+"/v1")
	t.Setenv("OMNIROUTE_MODEL", "test-model")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, ai.NewClient(logger), logger)
	jobID, err := svc.Enqueue(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "news",
		Selection: "kuyruk testi",
	})
	require.NoError(t, err)
	job, err := repo.GetAIJob(ctx, pool, jobID)
	require.NoError(t, err)
	assert.Equal(t, "queued", job.Status)

	ok, err := svc.ProcessNext(ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	job, err = repo.GetAIJob(ctx, pool, jobID)
	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status)
	res, err := DecodeResult(job.Result)
	require.NoError(t, err)
	assert.Equal(t, "Kuyruk Başlığı", res.Fields["title"])

	// İkinci iş: kuyruktayken iptal edilirse worker almamalı.
	job2, err := svc.Enqueue(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "news",
		Selection: "iptal testi",
	})
	require.NoError(t, err)
	cancelled, err := repo.CancelAIJob(ctx, pool, job2)
	require.NoError(t, err)
	assert.True(t, cancelled)
	ok, err = svc.ProcessNext(ctx)
	require.NoError(t, err)
	assert.False(t, ok, "iptal edilen iş worker tarafından alınmamalı")
	job2row, err := repo.GetAIJob(ctx, pool, job2)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", job2row.Status)

	// Çökmüş süreçten kalan running iş stale temizliğiyle kuyruğa dönmeli.
	job3, err := svc.Enqueue(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "news",
		Selection: "stale testi",
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE ai_jobs SET status='running', started_at=now()-interval '20 minutes' WHERE id=$1`, job3)
	require.NoError(t, err)
	n, err := repo.RecoverStaleAIJobs(ctx, pool, 15*time.Minute)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1))
	job3row, err := repo.GetAIJob(ctx, pool, job3)
	require.NoError(t, err)
	assert.Equal(t, "queued", job3row.Status)

	for _, id := range []int64{jobID, job2, job3} {
		_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, id)
	}
}
