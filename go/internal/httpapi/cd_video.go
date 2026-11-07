package httpapi

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
	"theblock/site/internal/youtube"
)

// YouTube video sayfaları.
//
// NEDEN AYRI DOSYA: /videos eskiden `contents` tablosundaki tür="video"
// kayıtlarını listeliyordu; yayında video içeriği hiç yoktu (boş durum).
// Kullanıcı kararı (7 Kasım 2026): video alanı GERÇEK YouTube kanalından
// beslenecek — /videos → kanal videoları, /videos/shorts → shorts,
// /canli-yayin → canlı yayınlar. Veri scheduler'ın doldurduğu
// `youtube_icerik` tablosundan okunur (migrations/0122).
//
// KANAL AYARI: `settings/youtube` (panel: /admin/video-yayin). Kanal
// değiştirilirse kod değişmez.

// cdVideoSekmeler üç video bölümünün üst gezinmesi.
func (s *Server) cdVideoSekmeler(aktif string) []templkit.CDVideoSekme {
	hepsi := []struct{ Kod, Ad, Href string }{
		{"video", "Videolar", "/videos"},
		{"short", "Shorts", "/videos/shorts"},
		{"canli", "Canlı Yayın", "/canli-yayin"},
	}
	out := make([]templkit.CDVideoSekme, 0, len(hepsi))
	for _, h := range hepsi {
		out = append(out, templkit.CDVideoSekme{Label: h.Ad, Href: h.Href, Aktif: h.Kod == aktif})
	}
	return out
}

// ytHref kaydın site içi adresi. Tüm türler tek izleme sayfasına gider;
// short'lar dikey oynatıcıyla açılır.
func ytHref(k repo.YoutubeKayit) string { return "/video/" + k.VideoID }

// ytZaman göreli zaman metni (yayın tarihi yoksa boş).
func ytZaman(k repo.YoutubeKayit) string {
	if k.YayinAt == nil {
		return ""
	}
	return trGoreliZaman(*k.YayinAt, time.Now())
}

// ytKutu kaydı ızgara kartına çevirir.
func ytKutu(k repo.YoutubeKayit) templkit.CDVideoKutu {
	return templkit.CDVideoKutu{
		VideoID: k.VideoID,
		Baslik:  k.Baslik,
		Image:   k.Gorsel,
		Sure:    youtube.Sure(k.SureSn),
		Zaman:   ytZaman(k),
		Href:    ytHref(k),
		Dikey:   k.Tur == "short",
		Canli:   k.Canli,
	}
}

// ytKutular toplu çeviri.
func ytKutular(kayitlar []repo.YoutubeKayit) []templkit.CDVideoKutu {
	out := make([]templkit.CDVideoKutu, 0, len(kayitlar))
	for _, k := range kayitlar {
		out = append(out, ytKutu(k))
	}
	return out
}

// ytSatir "Sıradaki" satırı.
func ytSatir(k repo.YoutubeKayit) templkit.CDVideoSatir {
	return templkit.CDVideoSatir{
		VideoID: k.VideoID,
		Baslik:  k.Baslik,
		Image:   k.Gorsel,
		Sure:    youtube.Sure(k.SureSn),
		Zaman:   ytZaman(k),
		Href:    ytHref(k),
		Canli:   k.Canli,
	}
}

// ytSatirlar toplu çeviri (en fazla adet).
func ytSatirlar(kayitlar []repo.YoutubeKayit, haric string, adet int) []templkit.CDVideoSatir {
	out := make([]templkit.CDVideoSatir, 0, adet)
	for _, k := range kayitlar {
		if k.VideoID == haric {
			continue
		}
		out = append(out, ytSatir(k))
		if len(out) >= adet {
			break
		}
	}
	return out
}

// ytGridHeroHariç ızgara kartları (kahraman hariç, en fazla adet).
func ytGridHeroHariç(kayitlar []repo.YoutubeKayit, haric string, adet int) []templkit.CDVideoKutu {
	out := make([]templkit.CDVideoKutu, 0, adet)
	for _, k := range kayitlar {
		if k.VideoID == haric {
			continue
		}
		out = append(out, ytKutu(k))
		if len(out) >= adet {
			break
		}
	}
	return out
}

