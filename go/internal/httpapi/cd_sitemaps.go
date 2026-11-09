package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// Sitemap ailesi — CoinDesk yapısıyla birebir.
//
// KAYNAK (ölçüldü, 7 Kasım 2026):
//
//	/sitemap-index.xml      → <sitemapindex>: static-1, articles-1…16, crypto-1…12
//	/sitemaps/static-1.xml  → sabit sayfalar (changefreq daily, priority 1)
//	/sitemaps/articles-N.xml→ haberler (lastmod ISO-8601, changefreq yearly, priority 1)
//	/sitemaps/crypto-N.xml  → fiyat sayfaları (changefreq hourly, priority 1)
//	/arc/outboundfeeds/news-sitemap-index → <sitemapindex> haber sitemap'leri
//	/arc/outboundfeeds/news-sitemap      → Google News urlset
//
// Bizde tek dosyalık `/sitemap.xml` (BİK resmî ilanları da taşıyor) korunuyor;
// üstüne CoinDesk'in BÖLÜNMÜŞ yapısı eklendi. Neden bölünür: Google bir
// sitemap dosyasında en çok 50.000 URL okur; arşiv büyüdükçe tek dosya
// sessizce kırpılır.

// sitemapSayfaBoyu bir bölünmüş sitemap dosyasındaki URL sayısı.
const sitemapSayfaBoyu = 1000

// handleSitemapIndex /sitemap-index.xml — alt sitemap'lerin listesi.
func (s *Server) handleSitemapIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	abs := s.absBase(r)
	site := s.siteByDomain(ctx, r.Host)

	sayfa := 1
	if _, toplam, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status: "yayinda", SiteID: site.ID, Locale: "tr",
		Page: 1, PerPage: 1, ExcludeGallery: true,
	}); err == nil && toplam > 0 {
		sayfa = (toplam + sitemapSayfaBoyu - 1) / sitemapSayfaBoyu
	}
	if sayfa > 500 {
		sayfa = 500
	}

	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	yaz := func(loc string) {
		sb.WriteString("<sitemap><loc>" + htmlEscape(loc) + "</loc></sitemap>")
	}
	yaz(abs + "/sitemaps/static-1.xml")
	for i := 1; i <= sayfa; i++ {
		yaz(abs + "/sitemaps/articles-" + strconv.Itoa(i) + ".xml")
	}
	yaz(abs + "/sitemaps/crypto-1.xml")
	yaz(abs + "/sitemaps/stocks-1.xml")
	sb.WriteString("</sitemapindex>")

	s.writeXML(w, sb.String())
}

// handleSitemapStatic /sitemaps/static-1.xml — sabit hizmet sayfaları.
func (s *Server) handleSitemapStatic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	abs := s.absBase(r)
	site := s.siteByDomain(ctx, r.Host)

	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	ekle := func(yol, freq, prio string) {
		sb.WriteString("<url><loc>" + htmlEscape(abs+yol) + "</loc>")
		if freq != "" {
			sb.WriteString("<changefreq>" + freq + "</changefreq>")
		}
		if prio != "" {
			sb.WriteString("<priority>" + prio + "</priority>")
		}
		sb.WriteString("</url>")
	}
	ekle("/", "daily", "1")
	// "Son Haberler" hiç bitmez: bölüm doluluğu tablosunda görünmese de
	// gezinmede duruyor, dizinde de olmalı.
	ekle("/latest-crypto-news", "daily", "1")
	// Diğer bölümler yalnız YAYINDA İÇERİĞİ VARSA (boş bölüm dizine
	// girmemeli — ölçüm: Focus/Videos/Podcasts sıfır içerikliydi).
	if dolu, err := repo.DoluBolumler(ctx, s.pool, site.ID); err == nil {
		for _, b := range []string{"markets", "business", "tech", "policy", "opinion", "news-analysis", "focus", "research", "press-release"} {
			if _, ok := dolu[b]; ok {
				ekle("/"+b, "daily", "1")
			}
		}
	}
	ekle("/price", "daily", "1")
	ekle("/videos", "daily", "1")
	ekle("/podcasts", "daily", "1")
	ekle("/newsletters", "weekly", "0.8")
	ekle("/sponsored-content", "weekly", "0.8")
	ekle("/yazarlar", "weekly", "0.8")
	ekle("/arsiv", "daily", "1")
	ekle("/hakkimizda", "monthly", "0.8")
	ekle("/iletisim", "monthly", "0.8")
	ekle("/gizlilik", "monthly", "0.5")
	ekle("/kullanim-kosullari", "monthly", "0.5")
	ekle("/resmi-ilanlar", "daily", "0.8")
	ekle("/rss-listesi", "monthly", "0.5")
	// İçeriği olan kategoriler (boş kategori dizine girmemeli).
	if cs, err := repo.ListCategories(ctx, s.pool, site.ID); err == nil {
		for _, c := range cs {
			if c.ContentCount <= 0 {
				continue
			}
			ekle("/kategori/"+c.Slug, "daily", "0.8")
		}
	}
	sb.WriteString("</urlset>")
	s.writeXML(w, sb.String())
}

