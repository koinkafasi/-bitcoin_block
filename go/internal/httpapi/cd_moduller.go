package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

func (s *Server) cdModuller(ctx context.Context, r *http.Request) []templkit.CDModul {
	site := s.siteByDomain(ctx, r.Host)
	raw, err := repo.GetLayout(ctx, s.pool, site.ID, "home")
	if err != nil || len(raw) == 0 {
		return nil
	}
	var bloklar []struct {
		Type    string `json:"type"`
		Visible *bool  `json:"visible"`
		Count   int    `json:"count"`
		Opts    struct {
			Count int `json:"count"`
		} `json:"opts"`
	}
	if err := json.Unmarshal(raw, &bloklar); err != nil {
		return nil
	}
	var out []templkit.CDModul
	for _, b := range bloklar {
		if !templkit.CDModulGecerli(b.Type) {
			continue
		}
		goster := true
		if b.Visible != nil {
			goster = *b.Visible
		}
		adet := b.Count
		if adet == 0 {
			adet = b.Opts.Count
		}
		out = append(out, templkit.CDModul{Tip: b.Type, Goster: goster, Adet: adet})
	}
	return out
}
