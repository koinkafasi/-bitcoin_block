package httpapi

import (
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ZİYARETÇİ PROFİLİ (/admin/trafik-guvenligi/ziyaretci/{anahtar}).
//
// SORU: "bu ziyaretçi ne yapmaya çalışıyordu?" Panel listesi sınıfı ve
// puanı veriyordu; ikisi de tek satırlık bir yargı. Niyet ise istek
// dizisinden çıkar — bu sayfa o diziyi gösterir.
//
// NE GÖSTERİR:
//   · kimlik (maskesiz IP — yönetici bu sayfada istedi; liste maskeli kalır)
//   · ağ (ASN, sağlayıcı, ülke, aynı IP'den gelen DİĞER kimlikler)
//   · davranış (tür dağılımı, en sık yollar, son 200 istek)
//   · çıkarım (niyetCikar: tek cümlelik yorum + gerekçe)
//
// GİZLİLİK: maskesiz IP yalnız oturum açmış yöneticiye, yalnız bu
// sayfada. Liste sayfası maskeli kalıyor (omuz üstünden okuma riski).
// Sorgu dizgeleri yazarken maskelenmişti (sorguTemizle); burada ham
// jeton görünmez.

// handleTrafikZiyaretci tek ziyaretçinin profilini çizer.
func (s *Server) handleTrafikZiyaretci(w http.ResponseWriter, r *http.Request) {
	anahtar := strings.TrimSpace(r.PathValue("anahtar"))
	if anahtar == "" || len(anahtar) > 200 {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	risk, var_, err := repo.RiskTek(ctx, s.pool, anahtar)
	if err != nil {
		s.logger.Error("ziyaretçi profili okunamadı", "error", err)
	}

	olaylar, err := repo.Olaylar(ctx, s.pool, anahtar, 200)
	if err != nil {
		s.logger.Error("ziyaretçi olayları okunamadı", "error", err)
	}
	sayim, toplam, engellenen, err := repo.OlaySayim(ctx, s.pool, anahtar)
	if err != nil {
		s.logger.Error("ziyaretçi olay sayımı", "error", err)
	}
	yollar, _ := repo.OlayYolSayim(ctx, s.pool, anahtar, 15)
	akrabalar, _ := repo.OlayIPAkrabalari(ctx, s.pool, risk.IP, anahtar, 10)

	niyet := NiyetCikar(sayim, toplam, engellenen)

	v := templkit.TrafikZiyaretciView{
		Anahtar:     anahtar,
		Var:         var_,
		MaskeliIP:   maskeliIP(risk.IP),
		IP:          risk.IP,
		IPAgg:       risk.IPAgg,
		Ulke:        ulkeAdi(risk.Ulke),
		UlkeKod:     risk.Ulke,
		ASN:         risk.ASN,
		Saglayici:   asnSaglayiciEtiket(risk.ASNAd),
		ASNAd:       risk.ASNAd,
		Tur:         risk.ASNTur,
		TurAd:       turAdlari[risk.ASNTur],
		Sinif:       risk.Sinif,
		SinifAd:     sinifGoster(risk.Sinif),
		Puan:        risk.Puan,
		IstekSayisi: risk.IstekSayisi,
		IPDegisim:   risk.IPDegisim,
		UlkeDegisim: risk.UlkeDegisim,
		IlkGorulme:  yerelZaman(risk.IlkGorulme),
		SonGorulme:  yerelZaman(risk.SonGorulme),
		UV:          uaOzet(risk.UserAgent),
		UserAgent:   risk.UserAgent,
		WebRTCIP:    maskeliIP(risk.WebRTCIP),
		Temiz:       risk.Sinif == "temiz",
		VPN:         risk.VPN,
		Proxy:       risk.Proxy,
		Hosting:     risk.Hosting,
		Tor:         risk.Tor,
		Bot:         risk.Bot,

		NiyetBaslik:   niyet.Baslik,
		NiyetAciklama: niyet.Aciklama,
		NiyetSeviye:   niyet.Seviye,
		OlaySayisi:    toplam,
		Engellenen:    engellenen,
	}

	v.Sinyaller = append(v.Sinyaller, sinyalListesi(risk.Sinyaller)...)

	// Tür dağılımı — çubuklu.
	for _, t := range []string{turIcerik, turBesleme, turArama, turAPI,
		turGiris, turYonetim, turZafiyet, turTuzak, turDiger} {
		n := sayim[t]
		if n == 0 {
			continue
		}
		pay := 0
		if toplam > 0 {
			pay = (n*100 + toplam/2) / toplam
		}
		v.Turler = append(v.Turler, templkit.ZiyaretciTurSatir{
			Kod: t, Ad: olayTurAdlari[t], Adet: n, Pay: pay,
		})
	}

	for _, y := range yollar {
		v.Yollar = append(v.Yollar, templkit.ZiyaretciYolSatir{
			Yol: y.Yol, Adet: y.Adet, Durum: durumMetni(y.DurumMin, y.DurumMax),
		})
	}

	for _, a := range akrabalar {
		v.Akrabalar = append(v.Akrabalar, templkit.ZiyaretciAkrabaSatir{
			Anahtar: a.Anahtar, Adet: a.Adet, Son: yerelZaman(a.Son),
		})
	}

	for _, o := range olaylar {
		v.Olaylar = append(v.Olaylar, templkit.ZiyaretciOlaySatir{
			Zaman:      o.Zaman.Local().Format("02.01 15:04:05"),
			Yontem:     o.Yontem,
			Yol:        o.Yol,
			Sorgu:      o.Sorgu,
			Durum:      o.Durum,
			Tur:        o.Tur,
			TurAd:      olayTurAdlari[o.Tur],
			Kaynak:     o.Kaynak,
			KaynakAd:   kaynakAdlari[o.Kaynak],
			Referer:    kirp(o.Referer, 120),
			Engellendi: o.Engellendi,
		})
	}

	d := s.baseView("tr", "/admin", "Ziyaretçi profili")
	s.renderTemplPage(w, r, d, templkit.TrafikZiyaretciPanel(v))
}

// durumMetni min..max HTTP kodlarını tek etikete indirger.
func durumMetni(min, max int) string {
	if min == max {
		return itoa(min)
	}
	return itoa(min) + "–" + itoa(max)
}

// yerelZaman zamanı panel biçimine çevirir; sıfır zaman "—"
// (panelde 0001-01-01 görünmesin).
func yerelZaman(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("02.01.2006 15:04")
}

// uaOzet user-agent'tan okunur kısa kimlik çıkarır.
//
// NEDEN TAM UA YETMİYOR: tam dize 150+ karakter ve panelde hücreyi
// bozuyordu (sağlayıcı hücresiyle aynı kusur, 14 Kasım 2026). Kısa özet
// satırda, tam dize başlık (title) olarak durur — hem taranabilir hem
// kopyalanabilir.
//
// SIRALAMA: bot imzaları EN ÖNCE. Chrome UA'sı içinde "Safari" de yazar;
// naif sıralama her Chrome'u Safari gösterirdi (klasik hata).
func uaOzet(ua string) string {
	u := strings.ToLower(ua)
	if strings.TrimSpace(u) == "" {
		return "boş"
	}
	// 1) Otomasyon / bot
	for _, b := range []struct{ imza, ad string }{
		{"googlebot", "Googlebot"},
		{"bingbot", "Bingbot"},
		{"yandexbot", "YandexBot"},
		{"duckduckbot", "DuckDuckBot"},
		{"applebot", "Applebot"},
		{"baiduspider", "Baiduspider"},
		{"curl/", "curl"},
		{"wget/", "wget"},
		{"python-requests", "python-requests"},
		{"python-urllib", "python-urllib"},
		{"go-http-client", "go-http-client"},
		{"scrapy", "Scrapy"},
		{"okhttp", "OkHttp"},
		{"java/", "Java"},
		{"libwww-perl", "libwww-perl"},
		{"axios", "axios"},
		{"node-fetch", "node-fetch"},
		{"headlesschrome", "HeadlessChrome"},
		{"phantomjs", "PhantomJS"},
		{"selenium", "Selenium"},
		{"puppeteer", "Puppeteer"},
		{"playwright", "Playwright"},
	} {
		if strings.Contains(u, b.imza) {
			return b.ad
		}
	}
	// 2) Tarayıcı + işletim sistemi
	tarayici := ""
	switch {
	case strings.Contains(u, "edg/"):
		tarayici = "Edge"
	case strings.Contains(u, "opr/") || strings.Contains(u, "opera"):
		tarayici = "Opera"
	case strings.Contains(u, "firefox/"):
		tarayici = "Firefox"
	case strings.Contains(u, "chrome/") || strings.Contains(u, "crios/"):
		tarayici = "Chrome"
	case strings.Contains(u, "safari/"):
		tarayici = "Safari"
	}
	isletim := ""
	switch {
	case strings.Contains(u, "windows nt 10"), strings.Contains(u, "windows nt 11"):
		isletim = "Windows"
	case strings.Contains(u, "android"):
		isletim = "Android"
	case strings.Contains(u, "iphone"), strings.Contains(u, "ipad"), strings.Contains(u, "ios"):
		isletim = "iOS"
	case strings.Contains(u, "mac os x"), strings.Contains(u, "macintosh"):
		isletim = "macOS"
	case strings.Contains(u, "linux"):
		isletim = "Linux"
	}
	switch {
	case tarayici != "" && isletim != "":
		return tarayici + " · " + isletim
	case tarayici != "":
		return tarayici
	case isletim != "":
		return isletim
	}
	return "tanınmayan"
}
