package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
	"theblock/site/internal/youtube"
)

type videoYayinData struct {
	ayar    youtube.Ayar
	items   []repo.YoutubeKayit
	sayilar map[string]int
	tur     string
}

// handleVideoYayinGet panel sayfası (HTMX'te yalnız panel gövdesi).
func (s *Server) handleVideoYayinGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildVideoYayinData(r)
	if IsHTMX(r) {
		s.renderTemplPanel(w, r, templkit.VideoYayinPanel(videoYayinViewFrom(d, nil)))
		return
	}
	s.renderVideoYayinPage(w, r, d, nil)
}

// buildVideoYayinData panel verisini toplar.
func (s *Server) buildVideoYayinData(r *http.Request) *videoYayinData {
	ctx := r.Context()
	tur := strings.TrimSpace(r.URL.Query().Get("tur"))
	switch tur {
	case "", "video", "short", "stream":
	default:
		tur = ""
	}
	ayar := youtube.AyarlariOku(ctx, s.pool)
	items, _ := repo.YoutubeListeleHepsi(ctx, s.pool, tur, 200)
	sayilar, _ := repo.YoutubeSayilar(ctx, s.pool)
	return &videoYayinData{ayar: ayar, items: items, sayilar: sayilar, tur: tur}
}

// renderVideoYayinPage tam sayfa.
func (s *Server) renderVideoYayinPage(w http.ResponseWriter, r *http.Request, d *videoYayinData, fl *flashMsg) {
	td := s.baseView("tr", "/admin", "Video & Canlı Yayın")
	td.User = ptr(userFromContext(r.Context()))
	td.Flash = fl
	s.renderTemplPage(w, r, td, templkit.VideoYayinPanel(videoYayinViewFrom(d, fl)))
}

// videoYayinSonuc HTMX'te panel tazeleme, normal istekte flash yönlendirme.
func (s *Server) videoYayinSonuc(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		d := s.buildVideoYayinData(withHTMXCurrentQuery(r))
		s.renderTemplPanel(w, r, templkit.VideoYayinPanel(videoYayinViewFrom(d, &flashMsg{Kind: kind, Text: text})))
		return
	}
	s.flashRedirect(w, r, "/admin/video-yayin", kind, text)
}

// handleVideoYayinAyar ayarları kaydeder.
func (s *Server) handleVideoYayinAyar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	kanalID := strings.TrimSpace(r.FormValue("kanal_id"))
	handle := strings.TrimPrefix(strings.TrimSpace(r.FormValue("handle")), "@")
	if !youtube.GecerliKanalID(kanalID) {
		s.videoYayinSonuc(w, r, "error", "Kanal kimliği geçersiz. YouTube kanal adresindeki UC… değerini girin.")
		return
	}
	if !youtube.GecerliHandle(handle) {
		s.videoYayinSonuc(w, r, "error", "Kanal adı geçersiz (örn. bitcosar).")
		return
	}
	dakika, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("dakika")))
	if dakika < 5 || dakika > 1440 {
		dakika = 15
	}
	a := youtube.Ayar{
		KanalID: kanalID, Handle: handle, Dakika: dakika,
		Aktif:       r.FormValue("aktif") == "1",
		VideoGoster: r.FormValue("video_goster") == "1",
		ShortGoster: r.FormValue("short_goster") == "1",
		CanliGoster: r.FormValue("canli_goster") == "1",
		// Elle işaretleme: video kimliği ya da tam bağlantı kabul edilir.
		CanliVideoID: youtube.VideoIDCikar(r.FormValue("canli_video_id")),
	}
	if err := a.Kaydet(r.Context(), s.pool); err != nil {
		s.logger.Error("video yayın ayar kaydetme", "error", err)
		s.videoYayinSonuc(w, r, "error", "Ayarlar kaydedilemedi.")
		return
	}
	s.audit(r, "video_yayin_ayar", kanalID)
	// Ayarlar değişti: kısa süreli önbellek bir sonraki istekte tazelenir.
	s.invalidateContentCache()
	s.videoYayinSonuc(w, r, "info", "Ayarlar kaydedildi.")
}

// handleVideoYayinCek kanalı hemen çeker.
func (s *Server) handleVideoYayinCek(w http.ResponseWriter, r *http.Request) {
	ctx, iptal := context.WithTimeout(r.Context(), 45*time.Second)
	defer iptal()
	sn, err := youtube.Senkron(ctx, s.pool, s.logger)
	if err != nil {
		s.videoYayinSonuc(w, r, "error", "Çekme başarısız: "+err.Error())
		return
	}
	s.audit(r, "video_yayin_cek", "elle")
	s.videoYayinSonuc(w, r, "info", "Çekme tamam: "+
		strconv.Itoa(len(sn.Videolar))+" video, "+
		strconv.Itoa(len(sn.Shorts))+" short, "+
		strconv.Itoa(len(sn.Yayinlar))+" yayın.")
}

