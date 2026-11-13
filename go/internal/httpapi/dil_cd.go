package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/webui/templkit"
)

// dilIcerikTTL dil seçicisinin içerik envanteri tazeliği. Yeni bir dilde ilk
// haber yayınlandığında seçenek bu süre içinde kendiliğinden görünür.
const dilIcerikTTL = 60 * time.Second

// CD TEMASI DİL DESTEĞİ — bu dosya, CoinDesk temasının çok dilli olmasını
// sağlayan tek merkezdir.
//
// ÖLÇÜLEN SORUN (13 Kasım 2026): DB'de 24 dil (`locales`, hepsi enabled),
// 179 çeviri anahtarı vardı; buna rağmen CD temasında dil SEÇİCİ hiç
// basılmıyordu, yalnız tıklanamaz bir "TR" etiketi vardı. Neden:
// `cdShellCtx` dil listesini SABİT kodluyordu:
//
//	Diller: []templkit.CDDil{{Kod: "tr", Ad: "Türkçe", Aktif: true}}
//
// Şablon `len(s.Diller) > 1` koşuluna baktığı için 1 > 1 hiç sağlanmıyordu.
// Yani altyapı (DB + i18n paketi) hazırdı, tema onu OKUMUYORDU.
//
// İkinci sorun: `Locale: "tr"` sabitti → her sayfa `<html lang="tr">`,
// `content_language=tr`, `og:locale=tr` basıyordu. `/en/` 200 dönüyordu ama
// İngilizce sayfa "Türkçe" olduğunu söylüyordu (Google için yanlış sinyal).

// cdYerel isteğin dil kodunu döndürür.
//
// Kaynak sırası: yol öneki (`/en/...`) → site varsayılanı. Yol öneki
// `handlePublicSegment` ve `localeAliasHandler` tarafından `lang` path
// değerine yazılır; CD rotaları öneksiz olduğunda bu değer boş kalır ve
// varsayılan dil kullanılır.
func (s *Server) cdYerel(r *http.Request) string {
	if r == nil {
		return "tr"
	}
	return langOf(r, s)
}

// cdDiller dil seçicisinin satırlarını üretir.
//
// KURAL (kullanıcı isteği, 13 Kasım 2026): seçenekler ELLE yazılmaz; yalnız
// O DİLDE YAYINDA İÇERİK OLAN diller listelenir. Ölçüm: tr 3361, en 479,
// diğer 22 dilde 0 içerik vardı — 24 dil göstermek boş sayfalara götürürdü
// (kullanıcı için kayıp, Google için kopya/boş dizin).
//
// Yeni bir dilde ilk haber yayınlandığında seçenek KENDİLİĞİNDEN belirir
// (önbellek 60 sn, bkz. dilIcerikTTL). Varsayılan dil (tr) her zaman kalır:
// ana dil hiçbir zaman kaybolmamalı.
//
// Her satır, AYNI SAYFANIN o dildeki adresini taşır (yol korunur); böylece
// dil değiştirmek kullanıcıyı ana sayfaya atmaz. Makale adresleri dile göre
// değiştiği (slug + tarih) için orada ana sayfaya düşülür — 404'e götürmek
// yol kaybından daha kötüdür.
func (s *Server) cdDiller(r *http.Request, aktif string) []templkit.CDDil {
	varsayilan := "tr"
	if s.i18n != nil {
		varsayilan = s.i18n.Default()
	}

	kalan := "/"
	if r != nil {
		kalan = r.URL.Path
		if kalan == "" {
			kalan = "/"
		}
		if s.i18n != nil {
			if loc, rest := s.i18n.StripLocale(kalan); loc != "" {
				kalan = rest
			}
		}
	}
	if cdMakaleYoluMu(kalan) {
		kalan = "/"
	}

	if s.i18n == nil {
		return []templkit.CDDil{{Kod: varsayilan, Ad: "Türkçe", Aktif: true, Href: kalan}}
	}
	yereller := s.i18n.Enabled()
	if len(yereller) == 0 {
		return []templkit.CDDil{{Kod: varsayilan, Ad: "Türkçe", Aktif: true, Href: kalan}}
	}

	dolu := s.cdIcerikDilleri(r)

	out := make([]templkit.CDDil, 0, len(yereller))
	for _, l := range yereller {
		// Varsayılan dil her zaman; diğerleri yalnız içeriği varsa.
		if l.Code != varsayilan && !dolu[l.Code] {
			continue
		}
		out = append(out, templkit.CDDil{
			Kod:   l.Code,
			Ad:    l.NativeName,
			Aktif: l.Code == aktif,
			Href:  cdDilHref(l.Code, kalan),
		})
	}
	// Aktif dil listede yoksa (ör. /de/ sayfası, Almanca içerik henüz yok)
	// etiketi kaybetmemek için ekle: kullanıcı hangi dilde olduğunu görmeli.
	bulundu := false
	for _, d := range out {
		if d.Kod == aktif {
			bulundu = true
			break
		}
	}
	if !bulundu && aktif != "" {
		ad := aktif
		for _, l := range yereller {
			if l.Code == aktif {
				ad = l.NativeName
				break
			}
		}
		out = append(out, templkit.CDDil{Kod: aktif, Ad: ad, Aktif: true, Href: cdDilHref(aktif, kalan)})
	}
	return out
}