// ytSonraki oynatıcının bitiş ekranı için "sıradaki" bilgisini doldurur.
func ytSonraki(h *templkit.CDVideoHero, satirlar []templkit.CDVideoSatir) {
	if len(satirlar) == 0 {
		return
	}
	h.SonrakiID = satirlar[0].VideoID
	h.SonrakiBaslik = satirlar[0].Baslik
}

// ytHero kahraman bloğu üretir.
func ytHero(k repo.YoutubeKayit, etiket string) templkit.CDVideoHero {
	spot := strings.TrimSpace(k.Aciklama)
	if len([]rune(spot)) > 180 {
		spot = string([]rune(spot)[:180]) + "…"
	}
	gorsel := k.Gorsel
	if k.Tur != "short" && k.VideoID != "" {
		// Büyük oynatıcı için yüksek çözünürlük; yoksa tablodaki kapak.
		gorsel = youtube.BuyukGorselURL(k.VideoID)
	}
	return templkit.CDVideoHero{
		VideoID: k.VideoID,
		Baslik:  k.Baslik,
		Spot:    spot,
		Image:   gorsel,
		Zaman:   ytZaman(k),
		Sure:    youtube.Sure(k.SureSn),
		Etiket:  etiket,
		Href:    ytHref(k),
		Canli:   k.Canli,
	}
}

// ytAyar panel ayarlarını istek bağlamından okur (DB hatasında varsayılan).
func (s *Server) ytAyar(r *http.Request) youtube.Ayar {
	return youtube.AyarlariOku(r.Context(), s.pool)
}

// handleCDVideos /videos
func (s *Server) handleCDVideos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ayar := s.ytAyar(r)

	var videolar, shorts, yayinlar []repo.YoutubeKayit
	if ayar.VideoGoster {
		videolar, _ = repo.YoutubeListele(ctx, s.pool, "video", 24)
	}
	if ayar.ShortGoster {
		shorts, _ = repo.YoutubeListele(ctx, s.pool, "short", 8)
	}
	if ayar.CanliGoster {
		yayinlar, _ = repo.YoutubeListele(ctx, s.pool, "stream", 8)
	}

	heroIdx := 0
	if one, err := repo.YoutubeOneCikan(ctx, s.pool, "video"); err == nil && one != nil {
		// Öne çıkarılan varsa listenin başına al.
		videolar = append([]repo.YoutubeKayit{*one}, videolar...)
	}

	var hero templkit.CDVideoHero
	heroID := ""
	if len(videolar) > 0 {
		hero = ytHero(videolar[heroIdx], "Son Videolar")
		heroID = videolar[heroIdx].VideoID
	}

	// "Sıradaki" listesi: SIRA ÖNEMLİ — önce videolar, sonra canlı yayınlar,
	// en son short'lar. Kanal ağırlıklı short üretiyor; yalnız videolara
	// bakılsaydı liste çoğu gün boş kalır ve sağ kolon (416 px) hiç
	// basılmazdı — oynatıcı 937 yerine 1377 px'e yayılıp CoinDesk
	// ölçüsünden sapardı (ölçüldü: 7 Kasım 2026).
	havuz := make([]repo.YoutubeKayit, 0, len(videolar)+len(yayinlar)+len(shorts))
	havuz = append(havuz, videolar...)
	havuz = append(havuz, yayinlar...)
	havuz = append(havuz, shorts...)
	siradaki := ytSatirlar(havuz, heroID, 6)
	ytSonraki(&hero, siradaki)

	v := templkit.CDVideoPage{
		Shell:     s.cdShellCtx(ctx, "videos"),
		Title:     "Videolar — " + s.siteName,
		Desc:      "Kripto piyasaları, borsa ve yatırım üzerine videolar; canlı yayınlar ve kısa klipler.",
		Canonical: s.cfg.BaseURL + "/videos",
		Baslik:    "Videolar",
		Sekmeler:  s.cdVideoSekmeler("video"),
		Hero:      hero,
		Siradaki:  siradaki,
		Videolar:  ytGridHeroHariç(videolar, heroID, 12),
		Shorts:    ytKutular(shorts),
		Yayinlar:  ytKutular(yayinlar),
		BosMetin:  "Henüz video çekilmedi. YouTube kanalına yeni video düştüğünde bu sayfa otomatik dolar.",
		JSONLD:    string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/videos")),
	}
	s.renderCDVideoPage(w, r, v)
}

