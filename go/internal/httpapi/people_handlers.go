package httpapi

import (
	"net/http"

	"theblock/site/internal/repo"
)

// handlePeopleList /yazarlar — yazar ve editör dizini (Faz 51).
func (s *Server) handlePeopleList(w http.ResponseWriter, r *http.Request) {
	// Yeni tasarıma devredildi. Eski şablon yolu (people_list) `return`
	// altında ölü kod olarak duruyordu; `go vet` bunu "unreachable"
	// diye işaretliyordu.
	s.cdYazarlarSayfasi(w, r)
}

// handlePerson /yazar/{slug} — kişi profili + yazıları (Faz 51).
func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, err := repo.GetPersonBySlug(r.Context(), s.pool, slug)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	s.cdYazarProfilSayfasi(w, r, p)
}