// cdIcerikDilleri yayında içeriği olan dil kodlarını döndürür (önbellekli).
func (s *Server) cdIcerikDilleri(r *http.Request) map[string]bool {
	siteID := int64(1)
	if r != nil {
		siteID = siteIDOr1(s.curSiteID(r))
	}
	v, ok := s.dilIcerikCache.Load(r.Context(), "diller:"+strconv.FormatInt(siteID, 10),
		func(ctx context.Context) (map[string]bool, bool) {
			if s.pool == nil {
				return nil, false
			}
			rows, err := s.pool.Query(ctx,
				`SELECT DISTINCT locale FROM contents
				  WHERE site_id=$1 AND status='yayinda' AND locale <> ''`, siteID)
			if err != nil {
				s.logger.Warn("dil listesi: içerik dilleri okunamadı", "error", err)
				return nil, false
			}
			defer rows.Close()
			m := map[string]bool{}
			for rows.Next() {
				var k string
				if rows.Scan(&k) == nil {
					m[k] = true
				}
			}
			return m, true
		})
	if !ok {
		return map[string]bool{}
	}
	return v
}

// cdDilHref bir dil kodunu yola uygular. Türkçe öneksizdir (varsayılan dil).
func cdDilHref(kod, kalan string) string {
	if kalan == "" {
		kalan = "/"
	}
	if !strings.HasPrefix(kalan, "/") {
		kalan = "/" + kalan
	}
	if kod == "" || kod == "tr" {
		return kalan
	}
	if kalan == "/" {
		return "/" + kod + "/"
	}
	return "/" + kod + kalan
}

// cdMakaleYoluMu yolun bir makale adresi olup olmadığını söyler.
//
// İki biçim var: tarihli kanonik (`/kategori/2026/11/13/slug`) ve eski
// kimlikli (`/slug-12345`). İkisinde de slug dile göre değişir.
func cdMakaleYoluMu(yol string) bool {
	p := strings.Trim(yol, "/")
	if p == "" {
		return false
	}
	segs := strings.Split(p, "/")
	if _, ok := makaleTarihDeseni(segs); ok {
		return true
	}
	return articleIDFromSegment(segs[len(segs)-1]) > 0
}

// cdDilYolu dil önekli CD yollarını (`/en/markets`, `/en/price/bitcoin` …)
// doğru CD handler'ına dağıtır. İstek karşılandıysa true döner.
//
// NEDEN HANDLER'DA: Go mux bu desenleri ayrı ayrı kaydetmeye izin vermiyor
// (`/{lang}/markets` ile `/price/{coin}` çakışan sayılıyor ve routes()
// panikliyor — ölçüldü 13 Kasım 2026). Tek joker `/{lang}/{rest...}`
// altında dağıtmak çakışmayı tümüyle ortadan kaldırıyor.
func (s *Server) cdDilYolu(w http.ResponseWriter, r *http.Request, segs []string) bool {
	if len(segs) == 0 {
		return false
	}
	seg := segs[0]

	for _, b := range cdBolumler {
		if seg == b && len(segs) == 1 {
			s.cdBolumHandler(b)(w, r)
			return true
		}
	}

	switch seg {
	case "price":
		if len(segs) == 1 {
			s.handleCDPrice(w, r)
			return true
		}
		// `/price/stock…` Türkiye'ye özgü bir dizin; dil önekiyle anlamlı
		// değil, eski ağaca bırakılır.
		if len(segs) == 2 && segs[1] != "stock" {
			r.SetPathValue("coin", segs[1])
			s.handleCDCoin(w, r)
			return true
		}
	case "videos":
		if len(segs) == 1 {
			s.handleCDVideos(w, r)
			return true
		}
		if len(segs) == 2 && segs[1] == "shorts" {
			s.handleCDShorts(w, r)
			return true
		}
	case "tag":
		if len(segs) == 2 {
			r.SetPathValue("slug", segs[1])
			s.handleCDTag(w, r)
			return true
		}
	case "podcasts":
		if len(segs) == 1 {
			s.handleCDPodcasts(w, r)
			return true
		}
	case "newsletters":
		if len(segs) == 1 {
			s.handleCDNewsletters(w, r)
			return true
		}
	case "canli-yayin":
		if len(segs) == 1 {
			s.handleCDCanli(w, r)
			return true
		}
	case "sponsored-content":
		if len(segs) == 1 {
			s.handleCDSponsored(w, r)
			return true
		}
	}
	return false
}

// cdDilDususu, içeriği OLMAYAN bir dil önekini varsayılan dile yönlendirir.
//
// ÖLÇÜLEN SORUN (13 Kasım 2026, canlı): `/de/markets` 200 dönüyordu ama
// arayüz Türkçe ve `<html lang="de">` — yani sayfa Almanca olduğunu
// söylüyordu. Google için yanlış dil sinyali, kullanıcı için anlamsız
// adres. Seçicide zaten görünmüyor (içeriği olan diller listelenir), bu
// yüzden buraya yalnız elle/ dış bağlantıyla gelen düşer.
//
// Davranış: aynı sayfanın öneksiz adresine 301. İçerik kaybı yok, dil
// iddiası ortadan kalkıyor, tek kanonik adres kalıyor.
// Varsayılan dil (tr) ve içeriği olan diller dokunulmaz.
func (s *Server) cdDilDususu(w http.ResponseWriter, r *http.Request, lang string) bool {
	if r == nil || lang == "" {
		return false
	}
	varsayilan := "tr"
	if s.i18n != nil {
		varsayilan = s.i18n.Default()
	}
	if lang == varsayilan {
		return false
	}
	if dolu := s.cdIcerikDilleri(r); dolu[lang] {
		return false
	}
	// Öneki soy: /de/markets -> /markets ; /de/ -> /
	hedef := "/"
	if s.i18n != nil {
		if _, rest := s.i18n.StripLocale(r.URL.Path); rest != "" {
			hedef = rest
		}
	}
	if r.URL.RawQuery != "" {
		hedef += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, hedef, http.StatusMovedPermanently)
	return true
}
