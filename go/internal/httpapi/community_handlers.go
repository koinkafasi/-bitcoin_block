package httpapi

import (
	"net/http"

	"theblock/site/internal/repo"
)

// ---- Topluluk: bookmarks / follow / bildirim / gamification (spec 06 K-G) ----

// handleBookmark kaydı açar/kapatır (JSON).
func (s *Server) handleBookmark(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "hata": "giriş gerekli"})
		return
	}
	contentID := parseID(r.PathValue("id"))
	saved, total, err := repo.ToggleBookmark(r.Context(), s.pool, u.ID, contentID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "hata": "içerik yok"})
		return
	}
	// gamification: kaydetme etkinliği puan kazandırır (rozet eşiği)
	var gained []string
	if _, gained, _ = repo.TouchActivity(r.Context(), s.pool, u.ID, 5); len(gained) > 0 {
		s.logger.Info("rozet kazanıldı", "user", u.ID, "rozet", gained[0])
	}
	s.audit(r, "bookmark", "icerik="+itoa(int(contentID))+" kayitli="+boolStr(saved))
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "kayitli": saved, "toplam": total})
}

func boolStr(b bool) string {
	if b {
		return "evet"
	}
	return "hayir"
}

// handleFollow takip açar/kapatır (JSON).
func (s *Server) handleFollow(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "hata": "giriş gerekli"})
		return
	}
	tt := r.PathValue("type")
	slug := r.PathValue("slug")
	if tt != "kategori" && tt != "etiket" && tt != "yazar" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "geçersiz tip"})
		return
	}
	var targetID *int64
	if tt == "kategori" {
		if n, err2 := repo.CategoryIDBySlug(r.Context(), s.pool, slug); err2 == nil {
			targetID = &n
		}
	}
	on, err := repo.ToggleFollow(r.Context(), s.pool, u.ID, tt, slug, targetID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	if on {
		_, _, _ = repo.TouchActivity(r.Context(), s.pool, u.ID, 3)
	}
	s.audit(r, "takip", tt+":"+slug+" takip="+boolStr(on))
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "takipli": on})
}

// handleMyBookmarks kullanıcının kaydettikleri.
func (s *Server) handleMyBookmarks(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "kaydedilenler")
	s.hesapKaydedilenRender(w, r, u, d, h)
}

// handleMyNotifications bildirim listesi.
func (s *Server) handleMyNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "bildirimler")
	s.hesapBildirimRender(w, r, u, d, h)
}

// handleAchievements rozetler + liderlik tablosu.
func (s *Server) handleAchievements(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "rozetler")
	s.hesapRozetRender(w, r, u, d, h)
}
