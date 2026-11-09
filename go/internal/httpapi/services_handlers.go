package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// servicesData servisler panel verisidir.
type servicesData struct {
	Cities  []repo.City
	Weather []WeatherS
	Kurlar  map[string]float64
	KurDemo bool
	Borsa   repo.BorsaData
	Fikstur repo.FiksturData
	Astro   repo.AstrolojiData
	// TheSportsDB API anahtarı (fikstür/puan kaynağı; boşsa test anahtarı)
	FiksturKey string
	// NosyAPI (RapidAPI) nöbetçi eczane anahtarı; boşsa kaynak kapalı.
	EczaneKey string
}

// WeatherS panel satırı (şehir + özet).
type WeatherS struct {
	City     string
	Slug     string
	Sicak    float64
	Aciklama string
	Demo     bool
}

// handleServicesGet servisler yönetimi.
func (s *Server) handleServicesGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildServicesData(r)
	if IsHTMX(r) {
		s.renderServislerPanelV2(w, r, d)
		return
	}
	s.renderServislerPageV2(w, r, d)
}

func (s *Server) buildServicesData(r *http.Request) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "Servisler (P3)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	sc := &servicesData{Kurlar: map[string]float64{}}
	cities, _ := repo.ListCities(ctx, s.pool)
	sc.Cities = cities
	rows, _ := repo.CacheList(ctx, s.pool, "hava")
	byKey := map[string]repo.CacheRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	for _, c := range cities {
		ws := WeatherS{City: c.Name, Slug: c.Slug}
		if rw, ok := byKey[strconv.Itoa(c.Plaka)]; ok {
			var w repo.WeatherInfo
			if json.Unmarshal(rw.Data, &w) == nil {
				ws.Sicak = w.Sicaklik
				ws.Aciklama = w.Aciklama
				ws.Demo = w.Demo
			}
		}
		sc.Weather = append(sc.Weather, ws)
	}
	if kb, err := repo.CacheGet(ctx, s.pool, "kur", "ana"); err == nil {
		var km map[string]any
		if json.Unmarshal(kb, &km) == nil {
			if f, ok := km["dolar"].(float64); ok {
				sc.Kurlar["dolar"] = f
			}
			if f, ok := km["euro"].(float64); ok {
				sc.Kurlar["euro"] = f
			}
			if f, ok := km["altin"].(float64); ok {
				sc.Kurlar["altin"] = f
			}
			sc.KurDemo, _ = km["demo"].(bool)
		}
	}
	sc.Borsa = repo.BorsaOf(ctx, s.pool)
	sc.Fikstur = repo.FiksturOf(ctx, s.pool)
	sc.Astro = repo.AstrolojiOf(ctx, s.pool)
	if b, err := repo.GetSetting(ctx, s.pool, "servis", "fikstur_key"); err == nil {
		sc.FiksturKey = strings.Trim(string(b), `"`)
	}
	if b, err := repo.GetSetting(ctx, s.pool, "servis", "eczane_api_key"); err == nil {
		sc.EczaneKey = strings.Trim(string(b), `"`)
	}
	d.Servis = sc
	return d
}

func (s *Server) renderServicesPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildServicesData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderServislerPanelV2(w, r, d)
}

func (s *Server) servicesResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderServicesPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/servisler", kind, text)
}

// handleServicesSettingsSave POST /admin/servisler/ayar — servis anahtarlarını kaydeder.
//
// Her alan BAĞIMSIZ kaydedilir: iki ayrı form aynı uca post ediyor; biri
// gönderilmediğinde diğerinin anahtarını silmemeli.
func (s *Server) handleServicesSettingsSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	kaydedilen := ""
	if r.Form.Has("fikstur_key") {
		k := strings.TrimSpace(r.FormValue("fikstur_key"))
		b, _ := json.Marshal(k)
		if err := repo.SetSetting(ctx, s.pool, "servis", "fikstur_key", b); err != nil {
			s.servicesResult(w, r, "error", "Kaydedilemedi.")
			return
		}
		kaydedilen = "TheSportsDB"
	}
	if r.Form.Has("eczane_api_key") {
		k := strings.TrimSpace(r.FormValue("eczane_api_key"))
		b, _ := json.Marshal(k)
		if err := repo.SetSetting(ctx, s.pool, "servis", "eczane_api_key", b); err != nil {
			s.servicesResult(w, r, "error", "Kaydedilemedi.")
			return
		}
		kaydedilen = "NosyAPI"
	}
	if kaydedilen == "" {
		s.servicesResult(w, r, "error", "Kaydedilecek alan yok.")
		return
	}
	s.audit(r, "servis_ayar", kaydedilen)
	s.servicesResult(w, r, "info", kaydedilen+" anahtarı kaydedildi. Bir sonraki yenilemede kullanılır; istersen 'Şimdi Yenile'ye bas.")
}

