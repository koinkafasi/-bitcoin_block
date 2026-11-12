package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

func (s *Server) handleAdminAra(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "gruplar": []any{}, "q": q})
		return
	}
	gruplar, err := repo.AdminArama(r.Context(), s.pool, s.curSiteID(r), q, 5)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError,
			map[string]any{"ok": false, "error": "Arama başarısız", "gruplar": []any{}})
		return
	}
	if gruplar == nil {
		gruplar = []repo.AdminAramaGrup{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "gruplar": gruplar, "q": q})
}
