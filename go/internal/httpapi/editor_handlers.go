package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/live"
	"theblock/site/internal/repo"
)

var (
	editorPresenceMu sync.Mutex
	editorPresence   = map[int64]map[string]string{}
)

func presenceAdd(contentID int64, connKey, name string) []live.PresenceUser {
	editorPresenceMu.Lock()
	defer editorPresenceMu.Unlock()
	if editorPresence[contentID] == nil {
		editorPresence[contentID] = map[string]string{}
	}
	editorPresence[contentID][connKey] = name
	return presenceListLocked(contentID)
}

func presenceRemove(contentID int64, connKey string) []live.PresenceUser {
	editorPresenceMu.Lock()
	defer editorPresenceMu.Unlock()
	if m := editorPresence[contentID]; m != nil {
		delete(m, connKey)
		if len(m) == 0 {
			delete(editorPresence, contentID)
		}
	}
	return presenceListLocked(contentID)
}

func presenceListLocked(contentID int64) []live.PresenceUser {
	seen := map[string]bool{}
	var out []live.PresenceUser
	for _, name := range editorPresence[contentID] {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, live.PresenceUser{Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Server) handleEditorLiveSSE(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		http.NotFound(w, r)
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		http.Error(w, "yetkisiz", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming desteklenmiyor", http.StatusInternalServerError)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	u := userFromContext(r.Context())
	_ = writeSSE(w, "snapshot", live.Event{
		SiteID: c.SiteID, ContentID: c.ID, Version: c.Version, Kind: "snapshot",
		UserID: u.ID, UserName: u.Name, At: time.Now().Unix(),
	})
	flusher.Flush()
	ch, cancel := live.EditorDefault.Subscribe(id)
	connKey := fmt.Sprintf("%d-%d", time.Now().UnixNano(), u.ID)
	users := presenceAdd(id, connKey, u.Name)
	defer func() {
		cancel()
		users := presenceRemove(id, connKey)
		live.EditorDefault.Publish(live.Event{
			ContentID: id, Kind: "presence", Action: "presence", Users: users, At: time.Now().Unix(),
		})
	}()
	_ = writeSSE(w, "presence", live.Event{
		ContentID: id, Kind: "presence", Action: "presence", Users: users, At: time.Now().Unix(),
	})
	live.EditorDefault.Publish(live.Event{
		ContentID: id, Kind: "presence", Action: "presence", Users: users, At: time.Now().Unix(),
	})
	flusher.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			event := "saved"
			if ev.Action == "presence" {
				event = "presence"
			}
			if err := writeSSE(w, event, ev); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) editorNoteContent(w http.ResponseWriter, r *http.Request, id int64) (repo.Content, bool) {
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "içerik bulunamadı"})
		return repo.Content{}, false
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		s.writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "yetkisiz"})
		return repo.Content{}, false
	}
	return c, true
}

func (s *Server) handleEditorNotesList(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if _, ok := s.editorNoteContent(w, r, id); !ok {
		return
	}
	notes, err := repo.ListEditorNotes(r.Context(), s.pool, id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "notlar yüklenemedi"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": notes})
}

func (s *Server) handleEditorNoteCreate(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if _, ok := s.editorNoteContent(w, r, id); !ok {
		return
	}
	_ = r.ParseForm()
	body := strings.TrimSpace(r.FormValue("body"))
	quote := strings.TrimSpace(r.FormValue("quote"))
	if body == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "not boş"})
		return
	}
	if len([]rune(body)) > 4000 {
		body = string([]rune(body)[:4000])
	}
	if len([]rune(quote)) > 500 {
		quote = string([]rune(quote)[:500])
	}
	uid := userFromContext(r.Context()).ID
	nid, err := repo.CreateEditorNote(r.Context(), s.pool, s.curSiteID(r), id, uid, body, quote)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "not eklenemedi"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": nid})
}

func (s *Server) handleEditorNoteResolve(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	nid := parseID(r.PathValue("note"))
	if id <= 0 || nid <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if _, ok := s.editorNoteContent(w, r, id); !ok {
		return
	}
	_ = r.ParseForm()
	resolved := strings.TrimSpace(r.FormValue("resolved")) != "0"
	if err := repo.ResolveEditorNote(r.Context(), s.pool, id, nid, resolved); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "resolved": resolved})
}

func (s *Server) handleEditorNoteDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	nid := parseID(r.PathValue("note"))
	if id <= 0 || nid <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if _, ok := s.editorNoteContent(w, r, id); !ok {
		return
	}
	if err := repo.DeleteEditorNote(r.Context(), s.pool, id, nid); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleInternalLinkSearch, editörde iç bağlantı seçmek için yayındaki
// içeriklerde başlık araması yapar.
func (s *Server) handleInternalLinkSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": []any{}})
		return
	}
	// Seçili metin genelde bir cümle parçası; tam alt dize araması
	// sonuçsuz kalıyordu. Kelime bazlı, skorlu öneri kullanılıyor.
	items, err := repo.OneriliContentLinks(r.Context(), s.pool, s.curSiteID(r), q, 12)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "arama başarısız"})
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":    it.ID,
			"title": it.Title,
			"url":   publicArticleURL(it.Locale, it.Slug, it.ID),
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out})
}
