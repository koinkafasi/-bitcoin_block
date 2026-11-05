package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AIUsage AI çağrı logudur.
type AIUsage struct {
	ID               int64
	UserID           *int64
	ContentID        *int64
	JobID            *int64
	Provider         string
	Action           string
	Model            *string
	PromptVersion    *string
	PromptTokens     int
	CompletionTokens int
	CostEst          float64
	LatencyMS        int
	CreatedAt        time.Time
}

// LogAIUsage bir AI çağrısını ölçümlendirir (denetim + maliyet raporu için).
func LogAIUsage(ctx context.Context, pool *pgxpool.Pool, u *AIUsage) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO ai_usage_logs
			(user_id, content_id, job_id, provider, action, model, prompt_version,
			 prompt_tokens, completion_tokens, cost_est, latency_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		nullableInt64P(u.UserID), nullableInt64P(u.ContentID), nullableInt64P(u.JobID),
		u.Provider, u.Action, nullableStr(u.Model), nullableStr(u.PromptVersion),
		u.PromptTokens, u.CompletionTokens, u.CostEst, u.LatencyMS,
	).Scan(&id)
	return id, err
}

// UsageRow rapor satırıdır (etiket + değer).
type UsageRow struct {
	Label string
	Value string
}

// AIUsageSummary son 30 gün toplamlarını döndürür.
func AIUsageSummary(ctx context.Context, pool *pgxpool.Pool) ([]UsageRow, error) {
	var totalCalls, totalTokens int
	var totalCost float64
	err := pool.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(SUM(prompt_tokens + completion_tokens),0),
		       COALESCE(SUM(cost_est),0)
		FROM ai_usage_logs WHERE created_at > now() - interval '30 days'`,
	).Scan(&totalCalls, &totalTokens, &totalCost)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT provider, count(*) FROM ai_usage_logs GROUP BY provider ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UsageRow{
		{Label: "Toplam çağrı (30g)", Value: itoa(totalCalls)},
		{Label: "Tahmini token (30g)", Value: itoa(totalTokens)},
		{Label: "Tahmini maliyet (30g)", Value: f2(totalCost)},
	}
	for rows.Next() {
		var p string
		var n int
		if err := rows.Scan(&p, &n); err != nil {
			return nil, err
		}
		out = append(out, UsageRow{Label: "Sağlayıcı: " + p, Value: itoa(n)})
	}
	return out, rows.Err()
}

// AIUsageCSV dışa aktarma için ham satırlar (en son 500).
func AIUsageCSV(ctx context.Context, pool *pgxpool.Pool) ([][]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(u.email,''), COALESCE(c.title,''), provider, action, COALESCE(model,''),
		       prompt_tokens, completion_tokens, cost_est, l.created_at
		FROM ai_usage_logs l
		LEFT JOIN users u ON u.id = l.user_id
		LEFT JOIN contents c ON c.id = l.content_id
		ORDER BY l.created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{{"email", "icerik", "provider", "action", "model", "prompt_tokens", "completion_tokens", "cost_est", "tarih"}}
	for rows.Next() {
		var email, title, provider, action, model string
		var pt, ct int
		var cost float64
		var t time.Time
		if err := rows.Scan(&email, &title, &provider, &action, &model, &pt, &ct, &cost, &t); err != nil {
			return nil, err
		}
		out = append(out, []string{email, title, provider, action, model, itoa(pt), itoa(ct), f2(cost), t.Format("2006-01-02 15:04")})
	}
	return out, rows.Err()
}

func nullableInt64P(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func f2(f float64) string {
	return strconvF(f)
}

func strconvF(f float64) string {
	// basit, 2 haneli
	neg := f < 0
	if neg {
		f = -f
	}
	i := int64(f)
	frac := int64((f-float64(i))*100 + 0.5)
	s := itoa(int(i)) + "." + pad2(frac)
	if neg {
		return "-" + s
	}
	return s
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + itoa(int(n))
	}
	return itoa(int(n))
}
