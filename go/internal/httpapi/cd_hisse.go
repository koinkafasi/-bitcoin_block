package httpapi

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

var hisseOnbellek = newTTLCache[repo.HisseDetay](60 * time.Second)

// hisseCek önbellekli çekim. İkinci değer false ise veri YOK.
func (s *Server) hisseCek(r *http.Request, sym, aralik string) (repo.HisseDetay, bool) {
	key := sym + "|" + aralik
	return hisseOnbellek.Load(r.Context(), key, func(ctx context.Context) (repo.HisseDetay, bool) {
		d, err := repo.HisseCek(ctx, sym, aralik)
		if err != nil {
			s.logger.Warn("hisse veri", "sembol", sym, "aralik", aralik, "error", err)
			return repo.HisseDetay{}, false
		}
		return d, true
	})
}

var hisseProfilOnbellek = newTTLCache[repo.HisseProfil](10 * time.Minute)

// hisseProfilCek önbellekli profil çekimi. İkinci değer false ise veri YOK.
func (s *Server) hisseProfilCek(ctx context.Context, sym string) (repo.HisseProfil, bool) {
	return hisseProfilOnbellek.Load(ctx, sym, func(ctx context.Context) (repo.HisseProfil, bool) {
		p, err := repo.HisseProfilCek(ctx, sym)
		if err != nil {
			// Uyarı seviyesinde: profil gelmemesi sayfayı düşürmez,
			// yalnız About/piyasa değeri satırları basılmaz.
			s.logger.Warn("hisse profil", "sembol", sym, "error", err)
			return repo.HisseProfil{}, false
		}
		return p, true
	})
}

