package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// handleCDArticle CoinDesk makale sayfası (§7).
//
// NEDEN AYRI: mevcut `handleArticle` theblock şablonunu (yorum, tepki,
// rozet, sesli okuma…) kullanıyor. CoinDesk yapısı sade ve sabit:
// breadcrumb → kicker → H1 → spot → yazar/tarih → gövde → kaynak atfı →
// etiketler → ilgili haberler. Çok dilli yönlendirme ortak yardımcıdan.
func (s *Server) handleCDArticle(w http.ResponseWriter, r *http.Request, lang, slug string, id int64) {
	ctx := r.Context()
	c, err := repo.ContentBySlugAndID(ctx, s.pool, slug, id)
	if err != nil || c.Status != "yayinda" {
		s.handleNotFound(w, r)
		return
	}
	// Dil eşleşmesi: istenen dilde çeviri varsa oraya, yoksa orijinal dile.
	if lang != "" && lang != c.Locale {
		if trs, err := repo.TranslationsOf(ctx, s.pool, c.TranslationRootID()); err == nil {
			for _, t := range trs {
				if t.Locale == lang && t.Status == "yayinda" {
					http.Redirect(w, r, publicArticleURL(lang, t.DisplaySlug(), t.ID), http.StatusFound)
					return
				}
			}
		}
		if lang != "tr" {
			http.Redirect(w, r, publicArticleURL(c.Locale, c.DisplaySlug(), c.ID), http.StatusFound)
			return
		}
	}

	_ = repo.IncrementHit(ctx, s.pool, c.ID)

	bolumAd, bolumSlug := "Markets", "markets"
	if c.CategoryName != nil && *c.CategoryName != "" {
		bolumAd = *c.CategoryName
	}
	if c.CategorySlug != nil && *c.CategorySlug != "" {
		bolumSlug = *c.CategorySlug
	}

	v := templkit.CDArticleView{
		Shell:       s.cdShell(bolumSlug),
		Title:       cdMakaleBaslik(c),
		Desc:        cdMakaleAciklama(c),
		Canonical:   s.cfg.BaseURL + publicArticleURL(c.Locale, c.DisplaySlug(), c.ID),
		Kicker:      strings.ToUpper(bolumAd),
		SectionName: bolumAd,
		SectionHref: "/" + bolumSlug,
		Headline:    c.Title,
		Spot:        c.Spot,
		BodyHTML:    string(s.blocksToHTML(c.Detail, nil)),
		Image:       c.ImageURL,
	}
	if c.AuthorName != "" {
		v.Author = c.AuthorName
	} else {
		v.Author = s.siteName
	}
	if c.PublishedAt != nil {
		v.PublishedAt = c.PublishedAt.Format("2 Ocak 2006 15:04 MST")
	}
	v.ReadMinutes = cdOkumaDakika(c.Spot, v.BodyHTML)
	if c.SourceName != nil {
		v.SourceName = *c.SourceName
	}
	if c.SourceURL != nil {
		v.SourceURL = *c.SourceURL
	}

	// Etiketler: kaynağın gerçek RSS kategorileri (migration 0115).
	if etler, err := repo.IcerikEtiketleri(ctx, s.pool, c.ID); err == nil {
		for _, e := range etler {
			v.Tags = append(v.Tags, templkit.CDTag{Ad: e.Ad, Slug: e.Slug})
		}
	}

	// İlgili haberler: alaka motoru → ilişki → kategori (kademeli düşüş).
	related, err := repo.SimilarTo(ctx, s.pool, c.ID, c.SiteID, c.Locale, 6)
	if err != nil || len(related) == 0 {
		related, _ = repo.RelatedContents(ctx, s.pool, c.SiteID, c.ID, 6)
	}
	if len(related) == 0 {
		related, _ = repo.PublishedBySlug(ctx, s.pool, c.SiteID, catSlugPtr(c.CategorySlug), c.Locale, 6)
	}
	simdi := time.Now()
	for _, rc := range related {
		if len(v.Related) >= 3 {
			break
		}
		v.Related = append(v.Related, cdStoryCevir(rc, simdi))
	}

	// Önceki/sonraki: kripto portalında "sıradaki haber" okuma süresini
	// uzatır. Zaman tabanlı komşuluk (ContentBefore/After) kullanılır.
	if c.PublishedAt != nil {
		catID := int64(0)
		if c.CategoryID != nil {
			catID = *c.CategoryID
		}
		if sonraki, err := repo.ContentAfter(ctx, s.pool, c.SiteID, *c.PublishedAt, c.Locale, c.ID, catID); err == nil {
			v.NextHref = publicArticleURL(sonraki.Locale, sonraki.DisplaySlug(), sonraki.ID)
			v.NextTitle = sonraki.Title
		}
		if onceki, err := repo.ContentBefore(ctx, s.pool, c.SiteID, *c.PublishedAt, c.Locale, c.ID, catID); err == nil {
			v.PrevHref = publicArticleURL(onceki.Locale, onceki.DisplaySlug(), onceki.ID)
			v.PrevTitle = onceki.Title
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDArticle(v).Render(ctx, w); err != nil {
		s.logger.Error("cd makale render", "error", err)
	}
}

// cdMakaleBaslik SEO başlığı: seo_title → başlık + site adı.
func cdMakaleBaslik(c repo.Content) string {
	if c.SeoTitle != nil && strings.TrimSpace(*c.SeoTitle) != "" {
		return *c.SeoTitle
	}
	return c.Title
}

// cdMakaleAciklama meta açıklama: seo_description → spot → kırpılmış gövde.
func cdMakaleAciklama(c repo.Content) string {
	if c.SeoDesc != nil && strings.TrimSpace(*c.SeoDesc) != "" {
		return *c.SeoDesc
	}
	if s := strings.TrimSpace(c.Spot); s != "" {
		return kirpRune(s, 160)
	}
	return c.Title
}

// cdOkumaDakika 200 kelime/dk varsayımıyla okuma süresi (en az 1).
func cdOkumaDakika(spot, govde string) int {
	kelime := len(strings.Fields(spot)) + len(strings.Fields(govde))
	if kelime <= 0 {
		return 1
	}
	d := kelime / 200
	if d < 1 {
		return 1
	}
	return d
}

// kirpRune metni rune sınırında kırpar (UTF-8 bozulmasın).
func kirpRune(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}
