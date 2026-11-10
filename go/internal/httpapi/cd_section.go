package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"theblock/site/internal/auth"
	"theblock/site/internal/i18n"
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
	"markets":            {"Piyasalar", "Kripto piyasaları: fiyat hareketleri, likidite, türevler ve kurumsal akış."},
	"business":           {"Finans", "Borsalar, kripto şirketleri, bankalar, ödeme ve kurumsal benimseme."},
	"tech":               {"Teknoloji", "Protokoller, ölçekleme, güvenlik açıkları ve altyapı gelişmeleri."},
	"policy":             {"Politika", "Düzenleme, yasa teklifleri, mahkemeler ve kamu kurumları."},
	"opinion":            {"Görüş", "Sektörden görüşler ve analizler."},
	"news-analysis":      {"Haber Analizi", "Derinlemesine haber analizleri."},
	"latest-crypto-news": {"Son Haberler", "En son kripto haberleri, dakika dakika."},
	"focus":              {"Odak", "Odak dosyaları ve uzun okumalar."},
	"research":           {"Araştırma", "Araştırma raporları ve veri analizleri."},
	"press-release":      {"Basın Bültenleri", "Basın bültenleri."},
	"learn":              {"Kripto 101", "Kripto para ve blokzincir temelleri, rehberler ve açıklayıcılar."},
}

// cdBolumHandler tek bir bölümün handler'ını üretir.
func (s *Server) cdBolumHandler(slug string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleCDSection(w, r, slug)
	}
}

// cdBolumMeta bölüm başlığı ve açıklaması.
//
// İKİ KAYNAK: önce sabit harita (CoinDesk bölümleri), sonra DB
// kategorileri (`/kategori/{slug}` adresleri buradan gelir).
//
// NEDEN: `/kategori/{slug}` sayfası theblock şablonunu kullanıyordu —
// menüden tıklanan kategori eski kabuğa düşüyordu (kullanıcı bildirimi,
// 9 Kasım 2026). Aynı içerik iki ayrı şablonla iki farklı görünümde
// basılıyordu. Artık tek şablon: `/tech` ve `/kategori/tech` birebir aynı.
func (s *Server) cdBolumMeta(ctx context.Context, slug string) (ad, aciklama string, ok bool) {
	if m, var_ := cdSectionMeta[slug]; var_ {
		return m.Ad, m.Aciklama, true
	}
	// DB kategorisi (panelden eklenen bölümler).
	var name string
	var status string
	err := s.pool.QueryRow(ctx,
		`SELECT name, status FROM categories WHERE slug=$1 AND site_id=1 LIMIT 1`, slug).Scan(&name, &status)
	if err != nil || status != "yayinda" {
		return "", "", false
	}
	return name, name + " kategorisindeki son haberler, analizler ve gelişmeler.", true
}

