package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// handleCardReaction kartlardaki "buna benzer daha fazla / daha az"
// geri bildirimini kaydeder. Anonim ziyaretçi tb_visitor çereziyle izlenir.
func (s *Server) handleCardReaction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID  int64  `json:"id"`
		Yon string `json:"yon"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&body); err != nil || body.ID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek"})
		return
	}
	var reaction int
	switch strings.TrimSpace(body.Yon) {
	case "fazla":
		reaction = 1
	case "az":
		reaction = -1
	case "iptal":
		reaction = 0
	default:
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz yön"})
		return
	}
	var exists bool
	if err := s.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM contents WHERE id=$1 AND status='yayinda')`, body.ID).Scan(&exists); err != nil || !exists {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "içerik bulunamadı"})
		return
	}
	if err := repo.SetContentReaction(r.Context(), s.pool, body.ID, s.cardVisitor(w, r), reaction); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "kaydedilemedi"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "reaction": reaction})
}

// cardVisitor ziyaretçi kimliğini döndürür; yoksa üretip çerez yazar.
func (s *Server) cardVisitor(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie("tb_visitor"); err == nil && len(c.Value) >= 16 {
		return c.Value
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	v := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: "tb_visitor", Value: v, Path: "/",
		MaxAge: 365 * 24 * 60 * 60, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode,
	})
	return v
}