// handleServicesRefresh tüm servis verilerini arka planda yeniler (istek bloke olmaz).
func (s *Server) handleServicesRefresh(w http.ResponseWriter, r *http.Request) {
	go func() {
		ctx := context.Background()
		_, _ = repo.RefreshWeather(ctx, s.pool)
		_ = repo.RefreshForex(ctx, s.pool)
		_ = repo.RefreshBorsa(ctx, s.pool)
		_ = repo.RefreshFikstur(ctx, s.pool)
		_ = repo.RefreshStandings(ctx, s.pool)
		_ = repo.RefreshAstroloji(ctx, s.pool)
		// Eczane yalnız anahtar varsa yenilenir: anahtarsız yol 81 ili
		// eczaneler.org'da 20'şer saniye bekletir (kümede 403).
		if b, err := repo.GetSetting(ctx, s.pool, "servis", "eczane_api_key"); err == nil &&
			strings.Trim(string(b), `"`) != "" {
			_, _ = repo.RefreshEczane(ctx, s.pool)
		}
	}()
	s.audit(r, "servis_yenile", "async")
	s.servicesResult(w, r, "info", "Servis yenileme arka planda başlatıldı; birkaç saniye içinde güncellenir.")
}

// handleBorsaPage piyasa (BIST + kripto) sayfası.
func (s *Server) handleBorsaPage(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Piyasa & Borsa")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)
	d.Borsa = ptr(repo.BorsaOf(r.Context(), s.pool))
	s.renderPublic(w, r, "borsa", d, cacheControlPublic)
}

// handleHisselerPage "Tüm Hisseler" sayfası: arama + A-Z dizini + tablo.
//
// Referans: haberturk.com/ekonomi/borsa/tum-hisseler. Fiyatlar hisse
// önbelleğinden gelir (istek yolunda ağa çıkılmaz); veri yoksa satır yine
// basılır ama fiyat "—" olur — uydurma değer gösterilmez.
//
// TASARIM (9 Kasım 2026): sayfa `site_public.html` içindeki eski
// `{{define "hisseler"}}` bloğundan geliyordu; yani theblock'tan kalan
// kabuk (site-header/site-footer, `hsp-*`). Yeni temaya geçmiş her sayfa
// `cd-*` kabuğunu kullanırken bu sayfa geride kalmıştı. Artık
// `templkit.CDHisselerPageView` basılıyor: aynı veri, bitcosar kabuğu.
func (s *Server) handleHisselerPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	harf := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("harf")))

	sayfa := hisseSayfaOlustur(repo.HisselerOf(ctx, s.pool), q, harf)

	rows := make([]templkit.CDHisseSatir, 0, len(sayfa.Satir))
	for _, h := range sayfa.Satir {
		row := templkit.CDHisseSatir{
			Code: h.Code, Name: h.Name, Ilk: h.Ilk, Href: h.Href,
			Arama: h.Arama, Var: h.Var,
		}
		if h.Var {
			row.Fiyat = piyasaNum(h.Value, 2)
			// piyasaChg "%" İÇERMEZ ("+1,25"); yüzde işareti burada eklenir.
			// (İlk sürüm "%%" yazmıştı — Go'da bu bir kaçış dizisi değil,
			// ekranda "%%1,25" çıkıyordu; ölçümle yakalandı.)
			row.Yuzde = "%" + piyasaChg(h.Delta)
			row.Up = h.Delta > 0
			// Tam sıfır "yükseliş" değildir: yeşil rozet yanıltıcı olur.
			row.Duz = h.Delta == 0
		}
		rows = append(rows, row)
	}

	guncel := ""
	if !sayfa.Guncel.IsZero() {
		guncel = sayfa.Guncel.Format("02.01.2006 15:04")
	}

	v := templkit.CDHisselerPage{
		Shell:     s.cdShellCtx(ctx, r, "fiyatlar"),
		Title:     "Tüm Hisseler — BIST hisse fiyatları ve arama",
		Desc:      metaDesc("BIST'te işlem gören hisselerin kod, ad, fiyat ve günlük değişimi; hisse arama ve A-Z dizini.", "Tüm hisseler, canlı fiyatlar ve hisse arama — "+s.siteName, s.siteName),
		Canonical: strings.TrimRight(s.cfg.BaseURL, "/") + "/borsa/hisseler",
		Satir:     rows,
		Harfler:   sayfa.Harfler,
		Q:         sayfa.Q,
		Harf:      sayfa.Harf,
		Toplam:    sayfa.Toplam,
		Adet:      sayfa.Adet,
		Guncel:    guncel,
		HomeURL:   "/",
		// CoinDesk SEO sınıflandırması: fiyat dizini = katalog sayfası.
		PageCategory:    "market_data_landing_page",
		ContentCategory: "markets",
		Locale:          "tr",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
	}

	w.Header().Set("Cache-Control", cacheControlPublic)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templkit.CDHisselerPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("hisseler sayfası render", "error", err)
	}
}

