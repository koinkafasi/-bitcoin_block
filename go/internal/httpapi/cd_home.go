package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"theblock/site/internal/i18n"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

func (s *Server) handleCDHome(w http.ResponseWriter, r *http.Request) {
	if wpID := wpKimlikSorgusu(r.URL.Query()); wpID > 0 {
		s.eskiKimlikYonlendir(w, r, wpID)
		return
	}
	ctx := r.Context()

	items := s.cdHomeIcerikleri(ctx)
	simdi := time.Now()
	cevir := func(c repo.Content) templkit.CDStory {
		st := templkit.CDStory{
			Title: c.Title,
			Spot:  c.Spot,
			Href:  makaleURL(c),
			Image: s.cdKapakGorsel(c),
			// Siyah-beyaz yalnız LİSTELERDE; yazı sayfası renkli.
			Gri: c.ImageGrayscale,
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

	var donus []templkit.CDStory
	for _, c := range items {
		donus = append(donus, cevir(c))
	}

	leadIdx := 0
	for i := range donus {
		if donus[i].Image != "" {
			leadIdx = i
			break
		}
	}
	var lead templkit.CDStory
	if len(donus) > 0 {
		lead = donus[leadIdx]
	}

	// Aynı haber iki modülde görünmesin: hero içinde tek kullanım.
	kullanildi := map[string]bool{}
	if lead.Href != "" {
		kullanildi[lead.Href] = true
	}

	var akis, gorselsiz []templkit.CDStory
	for i, st := range donus {
		if i == leadIdx || kullanildi[st.Href] {
			continue
		}
		if st.Image == "" {
			gorselsiz = append(gorselsiz, st)
			continue
		}
		akis = append(akis, st)
		kullanildi[st.Href] = true
		if len(akis) == 8 {
			break
		}
	}
	for _, st := range gorselsiz {
		if len(akis) >= 8 {
			break
		}
		if kullanildi[st.Href] {
			continue
		}
		akis = append(akis, st)
		kullanildi[st.Href] = true
	}

	// §7A sol ray: CoinDesk'in "Son Haberler" modülü — 5 satır.
	var railSol []templkit.CDStory
	for i, st := range donus {
		if i == leadIdx || kullanildi[st.Href] {
			continue
		}
		railSol = append(railSol, st)
		kullanildi[st.Href] = true
		if len(railSol) == 5 {
			break
		}
	}

	var tekGorsel templkit.CDStory
	for i, st := range donus {
		if i == leadIdx || st.Image == "" || kullanildi[st.Href] {
			continue
		}
		tekGorsel = st
		kullanildi[st.Href] = true
		break
	}

	// §22 bölüm blokları — CoinDesk 2×2 sırası: Piyasalar · Politika ·
	// Teknoloji · Finans (ölçüldü). Hücre başına 1 görselli + 3 satır.
	bolumler := []cdBolum{
		{"Piyasalar", "markets"},
		{"Politika", "policy"},
		{"Teknoloji", "tech"},
		{"Finans", "business"},
	}
	var sections []templkit.CDSection
	for _, b := range bolumler {
		var grup []templkit.CDStory
		for _, c := range items {
			if c.CategorySlug == nil || *c.CategorySlug != b.Slug {
				continue
			}
			grup = append(grup, cevir(c))
			if len(grup) >= 4 {
				break
			}
		}
		sec := templkit.CDSection{Title: b.Ad, Href: "/" + b.Slug}
		if len(grup) > 0 {
			sec.Lead = grup[0]
			sec.Rows = grup[1:]
		}
		sections = append(sections, sec)
	}

	d := templkit.CDHomeView{
		Shell:     s.cdShellCtx(r.Context(), r, ""),
		Title:     s.siteName + " — Kripto Haberleri ve Piyasa Verisi",
		Desc:      "Bitcoin, Ethereum ve kripto piyasalarından son haberler, analizler ve canlı fiyatlar.",
		Canonical: s.kanonikKok(r) + r.URL.Path,
		Lead:      lead,
		RailLeft:  railSol,
		Dusunce:   cdGeciciDusunce(),
		TekGorsel: tekGorsel,
		Akis:      akis,
		Basin:     cdGeciciBasin(),
		Videolar:  s.cdYoutubeVideolar(r, 3),
		Arastirma: cdGeciciArastirma(),
		Klipler:   s.cdYoutubeKlipler(r, 3),
		Canli:     s.cdYoutubeCanli(r),
		Sections:  sections,
		MostRead:  s.cdPopuler(r, 6),
		Opinion:   s.cdUzmanlar(r, 4),
		Hreflang:  map[string]string{"tr": s.kanonikKok(r) + "/"},
		JSONLD:    string(orgJSONLD(s.siteName, s.kanonikKok(r), s.kanonikKok(r)+r.URL.Path)),
		// "Explore more from CoinDesk" şeridi — gerçek veriyle.
		Explore: s.cdExploreDoldur(r, sections),
		// MODÜLER YERLEŞİM (9 Kasım 2026): sayfa tasarımcısının yazdığı
		// `cd_*` blokları. Liste boşsa şablon varsayılan sırayı uygular.
		Moduller: s.cdModuller(r.Context(), r),
		// Giriş hero'su: gerçek piyasa verisi (CoinGecko + alternative.me).
		Giris: s.cdGirisOlustur(ctx),
		// CoinDesk ana sayfa sınıflandırması.
		PageCategory:    "homepage",
		ContentCategory: "homepage",
		Locale:          "tr",
		BaseURL:         s.kanonikKok(r),
	}
	// Fiyat şeridi cdShellCtx içinde doldurulur (tüm sayfalar için tek yer).

	// Piyasa Nabzı: gerçek CryptoQuant verisi; anahtar yoksa boş kalır.
	d.NabizFiyat, d.Nabiz = s.cdNabiz(r.Context())

	// Analiz akışı (Piyasa Nabzı'nın üstü): son analiz yazıları.
	d.Analiz, d.AnalizSayi = s.cdAnalizAkisi(r, 15)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDHome(d).Render(ctx, w); err != nil {
		s.logger.Error("cd ana sayfa render", "error", err)
	}
}

// cdBolum ana sayfadaki bölüm bloklarının tanımı.
type cdBolum struct{ Ad, Slug string }

// cdPopuler §21 "En Çok Okunan" listesi (son 7 gün ağırlıklı).
func (s *Server) cdPopuler(r *http.Request, adet int) []templkit.CDStory {
	items, err := repo.PopulerIcerikler(r.Context(), s.pool, 1, "tr", adet)
	if err != nil || len(items) == 0 {
		return nil
	}
	simdi := time.Now()
	var out []templkit.CDStory
	for _, c := range items {
		out = append(out, s.cdStoryCevir(c, simdi))
	}
	return out
}

func (s *Server) cdUzmanlar(r *http.Request, adet int) []templkit.CDUzman {
	items, err := repo.Uzmanlar(r.Context(), s.pool, adet)
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]templkit.CDUzman, 0, len(items))
	for _, u := range items {
		out = append(out, templkit.CDUzman{
			Ad:     u.Ad,
			Unvan:  u.Unvan,
			Avatar: u.Avatar,
			Makale: u.Makale,
			Href:   u.MakaleURL,
		})
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
	case dk < 60*24*30:
		return strconv.Itoa(dk/(60*24)) + " gün önce"
	default:
		return repo.TarihTRUzun(t)
	}
}
