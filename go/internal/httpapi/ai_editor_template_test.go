package httpapi

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

func parseTestTemplates(t *testing.T) *template.Template {
	t.Helper()
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	tmpl, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)
	return tmpl
}

func TestAIEditorResultTemplateRenders(t *testing.T) {
	tmpl := parseTestTemplates(t)
	data := &tmplData{AI: &aiEditorView{
		ContentID: 42, JobID: 7, ActionLabel: "Bunu Haber Yap",
		Result: aieditor.Result{
			Action: "news",
			Fields: map[string]string{
				"title": "Başlık", "spot": "Spot", "body": "Gövde",
			},
			Warnings: []string{"deneme uyarısı"},
			Sources:  []aieditor.Source{{Ref: "S1", Title: "Kaynak", URL: "https://example.com"}},
			Claims:   []aieditor.Claim{{Text: "İddia", Status: "Destekleniyor"}},
		},
	}}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ai_editor_result", data))
	out := buf.String()
	assert.Contains(t, out, "Başlığa uygula")
	assert.Contains(t, out, "Seçili metnin yerine koy")
	assert.Contains(t, out, "Kaynaklar")
	assert.Contains(t, out, "deneme uyarısı")
}

func TestAIEditorPendingTemplateRenders(t *testing.T) {
	tmpl := parseTestTemplates(t)
	data := &tmplData{AI: &aiEditorView{
		ContentID: 42, JobID: 9, ActionLabel: "Araştırarak Haber Yap",
		Status: "Çalışıyor", PollURL: "/admin/icerikler/42/ai/is/9/durum",
		CancelURL: "/admin/icerikler/42/ai/is/9/iptal",
	}}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ai_editor_pending", data))
	out := buf.String()
	assert.Contains(t, out, "data-ai-pending")
	assert.Contains(t, out, "/ai/is/9/durum")
	assert.Contains(t, out, "İşi iptal et")
}

func TestAIEditorHistoryTemplateRenders(t *testing.T) {
	tmpl := parseTestTemplates(t)
	now := time.Now()
	data := &tmplData{AI: &aiEditorView{
		ContentID: 42,
		Jobs: []aiJobView{{
			Job:   repo.AIJob{ID: 1, Action: "news", Status: "completed", CreatedAt: now},
			Label: "Bunu Haber Yap", When: "20.10.2026 12:00", Status: "Tamamlandı",
		}},
		Revisions: []repo.ContentRevision{{
			ID: 3, Version: 4, Title: "Eski başlık", Source: "manuel", CreatedAt: now,
		}},
	}}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "ai_editor_history", data))
	out := buf.String()
	assert.Contains(t, out, "Bunu Haber Yap")
	assert.Contains(t, out, "v4")
	assert.Contains(t, out, "Geri yükle")
}
