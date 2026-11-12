package httpapi

import (
	"context"
	"net/http"

	"theblock/site/internal/repo"
)

const (
	gazeteActionView     = "goruntule"
	gazeteActionEdit     = "duzenle"
	gazeteActionPublish  = "yayinla"
	gazeteActionSubs     = "abone"
	gazeteActionTeam     = "ekip"
	gazeteActionLicense  = "lisans"
	gazeteActionTemplate = "sablon"
)

// gazeteRole kullanıcının sitedeki gazete rolünü döndürür (yetkisizse boş).
func (s *Server) gazeteRole(ctx context.Context, siteID int64) string {
	u := userFromContext(ctx)
	if u.ID == 0 {
		return ""
	}
	if u.Can("Yönetici") {
		return "yonetici"
	}
	if rol, err := repo.GazeteTeamRole(ctx, s.pool, siteID, u.ID); err == nil && rol != "" {
		return rol
	}
	if u.Can("Editör") {
		return "editor"
	}
	return ""
}

// gazeteCan eylem için yetkiyi doğrular (rol matrisi §8.1).
func (s *Server) gazeteCan(ctx context.Context, siteID int64, action string) bool {
	return gazeteRoleAllows(s.gazeteRole(ctx, siteID), action)
}

// gazeteRoleAllows rol → eylem matrisini uygular (saf; test edilebilir).
func gazeteRoleAllows(role, action string) bool {
	switch action {
	case gazeteActionView:
		return role != ""
	case gazeteActionEdit, gazeteActionPublish, gazeteActionTemplate:
		return role == "yonetici" || role == "sorumlu" || role == "editor"
	case gazeteActionSubs, gazeteActionTeam:
		return role == "yonetici" || role == "sorumlu"
	case gazeteActionLicense:
		return role == "yonetici"
	}
	return false
}

// gazeteLicensed sitenin gazete lisansı açık mı? (Yönetici yapılandırma için istisna)
func (s *Server) gazeteLicensed(ctx context.Context, siteID int64) bool {
	if s.gazeteRole(ctx, siteID) == "yonetici" {
		return true
	}
	on, err := repo.SiteFeatureEnabled(ctx, s.pool, siteID, "gazete")
	return err == nil && on
}

// requireGazete yetki + lisans kontrolü yapar; uygun değilse yanıt yazar ve false döner.
func (s *Server) requireGazete(w http.ResponseWriter, r *http.Request, action string) bool {
	sid := s.curSiteID(r)
	ctx := r.Context()
	if !s.gazeteCan(ctx, sid, action) {
		s.gazeteDenied(w, r)
		return false
	}
	if action != gazeteActionLicense && !s.gazeteLicensed(ctx, sid) {
		s.gazeteDeniedText(w, r, "Gazete modülü bu site için aktif değil. Yönetim → Lisans bölümünden açın.")
		return false
	}
	return true
}

func (s *Server) gazeteDenied(w http.ResponseWriter, r *http.Request) {
	s.gazeteDeniedText(w, r, "Bu işlem için yetkiniz yok.")
}

func (s *Server) gazeteDeniedText(w http.ResponseWriter, r *http.Request, text string) {
	if IsHTMX(r) {
		s.renderGazetePanelV2(w, r, s.gazeteView(r, &flashMsg{Kind: "error", Text: text}))
		return
	}
	s.flashRedirect(w, r, "/admin/gazete", "error", text)
}
