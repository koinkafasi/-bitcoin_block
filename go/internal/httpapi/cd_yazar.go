package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdYazarSosyalAd sosyal anahtarını görünen ada çevirir.
//
// Panelde saklanan anahtarlar: x, linkedin, website, email
// (people_admin_handlers.go). Bilinmeyen anahtar atlanır — bağlantı
// metni uydurulmaz.
func cdYazarSosyalAd(key string) string {
	switch key {
	case "x":
		return "X"
	case "linkedin":
		return "LinkedIn"
	case "website":
		return "Web sitesi"
	case "email":
		return "E-posta"
	}
	return ""
}

// cdYazarSosyalHref e-posta değerini mailto'ya çevirir.
func cdYazarSosyalHref(key, deger string) string {
	deger = strings.TrimSpace(deger)
	if deger == "" {
		return ""
	}
	if key == "email" && !strings.HasPrefix(deger, "mailto:") {
		return "mailto:" + deger
	}
	return deger
}

// cdYazarStory yazarın içeriğini akış satırına çevirir.
//
// Kategori CoinDesk yazar sayfasında BÜYÜK HARF DEĞİL ("Finance"), bu
// yüzden cdStoryCevir'in `i18n.BuyukHarf` çıktısı burada ham kategori
// adıyla değiştirilir (ölçüm 12 Kasım 2026).
func (s *Server) cdYazarStory(c repo.Content, simdi time.Time) templkit.CDStory {
	st := s.cdStoryCevir(c, simdi)
	if c.CategoryName != nil {
		st.Kicker = strings.TrimSpace(*c.CategoryName)
	}
	return st
}

// cdYazarProfilSayfasi /yazar/{slug} — CoinDesk yazar profili düzeni.
func (s *Server) cdYazarProfilSayfasi(w http.ResponseWriter, r *http.Request, p repo.Person) {
	ctx := r.Context()
	page := 1
	if n, e := strconv.Atoi(r.URL.Query().Get("sayfa")); e == nil && n > 0 {
		page = n
	}
	const per = 12
	items, total, _ := repo.PersonContents(ctx, s.pool, p.ID, per, (page-1)*per)

	v := templkit.CDYazarPage{
		Shell:           s.cdShellCtx(ctx, r, "yazarlar"),
		Ad:              p.Name,
		Unvan:           strings.TrimSpace(p.Title),
		Bio:             strings.TrimSpace(p.Bio),
		BasHarf:         ilkHarf(p.Name),
		YaziSayisi:      total,
		BosMetin:        "Bu yazarın henüz yayımlanmış yazısı yok.",
		PageCategory:    "author_page",
		ContentCategory: "author",
		Locale:          "tr",
		BaseURL:         s.kanonikKok(r),
	}
	if p.AvatarMediaID != nil {
		v.AvatarURL = s.mediaURL(p.AvatarMediaID)
		v.Image = strings.TrimRight(s.absBase(r), "/") + v.AvatarURL
	}
	for _, key := range []string{"x", "linkedin", "website", "email"} {
		ad := cdYazarSosyalAd(key)
		href := cdYazarSosyalHref(key, p.Socials[key])
		if ad == "" || href == "" {
			continue
		}
		v.Sosyal = append(v.Sosyal, templkit.CDSosyalBag{Ad: ad, Href: href})
	}

	simdi := time.Now()
	for _, c := range items {
		v.Yazilar = append(v.Yazilar, s.cdYazarStory(c, simdi))
	}

	kanonik := s.kanonikKok(r) + "/yazar/" + url.PathEscape(p.Slug)
	if page > 1 {
		v.OncekiURL = kanonik + "?sayfa=" + strconv.Itoa(page-1)
		v.Title = p.Name + " — Sayfa " + strconv.Itoa(page) + " | " + s.siteName
	} else {
		v.Title = p.Name + " | " + s.siteName + " Yazar"
	}
	if page*per < total {
		v.SonrakiURL = kanonik + "?sayfa=" + strconv.Itoa(page+1)
	}

	aciklama := v.Bio
	if aciklama == "" {
		aciklama = v.Unvan
	}
	if aciklama == "" {
		aciklama = p.Name + " — " + s.siteName + " yazar profili ve yazıları."
	}
	v.Desc = aciklama
	v.Canonical = kanonik

	var sameAs []string
	for _, s2 := range v.Sosyal {
		if strings.HasPrefix(s2.Href, "mailto:") {
			continue
		}
		sameAs = append(sameAs, s2.Href)
	}
	v.JSONLD = string(cdKisiJSONLD(s.siteName, kanonik, p.Name, v.Unvan, v.Bio, v.Image, sameAs))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDYazarPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("yazar profili render", "error", err)
	}
}

// cdYazarlarSayfasi /yazarlar — dizin.
func (s *Server) cdYazarlarSayfasi(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	people, _ := repo.ListPeople(ctx, s.pool, []string{"Yönetici", "Editör", "Muhabir", "Yazar"})

	kanonik := s.kanonikKok(r) + "/yazarlar"
	v := templkit.CDYazarlarPage{
		Shell:           s.cdShellCtx(ctx, r, "yazarlar"),
		Title:           "Yazarlar ve Editörler — " + s.siteName,
		Desc:            "Haberleri kaleme alan yazarlar, muhabirler ve editörler.",
		Canonical:       kanonik,
		PageCategory:    "author_index_page",
		ContentCategory: "author",
		Locale:          "tr",
		BaseURL:         s.kanonikKok(r),
	}
	for _, p := range people {
		k := templkit.CDYazarKart{
			Ad:         p.Name,
			Unvan:      strings.TrimSpace(p.Title),
			Bio:        strings.TrimSpace(p.Bio),
			Href:       "/yazar/" + url.PathEscape(p.Slug),
			BasHarf:    ilkHarf(p.Name),
			YaziSayisi: p.ArticleCount,
		}
		if p.AvatarMediaID != nil {
			k.AvatarURL = s.mediaURL(p.AvatarMediaID)
		}
		v.Kisiler = append(v.Kisiler, k)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDYazarlarPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("yazarlar dizini render", "error", err)
	}
}