// handleCDSection bölüm sayfası.
func (s *Server) handleCDSection(w http.ResponseWriter, r *http.Request, slug string) {
	ad, aciklama, ok := s.cdBolumMeta(r.Context(), slug)
	if !ok {
		s.handleNotFound(w, r)
		return
	}
	meta := struct{ Ad, Aciklama string }{ad, aciklama}

	sayfa := 1
	// `?sayfa=` kanonik; `?s=` eski `/kategori/{slug}` adreslerinden gelen
	// bağlantılar için korunuyor (onlar hâlâ dolaşımda — 301 üretmedik).
	for _, anahtar := range []string{"sayfa", "s"} {
		if v := r.URL.Query().Get(anahtar); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
				sayfa = n
				break
			}
		}
	}
	const perPage = 24

	// Liste önbellekten (10 sn). Hata durumunda boş liste + 0 toplam döner;
	// sayfa iskeleti yine basılır (eskiden de öyleydi).
	items, toplam := s.cdListeIcerikleri(r.Context(), slug, sayfa)

	simdi := time.Now()
	var kartlar []templkit.CDStory
	var ldItems []cdLDItem
	for i, c := range items {
		kartlar = append(kartlar, cdStoryCevir(c, simdi))
		if i < 11 {
			// CoinDesk CollectionPage ItemList: 11 en yeni haber.
			it := cdLDItem{Headline: c.Title, URL: s.absBase(r) + makaleURL(c), Author: c.AuthorName}
			if c.PublishedAt != nil {
				it.Published = c.PublishedAt.UTC().Format(time.RFC3339)
			}
			ldItems = append(ldItems, it)
		}
	}

	kanonik := s.kanonikKok(r) + "/" + slug

	// PARÇA (`?sayfa=N&parca=1`): cd-daha.js "Daha fazla haber"a basınca
	// bu yanıtı çeker ve satırları listenin sonuna ekler — sayfa geçişi
	// OLMAZ (CoinDesk davranışı: URL değişmez, sayfa yenilenmez).
	//
	// Parçada o sayfanın TÜM haberleri DÜZ satır olarak gider: lead/alt/
	// ray bölmesi yalnız tam sayfada (JS'siz fallback) uygulanır. Nav da
	// parçada: JS yeni href'lerle yeniler.
	if r.URL.Query().Get("parca") == "1" {
		var prev, next string
		if sayfa > 1 {
			prev = "/" + slug + sayfaURL(sayfa-1)
		}
		if toplam > sayfa*perPage {
			next = "/" + slug + sayfaURL(sayfa+1)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheControlPublic)
		if err := templkit.CDBolumParca(kartlar, prev, next).Render(r.Context(), w); err != nil {
			s.logger.Error("cd bölüm parça render", "error", err)
		}
		return
	}

	v := templkit.CDSectionPage{
		Shell:     s.cdShellCtx(r.Context(), r, slug),
		Title:     meta.Ad + " — " + s.siteName,
		Desc:      meta.Aciklama,
		Canonical: kanonik,
		Section:   meta.Ad,
		Href:      "/" + slug,
		Page:      sayfa,
		// CoinDesk bölüm sayfası: CollectionPage + mainEntity.ItemList.
		JSONLD:          string(cdCollectionJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), meta.Ad, kanonik, ldItems)),
		PageCategory:    "section_landing_page",
		ContentCategory: slug,
		Locale:          "tr",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
	}
	// Piyasa Nabzı yalnız /markets ilk sayfasında: gerçek CryptoQuant verisi.
	if slug == "markets" && sayfa == 1 {
		v.NabizFiyat, v.Nabiz = s.cdNabiz(r.Context())
	}

	// YERLEŞİM (CoinDesk /markets ölçümü, 8 Kasım 2026):
	//   lead      : 1 haber (yatay kart, metin %31 / görsel %67)
	//   alt       : sonraki 4 haber (239 px kart ızgarası)
	//   ray       : sonraki 6 haber (görselsiz başlık + saat)
	//   stories   : kalanlar ("More From" satırları, 1032 px)
	//
	// Önceki sürüm 1 lead + 3 yan + kalan dikey kart kullanıyordu; CoinDesk'te
	// bu sayfada yan sütun ve dikey kart YOK (ölçüldü).
	const altAdet, rayAdet = 4, 6
	if len(kartlar) > 0 {
		v.Lead = kartlar[0]
		kalan := kartlar[1:]
		if len(kalan) > altAdet {
			v.Alt = kalan[:altAdet]
			kalan = kalan[altAdet:]
		} else {
			v.Alt = kalan
			kalan = nil
		}
		if len(kalan) > rayAdet {
			v.Ray = kalan[:rayAdet]
			kalan = kalan[rayAdet:]
		} else {
			v.Ray = kalan
			kalan = nil
		}
		v.Stories = kalan
		v.MoreBaslik = meta.Ad + " bölümünden daha fazlası"
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
// kategoriID bölüm slug'ını kimliğe çevirir (10 sn önbellekli).
//
// NEDEN ÖNBELLEK: ana sayfa render'ı bölüm panelleri için AYNI dört slug'ı
// çeviriyor (4 sorgu/istek, ölçüldü 10 Kasım 2026). Kategori kimliği
// pratikte hiç değişmez; 10 sn'lik süreç içi katman yeter.
func (s *Server) kategoriID(ctx context.Context, slug string) int64 {
	if s.kategoriCache == nil {
		var id int64
		_ = s.pool.QueryRow(ctx,
			`SELECT id FROM categories WHERE slug=$1 LIMIT 1`, slug).Scan(&id)
		return id
	}
	v, ok := s.kategoriCache.Load(ctx, slug, func(ctx context.Context) (int64, bool) {
		var id int64
		if err := s.pool.QueryRow(ctx,
			`SELECT id FROM categories WHERE slug=$1 LIMIT 1`, slug).Scan(&id); err != nil {
			// "Yok" GEÇERLİ sonuçtur (0) ve önbelleğe yazılır: olmayan bir
			// slug her istekte yeniden sorgulanmasın. DB hatası yazılmaz.
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, true
			}
			return 0, false
		}
		return id, true
	})
	if !ok {
		return 0
	}
	return v
}

