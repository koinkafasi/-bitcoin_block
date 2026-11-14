package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdIlgiliHaberLimiti makale altındaki "İlgili Haberler" kart sayısı.
//
// 3 DEĞİL 6 (13 Kasım 2026): üç kademeli alaka motoru (SimilarTo →
// RelatedContents → PublishedBySlug) zaten 6 aday getiriyordu ve yarısı
// atılıyordu.
//
// NEDEN ÖNEMLİ: ölçümde yayındaki 3836 içeriğin yalnız 190'ında (%5)
// gövde içi iç bağlantı var; canlı makale sayfasında içerikten içeriğe
// giden bağlantı 2 taneydi, gerisi menü ve altbilgi. İç bağlantı site
// içi otoritenin tek taşıyıcısı ve 2130 tek-içerikli etiket sayfasının
// ölü kalmasının da sebebi. 3 yerine 6 ile içerik başına ek 3 bağlantı,
// toplamda ~11.500 yeni iç bağlantı demek.
//
// Düzen: cd-grid--3 ile iki satır olur (haber sitelerinde olağan).
const cdIlgiliHaberLimiti = 6

// cdAyniDil iki yerel kodun aynı dili gösterip göstermediğini söyler.
//
// Boş locale "tr" sayılır: eski kayıtlarda alan boş kalabiliyor ve bu
// içerikler Türkçe. Boşu "farklı dil" saymak onları ilgili listesinden
// tümüyle düşürürdü.
func cdAyniDil(a, b string) bool {
	norm := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return "tr"
		}
		// "tr-TR" ve "tr" aynı dil.
		if i := strings.IndexAny(s, "-_"); i > 0 {
			s = s[:i]
		}
		return s
	}
	return norm(a) == norm(b)
}

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

	// Okunma sayımı ve çizim ortak fonksiyonda (önizleme de aynı şablonu
	// kullanır; orada sayım YAPILMAZ).
	s.cdMakaleGoster(w, r, c, false)
}

func (s *Server) handleCDArticleOnizleme(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	s.cdMakaleGoster(w, r, c, true)
}

