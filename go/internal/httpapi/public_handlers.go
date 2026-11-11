package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/auth"

	"theblock/site/internal/cryptoicon"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
	"theblock/site/internal/webui/templkit"
)

// handleHome ana sayfa (yerelleştirilmiş).
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	d := s.homeView(r)
	// A/B testi: d.Blocks'u varyanta göre değiştirir (kapalıysa dokunulmaz).
	s.applyAB(w, r, d, "home")
	s.renderPublic(w, r, "home", d, cacheControlPublic)
}

// homeView ana sayfanın şablon verisini kurar. handleHome ve tasarımcının
// önizleme ucu AYNI veri yolunu kullanır; böylece önizlemede görünen ile
// sitede görünen birebir aynıdır (ayrı kod yolu = ayrı görünüm riski).
func (s *Server) homeView(r *http.Request) *tmplData {
	ctx := r.Context()
	lang := r.PathValue("lang")
	if lang == "" {
		lang = s.defaultLang(ctx)
	}
	if !s.i18n.IsSupported(lang) {
		lang = "tr"
	}
	locale := lang
	site := s.siteByDomain(ctx, r.Host)
	// İstenen dilde içerik yoksa varsayılan dile düşülür (Faz 189).
	//
	// Ölçüm: /en 200 dönüyordu ama manşet BOŞTU (TR'de 150 hero öğesi,
	// EN'de 0) çünkü sorgular `locale = $1` ile sıkı filtreleniyor ve hiç
	// İngilizce içerik yoktu. Boş manşet alanı tasarımı kırık gösteriyordu.
	// Düşüş varsa sayfa noindex'lenir ve canonical varsayılan dilin
	// adresine verilir: aynı Türkçe içerik kopya olarak dizinlenmesin.
	icerikLocale, dusus := s.icerikDili(ctx, site.ID, lang)

	// ---- Bağımsız okumalar PARALEL ----
	//
	// ÖLÇÜM (8 Kasım 2026): ana sayfa p50 51 ms, diğer sayfalar 21–29 ms.
	// Profil: istek başına CPU yalnızca ~4 ms; kalan süre PostgreSQL
	// gidiş-dönüşlerinin BEKLENMESİ. Tek istekte ~24 sorgu sırayla
	// çalışıyordu, her biri ~2 ms → ~48 ms.
	//
	// Bu bloklar birbirine bağımlı değil (hepsi site.ID + locale ile
	// çalışır), bu yüzden birlikte uçurulur. Derinlik 24 sıralı turdan
	// 2 tura iner; p50 en yavaş bloğun süresine yaklaşır.
	//
	// Hata davranışı DEĞİŞMEDİ: her çağrı eskisi gibi kendi hatasını
	// yutuyor (log/boş değer). Bir blok patlarsa yalnız o blok boş kalır;
	// sayfa eskiden de böyle davranıyordu.
	var (
		highlights  *repo.Highlights
		layoutRaw   []byte
		navCats     []repo.Category
		menu        []MenuRow
		footer      *FooterConfig
		hava        *WeatherS
		kurlar      []repo.Kur
		kurDemo     bool
		cities      []repo.City
		etkinlik    []repo.Event
		kategoriler []repo.Category
	)
	var wg sync.WaitGroup
	wg.Add(10)
	go func() { defer wg.Done(); layoutRaw, _ = repo.GetLayout(ctx, s.pool, site.ID, "home") }()
	go func() { defer wg.Done(); navCats = s.navCategoriesLang(ctx, locale) }()
	go func() { defer wg.Done(); menu = s.publicMenuLocale(ctx, locale) }()
	go func() { defer wg.Done(); footer = s.footerConfigLang(ctx, locale) }()
	go func() { defer wg.Done(); hava = s.havaFromCache(ctx) }()
	go func() { defer wg.Done(); kurlar, kurDemo = servisKurlar(ctx, s.pool) }()
	go func() { defer wg.Done(); cities, _ = repo.ListCities(ctx, s.pool) }()
	go func() { defer wg.Done(); etkinlik, _ = repo.UpcomingEvents(ctx, s.pool, site.ID, 14, 5) }()
	go func() { defer wg.Done(); kategoriler, _ = repo.ListCategories(ctx, s.pool, site.ID) }()
	// Manşet/son dakika/taze havuz: yalnız buna icerikLocale gerekir.
	go func() {
		defer wg.Done()
		hl, err := repo.HomeHighlights(ctx, s.pool, site.ID, icerikLocale, 8, 12, 48)
		if err != nil {
			s.logger.Warn("home yüklenemedi", "error", err)
			return
		}
		highlights = hl
	}()
	wg.Wait()

	langBase := s.langBase(locale)

	d := s.baseView(locale, langBase, s.siteName)
	d.LangBase = langBase
	d.HomeURL = homeURL(locale)
	d.Hreflangs = s.yerelHreflang(ctx, s.absBase(r), site.ID, lang, homeURL)
	if dusus {
		d.Robots = "noindex,follow"
		d.Canonical = strings.TrimRight(s.absBase(r), "/") + homeURL(s.i18n.Default())
	}
	d.NavCategories = navCats
	d.Menu = menu
	d.Footer = footer
	d.Theme = themeFromReq(r)
	d.Home = &homeData{SiteID: site.ID}
	d.Home.JSONLD = s.orgJSONLD()
	if highlights != nil {
		hl := highlights
		d.Home.Breaking = hl.Breaking
		d.Home.Lead = hl.Lead
		d.Home.Recent = hl.Recent
		d.Home.BreakingLen = len(hl.Breaking)
		// slider: manşet + son yayınlananlar; blok sayısı 20'ye kadar beslenir.
		d.Home.Slides = uniqueContent(append(append([]repo.Content{}, hl.Lead...), hl.Recent...))
		// trend numaralı (önce dış Google Trends, yoksa son 7 gün popüler etiketler)
		d.Home.Trends = repo.HomeTrendTags(ctx, s.pool, site.ID, locale, 5)
		// Kategori sekmeli bölümler tembel kurulur: yalnız bu blok sayfada
		// varsa buildBlocksFrom içinde TEK sorguyla doldurulur (performans).
		_, d.Home.TabCatsSelected = tabCatSlugs(ctx, s)
		// Eski tek bloklu kategori panelleri yalnız düzende hâlâ
		// "kategori_bloklari" varsa üretilir; yeni "kategori_panel"
		// modülleri kendi verisini buildBlocksFrom içinde çeker.
		if strings.Contains(string(layoutRaw), `"kategori_bloklari"`) {
			d.Home.CatBlocks = s.homeCatBlocks(ctx, site.ID, locale, 6)
		}
		// top read (en çok okunan) + galeri foto/video: bağımsız, paralel.
		var gw sync.WaitGroup
		gw.Add(3)
		go func() { defer gw.Done(); d.Home.TopRead, _ = repo.TopRead(ctx, s.pool, site.ID, 5) }()
		go func() {
			defer gw.Done()
			d.Home.GalleryPhotos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{TypeKey: "foto_galeri", Status: "yayinda", SiteID: site.ID, PerPage: 6, Page: 1})
		}()
		go func() {
			defer gw.Done()
			d.Home.GalleryVideos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{TypeKey: "video", Status: "yayinda", SiteID: site.ID, PerPage: 6, Page: 1})
		}()
		gw.Wait()
	}
	// hava + kur bandı (önbellekten; yoksa bas)
	d.Home.Hava = hava
	// USD/EUR/ALTIN + BIST/BTC/ONS; demo kurlar banda hiç alınmaz.
	d.Home.Kurlar, d.Home.KurDemo = kurlar, kurDemo
	// hava bandı için il/şehir seçimi
	d.Cities = cities
	// kategori sekmesi filtresi (spec 07): ana sayfada akış
	catID, fslug, cname, other, _, _, _ := s.resolveFeedQuery(r)
	d.Akis = &feedData{CatSlug: fslug, CatName: cname, Other: other, Guncel: fslug == "" && !other, TabBase: "/"}
	if fslug != "" || other {
		items, _, _ := repo.FeedPage(ctx, s.pool, repo.FeedQuery{SiteID: site.ID, CategoryID: catID, Other: other, Page: 1, PerPage: 18})
		d.Home.Recent = items
		d.Home.Lead = nil
		d.Home.Breaking = nil
		d.Home.BreakingLen = 0
		d.Home.FilterName = cname
	}
	// modüler sayfa düzeni (spec 08)
	d.Home.Categories = kategoriler
	if len(layoutRaw) > 0 {
		d.Blocks = s.buildBlocks(ctx, d, layoutRaw)
	}
	if fslug != "" || other {
		d.Blocks = nil // filtre aktifken bloklar yerine filtrelenmiş liste göster
	}
	// yaklaşan etkinlikler (hatırlatma şeridi)
	d.Etkinlik = etkinlik
	return d
}