// handleSitemapsDosya /sitemaps/{dosya} — bölünmüş sitemap dosyaları.
//
// Go mux joker'i segment ortasında kabul etmediği için (`articles-{n}.xml`
// kayıt anında panik üretiyor) tek segment alınıp burada çözülüyor.
func (s *Server) handleSitemapsDosya(w http.ResponseWriter, r *http.Request) {
	dosya := strings.TrimSpace(r.PathValue("dosya"))
	switch dosya {
	case "static-1.xml":
		s.handleSitemapStatic(w, r)
		return
	case "crypto-1.xml":
		s.handleSitemapCrypto(w, r)
		return
	case "stocks-1.xml":
		s.handleSitemapStocks(w, r)
		return
	}
	if n, ok := cdMakaleDilimi(dosya); ok {
		s.handleSitemapArticlesDilim(w, r, n)
		return
	}
	s.handleNotFound(w, r)
}

// cdMakaleDilimi "articles-7.xml" → 7.
func cdMakaleDilimi(dosya string) (int, bool) {
	const on, son = "articles-", ".xml"
	if !strings.HasPrefix(dosya, on) || !strings.HasSuffix(dosya, son) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(dosya, on), son))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// handleSitemapArticles eski imza (tek dilim) — dilim numarası handler'da
// çözüldüğü için bu sarmalayıcı yalnız testler/uyumluluk için duruyor.
func (s *Server) handleSitemapArticles(w http.ResponseWriter, r *http.Request, n int) {
	s.handleSitemapArticlesDilim(w, r, n)
}

// handleSitemapArticlesDilim /sitemaps/articles-N.xml — haber arşivi dilimi.
func (s *Server) handleSitemapArticlesDilim(w http.ResponseWriter, r *http.Request, n int) {
	ctx := r.Context()
	abs := s.absBase(r)
	site := s.siteByDomain(ctx, r.Host)

	items, _, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status: "yayinda", SiteID: site.ID, Locale: "tr",
		Page: n, PerPage: sitemapSayfaBoyu, ExcludeGallery: true,
	})
	if err != nil {
		s.logger.Warn("sitemap makale dilimi", "dilim", n, "error", err)
	}

	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	// CoinDesk articles-N.xml: yalnız loc + lastmod + changefreq + priority.
	// hreflang alternates BURADA basılmıyor (kaynak dosyada yok) ve sayfa
	// düzeyindeki x-default/canonical zaten doğru adresi veriyor.
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, it := range items {
		sb.WriteString("<url><loc>" + htmlEscape(abs+makaleURL(it)) + "</loc>")
		if !it.UpdatedAt.IsZero() {
			sb.WriteString("<lastmod>" + it.UpdatedAt.UTC().Format(time.RFC3339) + "</lastmod>")
		}
		sb.WriteString("<changefreq>yearly</changefreq><priority>1</priority></url>")
	}
	sb.WriteString("</urlset>")
	s.writeXML(w, sb.String())
}

// handleSitemapStocks /sitemaps/stocks-1.xml — hisse detayları (saatlik).
//
// NEDEN AYRI DOSYA: hisse sayfaları 631 URL; `static-1.xml` "sabit hizmet
// sayfaları" demek ve onu 90 KB'lik fiyat listesiyle şişirmek yanlış
// olurdu. CoinDesk de aynı desende `crypto-N.xml` ile ayırıyor.
//
// YALNIZ FİYATI OLAN HİSSELER yazılır: detay ucu veri bulamayınca 404
// döndüğü için fiyatsız sembolü dizine koymak arama motoruna kırık adres
// bildirmek olur. Böylece sitemap'teki her URL test edilebilir bir sayfa.
func (s *Server) handleSitemapStocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	abs := s.absBase(r)

	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	hd := repo.HisselerOf(ctx, s.pool)
	if len(hd.Rows) > 0 {
		// Dizin sayfası: hisse tablosunun tamamı burada listeleniyor.
		sb.WriteString("<url><loc>" + htmlEscape(abs+"/price") +
			"</loc><changefreq>hourly</changefreq><priority>1</priority></url>")
	}
	gorulen := map[string]bool{}
	for _, h := range hd.Rows {
		if h.Value <= 0 {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(h.Code))
		if code == "" || gorulen[code] {
			continue
		}
		gorulen[code] = true
		// ".IS" eki URL'de kalır (Yahoo BIST sembolü; kanonik adres bu).
		sb.WriteString("<url><loc>" + htmlEscape(abs+"/price/stock/"+hisseBorsaKodu(code)) +
			"</loc><changefreq>hourly</changefreq><priority>1</priority></url>")
	}
	sb.WriteString("</urlset>")
	s.writeXML(w, sb.String())
}

