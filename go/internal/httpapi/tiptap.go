package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var youtubeEmbedRE = regexp.MustCompile(`(?:youtube\.com/(?:watch\?v=|embed/|shorts/)|youtu\.be/)([A-Za-z0-9_-]{6,})`)

type tipTapNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []tipTapMark   `json:"marks"`
	Content []tipTapNode   `json:"content"`
}

type tipTapMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

func isTipTapDetail(detail []byte) bool {
	var probe struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(detail, &probe) != nil {
		return false
	}
	return probe.Type == "doc"
}

func isTipTapRaw(raw string) bool {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return false
	}
	var probe struct {
		Type    string `json:"type"`
		Content []any  `json:"content"`
	}
	if json.Unmarshal([]byte(raw), &probe) != nil {
		return false
	}
	return probe.Type == "doc" && probe.Content != nil && len(raw) < 2<<20
}

// parseDetailInput, editörden gelen sürümlü TipTap dokümanını olduğu gibi
// saklar; düz metin/legacy içe aktarımlarda eski blok dönüşümünü kullanır.
func parseDetailInput(raw string) []byte {
	raw = strings.TrimSpace(raw)
	if isTipTapRaw(raw) {
		return []byte(raw)
	}
	return textToBlocks(raw)
}

func (s *Server) detailToEditorJSON(detail []byte) string {
	if isTipTapDetail(detail) {
		return string(detail)
	}
	content := []map[string]any{}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		ID   *int64 `json:"id"`
	}
	if json.Unmarshal(detail, &blocks) == nil {
		for _, b := range blocks {
			switch b.Type {
			case "p":
				if strings.TrimSpace(b.Text) == "" {
					continue
				}
				content = append(content, map[string]any{
					"type":    "paragraph",
					"content": []map[string]any{{"type": "text", "text": b.Text}},
				})
			case "image":
				if b.ID != nil {
					content = append(content, map[string]any{
						"type": "image",
						"attrs": map[string]any{
							"mediaId": *b.ID,
							"src":     s.mediaURL(b.ID),
						},
					})
				}
			}
		}
	}
	if len(content) == 0 {
		content = append(content, map[string]any{"type": "paragraph"})
	}
	b, _ := json.Marshal(map[string]any{"type": "doc", "content": content})
	return string(b)
}

func (s *Server) renderTipTapHTML(detail []byte) template.HTML {
	var doc tipTapNode
	if json.Unmarshal(detail, &doc) != nil || doc.Type != "doc" {
		return ""
	}
	return template.HTML(renderTipTapNodes(doc.Content, func(id int64) string {
		return s.mediaPath(context.Background(), id)
	}))
}

func renderTipTapNodes(nodes []tipTapNode, mediaPath func(int64) string) string {
	var sb strings.Builder
	for _, n := range nodes {
		sb.WriteString(renderTipTapNode(n, mediaPath))
	}
	return sb.String()
}

