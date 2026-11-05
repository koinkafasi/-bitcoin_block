package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/aieditor"
)

// renderQuoteResult aieditor sonucunu ai_editor_result şablonuyla işler.
// parseTestTemplates (ai_editor_template_test.go) tüm şablonları ayrıştırır;
// burada yalnız ilgili parça çalıştırılır.
func renderQuoteResult(t *testing.T, res aieditor.Result) string {
	t.Helper()
	tmpl := parseTestTemplates(t)
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ai_editor_result", map[string]any{
		"AI": &aiEditorView{
			ContentID: 42, JobID: 7,
			ActionLabel: "Ses Kaydından Taslak", Result: res,
		},
	}))
	return buf.String()
}

// TestAIEditorResultRendersQuoteSources alıntı kaynaklarının saniye,
// konuşmacı, ses kaydı kimliği ve doğrulama işaretiyle basıldığını doğrular.
func TestAIEditorResultRendersQuoteSources(t *testing.T) {
	res := aieditor.Result{
		MediaID: 1337,
		Fields:  map[string]string{"title": "Başlık", "body": "Gövde"},
		QuoteSources: []aieditor.QuoteSource{
			{Quote: "Yatırım planı 2027 yılında başlayacak", StartMS: 0, EndMS: 4000,
				Speaker: "A", Verified: true, Overlap: 1},
			{Quote: "Bütçe detayları önümüzdeki ay açıklanacak", StartMS: 65000, EndMS: 70000,
				Speaker: "B", Verified: true, Overlap: 1},
			{Quote: "Bakan yeni bir vergi duyurdu", Verified: false},
		},
	}
	out := renderQuoteResult(t, res)

	assert.Contains(t, out, `class="ai-quote-sources"`, "kaynak bloğu basılmalı")
	assert.Contains(t, out, `data-quote-seek="0"`, "ilk saniye 0 olmalı")
	assert.Contains(t, out, `data-quote-seek="65000"`, "65. saniye basılmalı")
	assert.Contains(t, out, `data-quote-media="1337"`, "ses kaydı kimliği taşınmalı")
	assert.Contains(t, out, `>0:00<`, "0 ms 0:00 olarak gösterilmeli")
	assert.Contains(t, out, `>1:05<`, "65000 ms 1:05 olarak gösterilmeli")
	assert.Contains(t, out, `class="ai-quote-speaker">A<`)
	assert.Contains(t, out, `class="ai-quote-speaker">B<`)

	// Doğrulanmamış alıntı: kırmızı sınıf + data-quote-bad + uydurma saniye yok.
	assert.Contains(t, out, "ai-quote-src-bad")
	assert.Contains(t, out, `data-quote-bad="1"`)
	assert.Contains(t, out, "kaynak yok")

	// Doğrulama işaretleri.
	assert.Contains(t, out, "✓")
	assert.Contains(t, out, "✕")
}

// TestAIEditorResultWithoutQuoteSources alıntı kaynağı yoksa bloğun
// basılmadığını doğrular (gereksiz boş kart olmamalı).
func TestAIEditorResultWithoutQuoteSources(t *testing.T) {
	out := renderQuoteResult(t, aieditor.Result{
		Fields: map[string]string{"title": "B", "body": "G"},
	})
	assert.NotContains(t, out, "ai-quote-sources")
	assert.NotContains(t, out, "data-quote-seek")
}

// TestAIEditorResultUnverifiedLists doğrulanamayan alıntı ve sayı
// bloklarının listelendiğini doğrular.
func TestAIEditorResultUnverifiedLists(t *testing.T) {
	out := renderQuoteResult(t, aieditor.Result{
		Fields:            map[string]string{"title": "B", "body": "G"},
		UnverifiedQuotes:  []string{"uydurma alıntı"},
		UnverifiedNumbers: []string{"840"},
	})
	assert.Contains(t, out, "Doğrulanamayan alıntılar")
	assert.Contains(t, out, "uydurma alıntı")
	assert.Contains(t, out, "Doğrulanamayan sayılar")
	assert.Contains(t, out, "<b>840</b>")
	assert.Contains(t, out, "ai-quote-warn")
}
