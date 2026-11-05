package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// kumelerData Konu Kümesi sayfa verisidir.
type kumelerData struct {
	Clusters []repo.Cluster
	Active   *repo.Cluster
	Items    []repo.Content
	Pool     []repo.Content
	Warnings []repo.CannPair
}

// handleKumelerGet kümeleri listeler; ?kus=ID aktif küme detayını gösterir.
func (s *Server) handleKumelerGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildKumelerData(r)
	if IsHTMX(r) {
		s.renderKonuKumeleriPanel(w, r, d)
		return
	}
	s.renderKonuKumeleriPage(w, r, d)
}

func (s *Server) buildKumelerData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "Konu Kümeleri (SEO)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	kd := &kumelerData{}
	clusters, _ := repo.ListClusters(ctx, s.pool, sid)
	kd.Clusters = clusters

	kid := parseID(r.URL.Query().Get("kus"))
	for i := range clusters {
		if clusters[i].ID == kid {
			kd.Active = &clusters[i]
		}
	}
	if kd.Active != nil {
		kd.Items, _ = repo.ClusterContents(ctx, s.pool, kid, 100)
		kd.Pool, _ = repo.ClusterContentsPool(ctx, s.pool, sid, kid, 80)
		kd.Warnings, _ = repo.Cannibalization(ctx, s.pool, kid)
	} else if len(clusters) > 0 {
		kd.Active = &clusters[0]
		kid = clusters[0].ID
		kd.Items, _ = repo.ClusterContents(ctx, s.pool, kid, 100)
		kd.Pool, _ = repo.ClusterContentsPool(ctx, s.pool, sid, kid, 80)
		kd.Warnings, _ = repo.Cannibalization(ctx, s.pool, kid)
	}

	d.Kumeler = kd
	return d
}

func (s *Server) renderKumelerPanel(w http.ResponseWriter, r *http.Request, kid int64, kind, text string) {
	q := r.URL.Query()
	if kid != 0 {
		q.Set("kus", itoa(int(kid)))
	}
	r2 := r.Clone(r.Context())
	r2.URL.RawQuery = q.Encode()
	d := s.buildKumelerData(r2)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderKonuKumeleriPanel(w, r, d)
}

func (s *Server) kumelerResult(w http.ResponseWriter, r *http.Request, kid int64, kind, text string) {
	if IsHTMX(r) {
		s.renderKumelerPanel(w, r, kid, kind, text)
		return
	}
	path := "/admin/kumeler"
	if kid != 0 {
		path += "?kus=" + itoa(int(kid))
	}
	s.flashRedirect(w, r, path, kind, text)
}

// handleKumeEkle yeni küme ekler.
func (s *Server) handleKumeEkle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		s.kumelerResult(w, r, parseID(r.URL.Query().Get("kus")), "error", "Ad gerekli.")
		return
	}
	intent := r.FormValue("intent")
	if intent == "" {
		intent = "bilgi"
	}
	if _, err := repo.CreateCluster(r.Context(), s.pool, s.curSiteID(r), name, intent); err != nil {
		s.kumelerResult(w, r, 0, "error", "Eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "kume_ekle", name)
	s.kumelerResult(w, r, parseID(r.URL.Query().Get("kus")), "info", name+" eklendi.")
}

// handleKumeSil küme siler.
func (s *Server) handleKumeSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteCluster(r.Context(), s.pool, id); err != nil {
		s.kumelerResult(w, r, 0, "error", "Silinemedi.")
		return
	}
	s.kumelerResult(w, r, 0, "info", "Silindi.")
}

// handleKumeIcerikEkle içeriği kümeye bağlar (islem=cikar ise çıkarır).
func (s *Server) handleKumeIcerikEkle(w http.ResponseWriter, r *http.Request) {
	kid := parseID(r.PathValue("id"))
	cid := parseID(r.PathValue("cid"))
	kme := kid
	if r.URL.Query().Get("islem") == "cikar" {
		kme = 0
	}
	_ = repo.SetContentCluster(r.Context(), s.pool, cid, kme)
	s.audit(r, "kume_icerik", "kume="+itoa(int(kme))+" icerik="+itoa(int(cid)))
	s.kumelerResult(w, r, kid, "info", "İçerik kümeye eklendi / çıkarıldı.")
}