func renderTipTapNode(n tipTapNode, mediaPath func(int64) string) string {
	switch n.Type {
	case "text":
		for _, m := range n.Marks {
			if m.Type == "suggestionInsert" {
				return ""
			}
		}
		text := html.EscapeString(n.Text)
		for _, m := range n.Marks {
			switch m.Type {
			case "bold":
				text = "<strong>" + text + "</strong>"
			case "italic":
				text = "<em>" + text + "</em>"
			case "underline":
				text = "<u>" + text + "</u>"
			case "strike":
				text = "<s>" + text + "</s>"
			case "code":
				text = "<code>" + text + "</code>"
			case "link":
				href := safeTipTapHref(stringAttr(m.Attrs, "href"))
				if href != "" {
					text = `<a href="` + href + `" target="_blank" rel="noopener noreferrer">` + text + `</a>`
				}
			}
		}
		return text
	case "paragraph":
		return "<p>" + renderTipTapNodes(n.Content, mediaPath) + "</p>"
	case "heading":
		level := intAttr(n.Attrs, "level")
		if level < 2 {
			level = 2
		}
		if level > 4 {
			level = 4
		}
		tag := fmt.Sprintf("h%d", level)
		return "<" + tag + ">" + renderTipTapNodes(n.Content, mediaPath) + "</" + tag + ">"
	case "bulletList":
		return "<ul>" + renderTipTapNodes(n.Content, mediaPath) + "</ul>"
	case "orderedList":
		start := intAttr(n.Attrs, "start")
		if start > 1 {
			return `<ol start="` + strconv.Itoa(start) + `">` + renderTipTapNodes(n.Content, mediaPath) + "</ol>"
		}
		return "<ol>" + renderTipTapNodes(n.Content, mediaPath) + "</ol>"
	case "listItem":
		return "<li>" + renderTipTapNodes(n.Content, mediaPath) + "</li>"
	case "blockquote":
		return "<blockquote>" + renderTipTapNodes(n.Content, mediaPath) + "</blockquote>"
	case "codeBlock":
		lang := safeLanguage(stringAttr(n.Attrs, "language"))
		class := "be-code"
		if lang != "" {
			class += " language-" + lang
		}
		return `<pre><code class="` + class + `">` + html.EscapeString(tipTapNodeText(n)) + "</code></pre>"
	case "horizontalRule":
		return "<hr>"
	case "hardBreak":
		return "<br>"
	case "table":
		return `<table class="editor-table"><tbody>` + renderTipTapNodes(n.Content, mediaPath) + "</tbody></table>"
	case "tableRow":
		return "<tr>" + renderTipTapNodes(n.Content, mediaPath) + "</tr>"
	case "tableHeader":
		return "<th" + tableSpanAttrs(n.Attrs) + ">" + renderTipTapNodes(n.Content, mediaPath) + "</th>"
	case "tableCell":
		return "<td" + tableSpanAttrs(n.Attrs) + ">" + renderTipTapNodes(n.Content, mediaPath) + "</td>"
	case "callout":
		kind := stringAttr(n.Attrs, "kind")
		label := map[string]string{"info": "Bilgi", "warning": "Uyarı", "success": "Başarı", "danger": "Dikkat"}[kind]
		if label == "" {
			kind, label = "info", "Bilgi"
		}
		return `<aside class="callout callout-` + kind + `"><div class="callout-title">` + label + `</div>` +
			renderTipTapNodes(n.Content, mediaPath) + "</aside>"
	case "toggle":
		open := ""
		if v, ok := n.Attrs["open"].(bool); ok && v {
			open = " open"
		}
		return `<details class="editor-toggle"` + open + `><summary>` +
			html.EscapeString(stringAttr(n.Attrs, "title")) + `</summary>` +
			renderTipTapNodes(n.Content, mediaPath) + `</details>`
	case "uploadPlaceholder":
		return ""
	case "mediaCard":
		href := safeTipTapHref(stringAttr(n.Attrs, "url"))
		if href == "" {
			return ""
		}
		kind := strings.ToLower(stringAttr(n.Attrs, "kind"))
		title := html.EscapeString(stringAttr(n.Attrs, "title"))
		filename := html.EscapeString(stringAttr(n.Attrs, "filename"))
		switch kind {
		case "audio":
			out := `<figure class="editor-media-card editor-audio-card"><audio controls preload="metadata" src="` + href + `"></audio>`
			if title != "" {
				out += `<figcaption>` + title + `</figcaption>`
			}
			return out + `</figure>`
		case "video":
			out := `<figure class="editor-media-card editor-video-card"><video controls preload="metadata" playsinline src="` + href + `"></video>`
			if title != "" {
				out += `<figcaption>` + title + `</figcaption>`
			}
			return out + `</figure>`
		default:
			label := title
			if label == "" {
				label = filename
			}
			if label == "" {
				label = "Dosya indir"
			}
			download := ""
			if filename != "" {
				download = ` download="` + filename + `"`
			}
			out := `<div class="editor-media-card editor-file-card"><a href="` + href + `"` + download + ` target="_blank" rel="noopener noreferrer"><span class="editor-file-icon">📄</span><span class="editor-file-body"><strong>` + label + `</strong>`
			if filename != "" {
				out += `<small>` + filename + `</small>`
			}
			return out + `</span></a></div>`
		}
	case "bookmark":
		href := safeTipTapHref(stringAttr(n.Attrs, "url"))
		if href == "" {
			return ""
		}
		title := html.EscapeString(stringAttr(n.Attrs, "title"))
		desc := html.EscapeString(stringAttr(n.Attrs, "description"))
		site := html.EscapeString(stringAttr(n.Attrs, "site"))
		image := safeTipTapHref(stringAttr(n.Attrs, "image"))
		out := `<a class="editor-bookmark" href="` + href + `" target="_blank" rel="noopener noreferrer" data-bookmark="" data-title="` + title +
			`" data-description="` + desc + `" data-site="` + site + `" data-image="` + image + `">`
		if image != "" {
			out += `<img src="` + image + `" alt="" loading="lazy">`
		}
		return out + `<div class="editor-bookmark-body"><strong>` + title + `</strong><p>` + desc + `</p><span>` + site + `</span></div></a>`
	case "embed":
		return renderTipTapEmbed(n.Attrs)
	case "gallery":
		return renderTipTapGallery(n.Attrs, mediaPath)
	case "image":
		mediaID := int64Attr(n.Attrs, "mediaId")
		var src string
		if mediaID > 0 {
			p := ""
			if mediaPath != nil {
				p = mediaPath(mediaID)
			}
			if p == "" {
				return ""
			}
			src = "/media/" + html.EscapeString(p)
		} else {
			src = safeTipTapHref(stringAttr(n.Attrs, "src"))
			if src == "" {
				return ""
			}
		}
		alt := html.EscapeString(stringAttr(n.Attrs, "alt"))
		title := html.EscapeString(stringAttr(n.Attrs, "title"))
		if alt == "" {
			alt = title
		}
		caption := html.EscapeString(stringAttr(n.Attrs, "caption"))
		credit := html.EscapeString(stringAttr(n.Attrs, "credit"))
		creditURL := safeTipTapHref(stringAttr(n.Attrs, "creditUrl"))
		layout := stringAttr(n.Attrs, "layout")
		klass := "editor-image"
		if layout == "wide" || layout == "full" {
			klass += " editor-image-" + layout
		}
		styles := []string{}
		if layout == "" || layout == "normal" {
			if width := int64Attr(n.Attrs, "width"); width >= 25 && width < 100 {
				styles = append(styles, "width:"+fmt.Sprint(width)+"%")
			}
		}
		imgStyle := ""
		if crop := safeCropRatio(stringAttr(n.Attrs, "crop")); crop != "" {
			fx, fy := int64Attr(n.Attrs, "focalX"), int64Attr(n.Attrs, "focalY")
			if _, ok := n.Attrs["focalX"]; !ok {
				fx = 50
			}
			if _, ok := n.Attrs["focalY"]; !ok {
				fy = 50
			}
			if fx < 0 || fx > 100 {
				fx = 50
			}
			if fy < 0 || fy > 100 {
				fy = 50
			}
			imgStyle = ` style="aspect-ratio:` + crop + `;object-fit:cover;object-position:` + fmt.Sprint(fx) + `% ` + fmt.Sprint(fy) + `%"`
		}
		style := ""
		if len(styles) > 0 {
			style = ` style="` + strings.Join(styles, ";") + `"`
		}
		overlay := html.EscapeString(stringAttr(n.Attrs, "overlay"))
		titleAttr := ""
		if title != "" {
			titleAttr = ` title="` + title + `"`
		}
		out := `<figure class="` + klass + `"` + style + `><div class="editor-image-frame"><img src="` + src + `" alt="` + alt + `"` + titleAttr + ` loading="lazy"` + imgStyle + `>`
		if overlay != "" {
			out += `<div class="editor-image-overlay">` + overlay + `</div>`
		}
		out += `</div>`
		if caption != "" || credit != "" {
			out += "<figcaption>"
			if caption != "" {
				out += caption
			}
			if credit != "" {
				if caption != "" {
					out += " — "
				}
				if creditURL != "" {
					out += `<a class="editor-image-credit" href="` + creditURL + `" target="_blank" rel="noopener noreferrer">` + credit + `</a>`
				} else {
					out += `<span class="editor-image-credit">` + credit + `</span>`
				}
			}
			out += "</figcaption>"
		}
		return out + "</figure>"
	default:
		return renderTipTapNodes(n.Content, mediaPath)
	}
}