// handleCDShorts /videos/shorts
func (s *Server) handleCDShorts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ayar := s.ytAyar(r)

	var shorts, videolar []repo.YoutubeKayit
	if ayar.ShortGoster {
		shorts, _ = repo.YoutubeListele(ctx, s.pool, "short", 24)
	}
	if ayar.VideoGoster {
		videolar, _ = repo.YoutubeListele(ctx, s.pool, "video", 8)
	}
	if one, err := repo.YoutubeOneCikan(ctx, s.pool, "short"); err == nil && one != nil {
		shorts = append([]repo.YoutubeKayit{*one}, shorts...)
	}

	var hero templkit.CDVideoHero
	heroID := ""
	if len(shorts) > 0 {
		hero = ytHero(shorts[0], "Shorts")
		heroID = shorts[0].VideoID
	}

	// Sıradaki: short'lar önce, sonra normal videolar.
	havuz := make([]repo.YoutubeKayit, 0, len(shorts)+len(videolar))
	havuz = append(havuz, shorts...)
	havuz = append(havuz, videolar...)
	siradaki := ytSatirlar(havuz, heroID, 6)
	ytSonraki(&hero, siradaki)

	v := templkit.CDVideoPage{
		Shell:     s.cdShellCtx(ctx, "videos"),
		Title:     "Shorts — " + s.siteName,
		Desc:      "Kısa videolar: piyasa hareketleri, günün öne çıkan gelişmeleri ve yatırım notları.",
		Canonical: s.cfg.BaseURL + "/videos/shorts",
		Baslik:    "Shorts",
		Sekmeler:  s.cdVideoSekmeler("short"),
		Hero:      hero,
		Siradaki:  siradaki,
		Videolar:  ytKutular(videolar),
		Shorts:    ytGridHeroHariç(shorts, heroID, 12),
		BosMetin:  "Henüz kısa video yayınlanmadı. Kanal yeni bir short yüklediğinde burada görünür.",
		Dikey:     true,
		JSONLD:    string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/videos/shorts")),
	}
	s.renderCDVideoPage(w, r, v)
}

// handleCDCanli /canli-yayin
//
// OYNATICI KANALA BAĞLANIR (`live_stream?channel=`): yayın kimliği her gün
// değişse de doğru akış açılır. Canlı yoksa YouTube "yayın yok" çerçevesi
// gösterir; sayfa ayrıca son yayınları listeler.
func (s *Server) handleCDCanli(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ayar := s.ytAyar(r)

	var yayinlar []repo.YoutubeKayit
	yayinlar, _ = repo.YoutubeListele(ctx, s.pool, "stream", 24)

	canliVar := false
	var hero templkit.CDVideoHero
	heroID := ""
	for _, k := range yayinlar {
		if k.Canli {
			canliVar = true
			hero = ytHero(k, "Canlı Yayın")
			heroID = k.VideoID
			break
		}
	}
	// Elle işaretleme: settings/youtube.canli_video_id doluysa o kimlik canlı
	// sayılır (panelden yönetilir).
	if ayar.CanliVideoID != "" {
		canliVar = true
		hero.VideoID = ayar.CanliVideoID
		if hero.Baslik == "" {
			hero.Baslik = "Canlı Yayın"
			for _, k := range yayinlar {
				if k.VideoID == ayar.CanliVideoID {
					hero.Baslik = k.Baslik
					if hero.Image == "" {
						hero.Image = k.Gorsel
					}
				}
			}
		}
	}
	if heroID == "" && len(yayinlar) > 0 {
		heroID = yayinlar[0].VideoID
	}
	if hero.Baslik == "" {
		hero = templkit.CDVideoHero{
			Baslik: "Canlı Yayın",
			Spot:   "Kanalın canlı yayınları burada açılır. Şu an yayın yoksa YouTube'un canlı akış çerçevesi görünür.",
			Image:  "",
			Etiket: "Canlı Yayın",
		}
	}
	hero.Canli = canliVar
	hero.Sure = ""

	kanalID := ""
	if ayar.CanliGoster && youtube.GecerliKanalID(ayar.KanalID) {
		kanalID = ayar.KanalID
	}

	v := templkit.CDVideoPage{
		Shell:           s.cdShellCtx(ctx, "videos"),
		Title:           "Canlı Yayın — " + s.siteName,
		Desc:            "Bitcoin, borsa ve piyasa değerlendirmeleri; canlı yayın kayıtları ve güncel yayın akışı.",
		Canonical:       s.cfg.BaseURL + "/canli-yayin",
		Baslik:          "Canlı Yayın",
		Sekmeler:        s.cdVideoSekmeler("canli"),
		Hero:            hero,
		Siradaki:        ytSatirlar(yayinlar, heroID, 6),
		Yayinlar:        ytGridHeroHariç(yayinlar, heroID, 12),
		KanalID:         kanalID,
		CanliYayinAktif: canliVar,
		BosMetin:        "Kanalda henüz canlı yayın kaydı yok.",
		JSONLD:          string(siteAdiJSONLD(s.siteName, s.cfg.BaseURL+"/canli-yayin")),
	}
	s.renderCDVideoPage(w, r, v)
}