// homeCatBlocks ana sayfa için kategori panellerini üretir.
// Artık hangi kategoriler, sıra ve düzen Sayfa Tasarımcısı'ndan yönetilir
// (kategori_panelleri ayarı). Ayar yoksa tüm yayındaki kategoriler varsayılan
// düzenle gösterilir.
func (s *Server) homeCatBlocks(ctx context.Context, siteID int64, locale string, maxBlocks int) []catBlock {
	cats, err := repo.ListCategories(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	bySlug := map[string]repo.Category{}
	for _, c := range cats {
		bySlug[c.Slug] = c
	}

	type want struct {
		cat   repo.Category
		duzen string
		tone  string
	}
	var wants []want
	adet := 9
	cfg, hasCfg := s.panelConfigOf(ctx)

	if hasCfg {
		if cfg.Adet > 0 {
			adet = cfg.Adet
		}
		for _, p := range cfg.Panels {
			if !p.Show {
				continue
			}
			c, exists := bySlug[p.Slug]
			if !exists || c.Status != "yayinda" {
				continue
			}
			du := p.Duzen
			if !validPanelDuzen(du) {
				du = defaultPanelDuzen(p.Slug)
			}
			tone := p.Tone
			if tone == "" {
				tone = catTone(p.Slug)
			}
			wants = append(wants, want{cat: c, duzen: du, tone: tone})
		}
	} else {
		for _, c := range cats {
			if c.Status != "yayinda" {
				continue
			}
			wants = append(wants, want{cat: c, duzen: defaultPanelDuzen(c.Slug), tone: catTone(c.Slug)})
		}
	}

	limit := maxBlocks
	if limit <= 0 {
		limit = 6
	}
	if hasCfg {
		limit = 20 // yönetimden seçilenleri kırpma
	}
	if len(wants) > limit {
		wants = wants[:limit]
	}

	perPage := adet + 3
	if perPage < 6 {
		perPage = 6
	}

	out := make([]catBlock, 0, len(wants))
	for _, w := range wants {
		items, _, err := repo.FeedPage(ctx, s.pool, repo.FeedQuery{
			SiteID: siteID, CategoryID: w.cat.ID, Page: 1, PerPage: perPage,
		})
		if err != nil {
			continue
		}
		if b, ok := catBlockFromItems(w.cat.Name, w.cat.Slug, items, w.duzen, w.tone, adet); ok {
			out = append(out, b)
		}
	}
	return out
}

// catTone kategori başlığının rengini belirler (haberpi: Güncel yeşil).
func catTone(slug string) string {
	if slug == "gundem" || slug == "guncel" {
		return "yesil"
	}
	return "kirmizi"
}

func num(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// staticCacheControl statik dosyanın önbellek politikasını seçer.
//
// asset() sürüm parametresi (?v=…) ekler; o adresler içerik değişince
// değiştiği için sonsuza kadar önbelleklenebilir. Ama her statik dosya
// asset()'ten geçmiyor: ikon sprite'ı şablonlarda 40 yerde sürümsüz
// çağrılıyor (<use href="/static/vendor/tabler-icons.svg?v=b89545a6#ti-bell">).
//
// Önceden ikisine de "immutable" veriliyordu ve `immutable` tarayıcıya
// "bir daha sorma" demektir. Ölçüm: sprite 41 ikondan 621'e çıkarıldıktan
// sonra taze istek 239.282 bayt ve ti-bell içeriyordu; önbellekten gelen
// 11.412 bayt ve içermiyordu. Olmayan ikon hata vermez, SESSİZCE boş kutu
// basılır — kullanıcı "ikonlar kayboldu" der, günlükte iz olmaz.
//
// Sürümsüz dosya için kısa süre + revalidate: bayatlık beş dakikayla
// sınırlı kalır, her sayfada yeniden indirme olmaz.
func staticCacheControl(rawQuery string) string {
	if strings.Contains(rawQuery, "v=") {
		return "public, max-age=31536000, immutable"
	}
	return "public, max-age=300, must-revalidate"
}

func (s *Server) handlePublicSegment(w http.ResponseWriter, r *http.Request) {
	// statik/medya sahte kökleri (literal ağaç yok): burada çöz.
	if strings.HasPrefix(r.URL.Path, "/static/") {
		if strings.HasSuffix(r.URL.Path, ".webmanifest") {
			w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=3600")
		} else {
			w.Header().Set("Cache-Control", staticCacheControl(r.URL.RawQuery))
		}
		staticFS, err := fs.Sub(webui.FS, "static")
		if err == nil {
			http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/media/") {
		s.handleMedia(w, r)
		return
	}
	if r.URL.Path == "/" {
		s.handleHome(w, r)
		return
	}

	// bot tuzağı (honeypot): robots'ta yasaklı + insandan gizli — çiğneyen botlar buraya düşer
	if isTrapPath(r.URL.Path) {
		s.handleBotTrap(w, r)
		return
	}

	// IndexNow anahtar dosyası (kök): /<key>.txt — doğrulama için kökte sunulur
	if strings.HasSuffix(r.URL.Path, ".txt") && strings.TrimPrefix(r.URL.Path, "/") == s.indexNowKey(r.Context())+".txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write([]byte(s.indexNowKey(r.Context())))
		return
	}

	// 301/302 yönlendirme koruması (spec 01 §D7/J2) — eski URL'ler burada yakalanır.
	if s.maybeRedirect(w, r, s.siteByDomain(r.Context(), r.Host).ID) {
		return
	}

	// WordPress kimlikli adres: /?p=<yazı kimliği> — kanonik adrese taşınır.
	if wpID := wpKimlikSorgusu(r.URL.Query()); wpID > 0 {
		s.eskiKimlikYonlendir(w, r, wpID)
		return
	}

	p := strings.Trim(r.URL.Path, "/")
	if p == "" {
		s.handleHome(w, r)
		return
	}
	segs := strings.Split(p, "/")

	lang := s.defaultLang(r.Context())
	i := 0
	if s.i18n.IsSupported(segs[0]) {
		lang = segs[0]
		i = 1
	}
	if i >= len(segs) {
		r.SetPathValue("lang", lang)
		s.handleHome(w, r)
		return
	}

	seg := segs[i]
	r.SetPathValue("lang", lang)

	// MAKALE YOLU BÖLÜM ADINDAN ÖNCE ÇÖZÜLÜR.
	//
	// Tuzak (canlıda görüldü, 10 Kasım 2026): `/borsa/2023/04/17/<slug>`
	// adresi makale değil BORSA fiyat sayfası olarak açılıyordu — çünkü
	// aşağıdaki `case seg == "borsa"` bölüm eşleşmesi, tarihli makale
	// deseninden önce geliyordu. Aynı hata `etkinlikler`, `sinemalar`,
	// `arsiv` gibi tüm literal bölüm adları için geçerliydi: o adlarla
	// başlayan makaleler yanlış sayfaya (ve yanlış canonical'a) düşüyordu.
	//
	// Çözüm: tarihli makale deseni (/{bölüm}/{YYYY}/{MM}/{DD}/{slug})
	// literal eşleşmelerden ÖNCE denenir. Desen takvimsel olarak geçerli
	// olmalı (makaleTarihDeseni), bu yüzden `/borsa/hisseler` gibi gerçek
	// bölüm adresleri etkilenmez.
	if s.makaleYolu(w, r, segs[i:], lang) {
		return
	}

	switch {
	case seg == "kategori" && i+1 < len(segs):
		r.SetPathValue("slug", segs[i+1])
		s.handleCategory(w, r)
	case seg == "son-dakika" || seg == "fotograflar" || seg == "videolar":
		s.handleBreakingPage(w, r)
	case seg == "arsiv":
		s.handleArchive(w, r)
	case seg == "hava-durumu":
		s.handleHavaDurumu(w, r)
	case seg == "namaz-vakitleri":
		s.handleNamazDurumu(w, r)
	case seg == "imsakiye":
		s.handleImsakiye(w, r)
	case seg == "borsa" && i+1 < len(segs) && segs[i+1] == "hisseler":
		s.handleHisselerPage(w, r)
	case seg == "borsa" && i+1 < len(segs) && segs[i+1] == "kripto":
		s.handleKriptoPage(w, r)
	case seg == "borsa":
		s.handleBorsaPage(w, r)
	case seg == "fikstur":
		s.handleFiksturPage(w, r)
	case seg == "astroloji":
		s.handleAstrolojiPage(w, r)
	case seg == "etkinlikler":
		s.handleEventsPublic(w, r)
	case seg == "anketler":
		s.handlePollsPublic(w, r)
	case seg == "gazete-mansetleri":
		s.handleNewspapersPublic(w, r)
	case seg == "sinemalar":
		s.handleFilmsPublic(w, r)
	case seg == "resmi-ilanlar" && i+1 < len(segs):
		s.handleNoticeDetail(w, r, segs[i+1])
	case seg == "resmi-ilanlar":
		s.handleNoticesPublic(w, r)
	case seg == "akis":
		s.handleAkisPage(w, r)
	case seg == "nobetci-eczaneler":
		s.handleEczanePage(w, r)
	case seg == "arama":
		s.handleSearch(w, r)
	case seg == "yazarlar":
		s.handlePeopleList(w, r)
	case seg == "yazar" && i+1 < len(segs):
		r.SetPathValue("slug", segs[i+1])
		s.handlePerson(w, r)
	case seg == "sayfa" && i+1 < len(segs):
		// Kurumsal sayfalar kök adreste yayımlanır (Faz 143): eski
		// /sayfa/<slug> adresi kalıcı olarak /<slug> adresine yönlenir.
		if _, err := repo.GetPageBySlug(r.Context(), s.pool, segs[i+1], true); err != nil {
			s.handleNotFound(w, r)
			return
		}
		r.SetPathValue("slug", segs[i+1])
		s.handlePageLegacyRedirect(w, r, lang, segs[i+1])
	case seg == "sitemap.xml":
		s.handleSitemap(w, r)
	case seg == "rss.xml":
		s.handleFeed(w, r)
	case seg == "feed" || (seg == "comments" && i+1 < len(segs) && segs[i+1] == "feed"):
		// WordPress besleme adresleri RSS'e taşınır.
		http.Redirect(w, r, homeURL(lang)+"rss.xml", http.StatusMovedPermanently)
	default:
		// WordPress tarih arşivi (/2019, /2019/12, /2019/12/11) yeni sitede
		// ayrı sayfa değil; arşive taşınır.
		if tarihArsivMi("/" + p) {
			http.Redirect(w, r, homeURL(lang)+"arsiv", http.StatusMovedPermanently)
			return
		}

		// Tarihli makale deseni switch'ten ÖNCE çözüldü (bkz. makaleYolu).

		id := articleIDFromSegment(seg)
		slug := articleSlugFromSegment(seg)
		if id > 0 {
			// Eski (tarihsiz) adres: kalıcı olarak tarihli kanonik adrese
			// yönlendirilir. 301 bilinçli: SEO'da bağlantı değeri taşınır.
			if c, err := repo.ContentBySlugAndID(r.Context(), s.pool, slug, id); err == nil && c.Status == "yayinda" {
				hedef := makaleURL(c)
				if r.URL.RawQuery != "" {
					hedef += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, hedef, http.StatusMovedPermanently)
				return
			}
			// Yönlendirme kurulamadıysa eski davranış korunur.
			s.handleCDArticle(w, r, lang, slug, id)
			return
		}
		// Kök adresli kurumsal sayfa (Faz 143): /kullanim-kosullari gibi.
		// Rezerve segmentler yukarıda tükendiği için burada yalnız kalan
		// adlar denenir; içerik adresleri her zaman "-<id>" ile biter.
		if p, err := repo.GetPageBySlug(r.Context(), s.pool, seg, true); err == nil {
			r.SetPathValue("slug", p.Slug)
			s.handlePage(w, r)
			return
		}
		s.handleNotFound(w, r)
	}
}

// Örnek: "turkiye-gundem-42" → slug "turkiye-gundem", id 42
func articleIDFromSegment(seg string) int64 {
	i := strings.LastIndex(seg, "-")
	if i <= 0 {
		return 0
	}
	id := parseID(seg[i+1:])
	if id <= 0 {
		return 0
	}
	return id
}

func articleSlugFromSegment(seg string) string {
	i := strings.LastIndex(seg, "-")
	if i <= 0 {
		return ""
	}
	return seg[:i]
}

// uniqueContent aynı içerik kimliğini yalnızca bir kez bırakır.
func uniqueContent(items []repo.Content) []repo.Content {
	seen := make(map[int64]bool, len(items))
	out := make([]repo.Content, 0, len(items))
	for _, it := range items {
		if seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		out = append(out, it)
	}
	return out
}

// handleContentPreview taslağı/yayın dışı içeriği okuyucu görünümüyle önizler
// (Faz 46). Yönetim rotası arkasındadır; yayın kapısı atlanır, hit SAYILMAZ,
// cache no-store.
func (s *Server) handleContentPreview(w http.ResponseWriter, r *http.Request) {
	// ÖNİZLEME = CANLI ŞABLON (kullanıcı bildirimi, 10 Kasım 2026:
	// "ön izlemde yeni resim var ve bu önizleme eski theblock tasarımına
	// ait"). Burada `handleArticle` (theblock) çağrılıyordu; editör yeni
	// görseli eklediğinde önizlemede eski tasarımı görüyordu.
	//
	// Artık canlı makale ile AYNI cd şablonu kullanılıyor. Tek fark:
	// durum kapısı yok (taslak/planlanmış da açılır) ve okunma sayılmaz.
	s.handleCDArticleOnizleme(w, r)
}

// havaFromCache servis bandındaki varsayılan şehir (Ankara) havasını
// önbellekten okur. Ana sayfa dışındaki sayfalarda veri havuzunda hava
// olmadığı için tasarımcı/blok yolu da bu yardımcıyı kullanır.
func (s *Server) havaFromCache(ctx context.Context) *WeatherS {
	ak, err := repo.GetCitySlug(ctx, s.pool, "ankara")
	if err != nil {
		return nil
	}
	b, err := repo.CacheGet(ctx, s.pool, "hava", strconv.Itoa(ak.Plaka))
	if err != nil {
		return nil
	}
	var w repo.WeatherInfo
	if json.Unmarshal(b, &w) != nil {
		return nil
	}
	return &WeatherS{City: w.City, Slug: ak.Slug, Sicak: w.Sicaklik, Aciklama: w.Aciklama, Demo: w.Demo}
}

// categoryBlockHome kategori liste sayfası bloklarının veri havuzunu kurar.
// Ana sayfayla aynı homeData şekli kullanılır; liste içerikleri o kategoriye
// ait olduğundan top/sağ bloklar (slider, 6'lı manşet, son haberler) doğru
// içerikle dolar.
func (s *Server) categoryBlockHome(ctx context.Context, siteID int64, locale string, items []repo.Content) *homeData {
	h := &homeData{SiteID: siteID}
	h.Recent = items
	h.Lead = firstContent(items, 5)
	h.Slides = items
	h.TopRead, _ = repo.TopRead(ctx, s.pool, siteID, 5)
	h.Categories, _ = repo.ListCategories(ctx, s.pool, siteID)
	h.GalleryPhotos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		TypeKey: "foto_galeri", Status: "yayinda", SiteID: siteID, PerPage: 6, Page: 1})
	h.GalleryVideos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		TypeKey: "video", Status: "yayinda", SiteID: siteID, PerPage: 6, Page: 1})
	h.Breaking, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status: "yayinda", SiteID: siteID, Locale: locale, PerPage: 8, Page: 1})
	return h
}

