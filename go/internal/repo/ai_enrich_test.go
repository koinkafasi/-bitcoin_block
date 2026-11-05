package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldUseEnrichedSummary(t *testing.T) {
	title := "Zonguldak'ta 2 katlı bina çöktü"
	assert.True(t, shouldUseEnrichedSummary("", title, "Enkaz altındaki iki kişiden biri yaralı kurtarıldı."))
	assert.True(t, shouldUseEnrichedSummary(title, title, "Enkaz altındaki iki kişiden biri yaralı kurtarıldı."))
	assert.False(t, shouldUseEnrichedSummary("Mevcut açıklama yeterince uzun ve başlıktan farklı bir metin içeriyor; bu nedenle yeni kısa açıklama ile değiştirilmemeli.", title, "Yeni açıklama."))
	assert.False(t, shouldUseEnrichedSummary(title, title, ""))
}

func TestShouldUpgradeHTMLTitle(t *testing.T) {
	assert.True(t, shouldUpgradeHTMLTitle("Kısa başlık", "Çok daha açıklayıcı ve uzun bir haber başlığı"))
	assert.False(t, shouldUpgradeHTMLTitle("Aynı haber başlığı", "Aynı haber başlığı"))
	assert.False(t, shouldUpgradeHTMLTitle("Eski başlık", "Kısa"))
}

func TestEnrichedCategoryID(t *testing.T) {
	cats := map[string]int64{"gundem": 1, "ekonomi": 2, "spor": 3, "dunya": 4, "yasam": 5}
	ekonomi := int64(2)

	got, changed := enrichedCategoryID("Spor", &ekonomi, cats)
	assert.True(t, changed)
	assert.NotNil(t, got)
	assert.Equal(t, int64(3), *got)

	got, changed = enrichedCategoryID("Ekonomi", &ekonomi, cats)
	assert.False(t, changed)
	assert.Equal(t, ekonomi, *got)

	got, changed = enrichedCategoryID("Bilinmeyen Bölüm", &ekonomi, cats)
	assert.False(t, changed)
	assert.Equal(t, ekonomi, *got)

	got, changed = enrichedCategoryID("Dünya", nil, cats)
	assert.True(t, changed)
	assert.Equal(t, int64(4), *got)
}
