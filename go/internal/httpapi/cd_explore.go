package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/webui/templkit"
	"theblock/site/internal/youtube"
)

// cdExploreDoldur ana sayfa şeridini kurar.
func (s *Server) cdExploreDoldur(r *http.Request, sections []templkit.CDSection) templkit.CDExplore {
	e := templkit.CDExplore{Baslik: s.siteName + "'dan daha fazlası"}

	// 1) Fiyatlar — canlı şeritten 10 satır.
	if fiyatlar := s.cdFiyatSatirlari(r, 10); len(fiyatlar) > 0 {
		e.Paneller = append(e.Paneller, templkit.CDExplorePanel{
			Tur:       "fiyat",
			Baslik:    "Fiyatlar",
			Href:      "/price",
			Aciklama:  "3000+ kripto varlığın canlı fiyatı, piyasa değeri ve 24 saatlik değişimi.",
			Fiyatlar:  fiyatlar,
			AltMetin:  "Daha fazla veri mi gerekiyor?",
			AltHref:   "/price",
			AltBaglam: "Tüm fiyatlar",
		})
	}

	if videolar := s.cdExploreVideolar(r, 5); len(videolar) > 0 {
		e.Paneller = append(e.Paneller, templkit.CDExplorePanel{
			Tur:      "video",
			Baslik:   "Videolar",
			Href:     "/videos",
			Videolar: videolar,
		})
	}

	// 3) Klipler — dikey short'lar. 3 sütun × 2 satır (117×208 kart).
	// Açıklama/alt bağlantı yok: 2 satır ancak böyle sığıyor.
	if klipler := s.cdYoutubeKlipler(r, 6); len(klipler) > 0 {
		e.Paneller = append(e.Paneller, templkit.CDExplorePanel{
			Tur:     "klip",
			Baslik:  "Klipler",
			Href:    "/videos",
			Klipler: klipler,
		})
	}

	simdi := time.Now()
	for _, sec := range sections {
		var haberler []templkit.CDStory
		slug := strings.TrimPrefix(sec.Href, "/")
		if kayitlar, _ := s.cdListeIcerikleri(r.Context(), slug, 1); len(kayitlar) > 0 {
			for _, c := range kayitlar {
				haberler = append(haberler, s.cdStoryCevir(c, simdi))
				if len(haberler) == 5 {
					break
				}
			}
		}
		if len(haberler) == 0 {
			if sec.Lead.Href != "" {
				haberler = append(haberler, sec.Lead)
			}
			haberler = append(haberler, sec.Rows...)
		}
		if len(haberler) == 0 {
			continue
		}
		e.Paneller = append(e.Paneller, templkit.CDExplorePanel{
			Tur:       "haber",
			Baslik:    sec.Title,
			Href:      sec.Href,
			Aciklama:  "Son haberler ve analizler.",
			Haberler:  haberler,
			AltMetin:  "Tümü",
			AltHref:   sec.Href,
			AltBaglam: sec.Title + " haberleri",
		})
	}

	return e
}

func (s *Server) cdExploreVideolar(r *http.Request, adet int) []templkit.CDVideo {
	ayar := s.ytAyar(r)
	if !ayar.VideoGoster {
		return nil
	}
	out := make([]templkit.CDVideo, 0, adet)
	ekle := func(tur string) {
		if len(out) >= adet {
			return
		}
		kayitlar := ytListeDilim(s.ytListe(r.Context(), tur), adet-len(out))
		if len(kayitlar) == 0 {
			return
		}
		for _, k := range kayitlar {
			etiket := k.Kanal
			if etiket == "" {
				etiket = "Video"
			}
			// Kapak kimlikten üretilir; DB'deki hqdefault (480×360)
			// bulanıktı (bkz. cd_video.go aynı düzeltme).
			img := k.Gorsel
			if k.VideoID != "" {
				if u := youtube.GorselURL(k.VideoID, k.Tur); u != "" {
					img = u
				}
			}
			out = append(out, templkit.CDVideo{
				Etiket:     etiket,
				Baslik:     k.Baslik,
				Image:      img,
				Href:       ytHref(k),
				Sure:       youtube.Sure(k.SureSn),
				Srcset:     youtube.GorselSrcset(k.VideoID, k.Tur),
				ImageYedek: youtube.GorselYedekZincir(k.VideoID, k.Tur),
				Onizleme:   youtube.OnizlemeURL(k.VideoID),
			})
		}
	}
	ekle("video")
	ekle("stream")
	return out
}