// ---- kategori / arşiv / son dakika / sayfa ----

func (s *Server) handleCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	locale := langOf(r, s)
	// Eski Güncel adresi yeni Gündem adresine kalıcı yönlensin.
	if slug == "guncel" {
		target := s.langBase(locale) + "/kategori/gundem"
		if q := r.URL.RawQuery; q != "" {
			target += "?" + q
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	// YENİ KABUK — kullanıcı bildirimi (9 Kasım 2026): "/kategori/ … hâlâ
	// theblock'tan kalma stil ve şablon". Aynı içerik iki ayrı şablonla iki
	// farklı görünümde basılıyordu; artık `/tech` ile `/kategori/tech`
	// birebir aynı CoinDesk bölüm sayfası.
	//
	// PANEL DÜZENİ KORUNUR: kategoriye özel bir düzen (`kategori:<slug>`)
	// varsa eski motor kullanılır — editörün elle kurduğu modüller sessizce
	// kaybolmasın. Ölçüm: page_layouts tablosu şu an BOŞ, yani her kategori
	// yeni kabuğa geçiyor.
	if !s.sayfaDuzeniVar(ctx, "kategori:"+slug) {
		s.handleCDSection(w, r, slug)
		return
	}
	// Akış vitrini (coinhit/enerji modeli) için HTML parçası: "Daha fazla"
	// ve sonsuz kaydırma bu uçtan 12'lik enerji bloklarını çeker.
	if r.URL.Query().Get("parca") == "1" {
		s.handleCategoryAkisParca(w, r, slug, locale)
		return
	}
	site := s.siteByDomain(ctx, r.Host)

	// ?s=<sayfa> ile sayfalama (haberpi düzeni)
	page := 1
	if v := r.URL.Query().Get("s"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 500 {
			page = n
		}
	}
	const perPage = 24
	cats := s.cachedCategories(ctx, site.ID)
	var cat *repo.Category
	for i := range cats {
		if cats[i].Slug == slug {
			cat = &cats[i]
			break
		}
	}
	if cat == nil || cat.Status != "yayinda" {
		s.handleNotFound(w, r)
		return
	}
	// Kategori adı yerelleştirilir (Faz 189): `translations` tablosunda
	// `category.<slug>` anahtarı. Çeviri yoksa Türkçe ad korunur.
	title := s.categoryLangName(ctx, locale, cat.Slug, cat.Name)
	// Şablon metinleri (Faz 142): başlık/açıklama/boş kutu metinleri Sayfa
	// Tasarımcısı'ndan düzenlenebilir. Kırıntı yolu her zaman gerçek adı taşır.
	pt := s.pageTextsFor(ctx, site.ID, "kategori:"+slug, locale, categoryTextDefs(slug),
		map[string]string{"kategori": cat.Name})
	intro := pt["kategori.intro"]
	// İstenen dilde içerik yoksa varsayılan dile düşülür: boş kategori
	// sayfası yerine anlamlı liste gösterilir, ama noindex + canonical
	// varsayılan dilin adresine verilir (kopya dizinlenmesin).
	icerikLocale, dusus := s.kategoriIcerikDili(ctx, site.ID, slug, locale)
	// Bir fazla içerik çekilir: "sonraki sayfa var mı" ancak böyle anlaşılır.
	items, err := repo.PublishedBySlug(ctx, s.pool, site.ID, slug, icerikLocale, perPage*page+1)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	hasNext := len(items) > perPage*page
	if hasNext {
		items = items[:perPage*page]
	}
	if (page-1)*perPage < len(items) {
		items = items[(page-1)*perPage:]
	}
	d := s.baseView(locale, s.langBase(locale), title)
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategoriesLang(ctx, locale)
	d.Menu = s.publicMenuLocale(ctx, locale)
	d.Footer = s.footerConfigLang(ctx, locale)
	d.Theme = themeFromReq(r)
	// SEO: kategori adı + açıklama + kendi canonical'ı (sayfa parametresi korunur).
	absBase := strings.TrimRight(s.absBase(r), "/")
	d.MetaTitle = title + " — " + d.SiteName
	d.MetaDesc = metaDesc(intro, title+" kategorisindeki son haberler, güncel gelişmeler ve analizler.", s.siteName)
	if len(items) == 0 || dusus {
		d.Robots = "noindex,follow"
	}
	if dusus {
		d.Canonical = absBase + homeURL(s.i18n.Default()) + "kategori/" + slug
	} else {
		d.Canonical = absBase + r.URL.Path
		if page > 1 {
			d.Canonical += "?s=" + itoa(page)
		}
	}
	d.Hreflangs = s.yerelHreflang(ctx, absBase, site.ID, locale, func(l string) string {
		return homeURL(l) + "kategori/" + slug
	})
	// Hiyerarşi: alt kategori sayfasında üst kategori + kardeş sekmeler;
	// üst kategori sayfasında kendi alt kategorileri sekme olarak gösterilir.
	var parent *repo.Category
	if cat.ParentSlug != nil && *cat.ParentSlug != "" {
		for i := range cats {
			if cats[i].Slug == *cat.ParentSlug {
				parent = &cats[i]
				break
			}
		}
	}
	var tabs []repo.Category
	tabParent := cat.Slug
	if parent != nil {
		tabParent = parent.Slug
		for _, c := range cats {
			if c.ParentSlug != nil && *c.ParentSlug == parent.Slug && c.Status == "yayinda" && (c.InMenu || c.ContentCount > 0) {
				tabs = append(tabs, c)
			}
		}
	} else {
		for _, c := range cats {
			if c.ParentSlug != nil && *c.ParentSlug == cat.Slug && c.Status == "yayinda" && (c.InMenu || c.ContentCount > 0) {
				tabs = append(tabs, c)
			}
		}
	}
	crumbs := []crumb{{Name: "Anasayfa", URL: absBase + homeURL(locale)}}
	if parent != nil {
		crumbs = append(crumbs, crumb{Name: parent.Name, URL: absBase + s.langBase(locale) + "/kategori/" + parent.Slug})
	}
	crumbs = append(crumbs, crumb{Name: title, URL: d.Canonical})
	d.BreadcrumbJSONLD = breadcrumbJSONLD(crumbs)
	d.Category = &categoryData{
		Title: title, Slug: slug, Items: items, Intro: intro,
		H1:         pt["kategori.baslik"],
		CrumbTitle: cat.Name,
		AllText:    pt["kategori.tumu"],
		EmptyTitle: pt["kategori.bos_baslik"],
		EmptyText:  pt["kategori.bos_metin"],
		Page:       page, PerPage: perPage, HasPrev: page > 1, HasNext: hasNext,
		Tabs: tabs, AllURL: d.LangBase + "/kategori/" + tabParent,
	}
	if parent != nil {
		d.Category.ParentName = parent.Name
		d.Category.ParentSlug = parent.Slug
	}
	d.Category.View = s.categoryListView(ctx)
	if validCategoryView(cat.ListView) && cat.ListView != "" {
		d.Category.View = cat.ListView
	}
	switch d.Category.View {
	case "liste", "izgara", "mozaik", "akis":
		// geçerli
	default:
		d.Category.View = "klasik"
	}
	// Liste kartlarında video süresi rozeti (Faz 21): tek sorgu.
	s.fillVideoDurations(ctx, d.Category.Items)
	// Kategori sayfası düzeni: önce bu kategoriye özel "kategori:<slug>",
	// yoksa genel "kategori" düzeni. Bloklar modüler gövdeyi oluşturur.
	raw, _ := repo.GetLayout(ctx, s.pool, site.ID, "kategori:"+slug)
	if layoutEmpty(raw) {
		raw, _ = repo.GetLayout(ctx, s.pool, site.ID, "kategori")
	}
	if !layoutEmpty(raw) {
		h := s.categoryBlockHome(ctx, site.ID, locale, items)
		// Eski kategori panelleri yalnız düzende hâlâ varsa üretilir.
		if strings.Contains(string(raw), `"kategori_bloklari"`) {
			h.CatBlocks = s.homeCatBlocks(ctx, site.ID, locale, 6)
		}
		d.Blocks = s.buildBlocksFrom(ctx, d, raw, h)
		d.CatRows = groupCatRows(d.Blocks)
		for _, b := range d.Blocks {
			if b.Visible && (b.Zone == "govde") {
				d.Category.Moduler = true
				break
			}
		}
	}
	s.renderPublic(w, r, "category", d, cacheControlPublic)
}

// handleCategoryAkisParca kategori akış vitrininin sonraki 12'lik bloğunu
// HTML olarak döndürür. Sınır/sayfa boyutu ve etiket gizleme ayarı kategori
// düzenindeki akış bloğundan okunur; istemci yalnız "bas" (mutlak sıra) verir.
// Yanıt: <div class="ak-grid">…</div> blokları + X-Akis-Son / X-Akis-Bas
// başlıkları (JS bunları okur).
func (s *Server) handleCategoryAkisParca(w http.ResponseWriter, r *http.Request, slug, locale string) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)

	// Akış ayarları: kategoriye özel düzen → genel kategori düzeni.
	limit, per, hideBadge := 60, akisBatchSize, false
	raw, _ := repo.GetLayout(ctx, s.pool, site.ID, "kategori:"+slug)
	if layoutEmpty(raw) {
		raw, _ = repo.GetLayout(ctx, s.pool, site.ID, "kategori")
	}
	if !layoutEmpty(raw) {
		for _, b := range parseHomeBlocks(raw) {
			if b.Type == "kategori_akis" || (b.Type == "kategori_liste" && (b.ListView == "akis" || b.CatHeroStyle == "akis")) {
				limit = b.AkisLimit
				per = b.AkisPer
				hideBadge = b.HideBadge
				break
			}
		}
	}
	if limit < 8 {
		limit = 60
	}
	if per < akisBatchSize {
		per = akisBatchSize
	}
	if per > 36 {
		per = 36
	}

	bas := parseIntOr(r.URL.Query().Get("bas"))
	if bas < 0 {
		bas = 0
	}
	if bas > limit {
		bas = limit
	}
	end := bas + per
	if end > limit {
		end = limit
	}
	if end <= bas {
		w.Header().Set("X-Akis-Son", "1")
		w.Header().Set("X-Akis-Bas", itoa(bas))
		return
	}

	// Kategorinin ilk `end` haberini çekip `bas`tan sonrasını al.
	items, err := repo.PublishedBySlug(ctx, s.pool, site.ID, slug, locale, end)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	fetched := len(items)
	// Sınır doldu ya da kategori bu kadar haber içeriyor → akış bitti.
	son := end >= limit || fetched < end
	if bas < fetched {
		items = items[bas:]
	} else {
		items = nil
	}
	if len(items) > per {
		items = items[:per]
	}
	s.fillVideoDurations(ctx, items)

	batches := buildAkisBatches(items, s.langBase(locale), hideBadge)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if son || len(batches) == 0 {
		w.Header().Set("X-Akis-Son", "1")
	}
	w.Header().Set("X-Akis-Bas", itoa(bas+len(items)))
	if len(batches) > 0 {
		var akBuf strings.Builder
		if err := s.tmpl.ExecuteTemplate(&akBuf, "ak_batches", batches); err != nil {
			http.Error(w, "render", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, akBuf.String())
	}
}

