package httpapi

import (
	"html/template"
	"net/http"
	"strings"

	"theblock/site/internal/webui/templkit"
)

func (s *Server) handleBilesenler(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "Bileşenler")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	var sb strings.Builder
	if err := templkit.Gallery().Render(r.Context(), &sb); err != nil {
		s.logger.Error("templ galeri", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}
