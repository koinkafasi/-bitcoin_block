package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Mega menünün bağlandığı liste sayfaları.
//
// NEDEN: menüdeki her satırın gerçek bir karşılığı olmalı. Ölçüm (5 Kasım
// 2026): menüdeki 49 hedefin 29'u 404'tü. Bu dosya o boşlukları **gerçek**
// sayfalarla doldurur; içerik yoksa sayfa dürüst boş durum gösterir
// (uydurma kart yok).
//
// /videos artık YouTube kanalından besleniyor — handler cd_video.go'da.
// /podcasts          — tür "podcast"
// /sponsored-content — tür "sponsorlu"
// /newsletters       — bülten listesi + kayıt

// handleCDPodcasts /podcasts
func (s *Server) handleCDPodcasts(w http.ResponseWriter, r *http.Request) {
	s.cdTurListesi(w, r, "podcast", templkit.CDListePage{
		Baslik:   "Podcast",
		Aciklama: "Kripto ve blockchain üzerine sohbetler, analizler ve röportajlar.",
		BosMetin: "Henüz podcast bölümü yayınlanmadı. Bölümler eklendikçe burada listelenir.",
	})
}

// handleCDSponsored /sponsored-content
func (s *Server) handleCDSponsored(w http.ResponseWriter, r *http.Request) {
	s.cdTurListesi(w, r, "sponsorlu", templkit.CDListePage{
		Baslik:   "Sponsorlu İçerik",
		Aciklama: "İş ortaklarımızın hazırladığı içerikler. Sponsorlu içerikler açıkça etiketlenir.",
		BosMetin: "Şu an yayında sponsorlu içerik yok.",
	})
}

// cdTurListesi ortak uç: tür filtresiyle içerik listeler.
func (s *Server) cdTurListesi(w http.ResponseWriter, r *http.Request, tur string, v templkit.CDListePage) {
	ctx := r.Context()
	items, _, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status:  "yayinda",
		TypeKey: tur,
		SiteID:  1,
		Locale:  "tr",
		Page:    1,
		PerPage: 30,
	})
	if err != nil {
		s.logger.Warn("cd tur listesi", "tur", tur, "error", err)
	}
	simdi := time.Now()
	var ldItems []cdLDItem
	for i, c := range items {
		v.Stories = append(v.Stories, cdStoryCevir(c, simdi))
		if i < 11 {
			it := cdLDItem{Headline: c.Title, URL: s.absBase(r) + makaleURL(c), Author: c.AuthorName}
			if c.PublishedAt != nil {
				it.Published = c.PublishedAt.UTC().Format(time.RFC3339)
			}
			ldItems = append(ldItems, it)
		}
	}

	yol := "/" + strings.TrimPrefix(tur, "")
	switch tur {
	case "video":
		yol = "/videos"
	case "podcast":
		yol = "/podcasts"
	case "sponsorlu":
		yol = "/sponsored-content"
	}
	v.Shell = s.cdShellCtx(ctx, strings.TrimPrefix(yol, "/"))
	v.Title = v.Baslik + " — " + s.siteName
	v.Desc = v.Aciklama
	v.Canonical = s.cfg.BaseURL + yol
	// CoinDesk video/podcast/sponsorlu sayfaları CollectionPage basıyor.
	v.JSONLD = string(cdCollectionJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), v.Baslik, v.Canonical, ldItems))
	v.PageCategory = "section_landing_page"
	v.ContentCategory = strings.TrimPrefix(yol, "/")
	v.Locale = "tr"
	v.BaseURL = strings.TrimRight(s.cfg.BaseURL, "/")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDListePageView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd liste render", "tur", tur, "error", err)
	}
}

// handleCDNewsletters /newsletters
func (s *Server) handleCDNewsletters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	v := templkit.CDNewsletterPage{
		Shell:           s.cdShellCtx(ctx, "newsletters"),
		Title:           "Bültenler — " + s.siteName,
		Desc:            "Günün kripto haberleri ve piyasa özeti e-postanızda. Ücretsiz abone olun.",
		Canonical:       s.cfg.BaseURL + "/newsletters",
		PageCategory:    "section_landing_page",
		ContentCategory: "newsletters",
		Locale:          "tr",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
	}
	v.JSONLD = string(siteAdiJSONLD(s.siteName, v.Canonical))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDNewsletterPageView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd bulten render", "error", err)
	}
}
