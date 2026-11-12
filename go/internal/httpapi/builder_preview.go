package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

func (s *Server) handleBuilderPreview(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	var req struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Blocks) == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bloklar yok"})
		return
	}

	d := s.homeView(r)
	if d == nil || d.Home == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "veri yolu kapalı"})
		return
	}
	// Kategori sayfası önizlemesi: bloklar kategori verisiyle beslenir.
	// Aksi halde kategori modülleri (slider/ray/4'lü/akış) boş görünürdü.
	if isKategoriPage(page) {
		if !s.previewCategoryView(r, page, d) {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "kategori bulunamadı"})
			return
		}
	}
	// Kurumsal sayfa önizlemesi (Faz 143): gövde bloğu gerçek içeriği göstersin.
	if isSayfaPage(page) {
		slug := sayfaSlug(page)
		if slug == "" {
			if pages, err := repo.ListPages(r.Context(), s.pool); err == nil && len(pages) > 0 {
				slug = pages[0].Slug
			}
		}
		if slug != "" {
			if p, err := repo.GetPageBySlug(r.Context(), s.pool, slug, false); err == nil {
				d.Page = &pageData{Title: p.Title, HTML: s.blocksToHTML(p.Detail, nil)}
			}
		}
	}

	blocks := s.buildBlocksFrom(r.Context(), d, platform.NormalizeDesign(req.Blocks), d.Home)
	items := make([]map[string]any, 0, len(blocks)+1)
	for i := range blocks {
		b := blocks[i]
		var buf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&buf, "render_block", &blocks[i]); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "render: " + err.Error()})
			return
		}
		h := sanitizePreviewHTML(buf.String())
		// Manşet satırına ait modüller önizlemede de .cat-hero içinde
		// gösterilir; yoksa slider slaytları mutlak konumlanmaz.
		if b.RowSlotClass != "" {
			rows := groupCatRows([]homeBlock{b})
			if len(rows) == 1 && rows[0].Group {
				h = `<section class="cat-hero ` + rows[0].Class + `" style="` + string(rows[0].Style) + `">` + h + `</section>`
			}
		}
		if b.Visible && !hasVisiblePreview(h) {
			h = `<div class="bp-prev-empty">Bu modül interaktif (betikle çalışıyor); önizlemesi site üzerinde yaşar — burada yer tutucu görünüyor.</div>`
		}
		items = append(items, map[string]any{
			"id":      b.ID,
			"type":    b.Type,
			"visible": b.Visible,
			"html":    h,
		})
	}
	s.audit(r, "tasarim_onizleme", page)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (s *Server) previewCategoryView(r *http.Request, page string, d *tmplData) bool {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	locale := langOf(r, s)
	slug := strings.TrimPrefix(page, "kategori:")
	if slug == "kategori" {
		slug = ""
	}
	cats := s.cachedCategories(ctx, site.ID)
	var cat *repo.Category
	for i := range cats {
		if cats[i].Status != "yayinda" {
			continue
		}
		if slug != "" {
			if cats[i].Slug == slug {
				cat = &cats[i]
				break
			}
			continue
		}
		if cat == nil || cats[i].ContentCount > cat.ContentCount {
			cat = &cats[i]
		}
	}
	if cat == nil {
		return false
	}
	items, err := repo.PublishedBySlug(ctx, s.pool, site.ID, cat.Slug, locale, 24)
	if err != nil {
		// Sessiz hata yok: önizleme boş kalırsa nedeni sunucu günlüğünde.
		s.logger.Warn("kategori önizlemesi yüklenemedi", "error", err, "slug", cat.Slug)
		return false
	}
	d.LangBase = s.langBase(locale)
	d.Category = &categoryData{
		Title: cat.Name, Slug: cat.Slug, Items: items,
		Page: 1, PerPage: 24,
	}
	d.Category.View = s.categoryListView(ctx)
	if validCategoryView(cat.ListView) && cat.ListView != "" {
		d.Category.View = cat.ListView
	}
	switch d.Category.View {
	case "liste", "izgara", "mozaik", "akis":
	default:
		d.Category.View = "klasik"
	}
	return true
}

func sanitizePreviewHTML(html string) string {
	var out strings.Builder
	for len(html) > 0 {
		i := strings.IndexByte(html, '<')
		if i < 0 {
			out.WriteString(html)
			break
		}
		out.WriteString(html[:i])
		rest := html[i:]
		low := strings.ToLower(rest)
		// <script ...>...</script>
		if strings.HasPrefix(low, "<script") {
			closeIdx := strings.Index(low, "</script>")
			if closeIdx < 0 {
				break
			}
			html = rest[closeIdx+len("</script>"):]
			continue
		}
		// <link ...>
		if strings.HasPrefix(low, "<link") {
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				out.WriteString(rest)
				break
			}
			html = rest[end+1:]
			continue
		}
		out.WriteByte('<')
		html = rest[1:]
	}
	return out.String()
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func hasVisiblePreview(html string) bool {
	if strings.Contains(html, "<img") || strings.Contains(html, "<video") ||
		strings.Contains(html, "<svg") || strings.Contains(html, "background-image") {
		return true
	}
	text := strings.TrimSpace(tagRe.ReplaceAllString(html, ""))
	return len([]rune(text)) >= 8
}