// cdShell ortak kabuk verisini kurar (nav + arama + dil + mega menü verisi).
//
// Mega menünün etiket ve fiyat sütunları da BURADA doldurulur: menü her
// sayfada aynı görünmeli. Ayrı bir "zengin" varyant tutmak, ana sayfada
// menünün eksik kalmasına yol açtı (ölçüm: etiket sütunu yalnız liste
// sayfalarında vardı).
//
// Maliyet: iki hafif sorgu — fiyat 60 sn önbellekli (repo.Fiyatlar), etiket
// sorgusu indeksli GROUP BY. Sayfa başına <1 ms; karşılığında menü tutarlı.
func (s *Server) cdShell(aktif string) templkit.CDShell {
	return s.cdShellCtx(context.Background(), nil, aktif)
}

// cdShellOturum oturum bilgisini kabuğa işler (üst çubuk menüsü).
//
// Giriş bağlantıları TEMİZLENİR: şablon `LoggedIn` dalına geçiyor ama
// başka bir tüketici (mobil panel) `LoginHref`'e bakarsa ölü bir "Giriş"
// bağlantısı basmasın.
func cdShellOturum(sh *templkit.CDShell, u auth.User) {
	sh.LoggedIn = true
	// Ad KIRPILIR: kenar boşluklu ad menüde ve aria-label'da aynen
	// görünürdü (memberInitial da kırpıyor — iki yerde tutarlı).
	sh.UserName = strings.TrimSpace(u.Name)
	sh.UserInitial = memberInitial(u.Name)
	sh.UserHref = "/hesabim"
	sh.LoginHref = ""
	sh.SignupHref = ""
}

