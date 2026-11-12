package httpapi

import (
	"context"
	"strings"

	"theblock/site/internal/repo"
)

// defaultLocale sitenin kök dili.
const defaultLocale = "tr"

// translationGateResult kapı sonucudur.
type translationGateResult struct {
	Allowed bool
	Reason  string
}

func checkTranslationGate(c repo.Content, sourceCount int) translationGateResult {
	if c.TranslationOf == nil {
		return translationGateResult{Allowed: true}
	}
	locale := strings.ToLower(strings.TrimSpace(c.Locale))
	if locale == "" || locale == defaultLocale {
		return translationGateResult{Allowed: true}
	}
	if c.SourceURL != nil && strings.TrimSpace(*c.SourceURL) != "" {
		return translationGateResult{Allowed: true}
	}
	if sourceCount > 0 {
		return translationGateResult{Allowed: true}
	}
	return translationGateResult{
		Allowed: false,
		Reason: "Çeviri yayına alınmadan önce kaynak gerekli: yerel kaynak bağlantısı " +
			"(kaynak URL) veya en az bir kaynak ekleyin. Salt makine çevirisi yayınlanmaz.",
	}
}

// translationGate içerik için kaynak sayısını okuyup kapıyı uygular.
func (s *Server) translationGate(ctx context.Context, c repo.Content) translationGateResult {
	if c.TranslationOf == nil || c.Locale == "" || strings.EqualFold(c.Locale, defaultLocale) {
		return translationGateResult{Allowed: true}
	}
	n := 0
	if srcs, err := repo.ListContentSources(ctx, s.pool, c.ID); err == nil {
		n = len(srcs)
	}
	return checkTranslationGate(c, n)
}
