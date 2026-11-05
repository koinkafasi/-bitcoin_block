package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Etiket yönetimi (Faz 41) ----

type tagsData struct {
	Items []repo.TagWithCount
	Q     string
	Total int
}

func (s *Server) handleTagsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildTagsData(r)
	if IsHTMX(r) {
		s.renderEtiketlerPanelV2(w, r, d)
		return
	}
	s.renderEtiketlerPageV2(w, r, d)
}

func (s *Server) buildTagsData(r *http.Request) *tmplData {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, _ := repo.ListTagsWithCount(ctx, s.pool, q, 200)
	total, _ := repo.CountTags(ctx, s.pool)

	d := s.baseView("tr", "/admin", "Etiketler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Tags = &tagsData{Items: items, Q: q, Total: total}
	return d
}

func (s *Server) renderTagsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildTagsData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderEtiketlerPanelV2(w, r, d)
}

// handleTagRename etiketi yeniden adlandırır (+slug tazeler).
func (s *Server) handleTagRename(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	name := strings.TrimSpace(r.FormValue("yeni_ad"))
	fail := func(msg string) {
		if IsHTMX(r) {
			s.renderTagsPanel(w, r, "error", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/etiketler", "error", msg)
	}
	if name == "" {
		fail("Yeni ad boş olamaz.")
		return
	}
	err := repo.RenameTag(ctx, s.pool, id, name, platform.Slugify(name))
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrDuplicate):
			fail("Bu ad/slug zaten kullanılıyor.")
		case errors.Is(err, repo.ErrNotFound):
			fail("Etiket bulunamadı.")
		default:
			fail("Adlandırılamadı: " + err.Error())
		}
		return
	}
	s.audit(r, "etiket_adlandir", name)
	if IsHTMX(r) {
		s.renderTagsPanel(w, r, "info", "Etiket \""+name+"\" olarak güncellendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/etiketler", "info", "Etiket \""+name+"\" olarak güncellendi.")
}

// handleTagMerge kaynak etiketi hedefe birleştirir.
func (s *Server) handleTagMerge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fromID := parseID(r.FormValue("kaynak"))
	hedef := strings.TrimSpace(r.FormValue("hedef"))
	fail := func(msg string) {
		if IsHTMX(r) {
			s.renderTagsPanel(w, r, "error", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/etiketler", "error", msg)
	}
	t, err := repo.GetTagByName(ctx, s.pool, hedef)
	if err != nil {
		fail("Hedef etiket bulunamadı: " + hedef)
		return
	}
	n, err := repo.MergeTag(ctx, s.pool, fromID, t.ID)
	if err != nil {
		fail("Birleştirilemedi: " + err.Error())
		return
	}
	s.audit(r, "etiket_birlestir", "id="+itoa(int(fromID))+"->"+itoa(int(t.ID)))
	if IsHTMX(r) {
		s.renderTagsPanel(w, r, "info", "Etiket birleştirildi ("+itoa(int(n))+" bağ taşındı).")
		return
	}
	s.flashRedirect(w, r, "/admin/etiketler", "info", "Etiket birleştirildi ("+itoa(int(n))+" bağ taşındı).")
}

// handleTagDelete etiketi siler.
func (s *Server) handleTagDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	n, err := repo.DeleteTag(ctx, s.pool, id)
	if err != nil {
		if IsHTMX(r) {
			s.renderTagsPanel(w, r, "error", "Silinemedi: "+err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/etiketler", "error", "Silinemedi: "+err.Error())
		return
	}
	s.audit(r, "etiket_sil", "id="+itoa(int(id)))
	if IsHTMX(r) {
		s.renderTagsPanel(w, r, "info", "Etiket silindi ("+itoa(int(n))+" bağ kaldırıldı).")
		return
	}
	s.flashRedirect(w, r, "/admin/etiketler", "info", "Etiket silindi ("+itoa(int(n))+" bağ kaldırıldı).")
}
