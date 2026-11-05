package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDetailInputTipTap(t *testing.T) {
	raw := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Merhaba"}]}]}`
	got := parseDetailInput(raw)
	assert.True(t, isTipTapDetail(got))

	plain := parseDetailInput("Birinci paragraf\n\nİkinci paragraf")
	assert.False(t, isTipTapDetail(plain))
	assert.JSONEq(t, `[{"type":"p","text":"Birinci paragraf"},{"type":"p","text":"İkinci paragraf"}]`, string(plain))
}

func TestRenderTipTapHTMLSafe(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Başlık"}]},
		{"type":"paragraph","content":[
			{"type":"text","text":"Kalın","marks":[{"type":"bold"}]},
			{"type":"text","text":" ve "},
			{"type":"text","text":"link","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]},
			{"type":"text","text":" kötü","marks":[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]}
		]},
		{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Madde"}]}]}]}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, "<h2>Başlık</h2>")
	assert.Contains(t, out, "<strong>Kalın</strong>")
	assert.Contains(t, out, `href="https://example.com"`)
	assert.NotContains(t, out, "javascript:")
	assert.Contains(t, out, "<ul><li><p>Madde</p></li></ul>")
}

func TestTipTapPlainText(t *testing.T) {
	doc := []byte(`{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Birinci"}]},
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Başlık"}]}
	]}`)
	got := blocksToText(doc)
	assert.Contains(t, got, "Birinci")
	assert.Contains(t, got, "Başlık")
}

func TestRenderTipTapBlocks(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"table","content":[{"type":"tableRow","content":[
			{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Başlık"}]}]},
			{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"Hücre"}]}]}
		]}]},
		{"type":"callout","attrs":{"kind":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Dikkat"}]}]},
		{"type":"embed","attrs":{"provider":"youtube","url":"https://www.youtube.com/watch?v=abc12345"}}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, `class="editor-table"`)
	assert.Contains(t, out, "<th>")
	assert.Contains(t, out, "callout-warning")
	assert.Contains(t, out, "youtube-nocookie.com/embed/abc12345")
}

func TestRenderTipTapToggleAndCode(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"toggle","attrs":{"title":"Detay","open":true},"content":[{"type":"paragraph","content":[{"type":"text","text":"İçerik"}]}]},
		{"type":"codeBlock","attrs":{"language":"js"},"content":[{"type":"text","text":"const x = 1"}]}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, `<details class="editor-toggle" open>`)
	assert.Contains(t, out, "<summary>Detay</summary>")
	assert.Contains(t, out, `class="be-code language-js"`)
}

func TestRenderTipTapBookmark(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"bookmark","attrs":{"url":"https://example.com/haber","title":"Başlık","description":"Açıklama","site":"Example","image":"https://example.com/a.jpg"}},
		{"type":"bookmark","attrs":{"url":"javascript:alert(1)","title":"Kötü","description":"","site":"","image":""}}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, `class="editor-bookmark"`)
	assert.Contains(t, out, `href="https://example.com/haber"`)
	assert.Contains(t, out, `<strong>Başlık</strong>`)
	assert.NotContains(t, out, "javascript:")
}

func TestRenderTipTapPendingSuggestions(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[{"type":"paragraph","content":[
		{"type":"text","text":"Yayında "},
		{"type":"text","text":"eklenen","marks":[{"type":"suggestionInsert","attrs":{"id":"a"}}]},
		{"type":"text","text":"silinen","marks":[{"type":"suggestionDelete","attrs":{"id":"b"}}]},
		{"type":"text","text":" metin"}
	]}]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, "Yayında ")
	assert.Contains(t, out, " metin")
	assert.NotContains(t, out, "eklenen")
	assert.Contains(t, out, "silinen")
	assert.NotContains(t, tipTapPlainText([]byte(doc)), "eklenen")
	assert.Contains(t, tipTapPlainText([]byte(doc)), "silinen")
}

