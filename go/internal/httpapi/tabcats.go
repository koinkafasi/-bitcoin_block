package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

const (
	tabCatsGroup = "home"
	tabCatsKey   = "kategori_sekmeli"
)

// tabCatsEditData yönetim paneli sayfası verisi.
type tabCatsEditData struct {
	Categories []repo.Category
	Slugs      []string
	Configured bool
}

// tabCatSlugs ayarlanmış sekme sırasını döndürür. Ayar yoksa nil,false.
func tabCatSlugs(ctx context.Context, s *Server) ([]string, bool) {
	v := s.cachedSetting(ctx, tabCatsGroup, tabCatsKey)
	if len(v) == 0 || string(v) == "null" {
		return nil, false
	}
	var slugs []string
	if json.Unmarshal(v, &slugs) != nil {
		return nil, false
	}
	// temizle: boş slug'ları ve tekrarları at
	seen := map[string]bool{}
	out := make([]string, 0, len(slugs))
	for _, sl := range slugs {
		sl = strings.TrimSpace(sl)
		if sl == "" || seen[sl] {
			continue
		}
		seen[sl] = true
		out = append(out, sl)
	}
	return out, true
}

// handleTabCatsGet /admin/tab-kategorileri — ana sayfa sekme editörü.
func (s *Server) handleTabCatsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildTabCatsData(r)
	if IsHTMX(r) {
		s.renderTabCatsPanelV2(w, r, d)
		return
	}
	s.renderTabCatsPageV2(w, r, d)
}

func (s *Server) buildTabCatsData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Sekme Kategorileri")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ctx := r.Context()
	cats, _ := repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	slugs, ok := tabCatSlugs(ctx, s)

	// Ayar yoksa ekranda mevcut (menü tabanlı) davranışı da göster.
	if !ok {
		for _, c := range s.navCategories(ctx) {
			slugs = append(slugs, c.Slug)
			if len(slugs) >= 8 {
				break
			}
		}
	}
	d.TabCatsEdit = &tabCatsEditData{Categories: cats, Slugs: slugs, Configured: ok}
	return d
}

func (s *Server) renderTabCatsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildTabCatsData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTabCatsPanelV2(w, r, d)
}

// handleTabCatsSave POST /admin/tab-kategorileri/kaydet
// JSON: {"slugs":["guncel","siyaset",...]}
func (s *Server) handleTabCatsSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var slugs []string
	if IsHTMX(r) {
		_ = r.ParseForm()
		if raw := r.FormValue("slugs"); raw != "" {
			if json.Unmarshal([]byte(raw), &slugs) != nil {
				s.renderTabCatsPanel(w, r, "error", "Sekme verisi geçersiz.")
				return
			}
		}
	} else {
		defer r.Body.Close()
		var body struct {
			Slugs []string `json:"slugs"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "JSON geçersiz: " + err.Error()})
			return
		}
		slugs = body.Slugs
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(slugs))
	for _, sl := range slugs {
		sl = strings.TrimSpace(sl)
		if sl == "" || seen[sl] {
			continue
		}
		seen[sl] = true
		clean = append(clean, sl)
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		if IsHTMX(r) {
			s.renderTabCatsPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := repo.SetSetting(ctx, s.pool, tabCatsGroup, tabCatsKey, raw); err != nil {
		if IsHTMX(r) {
			s.renderTabCatsPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderTabCatsPanel(w, r, "info", "Sekme kategorileri kaydedildi ("+itoa(len(clean))+" kategori).")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "adet": len(clean)})
}

// handleTabCatsReset POST /admin/tab-kategorileri/sifirla — menü tabanlı
// varsayılana döner (blok yine menüdeki ilk 8 kategoriyi gösterir).
func (s *Server) handleTabCatsReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := repo.SetSetting(ctx, s.pool, tabCatsGroup, tabCatsKey, []byte("null")); err != nil {
		if IsHTMX(r) {
			s.renderTabCatsPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderTabCatsPanel(w, r, "info", "Sekmeler menü tabanlı varsayılana döndü.")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
