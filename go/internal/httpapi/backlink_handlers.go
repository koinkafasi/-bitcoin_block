package httpapi

import (
	"net/http"
	"sort"

	"theblock/site/internal/repo"
)

// ---- D9: İç link (backlink) ağı yönetim görünümü ----

type backlinkData struct {
	Stats    repo.LinkNetworkStats
	Clusters []repo.Cluster
	Orphans  []repo.Content
}

func (s *Server) handleBacklinkAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildBacklinkData(r)
	if IsHTMX(r) {
		s.renderBacklinkPanelV2(w, r, d)
		return
	}
	s.renderBacklinkPageV2(w, r, d)
}

func (s *Server) buildBacklinkData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "İç Link Ağı (Backlink)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	stats, _ := repo.NetworkStats(ctx, s.pool, sid)
	clusters, _ := repo.ListClusters(ctx, s.pool, sid)
	sort.SliceStable(clusters, func(i, j int) bool { return clusters[i].ContentCount > clusters[j].ContentCount })
	orphans, _ := repo.OrphanContents(ctx, s.pool, sid, 50)
	d.Backlink = &backlinkData{Stats: stats, Clusters: clusters, Orphans: orphans}
	return d
}