// handleCDHisse /price/stock/{sembol}.
func (s *Server) handleCDHisse(w http.ResponseWriter, r *http.Request) {
	sym := strings.ToUpper(strings.TrimSpace(r.PathValue("sembol")))
	if !repo.HisseSembolGecerli(sym) {
		s.handleNotFound(w, r)
		return
	}
	ar := repo.HisseAralikBul(r.URL.Query().Get("aralik"))
	kip := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kip")))
	if kip != "advanced" {
		kip = "basic"
	}

	d, ok := s.hisseCek(r, sym, ar.Kod)
	if !ok {
		// Sembol gerçekten yoksa da uç erişilemezse de aynı 404: boş/
		// uydurma bir sayfa indekslenmesin.
		s.handleNotFound(w, r)
		return
	}

	base := strings.TrimRight(s.cfg.BaseURL, "/")
	slug := strings.ToLower(sym)
	yol := "/price/stock/" + slug
	kanonik := base + yol

	// Sekmeler: aktif sekme bağlantısız (aria-current), diğerleri ?aralik=.
	sekmeler := make([]templkit.CDHisseSekme, 0, len(repo.HisseAraliklari))
	for _, a := range repo.HisseAraliklari {
		s := templkit.CDHisseSekme{Kod: a.Kod, Ad: a.Ad, Aktif: a.Kod == ar.Kod}
		if !s.Aktif {
			s.Href = yol + "?aralik=" + a.Kod
			if kip == "advanced" {
				s.Href += "&kip=advanced"
			}
		}
		sekmeler = append(sekmeler, s)
	}

	// Grafik: gerçek kapanış serisi → inline SVG yolu (JS yok).
	var (
		yolSVG, alanSVG      string
		grafikVar            bool
		gmin, gmaks          float64
		grafikUp             bool
		etiketSol, etiketSag string
	)
	closes := make([]float64, 0, len(d.Noktalar))
	for _, n := range d.Noktalar {
		closes = append(closes, n.C)
	}
	if y, a, mn, mx, ok := templkit.CDGrafikYolu(closes); ok {
		yolSVG, alanSVG, gmin, gmaks, grafikVar = y, a, mn, mx, true
		grafikUp = closes[len(closes)-1] >= closes[0]
		// Gün içi aralıkta eksen SAAT yazar (CoinDesk 1D: 09:30–16:00);
		// günlük/haftalık aralıkta tarih.
		if ar.Kod == "1d" {
			etiketSol = hisseSaat(d.Noktalar[0].T, d.Dilim)
			etiketSag = hisseSaat(d.Noktalar[len(d.Noktalar)-1].T, d.Dilim)
		} else {
			etiketSol = hisseKisaTarih(d.Noktalar[0].T, d.Dilim)
			etiketSag = hisseKisaTarih(d.Noktalar[len(d.Noktalar)-1].T, d.Dilim)
			// Aralık yıl sınırını geçiyorsa yıl eklenir ("7 Kas 2025" —
			// "7 Kas … 6 Kas" iki yıl mı aynı yıl mı belirsizdi).
			if yilFarkli(d.Noktalar[0].T, d.Noktalar[len(d.Noktalar)-1].T, d.Dilim) {
				etiketSol = hisseKisaTarihYil(d.Noktalar[0].T, d.Dilim)
				etiketSag = hisseKisaTarihYil(d.Noktalar[len(d.Noktalar)-1].T, d.Dilim)
			}
		}
	}

	var (
		performans []templkit.CDHisseDonem
		profil     repo.HisseProfil
		profilOk   bool
		paralel    sync.WaitGroup
	)
	paralel.Add(2)
	go func() {
		defer paralel.Done()
		if g1, ok := s.hisseCek(r, sym, "1y"); ok {
			performans = hissePerformansSatiri(g1, d.Onceki)
		}
	}()
	go func() {
		defer paralel.Done()
		pctx, iptal := context.WithTimeout(r.Context(), 2*time.Second)
		defer iptal()
		profil, profilOk = s.hisseProfilCek(pctx, sym)
	}()
	paralel.Wait()

	// Advanced kipi: gerçek OHLC mumları (Basic ile aynı viewBox ölçeği).
	var mumlar []templkit.CDHisseMum
	if kip == "advanced" {
		ac := make([]float64, 0, len(d.Noktalar))
		yu := make([]float64, 0, len(d.Noktalar))
		du := make([]float64, 0, len(d.Noktalar))
		ka := make([]float64, 0, len(d.Noktalar))
		for _, n := range d.Noktalar {
			ac = append(ac, n.Acilis)
			yu = append(yu, n.Yuksek)
			du = append(du, n.Dusuk)
			ka = append(ka, n.C)
		}
		mumlar = templkit.CDMumlar(ac, yu, du, ka)
	}

	// Key Stats: yalnız uçtan GELEN alanlar. Değer yoksa satır basılmaz.
	var istatistik []templkit.CDHisseStat
	if profilOk && profil.PiyasaDeger > 0 {
		istatistik = append(istatistik, templkit.CDHisseStat{
			Etiket: "Piyasa değeri",
			Deger:  repo.SayiBicimKisa(profil.PiyasaDeger, d.Para),
		})
	}
	if d.GunMin > 0 && d.GunMaks > 0 {
		istatistik = append(istatistik, templkit.CDHisseStat{
			Etiket: "Gün aralığı",
			Deger:  repo.FiyatBicimP(d.GunMin, d.Para) + " – " + repo.FiyatBicimP(d.GunMaks, d.Para),
		})
	}
	if d.H52Min > 0 && d.H52Maks > 0 {
		istatistik = append(istatistik, templkit.CDHisseStat{
			Etiket: "52 hafta aralığı",
			Deger:  repo.FiyatBicimP(d.H52Min, d.Para) + " – " + repo.FiyatBicimP(d.H52Maks, d.Para),
		})
	}
	if d.Hacim > 0 {
		istatistik = append(istatistik, templkit.CDHisseStat{
			Etiket: "Hacim", Deger: repo.SayiBicim(d.Hacim, 0),
		})
	}
	if d.Onceki > 0 {
		istatistik = append(istatistik, templkit.CDHisseStat{
			Etiket: "Önceki kapanış", Deger: repo.FiyatBicimP(d.Onceki, d.Para),
		})
	}
	// Sektör ve çalışan sayısı da uçtan geliyor (profil).
	if profilOk {
		if profil.Sektor != "" {
			sektor := profil.Sektor
			if profil.Endustri != "" {
				sektor += " · " + profil.Endustri
			}
			istatistik = append(istatistik, templkit.CDHisseStat{Etiket: "Sektör", Deger: sektor})
		}
		if profil.Calisan > 0 {
			istatistik = append(istatistik, templkit.CDHisseStat{
				Etiket: "Çalışan", Deger: repo.SayiBicim(float64(profil.Calisan), 0),
			})
		}
	}

	var ohlcO, ohlcH, ohlcL string
	if o, h, l, ok := repo.HisseGunOHLC(d.Noktalar, d.Dilim); ok {
		if d.GunMaks > 0 {
			h = d.GunMaks
		}
		if d.GunMin > 0 {
			l = d.GunMin
		}
		ohlcO = repo.FiyatBicimP(o, d.Para)
		ohlcH = repo.FiyatBicimP(h, d.Para)
		ohlcL = repo.FiyatBicimP(l, d.Para)
	}

	// Kip bağlantıları mevcut aralığı korur.
	kipTemel := yol
	if ar.Kod != "1d" {
		kipTemel += "?aralik=" + ar.Kod
	}
	kipBasic, kipAdvanced := kipTemel, kipTemel
	if ar.Kod != "1d" {
		kipAdvanced += "&kip=advanced"
	} else {
		kipAdvanced += "?kip=advanced"
	}

	var ld strings.Builder
	ld.WriteString(string(cdHisseJSONLD(s.siteName, kanonik, d.Ad, d.Sembol, d.Borsa, d.Para,
		repo.FiyatBicimP(d.Fiyat, d.Para), repo.DegisimBicim(d.Degisim))))
	ld.WriteString(string(breadcrumbScript([]crumb{
		{Name: s.siteName, URL: base + "/"},
		{Name: "Fiyatlar", URL: base + "/price"},
		{Name: d.Ad, URL: kanonik},
	})))

	v := templkit.CDHisseSayfa{
		Shell:           s.cdShellCtx(r.Context(), r, "price"),
		Title:           d.Ad + " (" + d.Sembol + ") Hisse Fiyatı ve Grafiği — " + s.siteName,
		Desc:            d.Ad + " (" + d.Sembol + ") güncel hisse fiyatı, " + ar.Ad + " grafiği, gün aralığı, 52 hafta aralığı ve hacim.",
		Canonical:       kanonik,
		JSONLD:          ld.String(),
		BaseURL:         base,
		PageCategory:    "price_page",
		ContentCategory: "price",
		Locale:          "tr",

		Ad:      d.Ad,
		Sembol:  d.Sembol,
		Borsa:   d.Borsa,
		Para:    d.Para,
		Fiyat:   repo.FiyatBicimP(d.Fiyat, d.Para),
		Degisim: repo.DegisimBicim(d.Degisim),
		Up:      d.Degisim >= 0,
		AsOf:    hisseAsOf(d.Guncelleme, d.Dilim),

		Ozet: hisseOzet(profil, profilOk),

		Istatistik: istatistik,

		Yol:        yolSVG,
		Alan:       alanSVG,
		GrafikUp:   grafikUp,
		GrafikVar:  grafikVar,
		GrafikMin:  repo.FiyatBicimP(gmin, d.Para),
		GrafikMaks: repo.FiyatBicimP(gmaks, d.Para),
		EtiketSol:  etiketSol,
		EtiketSag:  etiketSag,

		Acilis: ohlcO,
		Yuksek: ohlcH,
		Dusuk:  ohlcL,

		Sekmeler:        sekmeler,
		AktifKod:        ar.Kod,
		Kip:             kip,
		KipHrefBasic:    kipBasic,
		KipHrefAdvanced: kipAdvanced,
		Mumlar:          mumlar,

		Performans: performans,
		Kaynak:     "Kaynak: Yahoo Finance · " + d.Borsa + " · 60 saniye önbellek",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := templkit.CDHisseSayfaView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("hisse render", "error", err)
	}
}