// cdShellCtx istek bağlamıyla çalışan sürüm (iptal edilebilir).
func (s *Server) cdShellCtx(ctx context.Context, r *http.Request, aktif string) templkit.CDShell {
	nav := []templkit.CDNavLink{
		{Label: "Son Haberler", Href: "/latest-crypto-news"},
		{Label: "Piyasalar", Href: "/markets"},
		{Label: "Finans", Href: "/business"},
		{Label: "Teknoloji", Href: "/tech"},
		{Label: "Politika", Href: "/policy"},
		{Label: "Görüş", Href: "/opinion"},
		{Label: "Kripto 101", Href: "/learn"},
		{Label: "Fiyatlar", Href: "/price"},
		{Label: "Video", Href: "/videos"},
	}
	for i := range nav {
		if strings.HasSuffix(nav[i].Href, "/"+aktif) {
			nav[i].Active = true
		}
	}
	sh := templkit.CDShell{
		SiteName:   s.siteName,
		SearchHref: "/arama",
		Locale:     "tr",
		Nav:        nav,
		LoginHref:  "/giris",
		SignupHref: "/kayit",
		// Dil seçici: yalnız GERÇEK diller listelenir. Şu an içerik Türkçe;
		// EN içerik hattı gelince "en" de eklenir (çevirisi olmayan dil
		// kullanıcıyı boş sayfaya götürür).
		Diller: []templkit.CDDil{{Kod: "tr", Ad: "Türkçe", Aktif: true}},
	}

	// ÜST ÇUBUK OTURUMU: giriş yapılmışken Oturum Aç/Kaydol yerine
	// kullanıcı simgesi + hesap menüsü basılır (bkz. cdHesapMenu).
	// Çerez YOKSA DB'ye gitmez — anonim ziyaretçi ek tur ödemez.
	if r != nil {
		if u, err := s.currentUser(r); err == nil && u.ID > 0 {
			cdShellOturum(&sh, u)
			sh.UserUnread = s.uyeBildirimSayisi(ctx, u.ID)
		}
	}

	// Kabuk verisi: etiketler + menü coinleri + bölüm doluluğu.
	// Üçü de istekten bağımsız ve her sayfada aynı → 10 sn önbellek +
	// singleflight (bkz. cd_cache.go). Tek tek sorgulamak istek başına
	// 3 tur ekliyordu.
	kabuk := s.cdShellVeri(ctx)

	// Etiket sütunu: yalnız İÇERİĞİ OLAN etiketler. Boş etiket sayfasına
	// bağlantı vermek 404'e götürür (ölçüm: /tag/xrp 404 idi).
	for _, e := range kabuk.Tags {
		ad := etiketMenuAdi(e.Ad)
		// Sütun başlığı zaten "Crypto News": adı yalnız "News"/"Haber"
		// olan genel etiket satırı bilgi katmıyor (ölçüm: menüde
		// "Crypto News → News").
		if ad == "" || strings.EqualFold(ad, "News") || strings.EqualFold(ad, "Haber") {
			continue
		}
		sh.MenuTags = append(sh.MenuTags, templkit.CDMenuItem{
			Label: ad,
			Href:  "/tag/" + e.Slug,
		})
		if len(sh.MenuTags) >= 5 {
			break
		}
	}

	// Fiyat sütunu: canlı ilk 5 coin.
	for i, f := range kabuk.Coins {
		if i >= 5 {
			break
		}
		sh.MenuCoins = append(sh.MenuCoins, templkit.CDMenuItem{
			Label: f.Name + " (" + f.Sym + ")",
			Href:  "/price/" + repo.CoinSlug(f.Name),
		})
	}

	// Bölüm doluluğu: menüye yalnız YAYINDA İÇERİĞİ OLAN bölüm yazılır.
	// Ölçüm (6 Kasım 2026): Focus · Videos · Podcasts · News Analysis ·
	// Press Releases bölümlerinde 0 içerik vardı; menü beş ölü bağlantı
	// veriyordu. Sorgu başarısızsa harita boş kalır ve filtre uygulanmaz.
	if kabuk.DoluOk {
		dolu := kabuk.Dolu
		sh.MenuBolumler = dolu
		// Üst gezinme de aynı kurala uyar: "Video" yayında videosu yokken
		// basılmaz. /latest-crypto-news ve /price gibi liste sayfaları
		// bölüm sayılmadığı için etkilenmez.
		nav2 := make([]templkit.CDNavLink, 0, len(sh.Nav))
		for _, n := range sh.Nav {
			if slug := strings.Trim(strings.TrimSpace(n.Href), "/"); slug != "" &&
				!strings.Contains(slug, "/") && !dolu[slug] {
				continue
			}
			nav2 = append(nav2, n)
		}
		sh.Nav = nav2
	}
	// PİYASA ŞERİDİ: tüm CD sayfalarında aynı veri. Tek yerden doldurulur
	// (kullanıcı bildirimi, 8 Kasım 2026: "diğer sayfalarda kur alanı yok").
	// Fiyatlar 60 sn önbellekli; sayfa başına maliyet bir DB okuması.
	//
	// SEKME: `?serit=stocks` istek parametresi okunur. Önce şablon sekmeyi
	// çiziyordu ama veri her zaman kripto kalıyordu — "Hisse Senetleri"
	// sekmesi ölü bağlantıydı.
	sekme := "markets"
	if r != nil {
		if strings.TrimSpace(r.URL.Query().Get("serit")) == "stocks" {
			sekme = "stocks"
		}
	}
	sh.TickerTab = sekme
	if sekme == "stocks" {
		sh.TickerLive = "BIST"
	} else {
		sh.TickerLive = "Canlı Piyasalar"
	}
	sh.Ticker = s.cdTickerDoldurSekme(ctx, sekme)

	return sh
}

// etiketMenuAdi menüdeki etiket satırının adını üretir.
//
// CoinDesk deseni "Bitcoin News", "Ethereum News" biçiminde. Ancak kaynaktan
// gelen etiket zaten "News" ise son ek eklemek "News News" üretiyordu
// (ölçüm: 5 Kasım 2026 menüde görüldü).
func etiketMenuAdi(ad string) string {
	ad = strings.TrimSpace(ad)
	if ad == "" {
		return ""
	}
	// "News", "news", "Haber" ile bitiyorsa dokunma.
	low := strings.ToLower(ad)
	for _, sonek := range []string{"news", "haber"} {
		if strings.HasSuffix(strings.TrimSpace(low), sonek) {
			return ad
		}
	}
	return ad + " News"
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
		st.Kicker = i18n.BuyukHarf(*c.CategoryName)
	}
	return st
}