// handleKriptoPage /borsa/kripto: arama + görünüm seçici + liste.
//
// Referans: haberturk.com/ekonomi/kripto-para (Kripto Para Ara + tablo).
// Görünüm `?stil=tablo|kart|kompakt` ile seçilir; arama `?q=` (sunucu) ve
// istemci tarafı canlı filtre ile çalışır.
func (s *Server) handleKriptoPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	q := r.URL.Query().Get("q")
	stil := strings.TrimSpace(r.URL.Query().Get("stil"))

	d := s.baseView(locale, s.langBase(locale), "Kripto Para")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.MetaTitle = "Kripto Para — canlı fiyatlar, 24s/7g değişim ve hacim"
	d.MetaDesc = metaDesc("Bitcoin, Ethereum ve diğer kripto paraların TL fiyatı, 24 saat ve 7 günlük değişimi, hacim ve piyasa değeri; kripto para arama.",
		"Kripto para fiyatları ve arama — "+s.siteName, s.siteName)
	d.Robots = "index,follow"
	kd := repo.KriptoOf(ctx, s.pool)
	d.Kripto = &kd
	d.KriptoSayfa = kriptoSayfaOlustur(kd, q, stil)
	s.renderPublic(w, r, "kripto", d, cacheControlPublic)
}

// handleAPIBorsa piyasa özeti (JSON).
func (s *Server) handleAPIBorsa(w http.ResponseWriter, r *http.Request) {
	bd := repo.BorsaOf(r.Context(), s.pool)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "borsa": bd})
}

// handleFiksturPage puan durumu + hafta bazlı fikstür sayfası (mynet tarzı).
func (s *Server) handleFiksturPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Puan Durumu & Fikstür")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Theme = themeFromReq(r)

	lg := repo.LeagueByKey(strings.TrimSpace(r.URL.Query().Get("lig")))
	d.Leagues = repo.Leagues
	d.CurLeague = lg
	d.Standings = repo.StandingsOfLeague(ctx, s.pool, lg).Rows

	meta := repo.LeagueMetaOf(ctx, s.pool, lg)
	d.FxMeta = meta
	round := parseFiksturRound(r.URL.Query().Get("hafta"), meta)
	d.FxRound = repo.FiksturRoundOf(ctx, s.pool, lg, meta.Season, round)
	for i := 1; i <= meta.Rounds; i++ {
		d.FxRounds = append(d.FxRounds, i)
	}
	s.renderPublic(w, r, "fikstur", d, cacheControlPublic)
}

// parseFiksturRound istekteki "hafta" parametresini doğrular.
//
// Kural: yalnızca 1..meta.Rounds aralığında sayı kabul edilir; boş, sayı
// olmayan, 0, negatif veya sezon dışı değerler meta.Round'a (güncel hafta)
// düşer. Böylece "?hafta=abc" veya "?hafta=999" ile boş hafta açılmaz.
func parseFiksturRound(q string, meta repo.FiksturMeta) int {
	q = strings.TrimSpace(q)
	if q == "" {
		return meta.Round
	}
	n, err := strconv.Atoi(q)
	if err != nil || n < 1 || n > meta.Rounds {
		return meta.Round
	}
	return n
}

