package aieditor

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/ai"
	"theblock/site/internal/auth"
)

// TestServiceLiveProvider gerçek OmniRoute uç noktasıyla tek bir metin işi
// çalıştırır. AI_LIVE=1 + deploy/.env yüklü olmalıdır; varsayılan atlanır.
func TestServiceLiveProvider(t *testing.T) {
	if os.Getenv("AI_LIVE") != "1" {
		t.Skip("AI_LIVE=1 değil")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || os.Getenv("OMNIROUTE_API_KEY") == "" {
		t.Skip("DATABASE_URL/OMNIROUTE_API_KEY yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	var contentID, authorID int64
	err = pool.QueryRow(ctx, `SELECT id, author_id FROM contents WHERE author_id IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&contentID, &authorID)
	if err != nil {
		t.Skipf("uygun içerik yok: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, ai.NewClient(logger), logger)
	res, jobID, err := svc.Run(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "summary",
		Selection: "Türkiye ekonomisi ve piyasalar hakkında kısa bir değerlendirme.",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Fields["body"])
	assert.Equal(t, "omniroute", res.Provider)
	assert.NotEmpty(t, res.Model)
	t.Logf("job=%d provider=%s model=%s summary_len=%d", jobID, res.Provider, res.Model, len(res.Fields["body"]))

	_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, jobID)
}

// TestServiceLiveResearch gerçek Brave + OmniRoute araştırma akışını doğrular.
func TestServiceLiveResearch(t *testing.T) {
	if os.Getenv("AI_LIVE") != "1" || os.Getenv("BRAVE_SEARCH_API_KEY") == "" {
		t.Skip("AI_LIVE=1 + BRAVE_SEARCH_API_KEY gerekli")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || os.Getenv("OMNIROUTE_API_KEY") == "" {
		t.Skip("DATABASE_URL/OMNIROUTE_API_KEY yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	var contentID, authorID int64
	err = pool.QueryRow(ctx, `SELECT id, author_id FROM contents WHERE author_id IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&contentID, &authorID)
	if err != nil {
		t.Skipf("uygun içerik yok: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, ai.NewClient(logger), logger)
	res, jobID, err := svc.Run(ctx, Request{
		ContentID: contentID,
		User:      auth.User{ID: authorID, Role: "Yönetici"},
		Action:    "research_news",
		Selection: "Türkiye ekonomisi faiz enflasyon son gelişmeler",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Sources)
	assert.NotEmpty(t, res.Fields["body"])
	assert.Equal(t, "omniroute", res.Provider)
	t.Logf("job=%d provider=%s model=%s sources=%d", jobID, res.Provider, res.Model, len(res.Sources))

	_, _ = pool.Exec(ctx, `DELETE FROM ai_usage_logs WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM ai_jobs WHERE id=$1`, jobID)
}
