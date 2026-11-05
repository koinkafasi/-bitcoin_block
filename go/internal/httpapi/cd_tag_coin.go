package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Etiket sayfası (§4 /tag/{tag}) ve tek varlık fiyat sayfası (/price/{coin}).
//
// NEDEN: makale altındaki etiket rozetleri ve fiyat tablosundaki satırlar
// bu adreslere bağlanıyor. Rota yokken ikisi de 404 dönüyordu (ölçüm:
// 5 Kasım 2026) — kullanıcı için "site yarım" demek.

// handleCDTag /tag/{slug}.
func (s *Server) handleCDTag(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		s.handleNotFound(w, r)
		return
	}
	ctx := r.Context()

	var ad string
	_ = s.pool.QueryRow(ctx, `SELECT name FROM tags WHERE slug = $1`, slug).Scan(&ad)
	if ad == "" {
		s.handleNotFound(w, r)
		return
	}

	sayfa := 1
	items, toplam, err := repo.TagIcerikleri(ctx, s.pool, 1, "tr", slug, sayfa, 24)
	if err != nil {
		s.logger.Warn("etiket sayfası", "etiket", slug, "error", err)
	}

	simdi := time.Now()
	var kartlar []templkit.CDStory
	for _, c := range items {
		kartlar = append(kartlar, cdStoryCevir(c, simdi))
	}

	v := templkit.CDSectionPage{
		Shell:     s.cdShell(""),
		Title:     "#" + ad + " — " + s.siteName,
		Desc:      ad + " etiketli son kripto haberleri ve analizler.",
		Canonical: s.cfg.BaseURL + "/tag/" + slug,
		Section:   "#" + ad,
		Href:      "/tag/" + slug,
		Page:      sayfa,
		Stories:   kartlar,
		JSONLD:    string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/tag/"+slug)),
	}
	if len(kartlar) > 0 {
		v.Lead = kartlar[0]
		v.Stories = kartlar[1:]
	}
	if toplam > 24 {
		v.NextHref = "/tag/" + slug + "?sayfa=2"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDSectionPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("etiket render", "error", err)
	}
}

// handleCDCoin /price/{coin} — tek varlık fiyat sayfası.
//
// Veri CoinGecko listesinden gelir (ayrı istek yok): sembol eşleşmesi
// yeterli. Bilinmeyen sembolde 404.
func (s *Server) handleCDCoin(w http.ResponseWriter, r *http.Request) {
	sym := strings.ToUpper(strings.TrimSpace(r.PathValue("coin")))
	if sym == "" {
		s.handleNotFound(w, r)
		return
	}
	fiyatlar := repo.Fiyatlar(r.Context(), s.pool)
	var bulunan *repo.FiyatSatiri
	for i := range fiyatlar {
		if fiyatlar[i].Sym == sym {
			bulunan = &fiyatlar[i]
			break
		}
	}
	if bulunan == nil {
		// Sembol listesi dışında: şimdilik 404 (CoinGecko tüm coin listesi
		// Faz 2'de; uydurma sayfa üretmiyoruz).
		s.handleNotFound(w, r)
		return
	}

	v := templkit.CDCoinPage{
		Shell:     s.cdShell("price"),
		Title:     bulunan.Name + " (" + sym + ") Fiyatı — " + s.siteName,
		Desc:      bulunan.Name + " canlı fiyatı, 24 saatlik değişimi ve piyasa sıralaması.",
		Canonical: s.cfg.BaseURL + "/price/" + strings.ToLower(sym),
		Sym:       sym,
		Name:      bulunan.Name,
		Rank:      bulunan.Rank,
		Price:     repo.FiyatBicim(bulunan.Price),
		Change:    repo.DegisimBicim(bulunan.Change),
		Up:        bulunan.Change >= 0,
		Others:    s.cdFiyatSatirlari(r, 12),
		JSONLD:    string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/price/"+strings.ToLower(sym))),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDCoinPageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("coin render", "error", err)
	}
}
