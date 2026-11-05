package httpapi

import (
	"encoding/xml"
	"strings"
	"time"
)

// Google News sitemap doğrulaması (Faz 23)
//
// Search Console'a gönderilen `sitemap-news.xml` bozuksa Google sessizce yok
// sayar — hata bildirilmez. Editör gönderemeden önce burada yapısal kontrol
// yapabilsin diye üretilen XML ayrıştırılıp eksik alanlar sayılır.
//
// NOT: Ağ isteği yapılmaz; doğrulanan şey sitemap'i ÜRETEN kodun çıktısıdır
// (aynı fonksiyon HTTP uç noktasında da kullanılır).

// NewsSitemapReport doğrulama sonucu.
type NewsSitemapReport struct {
	URL       string // mutlak sitemap adresi
	ToplamURL int    // <url> sayısı
	EksikAlan string // eksik alanlar ("3 kayıtta news:title yok")
	EnEski    string // en eski yayın tarihi
	EnYeni    string // en yeni yayın tarihi
	Durum     string // "ok" | "uyari" | "hata"
	Mesaj     string // tek satırlık özet
}

// validateNewsSitemapXML üretilen haber sitemap'ini ayrıştırıp denetler.
func validateNewsSitemapXML(data []byte) NewsSitemapReport {
	rep := NewsSitemapReport{Durum: "ok"}

	type alanlar struct {
		loc, name, lang, date, title string
	}
	var (
		cur     alanlar
		aktif   bool
		alan    string
		eksik   = map[string]int{}
		sayac   = map[string]int{}
		enEski  time.Time
		enYeni  time.Time
		alanAdi = map[string]string{
			"loc": "loc", "name": "news:publication/news:name",
			"lang": "news:publication/news:language", "date": "news:publication_date",
			"title": "news:title",
		}
	)

	Kaydet := func() {
		if !aktif {
			return
		}
		aktif = false
		rep.ToplamURL++
		for k, v := range map[string]string{
			"loc": cur.loc, "name": cur.name, "lang": cur.lang,
			"date": cur.date, "title": cur.title,
		} {
			sayac[k]++
			if strings.TrimSpace(v) == "" {
				eksik[alanAdi[k]]++
			}
		}
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(cur.date)); err == nil {
			if enEski.IsZero() || t.Before(enEski) {
				enEski = t
			}
			if t.After(enYeni) {
				enYeni = t
			}
		}
	}

	dec := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			alan = t.Name.Local
			if alan == "url" {
				aktif = true
				cur = alanlar{}
			}
		case xml.CharData:
			if !aktif {
				continue
			}
			v := strings.TrimSpace(string(t))
			if v == "" {
				continue
			}
			switch alan {
			case "loc":
				cur.loc = v
			case "name":
				cur.name = v
			case "language":
				cur.lang = v
			case "publication_date":
				cur.date = v
			case "title":
				cur.title = v
			}
		case xml.EndElement:
			if t.Name.Local == "url" {
				Kaydet()
			}
			alan = ""
		}
	}
	Kaydet() // kapanmamış <url> varsa

	// Eksik alanları okunur sırayla birleştir (deterministik çıktı).
	var parcalar []string
	for _, ad := range []string{"loc", "news:publication/news:name", "news:publication/news:language", "news:publication_date", "news:title"} {
		if n := eksik[ad]; n > 0 {
			parcalar = append(parcalar, itoa(n)+" kayıtta "+ad+" yok")
		}
	}
	rep.EksikAlan = strings.Join(parcalar, "; ")
	if !enEski.IsZero() {
		rep.EnEski = enEski.UTC().Format("2006-01-02 15:04")
	}
	if !enYeni.IsZero() {
		rep.EnYeni = enYeni.UTC().Format("2006-01-02 15:04")
	}

	switch {
	case !strings.Contains(string(data), "schemas/sitemap-news"):
		rep.Durum = "hata"
		rep.Mesaj = "News ad alanı (xmlns:news) yok — Google bu dosyayı haber sitemap'i olarak kabul etmez."
	case rep.ToplamURL == 0:
		rep.Durum = "uyari"
		rep.Mesaj = "Son 48 saatte yayınlanan içerik yok; sitemap boş. Google News'e kaynak eklemeden önce içerik yayınlayın."
	case rep.EksikAlan != "":
		rep.Durum = "hata"
		rep.Mesaj = "Eksik alan var: " + rep.EksikAlan
	default:
		rep.Mesaj = "Sorun yok: " + itoa(rep.ToplamURL) + " haber kaydı geçerli."
	}
	return rep
}
