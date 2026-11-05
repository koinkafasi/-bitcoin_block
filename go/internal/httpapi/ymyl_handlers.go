package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// handleSourceEkle içeriğe kanıt kaynağı ekler (erişim anında yoklanır).
func (s *Server) handleSourceEkle(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	url := strings.TrimSpace(r.FormValue("kaynak_url"))
	if id == 0 || url == "" {
		s.flashRedirect(w, r, "/admin/icerikler/"+r.PathValue("id"), "error", "URL gerekli.")
		return
	}
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		_, err := repo.AddContentSource(r.Context(), s.pool, id, url)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), "error", "Kaynak eklenemedi.")
			return
		}
		s.audit(r, "kaynak_ekle", itoa(int(id))+" "+url)
		s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), "info", "Kaynak eklendi ve yoklandı.")
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), "error", "Geçersiz URL.")
}

// handleSourceKontrol tüm kaynakları yeniden yoklar.
func (s *Server) handleSourceKontrol(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	n, _ := repo.CheckSources(r.Context(), s.pool, id)
	s.audit(r, "kaynak_kontrol", itoa(int(id)))
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), "info", "Kaynaklar yoklandı: "+itoa(n)+" erişilebilir.")
}

// handleSourceSil kaynağı siler.
func (s *Server) handleSourceSil(w http.ResponseWriter, r *http.Request) {
	sid := parseID(r.PathValue("sid"))
	geri := func(kind, msg string) {
		u := s.ayniSiteYolu(r, r.Referer())
		if u == "" {
			u = "/admin/icerikler"
		}
		s.flashRedirect(w, r, u, kind, msg)
	}
	var contentID int64
	if err := s.pool.QueryRow(r.Context(), `SELECT content_id FROM content_sources WHERE id=$1`, sid).Scan(&contentID); err != nil {
		geri("error", "Kaynak bulunamadı.")
		return
	}
	c, err := repo.GetContent(r.Context(), s.pool, contentID)
	if err != nil || !canEditContent(userFromContext(r.Context()), c) {
		geri("error", "Bu kaynağı silme yetkiniz yok.")
		return
	}
	_ = repo.DeleteContentSource(r.Context(), s.pool, sid)
	geri("info", "Kaynak silindi.")
}
