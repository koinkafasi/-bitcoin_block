package httpapi

import "net/http"

func (s *Server) handleMerkezKaldirildi(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusMovedPermanently)
}
