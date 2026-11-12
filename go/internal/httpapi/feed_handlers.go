package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// ---- Akış (feed) modu + kategori sekmeleri ----

type feedData struct {
	Items    []repo.Content
	CatSlug  string
	CatName  string
	Other    bool
	Guncel   bool
	Page     int
	PerPage  int
	Total    int
	HasMore  bool
	Mode     string // feed | manset | grid
	BaseQS   string // kategori/sekme (çipler için)
	FilterQS string // kategori/sekme + tarih/aralık (mod + sayfalama)
	Day      string // belirli gün (YYYY-MM-DD)
	Range    string // hafta | bugun | dun
	TabBase  string
}

const feedPerPage = 12

// feedMode çerezden görünüm modunu okur (varsayılan: feed/akış).
func feedMode(r *http.Request) string {
	if c, err := r.Cookie("tb_feed_mod"); err == nil && (c.Value == "grid" || c.Value == "feed" || c.Value == "manset") {
		return c.Value
	}
	return "feed"
}

func validDay(s string) bool {
	if len(s) != 10 {
		return false
	}
	for i, ch := range s {
		if i == 4 || i == 7 {
			if ch != '-' {
				return false
			}
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func dayMinus(n int) string { return time.Now().AddDate(0, 0, -n).Format("2006-01-02") }

// resolveFeedQuery kategori slug / sekme / mod / tarih bilgisini çözer.
func (s *Server) resolveFeedQuery(r *http.Request) (categoryID int64, slug, name string, other bool, mode string, day, rng string) {
	mode = r.URL.Query().Get("mod")
	if mode != "grid" && mode != "feed" && mode != "manset" {
		mode = feedMode(r)
	}
	if ks := strings.TrimSpace(r.URL.Query().Get("kategori")); ks != "" {
		if cats, err := repo.ListCategories(r.Context(), s.pool, siteIDFromCtx(r.Context())); err == nil {
			for _, c := range cats {
				if c.Slug == ks {
					categoryID, slug, name = c.ID, c.Slug, c.Name
					break
				}
			}
		}
	} else if r.URL.Query().Get("sekme") == "diger" {
		other, name = true, "Diğer"
	} else {
		name = "Güncel"
	}
	// tarih: aralik (bugun/dun/hafta) veya belirli gün
	switch r.URL.Query().Get("aralik") {
	case "bugun":
		rng, day = "bugun", dayMinus(0)
	case "dun":
		rng, day = "dun", dayMinus(1)
	case "hafta":
		rng = "hafta"
	}
	if d := strings.TrimSpace(r.URL.Query().Get("tarih")); validDay(d) {
		day, rng = d, ""
	}
	return
}

// handleAkisPage SSR akış sayfası (/akis) — yeni kabuk (bkz. handleCDAkis).
func (s *Server) handleAkisPage(w http.ResponseWriter, r *http.Request) {
	s.handleCDAkis(w, r)
}

// handleAkisAPI sonsuz kaydırma için JSON akış.
func (s *Server) handleAkisAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	catID, _, _, other, _, day, rng := s.resolveFeedQuery(r)
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
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	type fi struct {
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		Spot      string `json:"spot"`
		URL       string `json:"url"`
		Cover     string `json:"cover"`
		Category  string `json:"category"`
		CatSlug   string `json:"cat_slug"`
		Published string `json:"published"`
	}
	out := make([]fi, 0, len(items))
	for _, c := range items {
		cover := ""
		if c.CoverMediaID != nil {
			cover = s.mediaURL(c.CoverMediaID)
		}
		cat, catSlug := "", ""
		if c.CategoryName != nil {
			cat = *c.CategoryName
		}
		if c.CategorySlug != nil {
			catSlug = *c.CategorySlug
		}
		pub := ""
		if c.PublishedAt != nil {
			pub = c.PublishedAt.Local().Format("02.01.2006 15:04")
		}
		out = append(out, fi{
			ID: c.ID, Title: c.Title, Spot: c.Spot,
			URL:   publicArticleURL(c.Locale, c.DisplaySlug(), c.ID),
			Cover: cover, Category: cat, CatSlug: catSlug, Published: pub,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "items": out, "page": page, "total": total,
		"has_more": page*feedPerPage < total,
	})
}