func (s *Server) handleBreakingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	rows, err := repo.HomeHighlights(ctx, s.pool, site.ID, locale, 0, 60, 0)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	d := s.baseView(locale, s.langBase(locale), s.i18n.T(locale, "breaking"))
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategoriesLang(ctx, locale)
	d.Menu = s.publicMenuLocale(ctx, locale)
	d.Footer = s.footerConfigLang(ctx, locale)
	d.Theme = themeFromReq(r)
	d.Category = &categoryData{Title: s.i18n.T(locale, "breaking"), View: s.categoryListView(ctx)}
	if rows != nil {
		d.Category.Items = rows.Breaking
	}
	s.fillVideoDurations(ctx, d.Category.Items)
	s.renderPublic(w, r, "category", d, cacheControlPublic)
}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)

	// YENİ KABUK — kullanıcı bildirimi (9 Kasım 2026): theblock şablonu
	// kalan sayfalar yeni tasarıma geçsin. Arşiv eski kabuğun son büyük
	// listesiydi (site_header/site_footer). Panelde bu sayfa için özel
	// düzen varsa eski motor korunur.
	if !s.sayfaDuzeniVar(ctx, "arsiv") {
		s.cdArsivSayfasi(w, r, locale, site.ID)
		return
	}
	items, _ := repo.RecentPublished(ctx, s.pool, site.ID, locale, 100)
	d := s.baseView(locale, s.langBase(locale), s.i18n.T(locale, "recent"))
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategoriesLang(ctx, locale)
	d.Menu = s.publicMenuLocale(ctx, locale)
	d.Footer = s.footerConfigLang(ctx, locale)
	d.Theme = themeFromReq(r)
	d.Category = &categoryData{Title: s.i18n.T(locale, "recent"), Items: items, View: s.categoryListView(ctx), Page: 1, PerPage: 100}
	s.fillVideoDurations(ctx, d.Category.Items)
	s.renderPublic(w, r, "archive", d, cacheControlPublic)
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	p, err := repo.GetPageBySlug(ctx, s.pool, slug, true)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	locale := langOf(r, s)
	sid := s.curSiteID(r)
	// Şablon metinleri + gövde override'ı (Faz 142/143).
	pt := s.pageTextsFor(ctx, sid, "sayfa:"+p.Slug, locale, pageTextDefsFor("sayfa"),
		map[string]string{"baslik": p.Title})
	title := p.Title
	if v := strings.TrimSpace(pt["sayfa.baslik"]); v != "" {
		title = v
	}
	title = s.applyContactTokens(ctx, title)
	body := s.blocksToHTML(p.Detail, nil)
	if v := strings.TrimSpace(pt["sayfa.govde"]); v != "" {
		body = template.HTML(platform.SanitizeRichHTML(v))
	}
	// Yer tutucular ({domain}, {eposta_info}…) yönetimdeki iletişim
	// bilgileriyle doldurulur (Faz 145).
	body = template.HTML(s.applyContactTokens(ctx, string(body)))
	// Künye sayfası: BİK tebliğinin zorunlu tuttuğu on alan, Kurum
	// kontrolünün aradığı kimliklerle birlikte gövdenin sonuna eklenir.
	// Kimlikler koddan geldiği için editörün sayfa metnini değiştirmesi
	// uyumu bozmaz (bik_kunye.go).
	if p.Slug == "kunye" {
		body = template.HTML(string(body) + string(bikKunyeHTML(s.bikKunyeDoldur(ctx))))
	}
	d := s.baseView(locale, s.langBase(locale), title)
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategoriesLang(ctx, locale)
	d.Menu = s.publicMenuLocale(ctx, locale)
	d.Footer = s.footerConfigLang(ctx, locale)
	d.Theme = themeFromReq(r)
	d.Page = &pageData{Title: title, HTML: body}
	// İletişim sayfası: form + iletişim kartları (Faz 145).
	if p.Slug == "iletisim" {
		d.Page.Contact = contactViewOf(s.contactConfig(ctx))
	}
	if f := s.consumeFlash(r); f != nil {
		d.Page.FlashKind = f.Kind
		d.Page.FlashText = f.Text
	}
	// Kanonik adres kök slug'dır: /kullanim-kosullari (Faz 143).
	d.Canonical = strings.TrimRight(s.absBase(r), "/") + s.langBase(locale) + "/" + p.Slug
	// Kırıntı yolu yapısal verisi: Anasayfa › Sayfa Adı.
	d.BreadcrumbJSONLD = breadcrumbJSONLD([]crumb{
		{Name: "Anasayfa", URL: strings.TrimRight(s.absBase(r), "/") + homeURL(locale)},
		{Name: title, URL: d.Canonical},
	})
	d.MetaTitle = title + " — " + d.SiteName
	if p.Spot != nil && strings.TrimSpace(*p.Spot) != "" {
		d.MetaDesc = *p.Spot
	}
	// Sayfa düzeni: modüller gövdeyi sarabilir (yoksa sabit gövde basılır).
	var duzenVar bool
	if raw, err := repo.GetLayout(ctx, s.pool, sid, "sayfa:"+p.Slug); err == nil && !layoutEmpty(raw) {
		d.Blocks = s.buildBlocksFrom(ctx, d, raw, s.pageBlockHome(ctx, sid, locale))
		duzenVar = len(d.Blocks) > 0
		for i := range d.Blocks {
			if d.Blocks[i].Type == "sayfa_govde" && d.Blocks[i].Visible {
				d.Page.HasBodyBlock = true
				break
			}
		}
	}
	cc := cacheControlPublic
	if p.Slug == "iletisim" {
		cc = "no-store" // form + flash mesajı önbelleğe girmemeli
	}

	// YENİ TASARIM (9 Kasım 2026): kurumsal sayfalar theblock'un `page`
	// şablonundan (site-header/site-footer) geliyordu; alt bilgiden bir
	// bağlantıya basan kullanıcı başka bir siteye geçmiş gibi görüyordu.
	// Düzen (tasarımcı blokları) YOKSA bitcosar kabuğu basılır.
	//
	// NEDEN KOŞUL: sayfa tasarımcısı bu sayfalara modül eklediyse (ör.
	// "Son Haberler" ızgarası) o bloklar `site_public.html` motoruyla
	// çiziliyor. Blokları yeni kabuğa taşımadan zorla cd-* basmak, panelden
	// eklenen modülleri sessizce yok ederdi. Bugün `sayfa:*` için kayıtlı
	// düzen YOK (ölçüldü, 9 Kasım 2026) — yani tüm kurumsal sayfalar yeni
	// kabukta.
	if !duzenVar {
		// BreadcrumbJSONLD sadece JSON gövdesi döner (eski şablon `<script>`
		// etiketini kendisi sarıyordu). CDHead ham HTML beklediği için etiket
		// BURADA sarılır — yoksa JSON sayfanın tepesine düz metin olarak
		// basılıyordu (ölçüldü: /iletisim, 9 Kasım 2026).
		ld := ""
		if s := strings.TrimSpace(string(d.BreadcrumbJSONLD)); s != "" {
			ld = `<script type="application/ld+json">` + s + `</script>`
		}
		v := templkit.CDKurumsalSayfa{
			Shell:     s.cdShellCtx(ctx, r, ""),
			Title:     d.MetaTitle,
			Desc:      d.MetaDesc,
			Canonical: d.Canonical,
			JSONLD:    ld,
			BaseURL:   strings.TrimRight(s.cfg.BaseURL, "/"), Locale: locale,
			Baslik: title,
			Govde:  body, // body zaten template.HTML (templkit.SafeHTML eş adı)
			// Kurumsal sayfa = statik bilgi sayfası.
			PageCategory:    "static_page",
			ContentCategory: "about",
		}
		if p.Slug == "iletisim" {
			cv := contactViewOf(s.contactConfig(ctx))
			if cv != nil {
				v.Iletisim = &templkit.CDIletisim{
					Genel: cv.Info, Haber: cv.Haber, Reklam: cv.Reklam,
					Duzeltme: cv.Duzeltme, Telefon: cv.Telefon, Adres: cv.Adres,
					Calisma: cv.Calisma, FormAktif: cv.FormAktif,
				}
			}
			if d.Page.FlashKind != "" || d.Page.FlashText != "" {
				v.FlashTip = d.Page.FlashKind
				v.FlashMetin = d.Page.FlashText
			}
		}
		w.Header().Set("Cache-Control", cc)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templkit.CDKurumsalSayfaView(v).Render(ctx, w); err != nil {
			s.logger.Error("kurumsal sayfa render", "slug", p.Slug, "error", err)
		}
		return
	}
	s.renderPublic(w, r, "page", d, cc)
}