// cdMakaleGoster makale sayfasını basar.
func (s *Server) cdMakaleGoster(w http.ResponseWriter, r *http.Request, c repo.Content, onizleme bool) {
	dil := s.cdYerel(r)
	ctx := r.Context()
	if !onizleme {
		s.okunmaKaydet(ctx, c.ID)
	}

	bolumAd, bolumSlug := "Piyasalar", "markets"
	if c.CategoryName != nil && *c.CategoryName != "" {
		bolumAd = *c.CategoryName
	}
	if c.CategorySlug != nil && *c.CategorySlug != "" {
		bolumSlug = *c.CategorySlug
	}

	kanonik := s.kanonikKok(r) + makaleURL(c)
	if c.CanonicalURL != nil {
		if elle := strings.TrimSpace(*c.CanonicalURL); elle != "" {
			if strings.HasPrefix(elle, "http://") || strings.HasPrefix(elle, "https://") {
				kanonik = elle
			} else {
				kanonik = s.kanonikKok(r) + "/" + strings.TrimLeft(elle, "/")
			}
		}
	}

	v := templkit.CDArticleView{
		Shell:       s.cdShellCtx(ctx, r, bolumSlug),
		Title:       cdMakaleBaslik(c),
		Desc:        cdMakaleAciklama(c),
		Canonical:   kanonik,
		Kicker:      bolumAd,
		SectionName: bolumAd,
		SectionHref: s.bolumHref(bolumSlug),
		Headline:    c.Title,
		Spot:        c.Spot,
		BodyHTML:    string(s.blocksToHTML(c.Detail, nil)),
		Image:       s.cdMakaleGorsel(c),
		BaseURL:     s.kanonikKok(r),
	}
	if c.AuthorName != "" {
		v.Author = c.AuthorName
	} else {
		v.Author = s.siteAdi()
	}
	v.Dil = templkit.CDMakaleDilIcin(c.Locale)
	// İÇERİK DİLİ BURADA, EN ERKEN NOKTADA kuruluyor.
	//
	// Eskiden v.Locale yalnız satır ~360'ta atanıyordu; makaleJSONLD ise
	// satır ~300'de çağrılıyor. Yapısal veri boş locale görüp "tr"ye
	// düşüyordu: üretimde ölçüldü, İngilizce sayfa <html lang="en"> ile
	// basılıp inLanguage="tr" diyordu. Dili kullanan her şey bu satırın
	// ALTINDA olmalı.
	icerikDil := strings.TrimSpace(c.Locale)
	if icerikDil == "" {
		icerikDil = "tr"
	}
	v.Locale = icerikDil
	if c.PublishedAt != nil {
		v.PublishedAt = makaleTarih(*c.PublishedAt, c.Locale)
		v.PublishedISO = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() && c.PublishedAt != nil &&
		c.UpdatedAt.Sub(*c.PublishedAt) > time.Hour {
		v.UpdatedAt = makaleTarih(c.UpdatedAt, c.Locale)
		v.UpdatedISO = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	v.ReadMinutes = cdOkumaDakika(c.Spot, v.BodyHTML)

	// Özet kutusu (spec §10.2.2): yalnız ai_summary DOLUYSA.
	// Uydurma özet yazmak yanlış bilgi olurdu — boşsa kutu basılmaz.
	if ozet := repo.AiOzet(ctx, s.pool, c.ID); ozet != "" {
		v.Summary = ozet
		v.SummaryIlkSatir = ilkCumle(ozet)
		v.SummaryMaddeler = cumlelereBol(ozet, 3)
	}
	if c.TranslationOf != nil {
		v.CeviriEtiketi = v.Dil.Ceviri
		if kok, err := repo.GetContent(ctx, s.pool, *c.TranslationOf); err == nil {
			v.CeviriHref = makaleURL(kok)
		}
	}

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
	//
	// KADEME KARARI DİL SÜZGECİNDEN SONRA VERİLİR.
	//
	// Eskiden zincir "len(related) == 0" ile ilerliyordu ve orta halka
	// (RelatedContents) locale parametresi ALMADIĞI için Türkçe makaleye
	// İngilizce haber döndürebiliyordu. Liste "dolu" sayıldığından
	// locale'li son kademe (PublishedBySlug) hiç çalışmıyordu; sonuçta
	// ya yanlış dilde bağlantı basılıyor ya da bölüm boş kalıyordu
	// (yerelde ölçüldü: 6 bağlantının 6'sı /en/).
	//
	// Artık her kademenin çıktısı süzülüp sayılıyor; hedef dolmadıkça
	// sıradaki kademe denenir.
	related := make([]repo.Content, 0, cdIlgiliHaberLimiti)
	ekle := func(aday []repo.Content) {
		for _, rc := range aday {
			if len(related) >= cdIlgiliHaberLimiti {
				return
			}
			if !cdAyniDil(rc.Locale, c.Locale) {
				continue
			}
			if rc.ID == c.ID {
				continue
			}
			var zaten bool
			for _, v := range related {
				if v.ID == rc.ID {
					zaten = true
					break
				}
			}
			if !zaten {
				related = append(related, rc)
			}
		}
	}

	// Kademelerden daha geniş aday istenir (2 katı): süzgeç sonrası
	// hedefin altına düşmemek için.
	const adayCarpani = 2
	if aday, err := repo.SimilarTo(ctx, s.pool, c.ID, c.SiteID, c.Locale, cdIlgiliHaberLimiti*adayCarpani); err == nil {
		ekle(aday)
	}
	if len(related) < cdIlgiliHaberLimiti {
		if aday, err := repo.RelatedContents(ctx, s.pool, c.SiteID, c.ID, cdIlgiliHaberLimiti*adayCarpani); err == nil {
			ekle(aday)
		}
	}
	if len(related) < cdIlgiliHaberLimiti {
		if aday, err := repo.PublishedBySlug(ctx, s.pool, c.SiteID, catSlugPtr(c.CategorySlug), c.Locale, cdIlgiliHaberLimiti*adayCarpani); err == nil {
			ekle(aday)
		}
	}
	// Süzme ve tekilleştirme yukarıdaki ekle() içinde yapıldı.
	simdi := time.Now()
	for _, rc := range related {
		v.Related = append(v.Related, s.cdStoryCevir(rc, simdi, dil))
	}

	// Önceki/sonraki: kripto portalında "sıradaki haber" okuma süresini
	// uzatır. Zaman tabanlı komşuluk (ContentBefore/After) kullanılır.
	if c.PublishedAt != nil {
		catID := int64(0)
		if c.CategoryID != nil {
			catID = *c.CategoryID
		}
		if sonraki, err := repo.ContentAfter(ctx, s.pool, c.SiteID, *c.PublishedAt, c.Locale, c.ID, catID); err == nil {
			v.NextHref = makaleURL(sonraki)
			v.NextTitle = sonraki.Title
		}
		if onceki, err := repo.ContentBefore(ctx, s.pool, c.SiteID, *c.PublishedAt, c.Locale, c.ID, catID); err == nil {
			v.PrevHref = makaleURL(onceki)
			v.PrevTitle = onceki.Title
		}
	}

	// NewsArticle (CoinDesk alan seti) + ayrı BreadcrumbList bloğu.
	// Go tarafında üretiliyor; templ `<script>` içini ham sayar.
	var yazarSlug string
	if c.AuthorID > 0 {
		if ref, err := repo.PersonRefByID(ctx, s.pool, c.AuthorID); err == nil {
			yazarSlug = ref.Slug
			// Yazar adı CoinDesk'te yazar sayfasına bağlanır (spec §10.2).
			v.AuthorURL = "/yazar/" + ref.Slug
		}
	}
	var anahtar []string
	for _, e := range v.Tags {
		anahtar = append(anahtar, e.Ad)
	}
	yayinISO := makaleYayinISO(c)
	guncelISO := yayinISO
	if !c.UpdatedAt.IsZero() {
		guncelISO = c.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	// Kilit JSON-LD'DEN ÖNCE hesaplanıyor: yapısal veri
	// `isAccessibleForFree` alanını buradan okuyor.
	// Yazıda geçen varlıkları canlı fiyat rozetiyle işaretle (referans
	// davranış: coindesk makale gövdesi). Fiyat gövdeye yazılmıyor, her
	// render'da güncel veriden üretiliyor.
	if fiyatlar := repo.Fiyatlar(ctx, s.pool); len(fiyatlar) > 0 {
		v.BodyHTML = cdVarlikRozetle(v.BodyHTML, s.cdVarlikListesi(fiyatlar, 12))
	}

	v.ContentID = itoa64(c.ID)
	v.Yorum = s.cdYorumVerisi(r, c)

	// Paylaşım görseli: içerikte yoksa üretilen varsayılana düşer.
	// Boş bırakmak kartı düz bağlantıya çevirir; ölü adres vermek ise
	// kartı hiç oluşturmaz. Her iki durumda da paylaşım görünmez olur.
	v.OGImage = s.mutlakGorsel(s.kanonikKok(r), v.Image)
	if strings.TrimSpace(v.OGImage) == "" {
		v.OGImage = s.kanonikKok(r) + varsayilanOGYol()
	}

	// ABONELİK KİLİDİ: gövde kesilir ve kilit kutusu basılır. Karar
	// `cd_kilit.go` içinde; burada yalnız sonuç uygulanıyor.
	//
	// ÖNİZLEME 3 PARAGRAF: okuyucunun konuyu anlamasına yetecek kadar,
	// yazının tamamını vermeyecek kadar. Ghost ve CoinDesk de ilk bölümü
	// açık bırakıyor; sıfır önizleme hem okuyucuyu hem arama motorunu
	// kör bırakırdı.
	if kilit := s.cdKilitHesapla(r, c); kilit.Kilitli {
		v.Kilitli = true
		v.KilitKatman = kilit.Katman
		v.BodyHTML = cdOnizlemeKes(v.BodyHTML, 3)
	}

	v.JSONLD = string(makaleJSONLD(makaleLDGirdi{
		View: v, YayinISO: yayinISO, GuncelISO: guncelISO,
		ID: itoa64(c.ID), YazarSlug: yazarSlug, Keywords: anahtar,
		Kilitli:   v.Kilitli,
		GorselAlt: v.ImageCredit,
		Gorsel:    s.mutlakGorsel(s.kanonikKok(r), v.Image),
		Logo:      s.mutlakGorsel(s.kanonikKok(r), v.Shell.LogoURL),
	}))
	v.BreadcrumbJSONLD = string(makaleBreadcrumbJSONLD(v))
	// SSS bölümü olan rehberlerde FAQPage. Metin gövdeden çıkarılır;
	// bölüm yoksa ya da ikiden az soru varsa blok hiç basılmaz.
	v.FAQJSONLD = string(faqJSONLD(faqCikar(c.Detail)))

	// CoinDesk makale meta seti (Arc/Parse.ly). Yalnız gerçek veri.
	v.Metas = cdMakaleMetas(s.siteAdi(), makaleMetaGirdi{
		ID:         itoa64(c.ID),
		Baslik:     c.Title,
		Bolum:      bolumAd,
		BolumSlug:  bolumSlug,
		Yazar:      v.Author,
		YazarSlug:  yazarSlug,
		Kanonik:    v.Canonical,
		Gorsel:     s.mutlakGorsel(s.kanonikKok(r), v.Image),
		YayinISO:   yayinISO,
		YayinTarih: yayinTarihKisa(c),
		YayinSaat:  yayinSaatKisa(c),
		Etiketler:  anahtar,
	})
	// Sayfanın dili İÇERİKTEN gelir, sabit "tr" değil.
	//
	// Eskiden her makale Türkçe varsayılıyordu: İngilizce içerik kendi
	// URL'sini hreflang="tr" diye bildiriyor, gerçek Türkçe çevirisi de
	// `t.Locale == "tr"` kontrolüne takılıp hiç basılmıyordu. Sonuç:
	// karşılıklı olmayan hreflang (Google bunu bütünüyle yok sayar) ve
	// İngilizce sayfada <html lang="tr">. 12 Kasım 2026'da ölçüldü.
	diller := map[string]string{icerikDil: v.Canonical}
	if trs, err := repo.TranslationsOf(ctx, s.pool, c.TranslationRootID()); err == nil {
		for _, t := range trs {
			if t.ID == c.ID || t.Status != "yayinda" || t.Locale == icerikDil {
				continue
			}
			// hreflang, hedefin KANONİK adresini göstermeli.
			// publicArticleURL kısa biçimi (id'li) üretiyordu; hedef
			// sayfa kendini kategori+tarih yoluyla kanonikleştirdiği
			// için iki adres uyuşmuyor ve Google eşleşmeyi kuramıyordu.
			diller[t.Locale] = s.kanonikKok(r) + makaleURL(t)
		}
	}
	if len(diller) > 1 {
		v.Hreflang = diller
	}
	v.PageCategory = "article_page"
	v.ContentCategory = bolumSlug
	// v.Locale yukarıda, JSON-LD üretiminden ÖNCE kuruldu.
	v.ImageAlt = s.mediaAlt(ctx, c.CoverMediaID)
	if v.ImageAlt == "" {
		v.ImageAlt = s.mediaAlt(ctx, c.MansetMediaID)
	}
	if v.ImageAlt == "" {
		v.ImageAlt = c.Title
	}

	// Paylaşım bağlantıları: mutlak adres + başlık (URL-encoded).
	v.ShareURL = v.Canonical
	v.ShareX = "https://twitter.com/intent/tweet?url=" + url.QueryEscape(v.Canonical) +
		"&text=" + url.QueryEscape(c.Title)
	v.ShareWhatsApp = "https://wa.me/?text=" + url.QueryEscape(c.Title+" "+v.Canonical)
	v.ShareLinkedIn = "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(v.Canonical)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDArticle(v).Render(ctx, w); err != nil {
		s.logger.Error("cd makale render", "error", err)
	}
}

// cdMakaleBaslik SEO başlığı: seo_title → başlık + site adı.
func cdMakaleBaslik(c repo.Content) string {
	if c.SeoTitle != nil && strings.TrimSpace(*c.SeoTitle) != "" {
		return cdSiteSonekiAt(*c.SeoTitle)
	}
	return cdSiteSonekiAt(c.Title)
}

// cdSiteSonekiAt başlık sonundaki "| BitCoŞaR" ekini kaldırır.
//
// NEDEN: eski sistem site adını başlığın İÇİNE yazıyordu. Google
// başlığı ~60 karakterde kestiği için site adı arama sonucunda zaten
// görünmüyor; sadece yer kaplayıp başlığın anlamlı kısmını dışarı
// itiyor. Veritabanı 12 Kasım 2026'da temizlendi (3.074 satır); bu
// savunma, eski alışkanlıkla yeniden yazılırsa sayfayı korur.
//
// Yalnız SON ek atılır — "(BİTCOŞAR Kripto Bülteni)" gibi başlığın
// gerçek parçası olan kullanımlar korunur.
func cdSiteSonekiAt(baslik string) string {
	s := strings.TrimSpace(baslik)
	i := strings.LastIndex(s, "|")
	if i <= 0 {
		return s
	}
	kuyruk := strings.TrimSpace(s[i+1:])
	if !strings.EqualFold(kuyruk, "bitcoşar") && !strings.EqualFold(kuyruk, "bitcosar") {
		return s
	}
	return strings.TrimSpace(s[:i])
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
//
// YUKARI yuvarlanır. Aşağı yuvarlama 399 kelimelik bir yazıya "1 dk"
// diyordu: okuyucu iki dakikalık metni bir dakika sanıp giriyor,
// beklentisi tutmuyor. 201 kelime ikinci dakikaya girmiştir.
func cdOkumaDakika(spot, govde string) int {
	kelime := len(strings.Fields(spot)) + len(strings.Fields(govde))
	if kelime <= 0 {
		return 1
	}
	d := (kelime + 199) / 200
	if d < 1 {
		return 1
	}
	return d
}

func (s *Server) cdKapakGorsel(c repo.Content) string {
	// 1) Kapak görseli (medya yöneticisinden yüklenen)
	if u := s.medyaYoluCoz(c.CoverMediaID); u != "" {
		return u
	}
	if u := s.medyaYoluCoz(c.MansetMediaID); u != "" {
		return u
	}
	// 3) Kaynağın görseli (dış adres ya da /kg proxy yolu)
	return strings.TrimSpace(c.ImageURL)
}

// medyaYoluCoz medya kimliğini web yoluna çevirir; kayıt yoksa/silinmişse "".
func (s *Server) medyaYoluCoz(id *int64) string {
	if id == nil || *id <= 0 {
		return ""
	}
	u := s.mediaURL(id)
	if u == "" || u == "/media/none.svg" {
		return ""
	}
	return u
}

// cdMakaleGorsel makalenin kapak görseli (bkz. cdKapakGorsel).
func (s *Server) cdMakaleGorsel(c repo.Content) string {
	return s.cdKapakGorsel(c)
}

// yayinTarihKisa CoinDesk `create_date` biçimi (YYYYMMDD).
func yayinTarihKisa(c repo.Content) string {
	if c.PublishedAt == nil {
		return ""
	}
	return c.PublishedAt.Format("20060102")
}

// yayinSaatKisa CoinDesk `create_time` biçimi (HH:MM, yerel saat).
func yayinSaatKisa(c repo.Content) string {
	if c.PublishedAt == nil {
		return ""
	}
	return c.PublishedAt.Format("15:04")
}

// kirpRune metni rune sınırında kırpar (UTF-8 bozulmasın).
func kirpRune(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}

func makaleTarih(t time.Time, locale string) string {
	if strings.EqualFold(locale, "en") {
		return t.Format("Jan 2, 2006 3:04 PM MST")
	}
	// Ay adı eşlemesi TEK YERDE (repo.TarihTRSaat): burada ikinci bir
	// layout dizgisi tutmak aynı hatayı geri getirirdi.
	return repo.TarihTRSaat(t)
}
