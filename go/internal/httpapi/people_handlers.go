package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// handlePeopleList /yazarlar — yazar ve editör dizini (Faz 51).
func (s *Server) handlePeopleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Yazarlar")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.People, _ = repo.ListPeople(ctx, s.pool, []string{"Yönetici", "Editör", "Muhabir", "Yazar"})
	d.PeoplePage = true
	d.Canonical = strings.TrimRight(s.absBase(r), "/") + r.URL.Path
	s.renderPublic(w, r, "people_list", d, cacheControlPublic)
}

// handlePerson /yazar/{slug} — kişi profili + yazıları (Faz 51).
func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	p, err := repo.GetPersonBySlug(ctx, s.pool, slug)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	page := 1
	if n, e := strconv.Atoi(r.URL.Query().Get("sayfa")); e == nil && n > 0 {
		page = n
	}
	const per = 12
	items, total, _ := repo.PersonContents(ctx, s.pool, p.ID, per, (page-1)*per)

	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), p.Name)
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Person = &p
	d.PersonContents = items
	d.PersonTotal = total
	d.PersonPage = page
	d.PersonHasPrev = page > 1
	d.PersonHasNext = page*per < total
	d.PersonPrevPage = page - 1
	d.PersonNextPage = page + 1
	d.PersonDetail = true
	d.Canonical = strings.TrimRight(s.absBase(r), "/") + r.URL.Path
	s.renderPublic(w, r, "person", d, cacheControlPublic)
}
