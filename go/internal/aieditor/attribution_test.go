package aieditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/quality"
)

// TestAttributionToQuoteSources alıntı eşleştirmesinin Result.QuoteSources
// alanına doğru aktarıldığını doğrular (yardımcı fonksiyon üzerinden; servis
// DB gerektirdiği için ağır kısmı quality testleri kapsıyor).
func TestAttributionMappingToQuoteSource(t *testing.T) {
	segs := []quality.SourceSegment{
		{StartMS: 0, EndMS: 4000, Speaker: "A", Text: "Yatırım planı 2027 yılında başlayacak."},
		{StartMS: 4000, EndMS: 9000, Speaker: "B", Text: "Bütçe detayları önümüzdeki ay açıklanacak."},
	}
	got := quality.SortQuoteSourcesVerifiedFirst(quality.AttributeQuotes([]string{
		"Bütçe detayları önümüzdeki ay açıklanacak",
		"uydurma cümle burada geçmiyor",
	}, segs))
	require.Len(t, got, 2)

	r := &Result{}
	for _, src := range got {
		r.QuoteSources = append(r.QuoteSources, QuoteSource{
			Quote: src.Quote, StartMS: src.StartMS, EndMS: src.EndMS,
			Speaker: src.Speaker, Verified: src.Verified, Overlap: src.Overlap,
		})
	}
	// Doğrulanmış alıntı önce gelmeli.
	assert.True(t, r.QuoteSources[0].Verified)
	assert.Equal(t, int64(4000), r.QuoteSources[0].StartMS)
	assert.Equal(t, "B", r.QuoteSources[0].Speaker)
	assert.False(t, r.QuoteSources[1].Verified)
}

// TestAttachQuoteSourcesSkipsWithoutMediaID media_id yoksa sessizce
// döndüğünü (panik/hata olmadığını) doğrular.
func TestAttachQuoteSourcesSkipsWithoutMediaID(t *testing.T) {
	s := &Service{}
	r := &Result{Quotes: []string{"bir alıntı burada"}}
	s.attachQuoteSources(t.Context(), nil, r)
	assert.Empty(t, r.QuoteSources)

	s.attachQuoteSources(t.Context(), map[string]any{}, r)
	assert.Empty(t, r.QuoteSources)

	// media_id 0 / geçersiz tip: sessizce dön.
	s.attachQuoteSources(t.Context(), map[string]any{"media_id": float64(0)}, r)
	assert.Empty(t, r.QuoteSources)
	s.attachQuoteSources(t.Context(), map[string]any{"media_id": "abc"}, r)
	assert.Empty(t, r.QuoteSources)
}

// TestAttachQuoteSourcesNoQuotes alıntı yoksa hiçbir şey yapılmadığını
// doğrular.
func TestAttachQuoteSourcesNoQuotes(t *testing.T) {
	s := &Service{}
	r := &Result{}
	s.attachQuoteSources(t.Context(), map[string]any{"media_id": float64(7)}, r)
	assert.Empty(t, r.QuoteSources)
	assert.Zero(t, r.MediaID)
}
