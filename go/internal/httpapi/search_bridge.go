package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/searchsvc"
)

// arama motoru istemcisi nil ise SQL/pg_trgm kullanılır.
var _ = http.StatusOK

// contentsBySearchIDs arama sonucu id listesini DB sırasına göre içeriklere çevirir.
func (s *Server) contentsBySearchIDs(ctx context.Context, siteID int64, hits []searchsvc.Hit) ([]repo.Content, error) {
	if len(hits) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(hits))
	for _, h := range hits {
		if id, err := strconv.ParseInt(strings.TrimSpace(h.ID), 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := repo.RecentPublishedAllLang(ctx, s.pool, siteID, 2000)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]repo.Content, len(all))
	for _, c := range all {
		if c.SiteID == siteID {
			byID[c.ID] = c
		}
	}
	out := make([]repo.Content, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}