// handlePageLegacyRedirect eski /sayfa/<slug> adresini kök adrese taşır.
// 301 kalıcıdır: arama motorları yeni adresi öğrenir, eski bağlantılar bozulmaz.
func (s *Server) handlePageLegacyRedirect(w http.ResponseWriter, r *http.Request, lang, slug string) {
	target := s.langBase(lang) + "/" + slug
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// pageBlockHome kurumsal sayfa düzenindeki modüller için veri havuzu.
// Yazı sayfasıyla aynı homeData şekli kullanılır; içerik dışlama yoktur.
func (s *Server) pageBlockHome(ctx context.Context, siteID int64, locale string) *homeData {
	h := &homeData{SiteID: siteID}
	h.TopRead, _ = repo.TopRead(ctx, s.pool, siteID, 5)
	h.Categories, _ = repo.ListCategories(ctx, s.pool, siteID)
	h.GalleryPhotos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		TypeKey: "foto_galeri", Status: "yayinda", SiteID: siteID, PerPage: 6, Page: 1})
	h.GalleryVideos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		TypeKey: "video", Status: "yayinda", SiteID: siteID, PerPage: 6, Page: 1})
	h.Breaking, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status: "yayinda", SiteID: siteID, Locale: locale, PerPage: 24, Page: 1})
	h.Recent = h.Breaking
	h.Slides = h.Recent
	h.Lead = firstContent(h.Recent, 5)
	return h
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "404")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)
	w.WriteHeader(http.StatusNotFound)
	s.renderPublic(w, r, "notfound", d, "no-store")
}

// ---- yorum + hit ----

// buildCommentTree onaylı yorumları kök + yanıt zinciri olarak sıralar (Faz 36).
// Bir yorum, zincirinin EN ÜST KÖKÜNÜN altında toplanır (yanıt-yanıtı dahil);
// zinciri köke ulaşmayan (kök silinmiş/onaysız) yorum kök gibi gösterilir.
// Döngü koruması vardır. Sıralama: kökler yorum sırasına göre, yanıtlar kendi
// kökünün altında aynı akışla — sorgu created_at ASC döndürdüğü için doğaldır.
func buildCommentTree(comments []repo.Comment) []commentTreeNode {
	if len(comments) == 0 {
		return nil
	}
	nodes := make(map[int64]*commentTreeNode, len(comments))
	for i := range comments {
		nodes[comments[i].ID] = &commentTreeNode{C: comments[i]}
	}
	// rootOf zincirin en üst kök id'sini verir (döngü kırılınca güncel düğüm).
	rootOf := func(id int64) int64 {
		cur := id
		for hops := 0; ; hops++ {
			c, ok := nodes[cur]
			if !ok || c.C.ParentID == 0 || hops > len(comments) {
				return cur
			}
			cur = c.C.ParentID
		}
	}
	isRoot := func(cm repo.Comment) bool {
		if cm.ParentID == 0 {
			return true
		}
		if nodes[cm.ParentID] == nil {
			return true // ebeveyn listede yok
		}
		return rootOf(cm.ID) == cm.ID
	}

	var roots []commentTreeNode
	rootIdx := make(map[int64]int, len(comments))
	for i := range comments {
		if !isRoot(comments[i]) {
			continue
		}
		rootIdx[comments[i].ID] = len(roots)
		roots = append(roots, commentTreeNode{C: comments[i]})
	}
	for i := range comments {
		if isRoot(comments[i]) {
			continue
		}
		if idx, ok := rootIdx[rootOf(comments[i].ID)]; ok {
			roots[idx].Replies = append(roots[idx].Replies, comments[i])
		}
	}
	return roots
}

