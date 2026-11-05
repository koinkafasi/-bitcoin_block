package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// CoinDesk ana sayfası (bitcosar).
//
// Modül sırası docs/COINDESK-SPEC.md §6 ile BİREBİR; sapma yok.
// Veri yoksa modül basılmaz, sıra korunur.
//
// Ayrı kod yolu bilinçli: mevcut `homeView` tasarımcı/blok motoruna bağlı.
// CoinDesk düzeni SABİT sıra istiyor; blok motoruna sıkıştırmak o sırayı
// veriye bağlardı. Eski düzen `/klasik` altında korunur.

func (s *Server) handleCDHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, _, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status:         "yayinda",
		SiteID:         1,
		Locale:         "tr",
		Page:           1,
		PerPage:        60,
		ExcludeGallery: true,
	})
	if err != nil {
		// Boş site 500 dönmez: hata loglanır, iskelet basılır.
		s.logger.Warn("cd ana sayfa: içerik listesi", "error", err)
	}

	simdi := time.Now()
	cevir := func(c repo.Content) templkit.CDStory {
		st := templkit.CDStory{
			Title: c.Title,
			Spot:  c.Spot,
			Href:  publicArticleURL(c.Locale, c.DisplaySlug(), c.ID),
			Image: c.ImageURL,
		}
		if c.PublishedAt != nil {
			st.Time = trGoreliZaman(*c.PublishedAt, simdi)
		}
		if c.SourceName != nil {
			st.Source = *c.SourceName
		}
		if c.SourceURL != nil {
			st.SourceURL = *c.SourceURL
		}
		if c.CategoryName != nil {
			st.Kicker = strings.ToUpper(*c.CategoryName)
		}
		return st
	}

	var donus []templkit.CDStory
	for _, c := range items {
		donus = append(donus, cevir(c))
	}

	// §8 manşet: 1 büyük + 3 yan. §5 latest: ilk 12 (saat damgalı liste).
	var lead templkit.CDStory
	var side, latest []templkit.CDStory
	for i, st := range donus {
		if i == 0 {
			lead = st
		} else if i <= 3 {
			side = append(side, st)
		}
		if len(latest) < 12 {
			latest = append(latest, st)
		}
	}

	// §22 bölüm blokları: 1 büyük + 2 alt + 3 satır (CoinDesk deseni).
	bolumler := []struct{ Ad, Slug string }{
		{"Markets", "markets"},
		{"Finance", "business"},
		{"Tech", "tech"},
		{"Policy", "policy"},
	}
	var sections []templkit.CDSection
	for _, b := range bolumler {
		var grup []templkit.CDStory
		for _, c := range items {
			if c.CategorySlug == nil || *c.CategorySlug != b.Slug {
				continue
			}
			grup = append(grup, cevir(c))
			if len(grup) >= 6 {
				break
			}
		}
		sec := templkit.CDSection{Title: b.Ad, Href: "/" + b.Slug}
		if len(grup) > 0 {
			sec.Lead = grup[0]
		}
		for i := 1; i < len(grup) && i <= 2; i++ {
			sec.Second = append(sec.Second, grup[i])
		}
		for i := 3; i < len(grup); i++ {
			sec.Rows = append(sec.Rows, grup[i])
		}
		sections = append(sections, sec)
	}

	d := templkit.CDHomeView{
		Shell: templkit.CDShell{
			SiteName:   s.siteName,
			SearchHref: "/arama",
			Locale:     "tr",
			Nav: []templkit.CDNavLink{
				{Label: "Son Haberler", Href: "/latest-crypto-news"},
				{Label: "Markets", Href: "/markets"},
				{Label: "Finance", Href: "/business"},
				{Label: "Tech", Href: "/tech"},
				{Label: "Policy", Href: "/policy"},
				{Label: "Görüş", Href: "/opinion"},
				{Label: "Fiyatlar", Href: "/price"},
				{Label: "Video", Href: "/videos"},
			},
		},
		Title:     s.siteName + " — Kripto Haberleri ve Piyasa Verisi",
		Desc:      "Bitcoin, Ethereum ve kripto piyasalarından son haberler, analizler ve canlı fiyatlar.",
		Canonical: s.cfg.BaseURL + r.URL.Path,
		Lead:      lead,
		Side:      side,
		Latest:    latest,
		Sections:  sections,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDHome(d).Render(ctx, w); err != nil {
		s.logger.Error("cd ana sayfa render", "error", err)
	}
}

// trGoreliZaman "3 saat önce" biçiminde insan okur zaman üretir
// (CoinDesk'in "17 minutes ago" deseninin Türkçesi).
func trGoreliZaman(t, simdi time.Time) string {
	dk := int(simdi.Sub(t).Minutes())
	switch {
	case dk < 1:
		return "az önce"
	case dk < 60:
		return strconv.Itoa(dk) + " dakika önce"
	case dk < 60*24:
		return strconv.Itoa(dk/60) + " saat önce"
	default:
		return strconv.Itoa(dk/(60*24)) + " gün önce"
	}
}