func hissePerformansSatiri(g1 repo.HisseDetay, onceki float64) []templkit.CDHisseDonem {
	var out []templkit.CDHisseDonem
	for _, p := range repo.HissePerformans(g1.Noktalar, onceki, time.Now()) {
		if !p.Gecerli {
			continue
		}
		out = append(out, templkit.CDHisseDonem{
			Ad: p.Ad, Yuzde: repo.DegisimBicim(p.Yuzde), Up: p.Yuzde >= 0,
		})
	}
	return out
}

func hisseOzet(p repo.HisseProfil, ok bool) string {
	if !ok {
		return ""
	}
	return repo.OzetKirp(p.Ozet, 700)
}

// hisseKisaTarih grafik ekseni etiketi: "9 Kas".
func hisseKisaTarih(unix int64, dilim int) string {
	aylar := []string{"Oca", "Şub", "Mar", "Nis", "May", "Haz",
		"Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}
	t := time.Unix(unix, 0).In(time.FixedZone("borsa", dilim))
	return itoa(t.Day()) + " " + aylar[int(t.Month())-1]
}

// hisseKisaTarihYil grafik ekseni: "7 Kas 2025".
func hisseKisaTarihYil(unix int64, dilim int) string {
	t := time.Unix(unix, 0).In(time.FixedZone("borsa", dilim))
	return hisseKisaTarih(unix, dilim) + " " + itoa(t.Year())
}

// yilFarkli iki nokta farklı takvim yılında mı (borsa saatine göre).
func yilFarkli(a, b int64, dilim int) bool {
	yer := time.FixedZone("borsa", dilim)
	return time.Unix(a, 0).In(yer).Year() != time.Unix(b, 0).In(yer).Year()
}

// hisseSaat gün içi eksen etiketi: "09:30" (borsa saati).
func hisseSaat(unix int64, dilim int) string {
	t := time.Unix(unix, 0).In(time.FixedZone("borsa", dilim))
	return pad2(t.Hour()) + ":" + pad2(t.Minute())
}

// hisseAsOf "9 Kas 2026 15:59 EST".
func hisseAsOf(unix int64, dilim int) string {
	if unix <= 0 {
		return ""
	}
	t := time.Unix(unix, 0).In(time.FixedZone(hisseDilimAdi(dilim), dilim))
	aylar := []string{"Oca", "Şub", "Mar", "Nis", "May", "Haz",
		"Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}
	return itoa(t.Day()) + " " + aylar[int(t.Month())-1] + " " + itoa(t.Year()) +
		" " + pad2(t.Hour()) + ":" + pad2(t.Minute()) + " " + hisseDilimAdi(dilim)
}

// hisseDilimAdi borsa saat dilimi kısaltması (ölçülen kayma → ad).
func hisseDilimAdi(sn int) string {
	switch sn {
	case -5 * 3600:
		return "EST"
	case -4 * 3600:
		return "EDT"
	case -6 * 3600:
		return "CST"
	case -7 * 3600:
		return "MST"
	case -8 * 3600:
		return "PST"
	case 0:
		return "UTC"
	}
	return "UTC"
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