// handleCommentLike yorum beğenisini kaydeder (aynı IP 1 kez; Faz 36).
func (s *Server) handleCommentLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	likes, voted, err := repo.LikeComment(ctx, s.pool, id, clientIP(r))
	if err != nil || errors.Is(err, repo.ErrNotFound) {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "likes": likes, "begenildi": voted})
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// yol sonundaki id'yi al: /yorum/{id} veya /{lang}/yorum/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	id := parseID(parts[len(parts)-1])
	name := strings.TrimSpace(r.FormValue("isim"))
	body := strings.TrimSpace(r.FormValue("metin"))
	ip := clientIP(r)
	if id == 0 || name == "" || body == "" {
		s.flashRedirect(w, r, s.ayniSiteYolu(r, r.Referer()), "error", "Yorum eklenemedi.")
		return
	}
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil || c.Status != "yayinda" || c.CommentStatus == "kapali" {
		s.flashRedirect(w, r, s.ayniSiteYolu(r, r.Referer()), "error", "Yorum kapalı.")
		return
	}
	// geri yalnız aynı siteye yönlendirir (açık yönlendirme koruması).
	geri := func(kind, msg string) {
		url := s.ayniSiteYolu(r, r.Referer())
		if url == "" {
			url = publicArticleURL(c.Locale, c.DisplaySlug(), c.ID)
		}
		s.flashRedirect(w, r, url, kind, msg)
	}

	// Faz 35 — spam koruması.
	// 1) Honeypot: gizli "web" alanını dolduran bot sessizce düşer (kayıt YOK).
	//    Bota "işlem başarılı" sinyali gider ki davranışını değiştirmesin.
	if strings.TrimSpace(r.FormValue("web")) != "" {
		_ = repo.LogBotTrap(ctx, s.pool, ip, r.UserAgent(), "/yorum/honeypot", r.Referer())
		geri("info", "Yorumunuz onay sonrası yayınlanacak.")
		return
	}
	// Faz 36 — yanıt: parent_id verildiyse aynı içerikte gerçek bir yorum olmalı.
	parentID := parseID(r.FormValue("parent_id"))
	if parentID > 0 {
		pc, perr := repo.GetComment(ctx, s.pool, parentID)
		if perr != nil || pc.ContentID != id {
			parentID = 0 // geçersiz ebeveyn → en üst yorum gibi işlenir
		}
	}
	// 2) IP hız sınırı: aynı IP, saatlik en fazla 5 yorum (spam patlaması).
	const maxPerHour = 5
	sonSaat, _ := repo.CountCommentsSince(ctx, s.pool, ip, time.Now().Add(-time.Hour))
	if sonSaat >= maxPerHour {
		_ = repo.LogBotTrap(ctx, s.pool, ip, r.UserAgent(), "/yorum/rate", r.Referer())
		geri("error", "Çok fazla yorum gönderdiniz; lütfen biraz bekleyin.")
		return
	}

	if err := repo.AddComment(ctx, s.pool, id, name, body, ip, parentID); err != nil {
		geri("error", "Hata.")
		return
	}
	geri("info", "Yorumunuz onay sonrası yayınlanacak.")
}

func (s *Server) handleHit(w http.ResponseWriter, r *http.Request) {
	id := parseID(strings.Trim(strings.TrimPrefix(r.URL.Path, "/hits/"), "/"))
	if id == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	if r.Method == http.MethodPost {
		_ = repo.IncrementShare(r.Context(), s.pool, id)
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	hits, _ := repo.Hits(r.Context(), s.pool, id)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hits": hits})
}

// ---- feed / sitemap / robots ----

// googleSourceURL Google News'te siteyi "kaynak" ekleme bağlantısını üretir:
// https://www.google.com/preferences/source?q=<domain>
//
// Sorgu değerinde YALNIZCA domain olur — şema (http/https) ve yol yok.
// Google bu biçimi bekler. Host'da port varsa (yalnız geliştirme) korunur.
func googleSourceURL(base string) string {
	host := strings.TrimSpace(base)
	if u, err := url.Parse(host); err == nil && u.Host != "" {
		host = u.Host
	} else {
		host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
		if i := strings.IndexAny(host, "/?#"); i >= 0 {
			host = host[:i]
		}
	}
	return "https://www.google.com/preferences/source?q=" + host
}

// absBase mutlak site kökü (sitemap/JSON-LD için); istek başlığından düşer.
func (s *Server) absBase(r *http.Request) string {
	// Alan adı farkındalığı: canlıda bitcosar.com, önizlemede demo.
	// (Eskiden koşulsuz cfg.BaseURL dönüyordu; sitemap/robots canlıya
	// geçildiğinde demo adresini gösterecekti.)
	if b := s.kanonikKok(r); b != "" {
		return b
	}
	sch := "https"
	if r.TLS == nil {
		sch = "http"
	}
	return sch + "://" + r.Host
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	abs := s.absBase(r)
	site := s.siteByDomain(ctx, r.Host)
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">`)
	// ana sayfa: dil alternatifleri (tr varsayılan, x-default)
	sb.WriteString("<url><loc>" + abs + "/</loc>")
	for _, l := range []string{"tr", "en"} {
		sb.WriteString(`<xhtml:link rel="alternate" hreflang="` + l + `" href="` + abs + homeURL(l) + `"/>`)
	}
	sb.WriteString(`<xhtml:link rel="alternate" hreflang="x-default" href="` + abs + `/"/></url>`)
	// statik hizmet sayfaları
	for _, p := range []string{"/bulten", "/ping", "/yazarlar", "/arsiv", "/borsa", "/fikstur", "/astroloji", "/etkinlikler", "/anketler", "/gazete-mansetleri", "/sinemalar", "/resmi-ilanlar", "/akis", "/hava-durumu", "/namaz-vakitleri", "/imsakiye", "/nobetci-eczaneler", "/rss-listesi"} {
		sb.WriteString("<url><loc>" + abs + p + "</loc></url>")
	}
	cs, _ := repo.ListCategories(ctx, s.pool, site.ID)
	// Boş kategoriler sitemap'e girmez; üst kategori, alt kategorilerinden
	// içerik alıyorsa yine de listelenir.
	catTotal := map[int64]int{}
	for _, c := range cs {
		catTotal[c.ID] = c.ContentCount
	}
	for _, c := range cs {
		if c.ParentID != nil {
			catTotal[*c.ParentID] += c.ContentCount
		}
	}
	for _, c := range cs {
		if catTotal[c.ID] <= 0 {
			continue
		}
		sb.WriteString("<url><loc>" + abs + "/kategori/" + html.EscapeString(c.Slug) + "</loc></url>")
	}
	// Yazar/editör profilleri (Faz 51) ve CMS sayfaları.
	if people, err := repo.ListPeople(ctx, s.pool, nil); err == nil {
		for _, p := range people {
			if p.Slug == "" {
				continue
			}
			sb.WriteString("<url><loc>" + abs + "/yazar/" + html.EscapeString(p.Slug) + "</loc></url>")
		}
	}
	if pages, err := repo.ListPages(ctx, s.pool); err == nil {
		for _, p := range pages {
			if p.Slug == "" {
				continue
			}
			sb.WriteString("<url><loc>" + abs + "/" + html.EscapeString(p.Slug) + "</loc></url>")
		}
	}
	// Resmî ilanlar: her ilan kendi kalıcı adresinde yayımlanıyor. BİK
	// tebliği, içeriklere ait TAM URL adreslerinin ve son güncelleme
	// zamanlarının site haritasında bulunmasını istiyor.
	if notices, err := repo.NoticesForSitemap(ctx, s.pool, site.ID, 2000); err == nil {
		for _, n := range notices {
			loc := abs + publicNoticeURL("", n)
			sb.WriteString("<url><loc>" + htmlEscape(loc) + "</loc>")
			if n.UpdatedAt.Year() > 2000 {
				sb.WriteString("<lastmod>" + n.UpdatedAt.UTC().Format("2006-01-02") + "</lastmod>")
			}
			sb.WriteString("</url>")
		}
	}
	items, _ := repo.RecentPublished(ctx, s.pool, site.ID, locale, 500)
	refs, _ := repo.PublishedLangRefs(ctx, s.pool, site.ID)
	byRoot := sitemapLangRefs(refs)
	for _, it := range items {
		loc := abs + makaleURL(it)
		sb.WriteString("<url><loc>" + htmlEscape(loc) + "</loc>")
		if it.UpdatedAt.Year() > 2000 {
			sb.WriteString("<lastmod>" + it.UpdatedAt.UTC().Format("2006-01-02") + "</lastmod>")
		}
		grp := byRoot[it.TranslationRootID()]
		if len(grp) > 1 {
			for _, g := range grp {
				sb.WriteString(`<xhtml:link rel="alternate" hreflang="` + htmlEscape(g.Locale) +
					`" href="` + htmlEscape(abs+publicArticleURL(g.Locale, g.Slug, g.ID)) + `"/>`)
			}
			sb.WriteString(`<xhtml:link rel="alternate" hreflang="x-default" href="` + htmlEscape(loc) + `"/>`)
		}
		sb.WriteString("</url>")
	}
	sb.WriteString("</urlset>")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	w.Write([]byte(sb.String()))
}

// handleSitemapNews Google News sitemap: son 48 saatte yayınlanan haberler.
func (s *Server) handleSitemapNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	abs := s.absBase(r)
	var items []repo.Content
	items, _ = repo.RecentPublishedAllLang(ctx, s.pool, site.ID, 200)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	w.Write([]byte(s.newsSitemap(abs, items)))
}

// handleRobots /robots.txt — CoinDesk yapısı.
//
// KAYNAK (ölçüldü, 7 Kasım 2026): genel kurallar (API/dahili yollar kapalı,
// arama parametreleri kapalı), ardından AI tarayıcıları için "yalnız fiyat
// sayfaları" bloğu, en sonda sitemap listesi.
//
// NEDEN AI BLOĞU: CoinDesk içerik lisansı gereği AI tarayıcılarına yalnız
// fiyat sayfalarını açıyor. Toplayıcı site olarak aynı politikayı
// uyguluyoruz — atıf vermeyen model eğitimi trafiği sunucu maliyeti.
func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	abs := s.absBase(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)

	var sb strings.Builder
	sb.WriteString("User-agent: *\nAllow: /\n")
	for _, yol := range []string{
		"/api/", "/admin/", "/internal/", "/hits/", "/webhooks/",
		"/arama?q=", "/?q=", "/?s=", "/giris", "/kayit", "/admin/giris",
		"/arsiv/dizin", "/gizli-dizin", "/*/amp",
	} {
		sb.WriteString("Disallow: " + yol + "\n")
	}

	sb.WriteString("\n# AI tarayıcıları — yalnız fiyat sayfaları\n")
	for _, bot := range []string{
		"Amazonbot", "anthropic-ai", "Applebot-Extended", "Bytespider",
		"CCBot", "ClaudeBot", "Claude-Web", "cohere-ai", "DataForSeoBot",
		"Diffbot", "FacebookBot", "FriendlyCrawler", "Google-Extended",
		"GPTBot", "Meta-ExternalAgent", "meta-externalagent", "omgili",
		"omgilibot", "PerplexityBot", "Quora-Bot", "Scrapy", "TurnitinBot",
	} {
		sb.WriteString("\nUser-agent: " + bot + "\nAllow: /price/\nDisallow: /\n")
	}

	sb.WriteString("\n# Sitemap indeksleri\n")
	sb.WriteString("Sitemap: " + abs + "/sitemap-index.xml\n")
	sb.WriteString("Sitemap: " + abs + "/news-sitemap-index\n")
	sb.WriteString("Sitemap: " + abs + "/sitemap-news.xml\n")
	sb.WriteString("Sitemap: " + abs + "/sitemap.xml\n")
	_, _ = w.Write([]byte(sb.String()))
}