// handleCDHisseDizin /price/stock — hisse bölümünün kökü.
//
// Ölü adres (404) yerine TEK kanonik sayfaya yönlendirilir: /price zaten
// "Hisse Senetleri" giriş listesini ve "Tümü" bağlantısını basıyor. İki
// ayrı dizin sayfası tutmak (biri kripto, biri hisse tablolu) arama
// motorunda yinelenen içerik olurdu.
func (s *Server) handleCDHisseDizin(w http.ResponseWriter, r *http.Request) {
	hedef := "/price"
	if r.URL.RawQuery != "" {
		hedef += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, hedef, http.StatusMovedPermanently)
}

// handleSitemapCrypto /sitemaps/crypto-1.xml — fiyat sayfaları (saatlik).
func (s *Server) handleSitemapCrypto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	abs := s.absBase(r)

	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sb.WriteString("<url><loc>" + abs + "/price</loc><changefreq>hourly</changefreq><priority>1</priority></url>")
	gorulen := map[string]bool{}
	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		slug := repo.CoinSlug(f.Name)
		if slug == "" || gorulen[slug] {
			continue
		}
		gorulen[slug] = true
		sb.WriteString("<url><loc>" + htmlEscape(abs+"/price/"+slug) +
			"</loc><changefreq>hourly</changefreq><priority>1</priority></url>")
	}
	sb.WriteString("</urlset>")
	s.writeXML(w, sb.String())
}

// handleNewsSitemapIndex /news-sitemap-index — haber sitemap'lerinin indeksi.
func (s *Server) handleNewsSitemapIndex(w http.ResponseWriter, r *http.Request) {
	abs := s.absBase(r)
	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sb.WriteString("<sitemap><loc>" + htmlEscape(abs+"/sitemap-news.xml") + "</loc></sitemap>")
	sb.WriteString("</sitemapindex>")
	s.writeXML(w, sb.String())
}

// sitemapBaslik her XML dosyasının ilk satırı.
const sitemapBaslik = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// newsSitemap Google News urlset — CoinDesk biçimi.
//
// KAYNAK (ölçüldü): her <url> şunları taşıyor:
//
//	<lastmod> tam ISO-8601
//	<loc>
//	<news:news><news:publication><news:name/><news:language/></…>
//	            <news:publication_date/><news:title><![CDATA[…]]></news:title>
//	<image:image><image:loc>
//
// Son 48 saat: Google News sitemap kuralı. `news:title` CDATA içinde
// veriliyor — CoinDesk de öyle; başlıktaki & ve < kaçışı elle yapılmıyor.
func (s *Server) newsSitemap(abs string, items []repo.Content) string {
	var sb strings.Builder
	sb.WriteString(sitemapBaslik)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">`)
	cut := time.Now().Add(-48 * time.Hour)
	for _, it := range items {
		if it.PublishedAt == nil || it.PublishedAt.Before(cut) {
			continue
		}
		loc := abs + makaleURL(it)
		sb.WriteString("<url><lastmod>" + it.UpdatedAt.UTC().Format(time.RFC3339) + "</lastmod>")
		sb.WriteString("<loc>" + htmlEscape(loc) + "</loc><news:news>")
		sb.WriteString("<news:publication><news:name>" + htmlEscape(s.siteName) + "</news:name><news:language>" + htmlEscape(it.Locale) + "</news:language></news:publication>")
		sb.WriteString("<news:publication_date>" + it.PublishedAt.UTC().Format(time.RFC3339) + "</news:publication_date>")
		sb.WriteString("<news:title><![CDATA[ " + strings.ReplaceAll(it.Title, "]]>", "]]]]><![CDATA[>") + " ]]></news:title>")
		sb.WriteString("</news:news>")
		if img := s.mutlakGorsel(abs, it.ImageURL); img != "" {
			sb.WriteString("<image:image><image:loc>" + htmlEscape(img) + "</image:loc></image:image>")
		}
		sb.WriteString("</url>")
	}
	sb.WriteString("</urlset>")
	return sb.String()
}

// mutlakGorsel göreli görsel yolunu mutlak adrese çevirir.
func (s *Server) mutlakGorsel(abs, gorsel string) string {
	gorsel = strings.TrimSpace(gorsel)
	if gorsel == "" {
		return ""
	}
	if strings.HasPrefix(gorsel, "http://") || strings.HasPrefix(gorsel, "https://") {
		return gorsel
	}
	return abs + "/" + strings.TrimPrefix(gorsel, "/")
}

// writeXML ortak XML yanıtı.
func (s *Server) writeXML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	_, _ = w.Write([]byte(body))
}
