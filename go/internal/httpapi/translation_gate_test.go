package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

func ptrInt64Local(v int64) *int64    { return &v }
func ptrStringLocal(v string) *string { return &v }

func TestTranslationGateRootContentAllowed(t *testing.T) {
	c := repo.Content{Locale: "de"}
	g := checkTranslationGate(c, 0)
	assert.True(t, g.Allowed, "kök içerik (çeviri değil) her dilde yayınlanabilir")
}

func TestTranslationGateDefaultLocaleAllowed(t *testing.T) {
	c := repo.Content{Locale: "tr", TranslationOf: ptrInt64Local(5)}
	g := checkTranslationGate(c, 0)
	assert.True(t, g.Allowed, "varsayılan dil kaynak sayılmaz")
}

func TestTranslationGateBlocksUnsourcedTranslation(t *testing.T) {
	c := repo.Content{Locale: "en", TranslationOf: ptrInt64Local(5)}
	g := checkTranslationGate(c, 0)
	assert.False(t, g.Allowed, "kaynaksız çeviri yayına alınamaz")
	assert.Contains(t, g.Reason, "kaynak gerekli")
}

func TestTranslationGateSourceURLAllows(t *testing.T) {
	c := repo.Content{
		Locale: "en", TranslationOf: ptrInt64Local(5),
		SourceURL: ptrStringLocal("https://example.com/local-report"),
	}
	g := checkTranslationGate(c, 0)
	assert.True(t, g.Allowed)
}

func TestTranslationGateContentSourcesAllow(t *testing.T) {
	c := repo.Content{Locale: "ar", TranslationOf: ptrInt64Local(5)}
	g := checkTranslationGate(c, 2)
	assert.True(t, g.Allowed)
}

func TestTranslationGateEmptyLocaleAllowed(t *testing.T) {
	// Locale boş eski kayıtlar: geriye dönük uyumluluk için geçer.
	c := repo.Content{TranslationOf: ptrInt64Local(5)}
	g := checkTranslationGate(c, 0)
	assert.True(t, g.Allowed)
}

func TestTranslationGateCaseInsensitiveLocale(t *testing.T) {
	c := repo.Content{Locale: "TR", TranslationOf: ptrInt64Local(5)}
	g := checkTranslationGate(c, 0)
	assert.True(t, g.Allowed, "büyük/küçük harf farkı varsayılan dili bozmamalı")
}

func TestTranslationGateBlankSourceURLIgnored(t *testing.T) {
	c := repo.Content{
		Locale: "en", TranslationOf: ptrInt64Local(5),
		SourceURL: ptrStringLocal("   "),
	}
	g := checkTranslationGate(c, 0)
	assert.False(t, g.Allowed, "boşluktan oluşan kaynak URL kaynak sayılmaz")
}