func (s *Server) handleAdsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	w.Write([]byte("# theblock ads.txt — reklam ağları eklenecek\n"))
}

// ---- medya dosyası servisi ----

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	// Private röportaj sesleri ayrı depolama kökünde tutulur ve bu uçtan
	// hiçbir koşulda servis edilmez. "audio/" öneki private ağaca aittir.
	if path == "" || strings.HasPrefix(path, "audio/") || strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(path, "thumb/") {
		id := parseID(strings.TrimPrefix(path, "thumb/"))
		s.serveMediaThumb(w, r, id)
		return
	}
	if strings.HasPrefix(path, "variant/") {
		parts := strings.Split(strings.TrimPrefix(path, "variant/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		id, errID := strconv.ParseInt(parts[0], 10, 64)
		width, errW := strconv.Atoi(parts[1])
		if errID != nil || errW != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
		s.serveMediaVariant(w, r, id, width)
		return
	}
	if path == "none.svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="675"><rect width="100%" height="100%" fill="#eef1f6"/></svg>`))
		return
	}
	// Kripto ikonları: gömülü CC0 kütüphaneden servis edilir. Dosya adı
	// tek bir yol parçası olmalı; cryptoicon.Bytes yalnız bilinen sembolü
	// döndürür, bilinmeyen ve "../" gibi girişler 404 olur.
	if strings.HasPrefix(path, "crypto/") {
		name := strings.TrimPrefix(path, "crypto/")
		name = strings.TrimSuffix(name, ".svg")
		if name == "" || strings.ContainsAny(name, "/\\") {
			http.NotFound(w, r)
			return
		}
		data, ok := cryptoicon.Bytes(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", cryptoicon.ContentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
		return
	}
	data, err := platform.ReadMedia(path)
	if err != nil {
		// kısa yol biçimi: /media/{yyyymm}/{dosya} → depolama "media/{yyyymm}/{dosya}"
		if data2, err2 := platform.ReadMedia("media/" + path); err2 == nil {
			data = data2
		} else {
			http.NotFound(w, r)
			return
		}
	}
	ct := mimeByExt(path)
	if strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".wav") {
		ct = platform.AudioContentType(path)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// serveMediaThumb medya önizlemesini döndürür; yoksa ilk istekte üretip kaydeder.
func (s *Server) serveMediaThumb(w http.ResponseWriter, r *http.Request, id int64) {
	if id <= 0 {
		http.NotFound(w, r)
		return
	}
	m, err := repo.GetMedia(r.Context(), s.pool, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if m.Visibility == "private" {
		http.NotFound(w, r)
		return
	}
	if m.Kind != "image" {
		// Ses/video/dosya için kırık görsel yerine tür ikonu.
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(mediaKindIcon(m.Kind))
		return
	}
	if m.ThumbPath != nil && *m.ThumbPath != "" {
		if data, err := platform.ReadMedia(*m.ThumbPath); err == nil {
			writeMediaBytes(w, data)
			return
		}
	}
	data, err := platform.ReadMedia(m.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	thumb, err := platform.CreateThumb(data, 480)
	if err != nil {
		w.Header().Set("Content-Type", mimeByExt(m.Path))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
		return
	}
	if p, err := platform.SaveMedia(thumb, "thumb.jpg"); err == nil {
		_ = repo.UpdateMediaThumb(r.Context(), s.pool, id, p)
	}
	writeMediaBytes(w, thumb)
}

// serveMediaVariant responsive görseli döndürür; yoksa ilk istekte üretir.
func (s *Server) serveMediaVariant(w http.ResponseWriter, r *http.Request, id int64, width int) {
	switch width {
	case 320, 480, 768, 960, 1200, 1600:
	default:
		http.NotFound(w, r)
		return
	}
	rel := platform.MediaVariantPath(id, width)
	if data, err := platform.ReadMedia(rel); err == nil {
		writeMediaBytes(w, data)
		return
	}
	m, err := repo.GetMedia(r.Context(), s.pool, id)
	if err != nil || m.Visibility == "private" {
		http.NotFound(w, r)
		return
	}
	srcPath := s.mediaPath(r.Context(), id)
	if srcPath == "" {
		http.NotFound(w, r)
		return
	}
	orig, err := platform.ReadMedia(srcPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := platform.ResizeToWidth(orig, width)
	if err != nil {
		w.Header().Set("Content-Type", mimeByExt(srcPath))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Length", strconv.Itoa(len(orig)))
		w.Write(orig)
		return
	}
	// Kaydetme başarısız olsa bile yanıtı döndür; sonraki istek yeniden üretir.
	_, _ = platform.SaveMediaVariant(id, width, data)
	writeMediaBytes(w, data)
}

func writeMediaBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// ---- yardımcılar ----

func (s *Server) defaultLang(ctx context.Context) string {
	if s.defaultLangCache != "" {
		return s.defaultLangCache
	}
	s.defaultLangCache = repo.DefaultLanguage(ctx, s.pool)
	return s.defaultLangCache
}

func (s *Server) langBase(locale string) string {
	if locale == "" || locale == "tr" {
		return ""
	}
	return "/" + locale
}

func homeURL(locale string) string {
	if locale == "" || locale == "tr" {
		return "/"
	}
	return "/" + locale + "/"
}

func langOf(r *http.Request, s *Server) string {
	lang := r.PathValue("lang")
	if lang == "" {
		lang = s.defaultLang(r.Context())
	}
	if !s.i18n.IsSupported(lang) {
		lang = "tr"
	}
	return lang
}

func publicArticleURL(locale, slug string, id int64) string {
	base := homeURL(locale)
	return base + slug + "-" + itoa(int(id))
}

// makaleURL CoinDesk URL deseniyle makale adresi üretir:
//
//	/{locale}/{section}/{YYYY}/{MM}/{DD}/{slug}
//
// NEDEN: CoinDesk deseni (spec §4) tarihi URL'de taşır — Google Haber'de
// tazelik sinyali ve kaynakla birebir uyum. Yayın tarihi yoksa (taslak,
// planlanmış) eski `slug-id` biçimine düşer; kırık bağlantı üretilmez.
//
// Bölüm (kategori) yoksa tarih köke yakın kalır: /{YYYY}/{MM}/{DD}/{slug}.
func makaleURL(c repo.Content) string {
	base := homeURL(c.Locale)
	yol := ""
	if c.CategorySlug != nil && *c.CategorySlug != "" {
		yol = *c.CategorySlug + "/"
	}
	t := c.PublishedAt
	if t == nil {
		t = &c.CreatedAt
	}
	if t == nil || t.IsZero() {
		return publicArticleURL(c.Locale, c.DisplaySlug(), c.ID)
	}
	return base + yol + t.Format("2006/01/02") + "/" + c.DisplaySlug()
}

// makaleIDMi /{section}/{y}/{m}/{d}/{slug} desenini çözer.
//
// (yil, ay, gun) değerleri takvimsel olarak geçerli olmalı; aksi hâlde
// desen sayılmaz (ör. /kategori/3/5 gibi başka bir yol olabilir).
func makaleTarihDeseni(segs []string) (slug string, ok bool) {
	if len(segs) < 4 {
		return "", false
	}
	n := len(segs)
	y, m, d, slug := segs[n-4], segs[n-3], segs[n-2], segs[n-1]
	if len(y) != 4 || len(m) != 2 || len(d) != 2 {
		return "", false
	}
	yi, err1 := strconv.Atoi(y)
	mi, err2 := strconv.Atoi(m)
	di, err3 := strconv.Atoi(d)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", false
	}
	if yi < 2000 || yi > 2100 || mi < 1 || mi > 12 || di < 1 || di > 31 {
		return "", false
	}
	if strings.TrimSpace(slug) == "" {
		return "", false
	}
	return slug, true
}

// hreflangLinks, yazı sayfasının <head> bölümüne basılacak alternatif dil
// etiketlerini mutlak adreslerle kurar. links aktif dilde URL taşımaz; kendi
// adresi selfURL'den gelir. Tek dilde hreflang anlamsız olduğu için nil döner.
func hreflangLinks(absBase, selfURL, defaultLocale string, links []langLink) []langLink {
	if len(links) < 2 {
		return nil
	}
	base := strings.TrimRight(absBase, "/")
	out := make([]langLink, 0, len(links)+1)
	def := ""
	for _, l := range links {
		if l.Code == "" {
			continue
		}
		u := l.URL
		if l.Active || u == "" {
			u = selfURL
		}
		if u == "" {
			continue
		}
		abs := base + u
		out = append(out, langLink{Code: l.Code, URL: abs})
		if l.Code == defaultLocale {
			def = abs
		}
	}
	if def == "" {
		def = base + selfURL
	}
	out = append(out, langLink{Code: "x-default", URL: def})
	return out
}

func catSlugPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Server) renderPublic(w http.ResponseWriter, r *http.Request, name string, d *tmplData, cc string) {
	w.Header().Set("Cache-Control", cc)
	// Sayfaya özel sütun şablonu: ana sayfa, yazı sayfası ve kategori
	// sayfalarında sağ sütun görünürlüğü/genişliği ayrı yönetilir.
	if d.PageLayout == nil {
		switch name {
		case "home", "article":
			d.PageLayout = s.pageLayoutFor(r.Context(), name)
		case "category":
			page := "kategori"
			if slug := r.PathValue("slug"); validCatSlug(slug) {
				page = "kategori:" + slug
			}
			d.PageLayout = s.pageLayoutFor(r.Context(), page)
		}
	}
	d.Theme = themeFromReq(r)
	// GA4 ölçüm kimliği yalnız public sayfalarda gerekir. Boşsa şablon
	// hiçbir üçüncü taraf izleme kodu basmaz (bkz. ga4.go).
	d.GA4Measurement = s.ga4Measurement(r.Context())
	// Search Console site doğrulama kodu (panelden girilir, boşsa basılmaz).
	d.GVerification = s.gVerification(r.Context())
	// AdSense yayıncı kimliği (panelden girilir, boşsa basılmaz).
	d.AdsPublisher = s.adsPublisher(r.Context())
	// Tüm public sayfalarda AYNI menu + footer (ficks-ectr dahil) garantisi
	if d.Header == nil {
		d.Header = s.headerConfig(r.Context())
	}
	// Header üye simgesi: oturum varsa avatar baş harfi, yoksa giriş ikonu.
	if !d.MemberLoggedIn {
		if u, err := s.currentUser(r); err == nil && u.ID > 0 {
			d.MemberLoggedIn = true
			d.MemberName = u.Name
			d.MemberInitial = memberInitial(u.Name)
			// Profil simgesi HER ZAMAN üyenin kendi alanına gider.
			//
			// ÖNCE personel rollerinde "/admin" yazıyordu: yönetici/editör
			// kendi avatarına basınca yönetim paneline düşüyordu (kullanıcı
			// bildirimi). Personel için yönetim kısayolu artık /hesabim
			// sayfasında ve çekmecede AÇIK bir bağlantı.
			d.MemberURL = "/hesabim"
			d.MemberUnread = s.uyeBildirimSayisi(r.Context(), u.ID)
			d.MemberStaff = auth.StaffRoles[u.Role]
		}
	}
	if d.Footer == nil {
		d.Footer = s.footerConfig(r.Context())
	}
	// Tek darboğaz: sayfa kendi alt bilgisini kursa da kurmasa da son söz
	// burada söylenir. Aksi hâlde /en/arama gibi sayfalarda alt bilgi
	// Türkçe kalıyordu (ölçüldü).
	d.Footer = s.footerYerellestir(d.Footer, d.Lang)
	if d.Menu == nil {
		d.Menu = s.publicMenuLocale(r.Context(), d.Lang)
	}
	if d.Brand == nil {
		d.Brand = s.brandConfig(r.Context())
	}
	// SEO varsayılanları: canonical (sayfa parametresi korunur) + marka OG görseli.
	if d.Canonical == "" {
		u := strings.TrimRight(s.absBase(r), "/") + r.URL.Path
		if sp := r.URL.Query().Get("s"); sp != "" {
			u += "?s=" + url.QueryEscape(sp)
		}
		d.Canonical = u
	}
	if d.OGImage != "" {
		// Sayfanın kendi kapak görseli (haber kapağı) geniş formatlıdır.
		d.OGImageGenis = true
	}
	if d.OGImage == "" && d.Brand != nil && d.Brand.OGImageID != nil {
		d.OGImage = strings.TrimRight(s.absBase(r), "/") + s.mediaURL(d.Brand.OGImageID)
		d.OGImageGenis = true
	}
	if d.OGImage == "" {
		// Sosyal kartlarda görselsiz kalmamak için kurumsal ikon yedeği.
		// 512x512 kare: geniş kart iddiasında bulunulmaz, aksi halde X/Facebook
		// görseli kırpıp ortasını gösterir. Panelden marka OG görseli (1200x630)
		// yüklenirse yukarıdaki dal devreye girer ve geniş karta dönülür.
		d.OGImage = strings.TrimRight(s.absBase(r), "/") + "/static/icon-512.png"
		d.OGImageGenis = false
	}
	if d.MetaDesc == "" {
		d.MetaDesc = d.SiteSlogan
	}
	// site arka planı (yönetici özel görsel yüklediyse)
	if d.SiteBg == "" {
		d.SiteBg = s.siteBgCSS(r.Context())
	}
	// ölçü/kenarlık token'ları (yönetici ayarladıysa)
	if d.DesignCSS == "" {
		d.DesignCSS = s.layoutTokensCSS(r.Context())
	}
	if t := s.themeTokensCSS(r.Context()); t != "" {
		d.DesignCSS += t
	}
	// aktif reklamları yerleşimlerine göre doldur (spec 01 §E)
	site := s.siteByDomain(r.Context(), r.Host)
	if site.ID > 0 {
		s.fillAdSlots(r.Context(), site.ID, d)
	}
	if err := s.tmpl.ExecuteTemplate(w, name, d); err != nil {
		s.logger.Error("şablon basılamadı", "template", name, "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
	}
}

// handleManifest PWA manifest'ini kök yoldan sunar (/manifest.webmanifest, /manifest.json).
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	staticFS, err := fs.Sub(webui.FS, "static")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(staticFS, "manifest.webmanifest")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Marka alanları sunucudaki ayarlardan gelir: ad değişince manifest
	// de değişir, iki yerde farklı isim kalması (PWA kurulumunda eski ad)
	// önlenir.
	var m map[string]any
	if json.Unmarshal(b, &m) == nil {
		m["name"] = s.siteName
		m["short_name"] = s.siteName
		m["start_url"] = "/"
		m["scope"] = "/"
		if nb, err := json.Marshal(m); err == nil {
			b = nb
		}
	}
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(b)
}

// handleFavicon /favicon.ico — kök ikon.
//
// NEDEN AYRI ROTA: tarayıcılar ve botlar `/favicon.ico`'yu head'de bağlantı
// olmasa da KÖKTEN ister. Rota yokken her sayfa yüklemesinde 404 üretiyordu
// (ölçüldü, 6 Kasım 2026 — sayfadaki tek 404 kaynağı buydu).
//
// ICO biçimi üretmiyoruz: PNG'yi `image/png` olarak sunuyoruz. Tarayıcılar
// favicon'da MIME'a bakar, uzantıya bakmaz; ayrı bir ikili üretmek
// gereksiz. Cache uzun: ikon içerik-adresli değil ama değişince `?v=`
// ile değil dosya değişimiyle gelir — bu yüzden 1 gün makul.
func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	staticFS, err := fs.Sub(webui.FS, "static")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(staticFS, "icon-192.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(b)
}

func themeFromReq(r *http.Request) string {
	if c, err := r.Cookie("tb_theme"); err == nil {
		if c.Value == "dark" || c.Value == "light" {
			return c.Value
		}
	}
	return "light"
}

func mimeByExt(path string) string {
	switch {
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".webp"):
		return "image/webp"
	case strings.HasSuffix(path, ".gif"):
		return "image/gif"
	case strings.HasSuffix(path, ".svg"):
		// Gömülü kripto ikonları ve none.svg. Tarayıcı türü uzantıdan
		// değil bu sabit değerden alır.
		return "image/svg+xml"
	default:
		if m := mime.TypeByExtension(extOf(path)); m != "" {
			return m
		}
		return "application/octet-stream"
	}
}

func extOf(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return strings.ToLower(path[i:])
	}
	return ""
}

// sayfaDuzeniVar bir sayfa anahtarı için panelden kaydedilmiş ÖZEL düzen
// olup olmadığını söyler.
//
// NEDEN: modüler şablonlar (page_layouts) panelde elle kurulmuş modüller
// taşıyor; yeni kabuk bunları basmıyor. Düzen VARSA eski motor korunur ki
// editörün işi sessizce yok olmasın. Düzen YOKSA yeni kabuk kullanılır.
func (s *Server) sayfaDuzeniVar(ctx context.Context, pageKey string) bool {
	b, err := repo.GetLayout(ctx, s.pool, 1, pageKey)
	if err != nil {
		return false
	}
	// GetLayout bulunamazsa "[]" döner: boş dizi = düzen yok.
	trim := strings.TrimSpace(string(b))
	return trim != "" && trim != "[]" && trim != "null"
}

// cdArsivSayfasi /arsiv — son yayınlanan içerikler, yeni kabukta.
//
// ESKİ ŞABLON: site_header + "cat_liste" ızgarası. Yeni kabukta aynı
// içerik `.cd-grid--3` kart ızgarasıyla basılır; üst çubuk, şerit ve alt
// bilgi ortak cd bileşenlerinden gelir.
func (s *Server) cdArsivSayfasi(w http.ResponseWriter, r *http.Request, locale string, siteID int64) {
	ctx := r.Context()
	items, _ := repo.RecentPublished(ctx, s.pool, siteID, locale, 100)

	simdi := time.Now()
	kartlar := make([]templkit.CDStory, 0, len(items))
	var ldItems []cdLDItem
	for i, c := range items {
		kartlar = append(kartlar, s.cdStoryCevir(c, simdi))
		if i < 11 {
			it := cdLDItem{Headline: c.Title, URL: s.absBase(r) + makaleURL(c), Author: c.AuthorName}
			if c.PublishedAt != nil {
				it.Published = c.PublishedAt.UTC().Format(time.RFC3339)
			}
			ldItems = append(ldItems, it)
		}
	}

	kanonik := s.kanonikKok(r) + "/arsiv"
	v := templkit.CDListePage{
		Shell:           s.cdShellCtx(ctx, r, "arsiv"),
		Title:           "Arşiv — " + s.siteName,
		Desc:            "Yayınlanan tüm haberler, en yeniden eskiye.",
		Canonical:       kanonik,
		Baslik:          "Arşiv",
		Aciklama:        "Son yayınlanan haberler, en yeniden eskiye.",
		Stories:         kartlar,
		BosMetin:        "Henüz yayınlanmış içerik yok.",
		JSONLD:          string(cdCollectionJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), "Arşiv", kanonik, ldItems)),
		PageCategory:    "section_landing_page",
		ContentCategory: "archive",
		Locale:          locale,
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDListePageView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd arşiv render", "error", err)
	}
}
