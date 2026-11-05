package quality

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var attrSegments = []SourceSegment{
	{StartMS: 0, EndMS: 4000, Speaker: "A", Text: "Yatırım planı 2027 yılında başlayacak."},
	{StartMS: 4000, EndMS: 9000, Speaker: "B", Text: "Bütçe detayları önümüzdeki ay açıklanacak."},
	{StartMS: 9000, EndMS: 14000, Speaker: "A", Text: "Bu konuda henüz resmi bir açıklama yapılmadı."},
}

func TestAttributeQuotesFindsSegment(t *testing.T) {
	got := AttributeQuotes([]string{"Bütçe detayları önümüzdeki ay açıklanacak"}, attrSegments)
	require.Len(t, got, 1)
	assert.True(t, got[0].Verified)
	assert.Equal(t, 1, got[0].Segment)
	assert.Equal(t, int64(4000), got[0].StartMS)
	assert.Equal(t, int64(9000), got[0].EndMS)
	assert.Equal(t, "B", got[0].Speaker)
	assert.InDelta(t, 1.0, got[0].Overlap, 0.001)
}

func TestAttributeQuotesMarkUnverified(t *testing.T) {
	got := AttributeQuotes([]string{"Bakan yeni bir vergi getireceğini duyurdu"}, attrSegments)
	require.Len(t, got, 1)
	assert.False(t, got[0].Verified)
	assert.Equal(t, -1, got[0].Segment)
	assert.Zero(t, got[0].StartMS)
}

func TestAttributeQuotesPicksBestSegment(t *testing.T) {
	// Aynı sözcükler iki segmentte de var; en iyi örtüşen seçilmeli.
	segs := []SourceSegment{
		{StartMS: 0, EndMS: 1000, Speaker: "A", Text: "Plan başlayacak."},
		{StartMS: 1000, EndMS: 5000, Speaker: "B", Text: "Yatırım planı 2027 yılında başlayacak ve sürecek."},
	}
	got := AttributeQuotes([]string{"Yatırım planı 2027 yılında başlayacak"}, segs)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Segment)
	assert.Equal(t, "B", got[0].Speaker)
}

func TestAttributeQuotesToleratesPunctuation(t *testing.T) {
	got := AttributeQuotes([]string{"“Yatırım planı 2027 yılında başlayacak!”"}, attrSegments)
	require.Len(t, got, 1)
	assert.True(t, got[0].Verified)
	assert.Equal(t, 0, got[0].Segment)
}

func TestAttributeQuotesEmptyInputs(t *testing.T) {
	assert.Nil(t, AttributeQuotes(nil, attrSegments))
	assert.Nil(t, AttributeQuotes([]string{"  "}, attrSegments))
	got := AttributeQuotes([]string{"Yatırım planı 2027 yılında başlayacak"}, nil)
	require.Len(t, got, 1)
	assert.False(t, got[0].Verified)
}

func TestAttributeQuotesMultiple(t *testing.T) {
	got := AttributeQuotes([]string{
		"Yatırım planı 2027 yılında başlayacak",
		"uydurma bir cümle burada",
		"Bütçe detayları önümüzdeki ay açıklanacak",
	}, attrSegments)
	require.Len(t, got, 3)
	assert.True(t, got[0].Verified)
	assert.False(t, got[1].Verified)
	assert.True(t, got[2].Verified)
	assert.Equal(t, 1, got[2].Segment)
}

func TestSortQuoteSourcesVerifiedFirst(t *testing.T) {
	in := []QuoteSource{
		{Quote: "c", Verified: false},
		{Quote: "b", Verified: true, StartMS: 5000},
		{Quote: "a", Verified: true, StartMS: 1000},
	}
	got := SortQuoteSourcesVerifiedFirst(in)
	require.Len(t, got, 3)
	assert.True(t, got[0].Verified)
	assert.Equal(t, int64(1000), got[0].StartMS)
	assert.True(t, got[1].Verified)
	assert.False(t, got[2].Verified)
	// Girdi değişmemeli.
	assert.False(t, in[0].Verified)
}
