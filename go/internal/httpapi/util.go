package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- medya URL çözümleme ----

// mediaPath DB'den medya depolama yolunu çözer (bellek önbellekli).
func (s *Server) mediaPath(ctx context.Context, id int64) string {
	s.mediaMu.RLock()
	p, ok := s.mediaCache[id]
	s.mediaMu.RUnlock()
	if ok {
		return p
	}
	m, err := repo.GetMedia(ctx, s.pool, id)
	if err != nil {
		return ""
	}
	s.mediaMu.Lock()
	s.mediaCache[id] = m.Path
	s.mediaMu.Unlock()
	return m.Path
}

// publicMediaURL depolama yolundan temiz web URL'si üretir (/media/{yyyymm}/{dosya}).
func publicMediaURL(stored *string) string {
	if stored == nil || *stored == "" {
		return ""
	}
	p := strings.TrimPrefix(strings.TrimPrefix(*stored, "/"), "media/")
	return "/media/" + p
}

// mediaURL şablonda kullanılır; id nil ise 1x1 boş döndürür.
func (s *Server) mediaURL(id *int64) string {
	if id == nil {
		return "/media/none.svg"
	}
	p := s.mediaPath(context.Background(), *id)
	if p == "" {
		return "/media/none.svg"
	}
	// depolama yolu "media/{yyyymm}/{dosya}" biçimindedir; web yolunda
	// "/media/" öneki tek sefer uygulanır.
	p = strings.TrimPrefix(strings.TrimPrefix(p, "/"), "media/")
	return "/media/" + p
}

// mediaThumbURL medya önizleme ucu. Thumbnail yoksa uç ilk istekte üretir.
func (s *Server) mediaThumbURL(id *int64) string {
	if id == nil {
		return "/media/none.svg"
	}
	return "/media/thumb/" + itoa64(*id)
}

// mediaVariantURL responsive görsel varyantı üretir. İlk istekte oluşturulur,
// sonraki isteklerde storage/media/variants altından servis edilir.
func (s *Server) mediaVariantURL(id *int64, width int) string {
	if id == nil {
		return "/media/none.svg"
	}
	return fmt.Sprintf("/media/variant/%d/%d", *id, width)
}

// mediaSrcset büyük içerik görselleri için tarayıcıya birden fazla genişlik
// sunar. Böylece mobilde 9 MB'lık orijinal yerine 480/768 px indirilir.
func (s *Server) mediaSrcset(id *int64) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("/media/variant/%d/320 320w, /media/variant/%d/480 480w, "+
		"/media/variant/%d/768 768w, /media/variant/%d/960 960w, "+
		"/media/variant/%d/1200 1200w, /media/variant/%d/1600 1600w",
		*id, *id, *id, *id, *id, *id)
}

