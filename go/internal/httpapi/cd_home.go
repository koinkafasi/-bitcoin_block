package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/i18n"
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

	items := s.cdHomeIcerikleri(ctx)

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
			st.Kicker = i18n.BuyukHarf(*c.CategoryName)
		}
		return st
	}

	var donus []templkit.CDStory
	for _, c := range items {
		donus = append(donus, cevir(c))
	}

	// §8 manşet: 1 büyük + 8 görselli akış + 5 satır "Son Haberler".
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
	if len(donus) > 0 {
		lead = donus[leadIdx]
	}

	// Aynı haber iki modülde görünmesin: hero içinde tek kullanım.
	kullanildi := map[string]bool{}
	if lead.Href != "" {
		kullanildi[lead.Href] = true
	}

	// Orta kolon akışı: manşetten sonraki 8 GÖRSELLİ haber (ölçüldü:
	// 152×86 küçük görsel + 20/26 başlık + 12/18 saat).
	var akis []templkit.CDStory
	for i, st := range donus {
		if i == leadIdx || st.Image == "" || kullanildi[st.Href] {
			continue
		}
		akis = append(akis, st)
		kullanildi[st.Href] = true
		if len(akis) == 8 {
			break
		}
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

	// Sol rayın altındaki tek görselli manşet (CoinDesk canlı blog kartı
	// yuvası; bizde canlı blog yok, aynı geometriyi görselli haberle
	// dolduruyoruz).
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
		Shell:     s.cdShellCtx(r.Context(), ""),
		Title:     s.siteName + " — Kripto Haberleri ve Piyasa Verisi",
		Desc:      "Bitcoin, Ethereum ve kripto piyasalarından son haberler, analizler ve canlı fiyatlar.",
		Canonical: s.cfg.BaseURL + r.URL.Path,
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
		// hreflang: yalnız GERÇEK diller. Şu an içerik Türkçe; EN içerik
		// üretildiğinde buraya eklenecek (çevirisi olmayan dili yazmak
		// Google'da "aynı içerik" cezası riski).
		Hreflang: map[string]string{"tr": s.cfg.BaseURL + "/"},
		JSONLD:   string(orgJSONLD(s.siteName, strings.TrimRight(s.cfg.BaseURL, "/"), s.cfg.BaseURL+r.URL.Path)),
		// CoinDesk ana sayfa sınıflandırması.
		PageCategory:    "homepage",
		ContentCategory: "homepage",
		Locale:          "tr",
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
	}
	// Fiyat şeridi (§3) — veri yoksa boş kalır, şablon basmaz.
	d.Shell.Ticker = s.cdTickerDoldur(r)

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
		out = append(out, cdStoryCevir(c, simdi))
	}
	return out
}

// cdUzmanlar §20 "Görüş": avatarı olan uzman/editoryal profiller.
//
// NEDEN içerik değil PROFİL: CoinDesk "Opinion" modülünde yazar adı +
// unvan + 48 px avatar basıyor; makale kartı değil. Profiller `users`
// tablosunda (rol = Yazar); makale başlığı profilin öne çıkan yazısıdır.
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
	default:
		return strconv.Itoa(dk/(60*24)) + " gün önce"
	}
}
