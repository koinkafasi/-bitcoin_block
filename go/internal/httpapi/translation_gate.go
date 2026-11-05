package httpapi

import (
	"context"
	"strings"

	"theblock/site/internal/repo"
)

// Çeviri kapısı (Faz 120).
//
// 24 dilli bir sitede aynı haberin makine çevirisiyle çoğaltılması, Google'ın
// "scaled content abuse" tanımına en yakın davranıştır: değer katmadan
// üretilmiş, başka yerde de bulunan içerik. Bu kapı, çeviri bir içeriğin
// yayına alınması için o dilde **bir kaynak/atıf** bulunmasını şart koşar.
//
// Kural:
//   - Kök içerik (translation_of yok) → geçer.
//   - Varsayılan dildeki (tr) içerik → geçer.
//   - Çeviri (translation_of var, locale != tr) → `source_url` veya en az bir
//     içerik kaynağı (kaynak ekle) olmalı.
//
// Kaynak, "bu bilgi nereden geldi" sorusunu o dilde de yanıtlar; salt çeviri
// metninden farklı olarak özgün bir katkı gösterir.

// defaultLocale sitenin kök dili.
const defaultLocale = "tr"

// translationGateResult kapı sonucudur.
type translationGateResult struct {
	Allowed bool
	Reason  string
}

// checkTranslationGate içeriğin yayına alınmasının çeviri açısından uygun
// olup olmadığını söyler. Saf mantık repo okumasına bağlıdır; testte
// sourceCount doğrudan verilebilir.
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
