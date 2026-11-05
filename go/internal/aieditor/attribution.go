package aieditor

import (
	"context"

	"theblock/site/internal/quality"
	"theblock/site/internal/repo"
)

// attachQuoteSources her alıntıyı transkript segmentine bağlar (Faz 124).
//
// `extra["media_id"]` ses taslağı akışında doldurulur. Segmentler yüklenir,
// alıntının en iyi örtüştüğü segment bulunur ve saniyesi/kullanıcısı sonuca
// yazılır. Böylece editör alıntının kaynağını görüp kaydı oradan dinleyebilir.
//
// Hata durumunda sessizce döner: alıntı listesi yine gösterilir, yalnız
// saniye bağlantısı olmaz.
func (s *Service) attachQuoteSources(ctx context.Context, extra map[string]any, r *Result) {
	if len(r.Quotes) == 0 || extra == nil {
		return
	}
	mediaID := int64(0)
	switch v := extra["media_id"].(type) {
	case float64:
		mediaID = int64(v)
	case int64:
		mediaID = v
	case int:
		mediaID = int64(v)
	}
	if mediaID <= 0 {
		return
	}
	tr, err := repo.GetAudioTranscriptByMedia(ctx, s.pool, mediaID)
	if err != nil || len(tr.Segments) == 0 {
		return
	}
	segs := make([]quality.SourceSegment, 0, len(tr.Segments))
	for _, seg := range tr.Segments {
		segs = append(segs, quality.SourceSegment{
			StartMS: seg.StartMS, EndMS: seg.EndMS,
			Speaker: seg.Speaker, Text: seg.Text,
		})
	}
	sources := quality.SortQuoteSourcesVerifiedFirst(quality.AttributeQuotes(r.Quotes, segs))
	if len(sources) == 0 {
		return
	}
	r.MediaID = mediaID
	for _, src := range sources {
		r.QuoteSources = append(r.QuoteSources, QuoteSource{
			Quote: src.Quote, StartMS: src.StartMS, EndMS: src.EndMS,
			Speaker: src.Speaker, Verified: src.Verified, Overlap: src.Overlap,
		})
	}
}
