package httpapi

import (
	"net/http"
	"strconv"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ZİYARETÇİ KONUM PANELİ (/admin/ziyaretciler).
//
// Veri: repo.ZiyaretciOzet — ülke/şehir kırılımı + izinli koordinatlar.
// Kaynak: visitor_locations (migration 0130), Cloudflare CF-IPCountry +
// tarayıcı geolocation.

// handleZiyaretciler paneli çizer.
func (s *Server) handleZiyaretciler(w http.ResponseWriter, r *http.Request) {
	gun := 30
	if g, err := strconv.Atoi(r.URL.Query().Get("gun")); err == nil {
		switch g {
		case 7, 30, 90:
			gun = g
		}
	}

	ozet, err := repo.ZiyaretciOzet(r.Context(), s.pool, gun)
	if err != nil {
		s.logger.Error("ziyaretçi özeti", "error", err)
	}

	v := templkit.ZiyaretcilerView{
		Gun:        gun,
		Toplam:     ozet.Toplam,
		Son24:      ozet.Son24,
		Konumlu:    ozet.Konumlu,
		UlkeSayisi: ozet.UlkeSayisi,
		IzinYuzde:  ozet.IzinOraniYuzde,
	}
	for _, u := range ozet.UlkeDagilim {
		v.Ulkeler = append(v.Ulkeler, templkit.ZiyaretciUlkeSatir{
			Kod:  u.Kod,
			Ad:   ulkeAdi(u.Kod),
			Adet: u.Adet,
			Pay:  u.Pay,
		})
	}
	for _, c := range ozet.SehirDagilim {
		v.Sehirler = append(v.Sehirler, templkit.ZiyaretciSehirSatir{
			Ulke:  c.Ulke,
			Sehir: c.Sehir,
			Adet:  c.Adet,
		})
	}
	for _, k := range ozet.Konumlular {
		v.Konumlular = append(v.Konumlular, ziyaretciSatirView(k, true))
	}
	for _, k := range ozet.SonGelenler {
		v.SonGelenler = append(v.SonGelenler, ziyaretciSatirView(k, false))
	}
	v.KonumZorunlu = s.konumZorunlu(r.Context())

	d := s.baseView("tr", "", s.siteName)
	d.Title = "Ziyaretçi Konumu — " + s.siteName
	s.renderTemplPage(w, r, d, templkit.ZiyaretcilerPanel(v))
}

func ziyaretciSatirView(k repo.ZiyaretciSatir, koordinat bool) templkit.ZiyaretciSatir {
	out := templkit.ZiyaretciSatir{
		Ulke:       ulkeAdi(k.Ulke),
		UlkeKod:    kodGoster(k.Ulke),
		Sehir:      k.Sehir,
		Kaynak:     k.Kaynak,
		Izin:       k.Izin,
		SonGorulme: k.SonGorulme.Local().Format("02.01.2006 15:04"),
		IP:         maskeliIP(k.MaskeliIP),
	}
	if koordinat && k.Enlem != nil && k.Boylam != nil {
		out.Enlem = strconv.FormatFloat(*k.Enlem, 'f', 4, 64)
		out.Boylam = strconv.FormatFloat(*k.Boylam, 'f', 4, 64)
		// OpenStreetMap: API anahtarı istemez, ticari kullanıma açık.
		out.HaritaURL = "https://www.openstreetmap.org/?mlat=" + out.Enlem +
			"&mlon=" + out.Boylam + "#map=15/" + out.Enlem + "/" + out.Boylam
		// Sayfa içi harita: Google Maps gömme adresi (anahtar istemez;
		// www.google.com CSP frame-src'de zaten izinli). Koordinat sayısal
		// biçimlendirildiği için adres kaçış gerektirmez.
		out.HaritaGomURL = "https://www.google.com/maps?q=" + out.Enlem + "," +
			out.Boylam + "&z=15&output=embed"
		if k.DogrulukM != nil {
			out.Dogruluk = "+-" + strconv.Itoa(*k.DogrulukM) + " m"
		}
	}
	return out
}

// ulkeAdi ISO-2 kodunu Türkçe ada çevirir.
//
// SÖZLÜK KÜÇÜK TUTULDU: en sık gelen ~45 ülke. Listede olmayan kod
// olduğu gibi gösterilir — eksik çeviri yanlış addan iyidir.
func ulkeAdi(kod string) string {
	if kod == "" || kod == "??" {
		return "Bilinmiyor"
	}
	if ad, ok := ulkeAdlari[kod]; ok {
		return ad
	}
	return kod
}

var ulkeAdlari = map[string]string{
	"TR": "Türkiye", "US": "ABD", "DE": "Almanya", "GB": "Birleşik Krallık",
	"FR": "Fransa", "NL": "Hollanda", "IT": "İtalya", "ES": "İspanya",
	"RU": "Rusya", "UA": "Ukrayna", "AZ": "Azerbaycan", "KZ": "Kazakistan",
	"UZ": "Özbekistan", "KG": "Kırgızistan", "TM": "Türkmenistan",
	"AE": "BAE", "SA": "Suudi Arabistan", "QA": "Katar", "KW": "Kuveyt",
	"IN": "Hindistan", "PK": "Pakistan", "BD": "Bangladeş", "ID": "Endonezya",
	"MY": "Malezya", "SG": "Singapur", "TH": "Tayland", "VN": "Vietnam",
	"CN": "Çin", "JP": "Japonya", "KR": "Güney Kore", "TW": "Tayvan",
	"HK": "Hong Kong", "AU": "Avustralya", "NZ": "Yeni Zelanda",
	"CA": "Kanada", "MX": "Meksika", "BR": "Brezilya", "AR": "Arjantin",
	"CL": "Şili", "CO": "Kolombiya", "PE": "Peru",
	"SE": "İsveç", "NO": "Norveç", "DK": "Danimarka", "FI": "Finlandiya",
	"BE": "Belçika", "AT": "Avusturya", "CH": "İsviçre", "PL": "Polonya",
	"CZ": "Çekya", "HU": "Macaristan", "RO": "Romanya", "BG": "Bulgaristan",
	"GR": "Yunanistan", "PT": "Portekiz", "IE": "İrlanda", "IL": "İsrail",
	"EG": "Mısır", "MA": "Fas", "DZ": "Cezayir", "TN": "Tunus", "LY": "Libya",
	"ZA": "Güney Afrika", "NG": "Nijerya", "KE": "Kenya", "ET": "Etiyopya",
	"IR": "İran", "IQ": "Irak", "SY": "Suriye", "LB": "Lübnan", "JO": "Ürdün",
	"GE": "Gürcistan", "AM": "Ermenistan", "MD": "Moldova", "RS": "Sırbistan",
	"HR": "Hırvatistan", "SI": "Slovenya", "SK": "Slovakya", "LT": "Litvanya",
	"LV": "Letonya", "EE": "Estonya", "BY": "Belarus", "AL": "Arnavutluk",
	"BA": "Bosna-Hersek", "MK": "Kuzey Makedonya", "XK": "Kosova",
}

// kodGoster ülke kodunu parantez içi gösterime hazırlar.
//
// ÖLÇÜLEN KUSUR (gerçek tarayıcı, 14 Kasım 2026): kod boşken panel
// "Bilinmiyor ()" basıyordu — boş parantez. Artık kod yoksa hiç
// basılmıyor.
func kodGoster(kod string) string {
	if kod == "" || kod == "??" {
		return ""
	}
	return kod
}
