package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Fiyat şeridi + /price sayfası (CoinDesk §3 ve §14).

// cdTickerDoldur şerit kalemlerini doldurur. Veri yoksa şerit boş kalır
// (boş şerit basılmaz — cdTicker len kontrolü yapar).
func (s *Server) cdTickerDoldur(r *http.Request) []templkit.CDTickerItem {
	fiyatlar := repo.Fiyatlar(r.Context(), s.pool)
	var out []templkit.CDTickerItem
	for _, f := range fiyatlar {
		if len(out) >= 7 {
			break
		}
		out = append(out, templkit.CDTickerItem{
			Sym:    f.Sym,
			Price:  repo.FiyatBicim(f.Price),
			Change: repo.DegisimBicim(f.Change),
			Up:     f.Change >= 0,
		})
	}
	return out
}

// cdFiyatSatirlari tablo satırlarını doldurur.
func (s *Server) cdFiyatSatirlari(r *http.Request, adet int) []templkit.CDPriceRow {
	fiyatlar := repo.Fiyatlar(r.Context(), s.pool)
	var out []templkit.CDPriceRow
	for _, f := range fiyatlar {
		if len(out) >= adet {
			break
		}
		out = append(out, templkit.CDPriceRow{
			Rank:   f.Rank,
			Sym:    f.Sym,
			Name:   f.Name,
			Price:  repo.FiyatBicim(f.Price),
			Change: repo.DegisimBicim(f.Change),
			Up:     f.Change >= 0,
			Href:   "/price/" + strings.ToLower(f.Sym),
		})
	}
	return out
}

// handleCDPrice /price — CoinDesk fiyat sayfası.
//
// Yapı: H1 + açıklama → fiyat tablosu (25 varlık) → bölüm bağlantıları.
func (s *Server) handleCDPrice(w http.ResponseWriter, r *http.Request) {
	rows := s.cdFiyatSatirlari(r, 50)

	v := templkit.CDPricePage{
		Shell:     s.cdShell("price"),
		Title:     "Kripto Fiyatları — " + s.siteName,
		Desc:      "Bitcoin, Ethereum ve önde gelen kripto paraların canlı fiyatları, 24 saatlik değişimleri ve piyasa sıralaması.",
		Canonical: s.cfg.BaseURL + "/price",
		Rows:      rows,
		JSONLD:    string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/price")),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDPricePageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd fiyat render", "error", err)
	}
}
