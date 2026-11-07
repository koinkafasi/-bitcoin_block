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
	var ldItems []cdLDItem
	for i, c := range items {
		kartlar = append(kartlar, cdStoryCevir(c, simdi))
		if i < 11 {
			it := cdLDItem{Headline: c.Title, URL: s.absBase(r) + makaleURL(c), Author: c.AuthorName}
			if c.PublishedAt != nil {
				it.Published = c.PublishedAt.UTC().Format(time.RFC3339)
			}
			ldItems = append(ldItems, it)
		}
	}

	kanonik := s.cfg.BaseURL + "/tag/" + slug
	v := templkit.CDSectionPage{
		Shell:     s.cdShellCtx(ctx, ""),
		Title:     "#" + ad + " — " + s.siteName,
		Desc:      ad + " etiketli son kripto haberleri ve analizler.",
		Canonical: kanonik,
		Section:   "#" + ad,
		Href:      "/tag/" + slug,
		Page:      sayfa,
		Stories:   kartlar,
		// CoinDesk etiket sayfasında JSON-LD basmıyor; biz CollectionPage
		// basıyoruz (Google'ın koleksiyon sayfası için doğru tipi).
		JSONLD:          string(cdCollectionJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), ad, kanonik, ldItems)),
		PageCategory:    "tag_page",
		ContentCategory: "tag",
		Locale:          "tr",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
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
// KANONİK ADRES TAM ADDIR (`/price/bitcoin`) — CoinDesk deseni (spec
// §10.5). SEO'da "bitcoin price" aramasıyla birebir eşleşiyor. Eski
// sembol adresleri (`/price/btc`) 301 ile buraya taşınır: bağlantı kırılmaz,
// iki adres ayrı sayfa olarak indekslenmez (kanonik tekilleşir).
//
// Veri CoinGecko listesinden gelir (ayrı istek yok): ad VEYA sembol
// eşleşmesi yeterli. Bilinmeyen değerde 404.
func (s *Server) handleCDCoin(w http.ResponseWriter, r *http.Request) {
	parca := strings.TrimSpace(r.PathValue("coin"))
	if parca == "" {
		s.handleNotFound(w, r)
		return
	}
	aranan := strings.ToLower(parca)

	fiyatlar := repo.Fiyatlar(r.Context(), s.pool)
	var bulunan *repo.FiyatSatiri
	for i := range fiyatlar {
		// Önce kanonik slug (tam ad), sonra sembol.
		if repo.CoinSlug(fiyatlar[i].Name) == aranan {
			bulunan = &fiyatlar[i]
			break
		}
	}
	if bulunan == nil {
		for i := range fiyatlar {
			if strings.ToLower(fiyatlar[i].Sym) == aranan {
				bulunan = &fiyatlar[i]
				// Sembol adresi kanonik değil: kalıcı yönlendir.
				hedef := "/price/" + repo.CoinSlug(fiyatlar[i].Name)
				if r.URL.RawQuery != "" {
					hedef += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, hedef, http.StatusMovedPermanently)
				return
			}
		}
	}
	if bulunan == nil {
		// Sembol listesi dışında: şimdilik 404 (CoinGecko tüm coin listesi
		// Faz 2'de; uydurma sayfa üretmiyoruz).
		s.handleNotFound(w, r)
		return
	}

	slug := repo.CoinSlug(bulunan.Name)
	sym := bulunan.Sym
	kanonik := s.cfg.BaseURL + "/price/" + slug
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	fiyatMetin := repo.FiyatBicim(bulunan.Price)
	degisimMetin := repo.DegisimBicim(bulunan.Change)
	digerleri := s.cdFiyatSatirlari(r, 12)

	// CoinDesk /price/{coin} yapısı: WebPage+FinancialProduct, BreadcrumbList,
	// ItemList. FAQPage BİLİNÇLİ OLARAK YOK: soru-cevap metnimiz yok,
	// uydurulmuş FAQ Google'ın spam politikasına girer.
	var ldItems []cdLDItem
	for _, row := range digerleri {
		ldItems = append(ldItems, cdLDItem{Headline: row.Name, URL: base + row.Href})
	}
	var ld strings.Builder
	ld.WriteString(string(cdCoinJSONLD(s.siteName, kanonik, bulunan.Name, sym, fiyatMetin, degisimMetin)))
	ld.WriteString(string(breadcrumbScript([]crumb{
		{Name: s.siteName, URL: base + "/"},
		{Name: "Fiyatlar", URL: base + "/price"},
		{Name: bulunan.Name, URL: kanonik},
	})))
	ld.WriteString(string(cdFiyatListesiJSONLD("Cryptocurrency Market Cap Rankings", ldItems)))

	v := templkit.CDCoinPage{
		Shell:           s.cdShellCtx(r.Context(), "price"),
		Title:           bulunan.Name + " (" + sym + ") Fiyatı — " + s.siteName,
		Desc:            bulunan.Name + " canlı fiyatı, 24 saatlik değişimi ve piyasa sıralaması.",
		Canonical:       kanonik,
		Sym:             sym,
		Name:            bulunan.Name,
		Rank:            bulunan.Rank,
		Price:           fiyatMetin,
		Change:          degisimMetin,
		Up:              bulunan.Change >= 0,
		Others:          digerleri,
		JSONLD:          ld.String(),
		PageCategory:    "price_page",
		ContentCategory: "price",
		Locale:          "tr",
		BaseURL:         base,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDCoinPageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("coin render", "error", err)
	}
}
