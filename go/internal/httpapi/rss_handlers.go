package httpapi

import (
	"net/http"
	"strings"
)

type rssFeedLink struct {
	Name string
	URL  string
}

type rssListData struct {
	Feeds []rssFeedLink
}

// handleRSSList /rss-listesi — tüm RSS akışlarını listeleyen sayfa.
func (s *Server) handleRSSList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	abs := strings.TrimRight(s.absBase(r), "/")

	cats := s.cachedCategories(ctx, site.ID)
	total := map[int64]int{}
	for _, c := range cats {
		total[c.ID] = c.ContentCount
	}
	for _, c := range cats {
		if c.ParentID != nil {
			total[*c.ParentID] += c.ContentCount
		}
	}

	feeds := []rssFeedLink{{
		Name: "Tüm haberler",
		URL:  abs + "/rss",
	}}
	for _, c := range cats {
		if c.ParentID != nil || c.Status != "yayinda" || total[c.ID] <= 0 {
			continue
		}
		feeds = append(feeds, rssFeedLink{
			Name: c.Name,
			URL:  abs + "/rss?kategori=" + c.Slug,
		})
	}

	d := s.baseView(locale, s.langBase(locale), "RSS Akışları")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.MetaTitle = "RSS Akışları — " + s.siteName
	d.MetaDesc = metaDesc(s.siteName+" RSS akışları: tüm haberler ve kategori bazlı beslemeler.", s.siteName)
	d.Canonical = abs + "/rss-listesi"
	d.RSSList = &rssListData{Feeds: feeds}
	s.renderPublic(w, r, "rss_list", d, cacheControlPublic)
}