func TestRenderTipTapMediaCard(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"mediaCard","attrs":{"kind":"file","url":"/media/2026/10/rapor.pdf","title":"Rapor","filename":"rapor.pdf"}},
		{"type":"mediaCard","attrs":{"kind":"audio","url":"/media/2026/10/ses.mp3","title":"Podcast"}},
		{"type":"mediaCard","attrs":{"kind":"video","url":"/media/2026/10/video.mp4","title":"Video"}},
		{"type":"mediaCard","attrs":{"kind":"video","url":"javascript:alert(1)","title":"Kötü"}},
		{"type":"uploadPlaceholder","attrs":{"id":"u1","filename":"bekleyen.jpg","status":"loading","message":"Yükleniyor…"}}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, "editor-file-card")
	assert.Contains(t, out, `href="/media/2026/10/rapor.pdf"`)
	assert.Contains(t, out, "Rapor")
	assert.Contains(t, out, `src="/media/2026/10/ses.mp3"`)
	assert.Contains(t, out, `src="/media/2026/10/video.mp4"`)
	assert.NotContains(t, out, "javascript:")
	assert.NotContains(t, out, "bekleyen.jpg")
	assert.Contains(t, tipTapPlainText([]byte(doc)), "Podcast")
	assert.NotContains(t, tipTapPlainText([]byte(doc)), "bekleyen.jpg")
}

func TestRenderTipTapRemoteImage(t *testing.T) {
	s := &Server{}
	doc := `{"type":"doc","content":[
		{"type":"image","attrs":{"mediaId":0,"src":"https://images.example.com/foto.jpg","alt":"Uzak","title":"Görsel başlığı","caption":"Açıklama","credit":"Fotoğraf: Ada / Unsplash","creditUrl":"https://unsplash.com/@ada","layout":"normal","width":100}},
		{"type":"image","attrs":{"mediaId":0,"src":"javascript:alert(1)","alt":"Kötü"}}
	]}`
	out := string(s.renderTipTapHTML([]byte(doc)))
	assert.Contains(t, out, `src="https://images.example.com/foto.jpg"`)
	assert.Contains(t, out, `title="Görsel başlığı"`)
	assert.Contains(t, out, `href="https://unsplash.com/@ada"`)
	assert.Contains(t, out, "Fotoğraf: Ada / Unsplash")
	assert.NotContains(t, out, "javascript:")
}

func TestRenderTipTapImageCropWidth(t *testing.T) {
	node := tipTapNode{Type: "image", Attrs: map[string]any{
		"mediaId": float64(7), "alt": "Kırpma", "width": float64(50),
		"crop": "16/9", "focalX": float64(20), "focalY": float64(30),
	}}
	out := renderTipTapNode(node, func(id int64) string { return "2026/01/a.jpg" })
	assert.Contains(t, out, `style="width:50%"`)
	assert.Contains(t, out, `style="aspect-ratio:16/9;object-fit:cover;object-position:20% 30%"`)
}

func TestRenderTipTapCustomCropOverlay(t *testing.T) {
	node := tipTapNode{Type: "image", Attrs: map[string]any{
		"mediaId": float64(7), "alt": "Serbest", "crop": "3/2", "overlay": "Manşet <metni>",
	}}
	out := renderTipTapNode(node, func(id int64) string { return "2026/01/a.jpg" })
	assert.Contains(t, out, `style="aspect-ratio:3/2;object-fit:cover;object-position:50% 50%"`)
	assert.Contains(t, out, `class="editor-image-overlay"`)
	assert.Contains(t, out, `Manşet &lt;metni&gt;`)
}

func TestRenderTipTapGallery(t *testing.T) {
	node := tipTapNode{Type: "gallery", Attrs: map[string]any{
		"items": []any{
			map[string]any{"mediaId": float64(7), "alt": "Bir", "caption": "Alt"},
			map[string]any{"mediaId": float64(8), "alt": "İki"},
		},
	}}
	out := renderTipTapNode(node, func(id int64) string {
		if id == 7 {
			return "2026/01/a.jpg"
		}
		return "2026/01/b.jpg"
	})
	assert.Contains(t, out, `class="editor-gallery"`)
	assert.Contains(t, out, `src="/media/2026/01/a.jpg"`)
	assert.Contains(t, out, "<figcaption>Alt</figcaption>")
}

func TestDetailToEditorJSONLegacy(t *testing.T) {
	s := &Server{}
	legacy := []byte(`[{"type":"p","text":"Eski paragraf"}]`)
	raw := s.detailToEditorJSON(legacy)
	var doc struct {
		Type    string `json:"type"`
		Content []any  `json:"content"`
	}
	assert.NoError(t, json.Unmarshal([]byte(raw), &doc))
	assert.Equal(t, "doc", doc.Type)
	assert.NotEmpty(t, doc.Content)
}