// handleCDVideoWatch /video/{id}
func (s *Server) handleCDVideoWatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" || len(id) != 11 {
		s.handleNotFound(w, r)
		return
	}
	k, err := repo.YoutubeOne(ctx, s.pool, id)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	ayar := s.ytAyar(r)

	// "Sıradaki" ve alt ızgara aynı türden gelir.
	ayni, _ := repo.YoutubeListele(ctx, s.pool, k.Tur, 24)
	hero := ytHero(*k, videoTurEtiket(k.Tur))
	siradaki := ytSatirlar(ayni, k.VideoID, 6)
	ytSonraki(&hero, siradaki)

	var videolar, shorts []repo.YoutubeKayit
	if ayar.VideoGoster {
		videolar, _ = repo.YoutubeListele(ctx, s.pool, "video", 8)
	}
	if ayar.ShortGoster {
		shorts, _ = repo.YoutubeListele(ctx, s.pool, "short", 8)
	}

	aciklama := strings.TrimSpace(k.Aciklama)
	canonical := s.cfg.BaseURL + ytHref(*k)

	v := templkit.CDVideoWatch{
		Shell:     s.cdShellCtx(ctx, "videos"),
		Title:     k.Baslik + " — " + s.siteName,
		Desc:      aciklama,
		Canonical: canonical,
		Sekmeler:  s.cdVideoSekmeler("video"),
		Video:     hero,
		Aciklama:  aciklama,
		Siradaki:  siradaki,
		Videolar:  ytGridHeroHariç(videolar, k.VideoID, 8),
		Shorts:    ytGridHeroHariç(shorts, k.VideoID, 8),
		Dikey:     k.Tur == "short",
		JSONLD:    string(videoJSONLD(s.siteName, hero, canonical, k.YayinAt)),
	}
	if v.Desc == "" {
		v.Desc = "BİTCOŞAR videosu: " + k.Baslik
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDVideoWatchView(v).Render(ctx, w); err != nil {
		s.logger.Error("cd video izleme render", "error", err)
	}
}

// renderCDVideoPage ortak render.
func (s *Server) renderCDVideoPage(w http.ResponseWriter, r *http.Request, v templkit.CDVideoPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDVideoPageView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd video sayfası render", "error", err)
	}
}

// ---- Ana sayfa modülleri (sol raydaki canlı + sağ raydaki video/klipler) ----

// cdYoutubeVideolar "Bugünün Videoları" — kanalın son videoları.
//
// ÖNCEKİ DURUM: modül CoinDesk'ten alınmış GEÇİCİ veriyle doluydu
// (cd_gecici.go). Artık gerçek kanal verisi: scheduler YouTube'a her yeni
// video düştüğünde tabloyu günceller, modül kendiliğinden tazelenir.
func (s *Server) cdYoutubeVideolar(r *http.Request, adet int) []templkit.CDVideo {
	ayar := s.ytAyar(r)
	if !ayar.VideoGoster {
		return nil
	}
	kayitlar, err := repo.YoutubeListele(r.Context(), s.pool, "video", adet)
	if err != nil || len(kayitlar) == 0 {
		return nil
	}
	out := make([]templkit.CDVideo, 0, len(kayitlar))
	for _, k := range kayitlar {
		etiket := k.Kanal
		if etiket == "" {
			etiket = "Video"
		}
		out = append(out, templkit.CDVideo{
			Etiket: etiket,
			Baslik: k.Baslik,
			Image:  k.Gorsel,
			Href:   ytHref(k),
			Sure:   youtube.Sure(k.SureSn),
		})
	}
	return out
}

