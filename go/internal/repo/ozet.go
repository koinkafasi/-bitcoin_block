package repo

import (
	"context"
	"strings"
)

func AiOzet(ctx context.Context, pool DB, contentID int64) string {
	var s *string
	if err := pool.QueryRow(ctx,
		`SELECT ai_summary FROM contents WHERE id = $1`, contentID).Scan(&s); err != nil || s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
