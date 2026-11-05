package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"theblock/site/internal/repo"
)

var personRoles = []string{"Yönetici", "Editör", "Muhabir", "Yazar"}

func validPersonRole(role string) bool {
	for _, r := range personRoles {
		if r == role {
			return true
		}
	}
	return false
}

// buildPeopleAdminData yönetim paneli verisini kurar.
func (s *Server) buildPeopleAdminData(r *http.Request) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "Yazarlar & Editörler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.AdminPeople, _ = repo.ListPeopleAdmin(ctx, s.pool)
	d.PersonMedia, _ = repo.ListMedia(ctx, s.pool, s.curSiteID(r), userFromContext(ctx).ID, 300)
	if id := parseID(r.PathValue("id")); id > 0 {
		if p, err := repo.GetPersonByID(ctx, s.pool, id); err == nil {
			d.EditPerson = &p
			if p.AvatarMediaID != nil {
				d.EditAvatarID = *p.AvatarMediaID
			}
		}
	}
	d.PersonPanel = true
	return d
}

// handlePeopleAdmin yazar/editör yönetim sayfası.
func (s *Server) handlePeopleAdmin(w http.ResponseWriter, r *http.Request) {
	if !userFromContext(r.Context()).Can("Yönetici") {
		s.flashRedirect(w, r, "/admin", "error", "Bu sayfa için yönetici yetkisi gerekir.")
		return
	}
	d := s.buildPeopleAdminData(r)
	if IsHTMX(r) {
		s.renderPeoplePanelV2(w, r, d)
		return
	}
	s.renderPeoplePageV2(w, r, d)
}

// handlePersonAdminSave profil alanlarını kaydeder.
func (s *Server) handlePersonAdminSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !userFromContext(ctx).Can("Yönetici") {
		s.flashRedirect(w, r, "/admin", "error", "Bu işlem için yönetici yetkisi gerekir.")
		return
	}
	id := parseID(r.PathValue("id"))
	p, err := repo.GetPersonByID(ctx, s.pool, id)
	if err != nil {
		s.flashRedirect(w, r, "/admin/yazarlar", "error", "Kişi bulunamadı.")
		return
	}

	p.Name = strings.TrimSpace(r.FormValue("ad"))
	if p.Name == "" {
		s.flashRedirect(w, r, "/admin/yazarlar/"+itoa(int(id)), "error", "Ad boş olamaz.")
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	if slug == "" {
		slug = repo.PersonSlug(p.Name)
	} else {
		slug = repo.PersonSlug(slug)
	}
	if slug == "" {
		slug = "yazar-" + itoa(int(id))
	}
	p.Slug = slug
	p.Title = strings.TrimSpace(r.FormValue("unvan"))
	p.Bio = strings.TrimSpace(r.FormValue("biyografi"))
	p.Role = r.FormValue("rol")
	if !validPersonRole(p.Role) {
		p.Role = "Yazar"
	}
	p.Status = r.FormValue("durum")
	if p.Status != "aktif" && p.Status != "pasif" && p.Status != "askida" {
		p.Status = "aktif"
	}
	p.AvatarMediaID = optID(r.FormValue("avatar_media_id"))
	p.Socials = map[string]string{}
	for _, key := range []string{"x", "linkedin", "website", "email"} {
		if v := strings.TrimSpace(r.FormValue("sosyal_" + key)); v != "" {
			p.Socials[key] = v
		}
	}

	if err := repo.UpdatePersonProfile(ctx, s.pool, p); err != nil {
		var pgErr *pgconn.PgError
		msg := "Kaydedilemedi."
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			msg = "Bu slug başka bir kişide kullanılıyor."
		}
		if IsHTMX(r) {
			d := s.buildPeopleAdminData(r)
			d.Flash = &flashMsg{Kind: "error", Text: msg}
			s.renderPeoplePanelV2(w, r, d)
			return
		}
		s.flashRedirect(w, r, "/admin/yazarlar/"+itoa(int(id)), "error", msg)
		return
	}
	s.audit(r, "person_update", p.Name+" ("+p.Role+")")

	if IsHTMX(r) {
		d := s.buildPeopleAdminData(r)
		d.Flash = &flashMsg{Kind: "info", Text: "Profil kaydedildi."}
		s.renderPeoplePanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/yazarlar/"+itoa(int(id)), "info", "Profil kaydedildi.")
}