// handleAPIFikstur fikstür özeti (JSON).
func (s *Server) handleAPIFikstur(w http.ResponseWriter, r *http.Request) {
	fd := repo.FiksturOf(r.Context(), s.pool)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "fikstur": fd})
}

// handleAstrolojiPage günlük burç yorumları sayfası.
func (s *Server) handleAstrolojiPage(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Astroloji")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)
	d.Astro = ptr(repo.AstrolojiOf(r.Context(), s.pool))
	s.renderPublic(w, r, "astroloji", d, cacheControlPublic)
}

// handleAPIAstroloji günlük burç yorumları (JSON).
func (s *Server) handleAPIAstroloji(w http.ResponseWriter, r *http.Request) {
	ad := repo.AstrolojiOf(r.Context(), s.pool)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "astroloji": ad})
}

// handleHavaDurumu genel hava durumu sayfası.
func (s *Server) handleHavaDurumu(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Hava Durumu")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)

	cities, _ := repo.ListCities(r.Context(), s.pool)
	rows, _ := repo.CacheList(r.Context(), s.pool, "hava")
	byKey := map[string]repo.CacheRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	var out []WeatherS
	for _, c := range cities {
		ws := WeatherS{City: c.Name, Slug: c.Slug}
		if rw, ok := byKey[strconv.Itoa(c.Plaka)]; ok {
			var w repo.WeatherInfo
			if json.Unmarshal(rw.Data, &w) == nil {
				ws.Sicak = w.Sicaklik
				ws.Aciklama = w.Aciklama
			}
		}
		out = append(out, ws)
	}
	d.Hava = out
	s.renderPublic(w, r, "hava", d, cacheControlPublic)
}

// NamazS panel/genel satır (şehir + vakitler).
type NamazS struct {
	City string
	Slug string
	repo.NamazVakit
}

// handleNamazDurumu genel "Namaz Vakitleri" sayfası.
func (s *Server) handleNamazDurumu(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Namaz Vakitleri")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)

	cities, _ := repo.ListCities(r.Context(), s.pool)
	rows, _ := repo.CacheList(r.Context(), s.pool, "namaz")
	byKey := map[string]repo.CacheRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	var out []NamazS
	for _, c := range cities {
		ns := NamazS{City: c.Name, Slug: c.Slug}
		if rw, ok := byKey[strconv.Itoa(c.Plaka)]; ok {
			_ = json.Unmarshal(rw.Data, &ns.NamazVakit)
		}
		out = append(out, ns)
	}
	d.NamazList = out
	s.renderPublic(w, r, "namaz", d, cacheControlPublic)
}

// handleImsakiye genel aylık imsakiye sayfası (varsayılan Ankara).
func (s *Server) handleImsakiye(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "İmsakiye")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)

	var list []repo.NamazVakit
	if b, err := repo.CacheGet(r.Context(), s.pool, "imsakiye", "6"); err == nil {
		_ = json.Unmarshal(b, &list)
	}
	d.Imsakiye = list
	s.renderPublic(w, r, "imsakiye", d, cacheControlPublic)
}

// handleAPINamaz şehir bazında bugünün vakitleri (JSON).
func (s *Server) handleAPINamaz(w http.ResponseWriter, r *http.Request) {
	c, err := repo.GetCitySlug(r.Context(), s.pool, r.PathValue("slug"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	var n repo.NamazVakit
	if b, err := repo.CacheGet(r.Context(), s.pool, "namaz", strconv.Itoa(c.Plaka)); err == nil {
		_ = json.Unmarshal(b, &n)
	}
	n.Il = c.Name
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "city": c.Name, "slug": c.Slug, "namaz": n})
}

// handleEczanePage nöbetçi eczaneler sayfası.
func (s *Server) handleEczanePage(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Nöbetçi Eczaneler")
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)

	cities, _ := repo.ListCities(r.Context(), s.pool)
	sel := r.URL.Query().Get("il")
	if sel == "" {
		sel = "6"
	}
	if es, err := repo.EczaneOf(r.Context(), s.pool, sel); err == nil {
		es.Plaka = int(parseID(sel))
		if es.Plaka == 0 {
			es.Plaka = 6
		}
		d.Eczane = &es
	}
	if d.Eczane == nil {
		d.Eczane = &repo.EczaneSehir{Plaka: 6, Sehir: "Ankara"}
	}
	d.Cities = cities
	s.renderPublic(w, r, "eczane", d, cacheControlPublic)
}

