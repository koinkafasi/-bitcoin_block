package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// CoinDesk ana sayfası (bitcosar).
//
// Modül sırası docs/COINDESK-SPEC.md §6 ile BİREBİR; sapma yok.
// Veri yoksa modül basılmaz, sıra korunur.
//
// Ayrı kod yolu bilinçli: mevcut `homeView` tasarımcı/blok motoruna bağlı.
// CoinDesk düzeni SABİT sıra istiyor; blok motoruna sıkıştırmak o sırayı
// veriye bağlardı. Eski düzen `/klasik` altında korunur.

func (s *Server) handleCDHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, _, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
		Status:         "yayinda",
		SiteID:         1,
		Locale:         "tr",
		Page:           1,
		PerPage:        60,
		ExcludeGallery: true,
	})
	if err != nil {
		// Boş site 500 dönmez: hata loglanır, iskelet basılır.
		s.logger.Warn("cd ana sayfa: içerik listesi", "error", err)
	}

	simdi := time.Now()
	cevir := func(c repo.Content) templkit.CDStory {
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

	var donus []templkit.CDStory
	for _, c := range items {
		donus = append(donus, cevir(c))
	}

	// §8 manşet: 1 büyük + 3 yan. §5 latest: ilk 12 (saat damgalı liste).
	//
	// MANŞET GÖRSELLİ OLMALI: CoinDesk manşeti büyük görselli basar.
	// Görselsiz haber manşete düşünce sayfanın üstü boş bir metin bloğu
	// oluyordu (ölçüm: Fed bülteni manşette, görsel yok). Görselli haber
	// yoksa ilk haber kullanılır — manşet yine dolar.
	leadIdx := 0
	for i := range donus {
		if donus[i].Image != "" {
			leadIdx = i
			break
		}
	}
	var lead templkit.CDStory
	var side, latest []templkit.CDStory
	if len(donus) > 0 {
		lead = donus[leadIdx]
	}
	for i, st := range donus {
		if i == leadIdx {
			continue
		}
		if len(side) < 3 {
			side = append(side, st)
		}
		if len(latest) < 12 {
			latest = append(latest, st)
		}
	}

	// §22 bölüm blokları: 1 büyük + 2 alt + 3 satır (CoinDesk deseni).
	bolumler := []struct{ Ad, Slug string }{
		{"Markets", "markets"},
		{"Finance", "business"},
		{"Tech", "tech"},
		{"Policy", "policy"},
	}
	var sections []templkit.CDSection
	for _, b := range bolumler {
		var grup []templkit.CDStory
		for _, c := range items {
			if c.CategorySlug == nil || *c.CategorySlug != b.Slug {
				continue
			}
			grup = append(grup, cevir(c))
			if len(grup) >= 6 {
				break
			}
		}
		sec := templkit.CDSection{Title: b.Ad, Href: "/" + b.Slug}
		if len(grup) > 0 {
			sec.Lead = grup[0]
		}
		for i := 1; i < len(grup) && i <= 2; i++ {
			sec.Second = append(sec.Second, grup[i])
		}
		for i := 3; i < len(grup); i++ {
			sec.Rows = append(sec.Rows, grup[i])
		}
		sections = append(sections, sec)
	}

	// §3 promo bandı: ayarla yönetilir. Kapalıysa/boşsa şablon basmaz.
	promo := repo.AktifPromo(ctx, s.pool, time.Now())

	d := templkit.CDHomeView{
		Shell:     s.cdShellCtx(r.Context(), ""),
		Title:     s.siteName + " — Kripto Haberleri ve Piyasa Verisi",
		Desc:      "Bitcoin, Ethereum ve kripto piyasalarından son haberler, analizler ve canlı fiyatlar.",
		Canonical: s.cfg.BaseURL + r.URL.Path,
		Promo: templkit.CDPromo{
			Aktif:    promo.Aktif,
			Metin:    promo.Metin,
			Baglanti: promo.Baglanti,
			CTA:      promo.CTA,
			Kalan:    promo.Kalan,
		},
		Lead:     lead,
		Side:     side,
		Latest:   latest,
		Prices:   s.cdFiyatSatirlari(r, 10),
		Sections: sections,
		MostRead: s.cdPopuler(r, 6),
		Opinion:  s.cdGorus(r, 4),
		// hreflang: yalnız GERÇEK diller. Şu an içerik Türkçe; EN içerik
		// üretildiğinde buraya eklenecek (çevirisi olmayan dili yazmak
		// Google'da "aynı içerik" cezası riski).
		Hreflang: map[string]string{"tr": s.cfg.BaseURL + "/"},
		JSONLD:   string(orgJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), s.cfg.BaseURL+r.URL.Path)),
	}
	// Fiyat şeridi (§3) — veri yoksa boş kalır, şablon basmaz.
	d.Shell.Ticker = s.cdTickerDoldur(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDHome(d).Render(ctx, w); err != nil {
		s.logger.Error("cd ana sayfa render", "error", err)
	}
}

// cdPopuler §21 "En Çok Okunan" listesi (son 7 gün ağırlıklı).
func (s *Server) cdPopuler(r *http.Request, adet int) []templkit.CDStory {
	items, err := repo.PopulerIcerikler(r.Context(), s.pool, 1, "tr", adet)
	if err != nil || len(items) == 0 {
		return nil
	}
	simdi := time.Now()
	var out []templkit.CDStory
	for _, c := range items {
		out = append(out, cdStoryCevir(c, simdi))
	}
	return out
}

// cdGorus §20: opinion kategorisindeki içerikler.
func (s *Server) cdGorus(r *http.Request, adet int) []templkit.CDStory {
	items, err := repo.BolumIcerikleri(r.Context(), s.pool, 1, "tr", "opinion", adet)
	if err != nil || len(items) == 0 {
		return nil
	}
	simdi := time.Now()
	var out []templkit.CDStory
	for _, c := range items {
		st := cdStoryCevir(c, simdi)
		// Yazar adı görüş satırında "kicker" olarak gösterilir.
		if c.AuthorName != "" {
			st.Source = c.AuthorName
		}
		out = append(out, st)
	}
	return out
}

// trGoreliZaman "3 saat önce" biçiminde insan okur zaman üretir
// (CoinDesk'in "17 minutes ago" deseninin Türkçesi).
func trGoreliZaman(t, simdi time.Time) string {
	dk := int(simdi.Sub(t).Minutes())
	switch {
	case dk < 1:
		return "az önce"
	case dk < 60:
		return strconv.Itoa(dk) + " dakika önce"
	case dk < 60*24:
		return strconv.Itoa(dk/60) + " saat önce"
	default:
		return strconv.Itoa(dk/(60*24)) + " gün önce"
	}
}