// catIconSVG kategori ikon anahtarlarını inline SVG çizimlerine eşler.
// Emoji fontuna bağımlı kalmamak için (platform fontu yoksa tofu kutu
// çıkabiliyor) vektör ikon kullanılır; viewBox/stroke tanımı ortaktır.
var catIconSVG = map[string]string{
	"news":      `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 8h6M7 12h6M7 16h4"/><path d="M17 8h1v8h-1z"/>`,
	"politics":  `<path d="M3 10h18L12 3 3 10z"/><path d="M5 21v-8M9.5 21v-8M14.5 21v-8M19 21v-8"/><path d="M3 21h18"/>`,
	"economy":   `<path d="M3 17l6-6 4 4 8-8"/><path d="M15 7h6v6"/><path d="M3 21h18"/>`,
	"sport":     `<circle cx="12" cy="12" r="9"/><path d="M12 3v5M12 16v5M3 12h5M16 12h5"/><path d="M5.6 5.6l3.5 3.5M14.9 14.9l3.5 3.5M18.4 5.6l-3.5 3.5M9.1 14.9l-3.5 3.5"/>`,
	"tech":      `<rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8M12 17v4"/><path d="M6 8h5M6 11h3"/>`,
	"health":    `<path d="M20.8 5.6a5 5 0 00-7.1 0L12 7.3l-1.7-1.7a5 5 0 10-7.1 7.1L12 21l8.8-8.3a5 5 0 000-7.1z"/>`,
	"game":      `<rect x="2" y="7" width="20" height="11" rx="4"/><path d="M7 10.5v3M5.5 12h3"/><circle cx="16" cy="11.5" r="1.1"/><circle cx="18.4" cy="14" r="1.1"/>`,
	"science":   `<path d="M9 3h6M10 3v6.5L4.7 18a2 2 0 001.7 3h11.2a2 2 0 001.7-3L14 9.5V3"/><path d="M7.4 15h9.2"/>`,
	"life":      `<circle cx="12" cy="12" r="4"/><path d="M12 2v2.5M12 19.5V22M4.2 4.2l1.8 1.8M18 18l1.8 1.8M2 12h2.5M19.5 12H22M4.2 19.8L6 18M18 6l1.8-1.8"/>`,
	"world":     `<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3c2.5 2.6 3.8 5.6 3.8 9s-1.3 6.4-3.8 9c-2.5-2.6-3.8-5.6-3.8-9S9.5 5.6 12 3z"/>`,
	"education": `<path d="M4 4.5A2.5 2.5 0 016.5 2H20v20H6.5A2.5 2.5 0 014 19.5z"/><path d="M8 7h8M8 11h6"/>`,
	"car":       `<path d="M5 11l1.5-4.5A2 2 0 018.4 5h7.2a2 2 0 011.9 1.5L19 11"/><rect x="3" y="11" width="18" height="6" rx="2"/><circle cx="7.5" cy="17.5" r="1.6"/><circle cx="16.5" cy="17.5" r="1.6"/>`,
	"shield":    `<path d="M12 3l8 3v6c0 5-3.4 8.4-8 9-4.6-.6-8-4-8-9V6z"/><path d="M9 12l2 2 4-4"/>`,
	"building":  `<path d="M4 21V5a2 2 0 012-2h8a2 2 0 012 2v16"/><path d="M16 9h2a2 2 0 012 2v10"/><path d="M2 21h20"/><path d="M8 7h2M8 11h2M8 15h2"/>`,
	"palette":   `<path d="M12 21a9 9 0 110-18c4.97 0 9 3.58 9 8 0 2.21-1.79 3-4 3h-1.5a2 2 0 00-1.4 3.42A2 2 0 0112 21z"/><circle cx="7.5" cy="11.5" r="1"/><circle cx="10.5" cy="7.5" r="1"/><circle cx="15" cy="8.5" r="1"/>`,
	"plane":     `<path d="M10.2 3.3a1.5 1.5 0 012.6 0l1.7 5.5 6.9 3.1a1.5 1.5 0 010 2.8l-6.9 1.3-1.7 5.5a1.5 1.5 0 01-2.6 0L8.5 15l-4.8 1a1.5 1.5 0 01-.7-2.9l4.8-1z"/>`,
	"grid":      `<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>`,
}

// catIconMap panelden gelen serbest metni (emoji veya anahtar) SVG ikon
// anahtarına eşler. Anahtar tanınmazsa değer olduğu gibi (kaçışlanmış) kullanılır.
var catIconMap = map[string]string{
	"news": "news", "guncel": "news", "gundem": "news", "haber": "news",
	"ilceler": "building", "building": "building", "belde": "building",
	"sanat": "palette", "kultur-sanat": "palette", "sanat-kultur": "palette",
	"turizm": "plane", "turizm-ic": "plane", "seyahat": "plane",
	"diger": "grid", "grid": "grid", "other": "grid",
	"egitim-kultur": "education",
	"siyaset":       "politics", "politics": "politics", "vote": "politics",
	"ekonomi": "economy", "economy": "economy", "coins": "economy",
	"spor": "sport", "sport": "sport", "ball": "sport",
	"teknoloji": "tech", "technology": "tech", "tech": "tech", "device": "tech",
	"saglik": "health", "health": "health",
	"oyun": "game", "game": "game",
	"bilim": "science", "science": "science", "flask": "science",
	"yasam": "life", "life": "life", "coffee": "life",
	"dunya": "world", "world": "world", "globe": "world",
	"egitim": "education", "education": "education", "book": "education",
	"otomobil": "car", "car": "car", "culture": "education", "kultur": "education",
	"asayis": "shield", "shield": "shield",
	"belgesel": "game", "video": "game", "magazin": "life",
}

const catIconSVGSkel = `<svg class="ci" viewBox="0 0 24 24" fill="none" stroke="currentColor" ` +
	`stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">`

// catIcon kategori ikon değerini inline SVG olarak döndürür. Panelden gelen
// metin anahtar tablosunda yoksa varsayılan "news" ikonu kullanılır — böylece
// çıktı her zaman güvenli, sabit ve kaçışlanmış HTML'dir.
func catIcon(key string) template.HTML {
	name := catIconMap[strings.ToLower(strings.TrimSpace(key))]
	body, ok := catIconSVG[name]
	if !ok {
		body = catIconSVG["news"]
	}
	return template.HTML(catIconSVGSkel + body + `</svg>`)
}

// ---- içerik detayı (JSON blok) dönüşümleri ----

