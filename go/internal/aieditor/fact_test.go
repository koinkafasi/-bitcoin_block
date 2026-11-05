package aieditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const sayiTranscript = `Konuşmacı A: Yatırım planı 2027 yılında başlayacak. Bütçe 480 milyon lira
olarak açıklandı ve çalışmalar 18 ay sürecek.`

// TestFactWarningsNumbersVerified kaynaktaki sayılarla yazılmış taslakta
// sayı uyarısı olmadığını doğrular.
func TestFactWarningsNumbersVerified(t *testing.T) {
	r := &Result{Fields: map[string]string{
		"title": "2027 Yatırım Planı",
		"spot":  "Bütçe 480 milyon lira.",
		"body":  "Çalışmalar 2027 yılında başlayacak ve 18 ay sürecek.",
	}}
	warns := factWarnings(r, sayiTranscript)
	assert.Empty(t, warns, "kaynaktaki sayılar uyarı üretmemeli: %v", warns)
	assert.Empty(t, r.UnverifiedNumbers)
}

// TestFactWarningsDetectsFabricatedNumber uydurma sayının işaretlendiğini
// ve Result.UnverifiedNumbers alanına yazıldığını doğrular.
func TestFactWarningsDetectsFabricatedNumber(t *testing.T) {
	r := &Result{Fields: map[string]string{
		"title": "Yatırım Planı",
		"body":  "Bütçe 840 milyon lira olarak açıklandı.",
	}}
	warns := factWarnings(r, sayiTranscript)
	assert.Len(t, warns, 1)
	assert.Contains(t, warns[0], "1 sayı")
	assert.Contains(t, warns[0], "840")
	assert.Equal(t, []string{"840"}, r.UnverifiedNumbers)
}

// TestFactWarningsBothGates alıntı ve sayı kapılarının birlikte çalıştığını
// doğrular (iki ayrı uyarı).
func TestFactWarningsBothGates(t *testing.T) {
	r := &Result{
		Fields: map[string]string{
			"title": "Plan",
			"body":  "Bütçe 840 milyon lira olacak.",
		},
		Quotes: []string{"Bakan ayrıca yeni bir vergi getirileceğini duyurdu"},
	}
	warns := factWarnings(r, sayiTranscript)
	assert.Len(t, warns, 2, "iki kapı da uyarı vermeli: %v", warns)
	assert.Len(t, r.UnverifiedQuotes, 1)
	assert.Equal(t, []string{"840"}, r.UnverifiedNumbers)
}

// TestFactWarningsNoNumbersNoQuotes hiç sayı/alıntı yoksa sessiz kalmalı.
func TestFactWarningsNoNumbersNoQuotes(t *testing.T) {
	r := &Result{Fields: map[string]string{"body": "Toplantı yapıldı ve kararlar alındı."}}
	assert.Empty(t, factWarnings(r, sayiTranscript))
}

// TestDraftNumbersTextIncludesAllFields başlık/spot/gövdenin birleştiğini
// doğrular: başlıktaki bir sayı da denetlenmeli.
func TestDraftNumbersTextIncludesAllFields(t *testing.T) {
	r := &Result{Fields: map[string]string{
		"title": "2027 bütçesi",
		"spot":  "480 milyon lira",
		"body":  "18 ay sürecek",
	}}
	text := draftNumbersText(r)
	assert.Contains(t, text, "2027")
	assert.Contains(t, text, "480")
	assert.Contains(t, text, "18")
}

// TestFactWarningsTitleNumberChecked başlıktaki uydurma sayı yakalanmalı.
func TestFactWarningsTitleNumberChecked(t *testing.T) {
	r := &Result{Fields: map[string]string{
		"title": "2035 Yatırım Planı",
		"body":  "Plan 2027 yılında başlayacak.",
	}}
	warns := factWarnings(r, sayiTranscript)
	assert.Len(t, warns, 1)
	assert.Equal(t, []string{"2035"}, r.UnverifiedNumbers)
}
