package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Akış (/akis) ve 404 — CoinDesk kabuğu (12 Kasım 2026).

// handleCDAkis /akis sayfasını yeni kabukla basar.
func (s *Server) handleCDAkis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	catID, slug, name, other, _, day, rng := s.resolveFeedQuery(r)

	page := parseIntOr(r.URL.Query().Get("sayfa"))
	if page <= 0 {
		page = 1
	}
	fq := repo.FeedQuery{SiteID: site.ID, CategoryID: catID, Other: other, Page: page, PerPage: feedPerPage}
	if day != "" {
		fq.Day = day
	} else if rng == "hafta" {
		fq.Since = dayMinus(6)
	}
	items, total, err := repo.FeedPage(ctx, s.pool, fq)
	if err != nil {
		s.logger.Warn("akis liste", "error", err)
	}

	// Bilinmeyen kategori adı boş H1 bırakmasın.
	if strings.TrimSpace(name) == "" {
		if slug != "" {
			name = "Akış"
		} else {
			name = "Güncel"
		}
	}

	simdi := time.Now()
	v := templkit.CDAkisPage{
		Baslik:   name,
		Aciklama: "Sitedeki en yeni içerikler, en yeniden eskiye.",
		Page:     page,
		BosMetin: "Bu filtrede henüz içerik yok.",
	}
	for _, c := range items {
		v.Stories = append(v.Stories, s.cdStoryCevir(c, simdi))
	}

	// Filtre sorgusu (sayfalama bağlantılarında korunur).
	temel := url.Values{}
	if slug != "" {
		temel.Set("kategori", slug)
	} else if other {
		temel.Set("sekme", "diger")
	}
	if day != "" && rng == "" {
		temel.Set("tarih", day)
	} else if rng != "" {
		temel.Set("aralik", rng)
	}

	// Kategori çipleri: yalnız içeriği olan kategoriler.
	v.Kategoriler = append(v.Kategoriler, templkit.CDCip{Ad: "Güncel", Href: "/akis", Secili: slug == "" && !other})
	if cats, err := repo.ListCategories(ctx, s.pool, site.ID); err == nil {
		for _, c := range cats {
			if c.ContentCount <= 0 {
				continue
			}
			v.Kategoriler = append(v.Kategoriler, templkit.CDCip{
				Ad: c.Name, Href: "/akis?kategori=" + url.QueryEscape(c.Slug), Secili: c.Slug == slug,
			})
		}
	}
	v.Kategoriler = append(v.Kategoriler, templkit.CDCip{Ad: "Diğer", Href: "/akis?sekme=diger", Secili: other})

	// Tarih çipleri.
	for _, a := range []struct{ Ad, K, Gun string }{
		{"Bugün", "bugun", dayMinus(0)},
		{"Dün", "dun", dayMinus(1)},
		{"Bu hafta", "hafta", ""},
	} {
		q := url.Values{}
		for k, vv := range temel {
			q[k] = vv
		}
		q.Del("tarih")
		q.Set("aralik", a.K)
		v.Araliklar = append(v.Araliklar, templkit.CDCip{
			Ad: a.Ad, Href: "/akis?" + q.Encode(), Secili: rng == a.K || (a.K == "bugun" && day == a.Gun && rng == ""),
		})
	}

	// Sayfalama.
	sayfaURL := func(n int) string {
		q := url.Values{}
		for k, vv := range temel {
			q[k] = vv
		}
		if n > 1 {
			q.Set("sayfa", itoa(n))
		}
		if len(q) == 0 {
			return "/akis"
		}
		return "/akis?" + q.Encode()
	}
	if page > 1 {
		v.PrevHref = sayfaURL(page - 1)
	}
	if page*feedPerPage < total {
		v.NextHref = sayfaURL(page + 1)
	}

	kanonik := s.kanonikKok(r) + "/akis"
	if len(temel) > 0 {
		kanonik += "?" + temel.Encode()
	}
	v.Shell = s.cdShellCtx(ctx, r, "akis")
	v.Title = name + " — " + s.siteName
	v.Desc = "Kripto ve blockchain haber akışı: " + strings.ToLower(name) + " içerikler."
	v.Canonical = kanonik
	v.PageCategory = "section_landing_page"
	v.ContentCategory = "akis"
	v.Locale = "tr"
	v.BaseURL = s.kanonikKok(r)

	// Yapısal veri: listenin ilk 12 öğesi.
	var ld []cdLDItem
	for i, st := range v.Stories {
		if i >= 12 {
			break
		}
		ld = append(ld, cdLDItem{Headline: st.Title, URL: s.absBase(r) + st.Href, Author: s.siteName})
	}
	v.JSONLD = string(cdCollectionJSONLD(s.siteName, s.kanonikKok(r), name, kanonik, ld))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDAkisPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd akis render", "error", err)
	}
}

// handleCD404 bulunamayan sayfa: yeni kabuk + gerçek son haberler.
func (s *Server) handleCD404(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	v := templkit.CD404Page{
		Shell:     s.cdShellCtx(ctx, r, ""),
		Title:     "Sayfa bulunamadı — " + s.siteName,
		Desc:      "Aradığınız sayfa bulunamadı. Ana sayfaya dönün veya sitede arama yapın.",
		Canonical: s.kanonikKok(r) + r.URL.Path,
		Locale:    "tr",
		BaseURL:   s.kanonikKok(r),
	}
	if items, err := repo.PopulerIcerikler(ctx, s.pool, site.ID, "tr", 6); err == nil {
		simdi := time.Now()
		for _, c := range items {
			v.Oneriler = append(v.Oneriler, s.cdStoryCevir(c, simdi))
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if err := templkit.CD404PageView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd 404 render", "error", err)
	}
}
