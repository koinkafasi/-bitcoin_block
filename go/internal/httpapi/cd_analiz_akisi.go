package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

func (s *Server) cdAnalizAkisi(r *http.Request, adet int) ([]templkit.CDAnalizKart, int) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	dil := s.cdYerel(r)
	rows, err := repo.AnalizAkisi(ctx, s.pool, sid, dil, adet)
	if err != nil || len(rows) < 6 {
		return nil, 0
	}
	simdi := time.Now()
	out := make([]templkit.CDAnalizKart, 0, len(rows))
	for _, k := range rows {
		spot := strings.TrimSpace(k.Spot)
		if spot == "" || k.Href == "" {
			continue
		}
		kart := templkit.CDAnalizKart{
			Baslik:   strings.TrimSpace(k.Baslik),
			Spot:     spot,
			Kategori: templkit.CDT(dil, strings.TrimSpace(k.Kategori)),
			Href:     k.Href,
		}
		if !k.Tarih.IsZero() {
			kart.Tarih = goreliZaman(k.Tarih, simdi, dil)
		}
		out = append(out, kart)
	}
	if len(out) < 6 {
		return nil, 0
	}
	return out, repo.AnalizSayisi(ctx, s.pool, sid, "tr", 7)
}