// handleAPIEczane şehir bazında nöbetçi eczane JSON.
func (s *Server) handleAPIEczane(w http.ResponseWriter, r *http.Request) {
	c, err := repo.GetCitySlug(r.Context(), s.pool, r.PathValue("slug"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	es, _ := repo.EczaneOf(r.Context(), s.pool, strconv.Itoa(c.Plaka))
	es.Sehir = c.Name
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "city": c.Name, "slug": c.Slug, "eczane": es})
}

// handleAMP makalenin AMP sürümünü yayınlar.
func (s *Server) handleAMP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seg := strings.Trim(r.PathValue("rest"), "/")
	id := articleIDFromSegment(seg)
	slug := articleSlugFromSegment(seg)
	if id == 0 {
		http.NotFound(w, r)
		return
	}
	c, err := repo.ContentBySlugAndID(ctx, s.pool, slug, id)
	if err != nil || c.Status != "yayinda" {
		http.NotFound(w, r)
		return
	}
	d := s.baseView(c.Locale, s.langBase(c.Locale), c.Title)
	d.HomeURL = homeURL(c.Locale)
	d.Theme = themeFromReq(r)
	d.Article = &articleData{Content: c, HTML: s.blocksToHTML(c.Detail, nil)}
	d.AMPCanonical = strings.TrimRight(s.cfg.BaseURL, "/") + publicArticleURL(c.Locale, c.DisplaySlug(), c.ID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := s.tmpl.ExecuteTemplate(w, "amp", d); err != nil {
		s.logger.Error("amp şablon", "error", err)
		http.Error(w, "hata", 500)
	}
}

// handleSearch genel site arama sayfası.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	locale := langOf(r, s)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	d := s.baseView(locale, s.langBase(locale), "Arama")
	// Arama sonuçları ince/tekrarlı içerik: dizine girmemeli.
	d.Robots = "noindex,follow"
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(r.Context())
	d.Theme = themeFromReq(r)
	site := s.siteByDomain(r.Context(), r.Host)
	if q != "" {
		repo.LogSearchQuery(r.Context(), s.pool, site.ID, q)
		items, _ := repo.SearchPublic(r.Context(), s.pool, site.ID, locale, q, 30)
		// Arama motorunun adı KULLANICIYA GÖSTERİLMEZ: hangi veritabanı/
		// indeks motorunun çalıştığı altyapı bilgisidir, ziyaretçiye faydası
		// yok, saldırgana ise sürüm/CVE hedeflemesi için ipucu verir.
		// Tantivy search-svc varsa ondan gelen sıralamayı tercih et
		// (Türkçe tokenizasyon).
		if s.search != nil {
			if hits, err := s.search.Search(r.Context(), q, 30); err == nil && len(hits) > 0 {
				if ordered, e := s.contentsBySearchIDs(r.Context(), site.ID, hits); e == nil && len(ordered) > 0 {
					items = ordered
				}
			}
		}
		d.Arama = items
		d.Category = &categoryData{Title: "Arama" + ": " + q, Items: items, View: s.categoryListView(r.Context()), Page: 1, PerPage: 30}
		s.fillVideoDurations(r.Context(), d.Category.Items)
	}
	d.Query = q
	s.renderPublic(w, r, "arama", d, cacheControlPublic)
}

// gerekli kokuyu giderme
func (s *Server) handleAPIHava(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	c, err := repo.GetCitySlug(r.Context(), s.pool, slug)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	var wi repo.WeatherInfo
	if b, err := repo.CacheGet(r.Context(), s.pool, "hava", strconv.Itoa(c.Plaka)); err == nil {
		_ = json.Unmarshal(b, &wi)
	}
	wi.City = c.Name
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "city": c.Name, "slug": c.Slug, "sicaklik": wi.Sicaklik, "aciklama": wi.Aciklama, "demo": wi.Demo, "lat": c.Lat, "lon": c.Lon})
}