// cdYoutubeKlipler "Bugünün Klipleri" — kanalın son short'ları (9:16).
func (s *Server) cdYoutubeKlipler(r *http.Request, adet int) []templkit.CDKlip {
	ayar := s.ytAyar(r)
	if !ayar.ShortGoster {
		return nil
	}
	kayitlar, err := repo.YoutubeListele(r.Context(), s.pool, "short", adet)
	if err != nil || len(kayitlar) == 0 {
		return nil
	}
	out := make([]templkit.CDKlip, 0, len(kayitlar))
	for _, k := range kayitlar {
		out = append(out, templkit.CDKlip{
			Baslik: k.Baslik,
			Image:  k.Gorsel,
			Href:   ytHref(k),
		})
	}
	return out
}

// cdYoutubeCanli sol kolonun üstündeki canlı yayın modülü.
//
// ÖNCELİK: (1) şu an canlı olan yayın, (2) panelden elle işaretlenen video,
// (3) en son yayın. Hiçbiri yoksa modül basılmaz — boş bir "canlı yayın"
// kutusu sayfayı bozuk gösterir.
func (s *Server) cdYoutubeCanli(r *http.Request) templkit.CDCanli {
	ayar := s.ytAyar(r)
	if !ayar.CanliGoster {
		return templkit.CDCanli{}
	}
	kayitlar, err := repo.YoutubeListele(r.Context(), s.pool, "stream", 12)
	if err != nil && len(kayitlar) == 0 {
		return templkit.CDCanli{}
	}

	var secim *repo.YoutubeKayit
	for i := range kayitlar {
		if kayitlar[i].Canli {
			secim = &kayitlar[i]
			break
		}
	}
	if secim == nil && ayar.CanliVideoID != "" {
		for i := range kayitlar {
			if kayitlar[i].VideoID == ayar.CanliVideoID {
				secim = &kayitlar[i]
				break
			}
		}
	}
	if secim == nil && len(kayitlar) > 0 {
		secim = &kayitlar[0]
	}
	if secim == nil {
		return templkit.CDCanli{}
	}

	zaman := ytZaman(*secim)
	if secim.Canli {
		zaman = "Şu an canlı yayında"
	} else if zaman != "" {
		zaman = "Son yayın · " + zaman
	}
	return templkit.CDCanli{
		VideoID: secim.VideoID,
		Baslik:  secim.Baslik,
		Image:   secim.Gorsel,
		Href:    "/canli-yayin",
		Zaman:   zaman,
		Sure:    youtube.Sure(secim.SureSn),
		Canli:   secim.Canli,
	}
}

// videoTurEtiket tür adı (hero etiketi).
func videoTurEtiket(tur string) string {
	switch tur {
	case "short":
		return "Shorts"
	case "stream":
		return "Canlı Yayın"
	default:
		return "Video"
	}
}

// videoJSONLD VideoObject — Google video sonuçları için.
func videoJSONLD(siteName string, h templkit.CDVideoHero, canonical string, yayin *time.Time) template.HTML {
	m := map[string]any{
		"@context":     "https://schema.org",
		"@type":        "VideoObject",
		"name":         h.Baslik,
		"description":  h.Spot,
		"thumbnailUrl": h.Image,
		"contentUrl":   "https://www.youtube.com/watch?v=" + h.VideoID,
		"embedUrl":     "https://www.youtube-nocookie.com/embed/" + h.VideoID,
		"publisher": map[string]any{
			"@type": "Organization",
			"name":  siteName,
		},
		"url": canonical,
	}
	// uploadDate ZORUNLU alan: gerçek yayın tarihi yoksa blok basılmaz
	// (uydurma tarih yapısal veriyi geçersiz kılar).
	if yayin == nil {
		return ""
	}
	m["uploadDate"] = yayin.UTC().Format("2006-01-02T15:04:05Z")
	return jsonldScript(m)
}
