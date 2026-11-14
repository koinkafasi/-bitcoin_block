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

// cdTickerOnemliHisse şeritte öncelikli gösterilecek BIST hisseleri.
//
// NEDEN SABİT LİSTE: "hisseler" önbelleği 631 satır ve fiyata göre sıralı;
// ilk 20 satır ISATR/ISKUR gibi düşük likidite araçlarıyla doluyordu.
// Şeritte "önemli" hisse görünmeli — hacim/piyasa değeri sırası elimizde
// olmadığı için piyasa tarafından bilinen büyük şirketler listeleniyor.
var cdTickerOnemliHisse = []string{
	"THYAO", "ASELS", "BIMAS", "TUPRS", "KCHOL", "GARAN",
	"AKBNK", "SISE", "EREGL", "SAHOL", "TCELL", "FROTO",
	"TOASO", "PGSUS", "KRDMD", "EKGYO", "ARCLK", "SASA",
}

// cdTickerDoldurSekme fiyat şeridini doldurur.
//
// KALEM SAYISI (12 Kasım 2026, kullanıcı isteği "sürekli aynı veriler
// dönmesin yeni coin ve bilgiler ekle"): eskiden 7 kalem vardı ve şerit
// kısa olduğu için aynı 7 fiyat dönüp duruyordu. CoinGecko önbelleğinde
// 25 coin var; 16'sı şeride giriyor. Hisse tarafında küratörlü 18 kod
// öncelikli, yetmezse borsa önbelleğindeki diğer hisseler ekleniyor.
func (s *Server) cdTickerDoldurSekme(ctx context.Context, sekme string) []templkit.CDTickerItem {
	const istenenKalem = 16

	if sekme == "stocks" {
		// BIST verisi `services` işinin önbelleğinden gelir (Yahoo Finance).
		// Önbellek boşsa şerit BOŞ kalır — uydurma fiyat basılmaz.
		bd := repo.BorsaOf(ctx, s.pool)
		if bd.Demo || len(bd.Rows) == 0 {
			return nil
		}

		// Koddan satıra hızlı erişim (hem borsa hem hisseler önbelleği).
		borsaSatir := map[string]repo.BorsaRow{}
		var endeksler []repo.BorsaRow
		for _, r := range bd.Rows {
			if r.Kind == "kripto" || r.Value <= 0 {
				continue
			}
			kod := strings.TrimSuffix(r.Code, ".IS")
			borsaSatir[kod] = r
			if r.Kind == "endeks" {
				endeksler = append(endeksler, r)
			}
		}

		hisseSatir := map[string]repo.HisseRow{}
		if hd := repo.HisselerOf(ctx, s.pool); len(hd.Rows) > 0 {
			for _, r := range hd.Rows {
				if r.Value > 0 {
					hisseSatir[strings.TrimSuffix(r.Code, ".IS")] = r
				}
			}
		}

		ekle := func(out []templkit.CDTickerItem, kod, ad string, fiyat, delta float64, para string) []templkit.CDTickerItem {
			if kod == "" || fiyat <= 0 {
				return out
			}
			return append(out, templkit.CDTickerItem{
				Sym:    kod,
				Href:   "/price/stock/" + hisseBorsaKodu(kod),
				Price:  repo.FiyatBicimP(fiyat, para),
				Change: repo.DegisimBicim(delta),
				Up:     delta >= 0,
			})
		}

		var out []templkit.CDTickerItem
		// 1) Endeksler (BIST 100 vb.) — şeridin başında bağlam verir.
		for _, r := range endeksler {
			out = ekle(out, strings.TrimSuffix(r.Code, ".IS"), r.Name, r.Value, r.Delta, r.Para)
		}
		// 2) Küratörlü önemli hisseler.
		for _, kod := range cdTickerOnemliHisse {
			if len(out) >= istenenKalem {
				break
			}
			if r, ok := borsaSatir[kod]; ok {
				out = ekle(out, kod, r.Name, r.Value, r.Delta, r.Para)
				continue
			}
			if r, ok := hisseSatir[kod]; ok {
				out = ekle(out, kod, r.Name, r.Value, r.Delta, "TRY")
			}
		}
		// 3) Yetmezse borsa önbelleğindeki diğer hisseler.
		for _, r := range bd.Rows {
			if len(out) >= istenenKalem {
				break
			}
			if r.Kind != "hisse" {
				continue
			}
			kod := strings.TrimSuffix(r.Code, ".IS")
			if _, varMi := borsaSatir[kod]; !varMi {
				continue
			}
			zaten := false
			for _, o := range out {
				if o.Sym == kod {
					zaten = true
					break
				}
			}
			if zaten {
				continue
			}
			out = ekle(out, kod, r.Name, r.Value, r.Delta, r.Para)
		}
		return out
	}

	fiyatlar := repo.Fiyatlar(ctx, s.pool)
	var out []templkit.CDTickerItem
	for _, f := range fiyatlar {
		if len(out) >= istenenKalem {
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
	ld.WriteString(string(siteAdiJSONLD(s.siteAdi(), kanonik)))
	ld.WriteString(string(breadcrumbScript([]crumb{
		{Name: s.siteAdi(), URL: base + "/"},
		{Name: templkit.CDT(s.cdYerel(r), "Fiyatlar"), URL: kanonik},
	})))
	ld.WriteString(string(cdFiyatListesiJSONLD("Cryptocurrency Market Cap Rankings", ldItems)))

	// Piyasa Nabzı: /price sayfası da grafik gösterir (gerçek veri).
	nbFiyat, nbKartlar := s.cdNabiz(r.Context())

	v := templkit.CDPricePage{
		Shell:           s.cdShellCtx(r.Context(), r, "price"),
		Title:           templkit.CDT(s.cdYerel(r), "Kripto Fiyatları") + " — " + s.siteAdi(),
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
		Locale:          s.cdYerel(r),
		BaseURL:         base,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDPricePageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd fiyat render", "error", err)
	}
}
