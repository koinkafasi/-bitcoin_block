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

	// Okunma sayımı ve çizim ortak fonksiyonda (önizleme de aynı şablonu
	// kullanır; orada sayım YAPILMAZ).
	s.cdMakaleGoster(w, r, c, false)
}

// handleCDArticleOnizleme /admin/icerikler/{id}/onizleme — editör önizlemesi.
//
// NEDEN: önizleme `handleArticle`ı (theblock şablonu) çağırıyordu; editör
// yeni görseli ekledikten sonra önizlemede ESKİ tasarımı görüyordu
// (kullanıcı bildirimi, 10 Kasım 2026). Artık canlı sayfayla AYNI şablon.
// Farklar yalnız: durum kapısı yok (taslak da açılır) ve okunma sayılmaz.
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
	ctx := r.Context()
	if !onizleme {
		// Okunma sayımı: istek yolunda DB yazımı YOK. Sayı bellekte
		// biriktirilir, ~2 sn'de bir tek turda yazılır (bkz. okunma.go).
		// Ölçüm: iki UPSERT makale sayfasında c=8'de 24 rps → 80 rps.
		s.okunmaKaydet(ctx, c.ID)
	}

	bolumAd, bolumSlug := "Piyasalar", "markets"
	if c.CategoryName != nil && *c.CategoryName != "" {
		bolumAd = *c.CategoryName
	}
	if c.CategorySlug != nil && *c.CategorySlug != "" {
		bolumSlug = *c.CategorySlug
	}

	// KANONİK ÖNCELİĞİ: içerikte elle atanmış `canonical_url` varsa o kazanır.
	// Kaynak sitede aynı haber birden çok kez yayınlanmış olabiliyor
	// (WordPress aktarımında 3 böyle grup var); sendika/kopya içerikte
	// kanonik adresi asıl yayına çevirmek yinelenen içeriği tek sayfada
	// toplar. Alan boşsa bugüne kadarki davranış korunur.
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
		Shell:     s.cdShellCtx(ctx, r, bolumSlug),
		Title:     cdMakaleBaslik(c),
		Desc:      cdMakaleAciklama(c),
		Canonical: kanonik,
		// Kicker: CoinDesk bölüm adını OLDUĞU GİBİ basıyor ("Markets"),
		// büyük harfe çevirmiyor (ölçüldü: computed `text-transform: none`).
		// Eskiden i18n.BuyukHarf uygulanıyordu ve "POLİTİKA" çıkıyordu.
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
		v.Author = s.siteName
	}
	v.Dil = templkit.CDMakaleDilIcin(c.Locale)
	if c.PublishedAt != nil {
		v.PublishedAt = makaleTarih(*c.PublishedAt, c.Locale)
		v.PublishedISO = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	// "Güncellendi" yalnız GERÇEKTEN sonradan düzenlendiyse.
	//
	// NEDEN EŞİK: `updated_at` crawler'ın her upsert'ünde tazeleniyor; ham
	// karşılaştırma 126/126 içeriği "güncellenmiş" gösteriyordu (ölçüldü,
	// 6 Kasım 2026). Oysa içerik aynıydı. 1 saatlik eşik yalnız gerçek
	// editoryal düzenlemeyi yakalar; altındaki fark tarama gürültüsüdür.
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
	// ÇEVİRİ ETİKETİ (§10.2): yalnız GERÇEK çeviri kayıtlarında.
	//
	// "translation_of" dolu = bu içerik başka bir dildeki içerikten
	// üretilmiş. Etiket okuyucuya sayfanın makine çevirisi olduğunu
	// SÖYLER — bu bir dürüstlük yükümlülüğü, süs değil.
	//
	// Bağlantı ORİJİNAL makaleye gider: okuyucu kaynağı tek tıkla
	// görebilmeli. Orijinal okunamazsa (silinmiş/taşınmış) etiket yine
	// basılır ama bağlantısız (kırık bağlantı üretilmez).
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
		v.Related = append(v.Related, s.cdStoryCevir(rc, simdi))
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
	v.JSONLD = string(makaleJSONLD(makaleLDGirdi{
		View: v, YayinISO: yayinISO, GuncelISO: guncelISO,
		ID: itoa64(c.ID), YazarSlug: yazarSlug, Keywords: anahtar,
		GorselAlt: v.ImageCredit,
		Gorsel:    s.mutlakGorsel(s.kanonikKok(r), v.Image),
	}))
	v.BreadcrumbJSONLD = string(makaleBreadcrumbJSONLD(v))

	// CoinDesk makale meta seti (Arc/Parse.ly). Yalnız gerçek veri.
	v.Metas = cdMakaleMetas(s.siteName, makaleMetaGirdi{
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
	// hreflang: yalnız GERÇEK çeviriler. /en/... adresleri theblock
	// şablonuyla servis ediliyor; adres biçimi publicArticleURL'den gelir.
	// Çeviri yoksa yalnız x-default basılır (cdHead kuralı).
	diller := map[string]string{"tr": v.Canonical}
	if trs, err := repo.TranslationsOf(ctx, s.pool, c.TranslationRootID()); err == nil {
		for _, t := range trs {
			if t.ID == c.ID || t.Status != "yayinda" || t.Locale == "tr" {
				continue
			}
			diller[t.Locale] = s.kanonikKok(r) +
				publicArticleURL(t.Locale, t.DisplaySlug(), t.ID)
		}
	}
	if len(diller) > 1 {
		v.Hreflang = diller
	}
	v.PageCategory = "article_page"
	v.ContentCategory = bolumSlug
	v.Locale = "tr"
	v.ImageAlt = v.ImageCredit

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

// cdKapakGorsel içeriğin kapak görselini çözer.
//
// ÖNCELİK SIRASI — editör kazanır:
//  1. `cover_media_id` (medya yöneticisinden yüklenen görsel)
//  2. `image_url` (kaynak siteden gelen görsel)
//
// NEDEN TERSİYDİ: önce `image_url` okunuyordu. Editör yeni görsel yükleyip
// kaydettiğinde canlı sayfa HÂLÂ eski CDN görselini gösteriyordu
// (kullanıcı bildirimi, 10 Kasım 2026: "yeni resim var ama eski resim
// imaj yolu görünüyor"). Aynı hata liste kartlarını da vuruyordu:
// category/section/tag/related kartları da `image_url` kullanıyordu.
// Diğer çıktılar (RSS, JSON-LD, canlı içerik) zaten `cover_media_id`yi
// önceleyiyordu — tek tutarsız yer buydu.
func (s *Server) cdKapakGorsel(c repo.Content) string {
	if c.CoverMediaID != nil && *c.CoverMediaID > 0 {
		if u := s.mediaURL(c.CoverMediaID); u != "" && u != "/media/none.svg" {
			return u
		}
	}
	return strings.TrimSpace(c.ImageURL)
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

// makaleTarih makale tarihini SAYFA DİLİNE göre biçimler.
//
// İKİ KUSUR DÜZELTİLDİ (ölçüldü, 9 Kasım 2026):
//  1. Biçim `"2 Ocak 2006 15:04 MST"` idi. Go'nun layout'unda ay belirteci
//     "Jan"/"January"dır; **"Ocak" tanınmaz ve DÜZ METİN olarak basılır**.
//     Sonuç: hangi ayda yayınlanırsa yayınlansın her Türkçe makale
//     "6 Ocak 2026" gösteriyordu (Kasım'daki haber Ocak görünüyordu).
//     Türkçe ay adları bu yüzden ELLE eşlenir.
//  2. İngilizce çeviri sayfası da aynı Türkçe biçimi kullanıyordu.
func makaleTarih(t time.Time, locale string) string {
	if strings.EqualFold(locale, "en") {
		return t.Format("Jan 2, 2006 3:04 PM MST")
	}
	// Ay adı eşlemesi TEK YERDE (repo.TarihTRSaat): burada ikinci bir
	// layout dizgisi tutmak aynı hatayı geri getirirdi.
	return repo.TarihTRSaat(t)
}