// textToBlocks paragraf metnini JSON blok dizisine çevirir.
// "[gorsel:ID]" satırı görsel bloğuna dönüşür.
func textToBlocks(text string) []byte {
	var blocks []map[string]any
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if strings.HasPrefix(para, "[gorsel:") && strings.HasSuffix(para, "]") {
			idStr := strings.TrimSuffix(strings.TrimPrefix(para, "[gorsel:"), "]")
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				blocks = append(blocks, map[string]any{"type": "image", "id": id})
				continue
			}
		}
		blocks = append(blocks, map[string]any{"type": "p", "text": para})
	}
	if blocks == nil {
		blocks = []map[string]any{}
	}
	b, _ := json.Marshal(blocks)
	return b
}

// blocksToText JSON blokları düz metne çevirir (form yüklerken).
func blocksToText(detail []byte) string {
	if len(detail) == 0 {
		return ""
	}
	if isTipTapDetail(detail) {
		return tipTapPlainText(detail)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		ID   *int64 `json:"id"`
	}
	if err := json.Unmarshal(detail, &blocks); err != nil {
		return string(detail)
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "p" {
			parts = append(parts, b.Text)
		} else if b.Type == "image" && b.ID != nil {
			parts = append(parts, fmt.Sprintf("[gorsel:%d]", *b.ID))
		}
	}
	return strings.Join(parts, "\n\n")
}

// blocksToHTML genel site için güvenli HTML üretir.
func (s *Server) blocksToHTML(detail []byte, mediaKey func(int64) string) template.HTML {
	if len(detail) == 0 {
		return template.HTML("")
	}
	if isTipTapDetail(detail) {
		return s.renderTipTapHTML(detail)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		ID   *int64 `json:"id"`
	}
	if err := json.Unmarshal(detail, &blocks); err != nil {
		return template.HTML("<p>" + html.EscapeString(string(detail)) + "</p>")
	}
	var sb strings.Builder
	for _, b := range blocks {
		switch b.Type {
		case "image":
			if b.ID != nil {
				p := s.mediaPath(context.Background(), *b.ID)
				if p != "" {
					sb.WriteString(`<p><img src="/media/` + html.EscapeString(p) + `" alt="" loading="lazy"></p>`)
				}
			}
		case "p":
			sb.WriteString("<p>" + html.EscapeString(b.Text) + "</p>")
		}
	}
	return template.HTML(sb.String())
}

// ---- istatistik yardımcıları (SEO puanı için) ----

func wordCount(s string) int {
	return len(strings.Fields(s))
}

func countTag(s, tag string) int {
	return strings.Count(strings.ToLower(s), "<"+tag)
}

// ---- tip etiketleri ----

// ---- sayısal yardımcılar ----

func parseID(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseLocalTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
	if err != nil {
		return nil
	}
	return &t
}

func optID(s string) *int64 {
	if s == "" {
		return nil
	}
	n := parseID(s)
	if n == 0 {
		return nil
	}
	return &n
}

func optID64(n int64) *int64 {
	if n == 0 {
		return nil
	}
	return &n
}

func optStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func rndSuffix() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// readFull io.ReadFull sarıcısı.
func readFull(r io.Reader, b []byte) (int, error) { return io.ReadFull(r, b) }

// platformFileExt dosya adından mime türü.
func platformFileExt(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if m := mime.TypeByExtension(ext); m != "" {
		return m, nil
	}
	return "application/octet-stream", nil
}

// mediaKindByExt dosya uzantısından medya türünü çıkarır.
// Görsel olmayanlar (ses/video/dosya) native yüklenir; önizleme yerine
// tür ikonu basılır. Listede olmayan uzantılar reddedilir.
func mediaKindByExt(filename string) (string, bool) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return "image", true
	case ".mp3", ".wav", ".ogg", ".oga", ".m4a", ".aac", ".flac":
		return "audio", true
	case ".mp4", ".webm", ".mov", ".m4v", ".ogv":
		return "video", true
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv":
		return "document", true
	}
	return "", false
}

// mediaKindIcon tür ikonu (SVG) üretir; görsel olmayan medya önizlemelerinde
// kırık görsel yerine kullanılır.
func mediaKindIcon(kind string) []byte {
	label, glyph, bg := "Dosya", "📄", "#eef1f6"
	switch kind {
	case "audio":
		label, glyph, bg = "Ses", "🎵", "#e8f2ff"
	case "video":
		label, glyph, bg = "Video", "🎬", "#f3e8ff"
	case "document":
		label, glyph = "Belge", "📄"
	}
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="360" viewBox="0 0 480 360">` +
		`<rect width="100%" height="100%" fill="` + bg + `"/>` +
		`<text x="240" y="185" font-size="120" text-anchor="middle">` + glyph + `</text>` +
		`<text x="240" y="265" font-size="34" text-anchor="middle" fill="#71717a" font-family="system-ui,sans-serif">` + label + `</text>` +
		`</svg>`)
}

var _ = platform.Slugify