func (s *Server) handleVideoYayinIslem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	islem := strings.TrimSpace(r.PathValue("islem"))
	if len(id) != 11 {
		s.videoYayinSonuc(w, r, "error", "Geçersiz video kimliği.")
		return
	}
	ctx := r.Context()
	k, err := repo.YoutubeOne(ctx, s.pool, id)
	if err != nil {
		s.videoYayinSonuc(w, r, "error", "Kayıt bulunamadı.")
		return
	}
	switch islem {
	case "one_cikan":
		if err := repo.YoutubeBayrak(ctx, s.pool, id, "one_cikan", !k.OneCikan); err != nil {
			s.videoYayinSonuc(w, r, "error", "İşlem başarısız.")
			return
		}
		s.audit(r, "video_yayin_one_cikan", id)
		s.videoYayinSonuc(w, r, "info", "Öne çıkan güncellendi.")
	case "canli":
		// Canlı işareti aynı anda TEK videoda kalır.
		_ = repo.YoutubeCanliAyarla(ctx, s.pool, "")
		if !k.Canli {
			_ = repo.YoutubeCanliAyarla(ctx, s.pool, id)
		}
		s.audit(r, "video_yayin_canli", id)
		s.videoYayinSonuc(w, r, "info", "Canlı yayın işareti güncellendi.")
	case "gizli":
		if err := repo.YoutubeBayrak(ctx, s.pool, id, "gizli", !k.Gizli); err != nil {
			s.videoYayinSonuc(w, r, "error", "İşlem başarısız.")
			return
		}
		s.audit(r, "video_yayin_gizli", id)
		s.videoYayinSonuc(w, r, "info", "Görünürlük güncellendi.")
	case "sil":
		if err := repo.YoutubeSil(ctx, s.pool, id); err != nil {
			s.videoYayinSonuc(w, r, "error", "Silinemedi.")
			return
		}
		s.audit(r, "video_yayin_sil", id)
		s.videoYayinSonuc(w, r, "info", "Kayıt silindi.")
	default:
		s.videoYayinSonuc(w, r, "error", "Bilinmeyen işlem.")
	}
}

// videoYayinViewFrom veri → templ görünümü.
func videoYayinViewFrom(d *videoYayinData, fl *flashMsg) templkit.VideoYayinView {
	kind, text := flashTo(fl)
	v := templkit.VideoYayinView{FlashKind: kind, FlashText: text}
	if d == nil {
		return v
	}
	a := d.ayar
	v.KanalID = a.KanalID
	v.Handle = a.Handle
	v.Dakika = strconv.Itoa(a.Dakika)
	v.Aktif = a.Aktif
	v.VideoGoster = a.VideoGoster
	v.ShortGoster = a.ShortGoster
	v.CanliGoster = a.CanliGoster
	v.CanliVideoID = a.CanliVideoID
	if a.SonCekme != "" {
		if t, err := time.Parse(time.RFC3339, a.SonCekme); err == nil {
			v.SonCekme = t.Local().Format("02.01.2006 15:04")
		} else {
			v.SonCekme = a.SonCekme
		}
	}
	v.SonHata = a.SonHata
	v.VideoAdet = d.sayilar["video"]
	v.ShortAdet = d.sayilar["short"]
	v.YayinAdet = d.sayilar["stream"]
	v.IcerikToplam = v.VideoAdet + v.ShortAdet + v.YayinAdet

	for _, ad := range []struct{ Kod, Ad string }{
		{"", "Tümü"}, {"video", "Videolar"}, {"short", "Shorts"}, {"stream", "Canlı Yayınlar"},
	} {
		adet := v.IcerikToplam
		switch ad.Kod {
		case "video":
			adet = v.VideoAdet
		case "short":
			adet = v.ShortAdet
		case "stream":
			adet = v.YayinAdet
		}
		v.Sekmeler = append(v.Sekmeler, templkit.VideoYayinSekme{
			Kod: ad.Kod, Ad: ad.Ad, Adet: adet, Aktif: ad.Kod == d.tur,
		})
	}

	simdi := time.Now()
	for _, k := range d.items {
		if k.Canli {
			v.CanliVar = true
		}
		zaman := ""
		if k.YayinAt != nil {
			zaman = trGoreliZaman(*k.YayinAt, simdi)
		}
		v.Satirlar = append(v.Satirlar, templkit.VideoYayinSatir{
			VideoID: k.VideoID, Tur: k.Tur, TurAd: videoTurEtiket(k.Tur),
			Baslik: k.Baslik, Gorsel: k.Gorsel,
			Sure: youtube.Sure(k.SureSn), Zaman: zaman,
			Canli: k.Canli, OneCikan: k.OneCikan, Gizli: k.Gizli,
		})
	}
	v.BosMetin = "Arşiv boş. \"Şimdi Çek\" düğmesiyle kanalı hemen okuyabilirsiniz."
	return v
}
