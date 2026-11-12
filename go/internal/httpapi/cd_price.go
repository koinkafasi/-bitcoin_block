package httpapi

import (
	"context"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Fiyat şeridi + /price sayfası (CoinDesk §3 ve §14).

// cdTickerDoldur istek bağlamından şerit sekmesini okuyup doldurur.
func (s *Server) cdTickerDoldur(r *http.Request) []templkit.CDTickerItem {
	sekme := "markets"
	if strings.TrimSpace(r.URL.Query().Get("serit")) == "stocks" {
		sekme = "stocks"
	}
	return s.cdTickerDoldurSekme(r.Context(), sekme)
}

func (s *Server) cdTickerDoldurSekme(ctx context.Context, sekme string) []templkit.CDTickerItem {
	if sekme == "stocks" {
		// BIST verisi `services` işinin önbelleğinden gelir (Yahoo Finance).
		// Önbellek boşsa şerit BOŞ kalır — uydurma fiyat basılmaz.
		bd := repo.BorsaOf(ctx, s.pool)
		if bd.Demo || len(bd.Rows) == 0 {
			return nil
		}
		out := make([]templkit.CDTickerItem, 0, len(bd.Rows))
		for _, r := range bd.Rows {
			if r.Kind == "kripto" {
				continue
			}
			if r.Value <= 0 {
				continue
			}
			out = append(out, templkit.CDTickerItem{
				Sym:    strings.TrimSuffix(r.Code, ".IS"),
				Href:   "/price/stock/" + hisseBorsaKodu(r.Code),
				Price:  repo.FiyatBicimP(r.Value, r.Para),
				Change: repo.DegisimBicim(r.Delta),
				Up:     r.Delta >= 0,
			})
			if len(out) >= 7 {
				break
			}
		}
		return out
	}

	fiyatlar := repo.Fiyatlar(ctx, s.pool)
	var out []templkit.CDTickerItem
	for _, f := range fiyatlar {
		if len(out) >= 7 {
			break
		}
		slug := repo.CoinSlug(f.Name)
		out = append(out, templkit.CDTickerItem{
			Sym:    f.Sym,
			Slug:   slug,
			Href:   "/price/" + slug,
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
			Href:   "/price/" + repo.CoinSlug(f.Name),
		})
	}
	return out
}

// cdDigerVarliklar kendi varlığını dışarıda bırakan fiyat listesi.
func (s *Server) cdDigerVarliklar(r *http.Request, adet int, haricSym string) []templkit.CDPriceRow {
	haricSym = strings.ToUpper(strings.TrimSpace(haricSym))
	fiyatlar := repo.Fiyatlar(r.Context(), s.pool)
	out := make([]templkit.CDPriceRow, 0, adet)
	for _, f := range fiyatlar {
		if len(out) >= adet {
			break
		}
		if haricSym != "" && strings.EqualFold(f.Sym, haricSym) {
			continue
		}
		out = append(out, templkit.CDPriceRow{
			Rank:   f.Rank,
			Sym:    f.Sym,
			Name:   f.Name,
			Price:  repo.FiyatBicim(f.Price),
			Change: repo.DegisimBicim(f.Change),
			Up:     f.Change >= 0,
			Href:   "/price/" + repo.CoinSlug(f.Name),
		})
	}
	return out
}

func hisseBorsaKodu(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" || strings.HasSuffix(c, ".IS") {
		return c
	}
	return c + ".IS"
}

func (s *Server) cdOndeGelenHisseler(ctx context.Context, adet int) []templkit.CDPriceRow {
	bd := repo.BorsaOf(ctx, s.pool)
	if bd.Demo {
		return nil
	}
	out := make([]templkit.CDPriceRow, 0, adet)
	for _, r := range bd.Rows {
		if len(out) >= adet {
			break
		}
		if r.Kind != "hisse" || r.Value <= 0 {
			continue
		}
		out = append(out, templkit.CDPriceRow{
			Sym:    strings.TrimSuffix(r.Code, ".IS"),
			Name:   r.Name,
			Price:  repo.FiyatBicimP(r.Value, r.Para),
			Change: repo.DegisimBicim(r.Delta),
			Up:     r.Delta >= 0,
			// Detay sayfası `.IS` ekiyle istenir (Yahoo BIST sembolü).
			Href: "/price/stock/" + hisseBorsaKodu(r.Code),
		})
	}
	return out
}

func (s *Server) handleCDPrice(w http.ResponseWriter, r *http.Request) {
	rows := s.cdFiyatSatirlari(r, 50)
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	kanonik := base + "/price"

	// CoinDesk /price: liste sayfası → BreadcrumbList + ItemList
	// (FinancialProduct). WebSite bloğu da korunuyor (marka/arama kutusu).
	var ldItems []cdLDItem
	for _, row := range rows {
		ldItems = append(ldItems, cdLDItem{Headline: row.Name, URL: base + row.Href})
	}
	var ld strings.Builder
	ld.WriteString(string(siteAdiJSONLD(s.siteName, kanonik)))
	ld.WriteString(string(breadcrumbScript([]crumb{
		{Name: s.siteName, URL: base + "/"},
		{Name: "Fiyatlar", URL: kanonik},
	})))
	ld.WriteString(string(cdFiyatListesiJSONLD("Cryptocurrency Market Cap Rankings", ldItems)))

	// Piyasa Nabzı: /price sayfası da grafik gösterir (gerçek veri).
	nbFiyat, nbKartlar := s.cdNabiz(r.Context())

	v := templkit.CDPricePage{
		Shell:           s.cdShellCtx(r.Context(), r, "price"),
		Title:           "Kripto Fiyatları — " + s.siteName,
		Desc:            "Bitcoin, Ethereum ve önde gelen kripto paraların canlı fiyatları, 24 saatlik değişimleri ve piyasa sıralaması.",
		Canonical:       kanonik,
		Rows:            rows,
		Hisseler:        s.cdOndeGelenHisseler(r.Context(), 12),
		HisselerHref:    "/borsa/hisseler",
		NabizFiyat:      nbFiyat,
		Nabiz:           nbKartlar,
		JSONLD:          ld.String(),
		PageCategory:    "price_page",
		ContentCategory: "price",
		Locale:          "tr",
		BaseURL:         base,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDPricePageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd fiyat render", "error", err)
	}
}