func tipTapNodeText(n tipTapNode) string {
	var sb strings.Builder
	var walk func(tipTapNode)
	walk = func(node tipTapNode) {
		if node.Type == "text" {
			for _, m := range node.Marks {
				if m.Type == "suggestionInsert" {
					return
				}
			}
			sb.WriteString(node.Text)
			return
		}
		if node.Type == "hardBreak" {
			sb.WriteString("\n")
			return
		}
		if node.Type == "uploadPlaceholder" {
			return
		}
		if node.Type == "mediaCard" {
			sb.WriteString(stringAttr(node.Attrs, "title"))
			sb.WriteString("\n")
			return
		}
		for _, c := range node.Content {
			walk(c)
		}
		if node.Type == "paragraph" || node.Type == "heading" || node.Type == "listItem" ||
			node.Type == "blockquote" || node.Type == "codeBlock" {
			sb.WriteString("\n")
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func tipTapPlainText(detail []byte) string {
	var doc tipTapNode
	if json.Unmarshal(detail, &doc) != nil || doc.Type != "doc" {
		return ""
	}
	return tipTapNodeText(doc)
}

func safeTipTapHref(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto", "tel":
		return html.EscapeString(raw)
	case "":
		if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
			return html.EscapeString(raw)
		}
	}
	return ""
}

func safeCropRatio(raw string) string {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "16/9", "4/3", "1/1":
		return raw
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return ""
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w < 1 || h < 1 || w > 100 || h > 100 {
		return ""
	}
	ratio := float64(w) / float64(h)
	if ratio < 0.2 || ratio > 5 {
		return ""
	}
	return fmt.Sprintf("%d/%d", w, h)
}

func safeLanguage(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '#' || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func tableSpanAttrs(attrs map[string]any) string {
	out := ""
	if n := intAttr(attrs, "colspan"); n > 1 {
		out += ` colspan="` + strconv.Itoa(n) + `"`
	}
	if n := intAttr(attrs, "rowspan"); n > 1 {
		out += ` rowspan="` + strconv.Itoa(n) + `"`
	}
	return out
}

func renderTipTapEmbed(attrs map[string]any) string {
	raw := stringAttr(attrs, "url")
	provider := stringAttr(attrs, "provider")
	id := youtubeEmbedRE.FindStringSubmatch(raw)
	if id == nil && isXStatusURL(raw) {
		provider = "x"
	}
	if len(id) > 1 {
		provider = "youtube"
	}
	switch provider {
	case "youtube":
		if len(id) < 2 {
			return ""
		}
		return `<div class="editor-embed"><iframe src="https://www.youtube-nocookie.com/embed/` +
			html.EscapeString(id[1]) + `" title="YouTube video" loading="lazy" allowfullscreen ` +
			`allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"></iframe></div>`
	case "x":
		if !isXStatusURL(raw) {
			return ""
		}
		return `<p class="editor-embed-link"><a href="` + html.EscapeString(raw) +
			`" target="_blank" rel="noopener noreferrer">X gönderisini aç</a></p>`
	}
	return ""
}

func isXStatusURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if host != "x.com" && host != "www.x.com" && host != "twitter.com" && host != "www.twitter.com" {
		return false
	}
	return strings.Contains(u.Path, "/status/")
}

func renderTipTapGallery(attrs map[string]any, mediaPath func(int64) string) string {
	items, _ := attrs["items"].([]any)
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<div class="editor-gallery">`)
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		id := int64Attr(m, "mediaId")
		p := ""
		if mediaPath != nil {
			p = mediaPath(id)
		}
		if p == "" {
			continue
		}
		alt := html.EscapeString(stringAttr(m, "alt"))
		caption := html.EscapeString(stringAttr(m, "caption"))
		sb.WriteString(`<figure><img src="/media/` + html.EscapeString(p) + `" alt="` + alt + `" loading="lazy">`)
		if caption != "" {
			sb.WriteString("<figcaption>" + caption + "</figcaption>")
		}
		sb.WriteString("</figure>")
	}
	sb.WriteString("</div>")
	return sb.String()
}

func stringAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if v, ok := attrs[key].(string); ok {
		return v
	}
	return ""
}

func intAttr(attrs map[string]any, key string) int {
	switch v := attrs[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

func int64Attr(attrs map[string]any, key string) int64 {
	switch v := attrs[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}
