package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// RSS 2.0 beslemesi — CoinDesk biçimi.
//
// KAYNAK (ölçüldü, 7 Kasım 2026 — /arc/outboundfeeds/rss):
// kanal: title, link, atom:link self, description, lastBuildDate,
// language, copyright, generator, ttl=5, sy:updatePeriod,
// sy:updateFrequency, image.
// öğe: title (CDATA), link, media:content (800×450), guid isPermaLink=false,
// pubDate, description (CDATA), dc:creator, content:encoded (boş),
// dc:description (boş), category (bölüm + etiket + "News").
//
// NEDEN ÖNEMLİ: Google News ve toplayıcılar (Feedly, Inoreader) beslemeyi
// bu alanlarla sınıflandırır. `dc:creator` yoksa yazar atfı kaybolur;
// `media:content` yoksa görsel kartı çıkmaz.

// cdRSSXML beslemeyi üretir.
func (s *Server) cdRSSXML(baslik, link, aciklama, abs, self string, items []repo.Content, etiketler map[int64][]repo.Etiket) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<rss xmlns:content="http://purl.org/rss/1.0/modules/content/" ` +
		`xmlns:wfw="http://wellformedweb.org/CommentAPI/" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
		`xmlns:atom="http://www.w3.org/2005/Atom" ` +
		`xmlns:sy="http://purl.org/rss/1.0/modules/syndication/" ` +
		`xmlns:slash="http://purl.org/rss/1.0/modules/slash/" ` +
		`xmlns:media="http://search.yahoo.com/mrss/" version="2.0">`)
	sb.WriteString("<channel>")
	sb.WriteString("<title>" + htmlEscape(baslik) + "</title>")
	sb.WriteString("<link>" + htmlEscape(link) + "</link>")
	sb.WriteString(`<atom:link rel="self" type="application/rss+xml" href="` + htmlEscape(self) + `"/>`)
	sb.WriteString(`<atom:link rel="hub" href="https://pubsubhubbub.appspot.com/"/>`)
	sb.WriteString("<description>" + htmlEscape(aciklama) + "</description>")
	sb.WriteString("<lastBuildDate>" + time.Now().UTC().Format(time.RFC1123Z) + "</lastBuildDate>")
	sb.WriteString("<language>tr-TR</language>")
	sb.WriteString("<copyright>© " + time.Now().Format("2006") + " " + htmlEscape(s.siteName) + "</copyright>")
	sb.WriteString("<generator>" + htmlEscape(s.siteName) + "</generator>")
	sb.WriteString("<ttl>5</ttl>")
	sb.WriteString("<sy:updatePeriod>hourly</sy:updatePeriod>")
	sb.WriteString("<sy:updateFrequency>1</sy:updateFrequency>")
	sb.WriteString("<image><url>" + htmlEscape(abs+"/static/icon-512.png") + "</url>")
	sb.WriteString("<title>" + htmlEscape(baslik) + "</title>")
	sb.WriteString("<link>" + htmlEscape(link) + "</link></image>")

	for _, it := range items {
		u := abs + makaleURL(it)
		sb.WriteString("<item>")
		sb.WriteString("<title><![CDATA[ " + cdata(it.Title) + " ]]></title>")
		sb.WriteString("<link>" + htmlEscape(u) + "</link>")
		if img := s.mutlakGorsel(abs, it.ImageURL); img != "" {
			sb.WriteString(`<media:content url="` + htmlEscape(img) + `" type="image/*" medium="image"/>`)
		}
		// guid içerik kimliği: adres değişse de (slug düzeltmesi) okuyucu
		// aynı öğeyi tekrar göstermez.
		sb.WriteString(`<guid isPermaLink="false">` + htmlEscape(itoa64(it.ID)) + `</guid>`)
		if it.PublishedAt != nil {
			sb.WriteString("<pubDate>" + it.PublishedAt.UTC().Format(time.RFC1123Z) + "</pubDate>")
		}
		if it.Spot != "" {
			sb.WriteString("<description><![CDATA[ " + cdata(it.Spot) + " ]]></description>")
		}
		if it.AuthorName != "" {
			sb.WriteString("<dc:creator>" + htmlEscape(it.AuthorName) + "</dc:creator>")
		}
		sb.WriteString("<content:encoded/><dc:description/>")
		if it.CategoryName != nil && *it.CategoryName != "" {
			domain := abs
			if it.CategorySlug != nil && *it.CategorySlug != "" {
				domain = abs + "/" + *it.CategorySlug
			}
			sb.WriteString(`<category domain="` + htmlEscape(domain) + `"><![CDATA[ ` + cdata(*it.CategoryName) + ` ]]></category>`)
		}
		for _, e := range etiketler[it.ID] {
			sb.WriteString(`<category domain="tag"><![CDATA[ ` + cdata(e.Ad) + ` ]]></category>`)
		}
		sb.WriteString(`<category domain="` + htmlEscape(abs+"/") + `">News</category>`)
		sb.WriteString("</item>")
	}
	sb.WriteString("</channel></rss>")
	return sb.String()
}

// cdata CDATA gövdesinde `]]>` dizisini güvenli biçime çevirir.
func cdata(s string) string {
	return strings.ReplaceAll(s, "]]>", "]]]]><![CDATA[>")
}

// etiketHaritasi öğe kimlikleri için toplu etiket sorgusu.
//
// NEDEN TOPLU: 50 öğe için 50 ayrı sorgu, beslemeyi 50 × gidiş-dönüş
// yapardı. Sorgu hatası beslemeyi düşürmez — kategori satırı eksik kalır.
func (s *Server) etiketHaritasi(ctx context.Context, items []repo.Content) map[int64][]repo.Etiket {
	out := map[int64][]repo.Etiket{}
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	if len(ids) == 0 {
		return out
	}
	m, err := repo.EtiketlerCoklu(ctx, s.pool, ids)
	if err != nil {
		s.logger.Warn("rss etiketleri", "error", err)
		return out
	}
	return m
}

// handleFeed /rss.xml — site ve kategori beslemesi.
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	abs := strings.TrimRight(s.absBase(r), "/")
	slug := strings.TrimSpace(r.URL.Query().Get("kategori"))
	if slug == "" {
		slug = strings.TrimSpace(r.URL.Query().Get("category"))
	}

	title, link, desc := s.siteName, abs+homeURL(locale), s.siteName+" son haberler"
	self := abs + "/rss.xml"
	var items []repo.Content
	if slug != "" {
		var cat *repo.Category
		cats := s.cachedCategories(ctx, site.ID)
		for i := range cats {
			if cats[i].Slug == slug && cats[i].Status == "yayinda" {
				cat = &cats[i]
				break
			}
		}
		if cat == nil {
			http.NotFound(w, r)
			return
		}
		items, _ = repo.PublishedBySlug(ctx, s.pool, site.ID, slug, locale, 50)
		title = cat.Name + " — " + s.siteName
		link = abs + s.langBase(locale) + "/kategori/" + slug
		desc = cat.Name + " kategorisindeki son haberler"
		self = abs + "/rss.xml?kategori=" + slug
	} else {
		items, _ = repo.RecentPublished(ctx, s.pool, site.ID, locale, 50)
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	_, _ = w.Write([]byte(s.cdRSSXML(title, link, desc, abs, self, items, s.etiketHaritasi(ctx, items))))
}
