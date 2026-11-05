package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// CoinDesk bölüm sayfaları (§8): /markets, /business, /tech, /policy,
// /opinion, /latest-crypto-news.
//
// URL deseni CoinDesk ile aynı: /{section}. Sayfalama `?sayfa=N`.
//
// NEDEN ayrı handler: mevcut /kategori/{slug} theblock şablonunu kullanıyor.
// CoinDesk yapısında bölüm adresi kökte ve tek segment; SEO'da kanonik
// adres budur.

// cdSectionMeta bölüm tanımı (slug → ad + açıklama).
var cdSectionMeta = map[string]struct{ Ad, Aciklama string }{
	"markets":            {"Markets", "Kripto piyasaları: fiyat hareketleri, likidite, türevler ve kurumsal akış."},
	"business":           {"Finance", "Borsalar, kripto şirketleri, bankalar, ödeme ve kurumsal benimseme."},
	"tech":               {"Tech", "Protokoller, ölçekleme, güvenlik açıkları ve altyapı gelişmeleri."},
	"policy":             {"Policy", "Düzenleme, yasa teklifleri, mahkemeler ve kamu kurumları."},
	"opinion":            {"Opinion", "Sektörden görüşler ve analizler."},
	"news-analysis":      {"News Analysis", "Derinlemesine haber analizleri."},
	"latest-crypto-news": {"Latest News", "En son kripto haberleri, dakika dakika."},
	"focus":              {"Focus", "Odak dosyaları ve uzun okumalar."},
	"research":           {"Research", "Araştırma raporları ve veri analizleri."},
	"press-release":      {"Press Releases", "Basın bültenleri."},
	"videos":             {"Videos", "Video haberler ve röportajlar."},
	"podcasts":           {"Podcasts", "Podcast bölümleri."},
}

// cdBolumHandler tek bir bölümün handler'ını üretir.
func (s *Server) cdBolumHandler(slug string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleCDSection(w, r, slug)
	}
}

// handleCDSection bölüm sayfası.
func (s *Server) handleCDSection(w http.ResponseWriter, r *http.Request, slug string) {
	meta, ok := cdSectionMeta[slug]
	if !ok {
		s.handleNotFound(w, r)
		return
	}

	sayfa := 1
	if v := r.URL.Query().Get("sayfa"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			sayfa = n
		}
	}
	const perPage = 24

	f := repo.ListFilter{
		Status:         "yayinda",
		SiteID:         1,
		Locale:         "tr",
		Page:           sayfa,
		PerPage:        perPage,
		ExcludeGallery: true,
	}
	// "latest-crypto-news" tüm bölümleri kapsar; diğerleri kategoriye süzülür.
	if slug != "latest-crypto-news" {
		if id := s.kategoriID(r.Context(), slug); id > 0 {
			f.CategoryID = id
		}
	}

	items, toplam, err := repo.ListContents(r.Context(), s.pool, f)
	if err != nil {
		s.logger.Warn("cd bölüm: liste", "bolum", slug, "error", err)
	}

	simdi := time.Now()
	var kartlar []templkit.CDStory
	for _, c := range items {
		kartlar = append(kartlar, cdStoryCevir(c, simdi))
	}

	v := templkit.CDSectionPage{
		Shell:     s.cdShell(slug),
		Title:     meta.Ad + " — " + s.siteName,
		Desc:      meta.Aciklama,
		Canonical: s.cfg.BaseURL + "/" + slug,
		Section:   meta.Ad,
		Href:      "/" + slug,
		Page:      sayfa,
	}
	if len(kartlar) > 0 {
		v.Lead = kartlar[0]
		// Yan sütun: manşetten sonraki 3.
		for i := 1; i < len(kartlar) && i <= 3; i++ {
			v.Side = append(v.Side, kartlar[i])
		}
		// Izgara: 4. haberden sonrası.
		if len(kartlar) > 4 {
			v.Stories = kartlar[4:]
		}
	}
	if sayfa > 1 {
		v.PrevHref = "/" + slug + sayfaURL(sayfa-1)
	}
	if toplam > sayfa*perPage {
		v.NextHref = "/" + slug + sayfaURL(sayfa+1)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDSectionPageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd bölüm render", "error", err)
	}
}

// sayfaURL sayfa 1 için boş, diğerleri için "?sayfa=N" üretir (kanonik
// adres temiz kalsın).
func sayfaURL(n int) string {
	if n <= 1 {
		return ""
	}
	return "?sayfa=" + strconv.Itoa(n)
}

// kategoriID slug'dan kategori kimliği bulur (yoksa 0).
func (s *Server) kategoriID(ctx context.Context, slug string) int64 {
	var id int64
	_ = s.pool.QueryRow(ctx,
		`SELECT id FROM categories WHERE slug=$1 LIMIT 1`, slug).Scan(&id)
	return id
}

// cdShell ortak kabuk verisini kurar (nav + arama + dil).
func (s *Server) cdShell(aktif string) templkit.CDShell {
	nav := []templkit.CDNavLink{
		{Label: "Son Haberler", Href: "/latest-crypto-news"},
		{Label: "Markets", Href: "/markets"},
		{Label: "Finance", Href: "/business"},
		{Label: "Tech", Href: "/tech"},
		{Label: "Policy", Href: "/policy"},
		{Label: "Görüş", Href: "/opinion"},
		{Label: "Fiyatlar", Href: "/price"},
		{Label: "Video", Href: "/videos"},
	}
	for i := range nav {
		if strings.HasSuffix(nav[i].Href, "/"+aktif) {
			nav[i].Active = true
		}
	}
	return templkit.CDShell{
		SiteName:   s.siteName,
		SearchHref: "/arama",
		Locale:     "tr",
		Nav:        nav,
	}
}

// cdStoryCevir içeriği CoinDesk kart verisine çevirir (ortak yol).
func cdStoryCevir(c repo.Content, simdi time.Time) templkit.CDStory {
	st := templkit.CDStory{
		Title: c.Title,
		Spot:  c.Spot,
		Href:  makaleURL(c),
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
