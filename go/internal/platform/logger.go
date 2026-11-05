package platform

import (
	"log/slog"
	"os"
)

// NewLogger yapısal (JSON) bir slog logger döndürür; prod'da JSON, debug'da saatli.
func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if os.Getenv("APP_ENV") == "production" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}
