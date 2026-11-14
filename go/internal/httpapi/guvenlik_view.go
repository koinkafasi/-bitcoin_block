package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// TRAFİK GÜVENLİĞİ PANELİ (/admin/trafik-guvenligi).
//
// Veri: repo.RiskOzet (visitor_risk) + ASN önbelleği.
// Amaç: "kaç okunma var" değil, "bu okumaların kaçı gerçek".

// sinyalAdlari sinyal kodlarının insan okur karşılığı.
//
// TEK YER: puan ağırlıkları guvenlik_skor.go'da, adlar burada. Kod
// eklenince burada karşılığı yoksa kod olduğu gibi gösterilir — eksik
// çeviri yanlış addan iyidir.
var sinyalAdlari = map[string]string{
	"ua_bot":         "Otomasyon aracı (UA)",
	"ua_bos":         "Boş user-agent",
	"asn_hosting":    "Veri merkezi ağı",
	"asn_vpn":        "Ticari VPN ağı",
	"tor":            "Tor çıkışı",
	"baslik_proxy":   "Proxy başlığı",
	"xff_celiski":    "X-Forwarded-For çelişkisi",
	"secfetch_yok":   "Tarayıcı gezinme başlığı yok",
	"dil_yok":        "Accept-Language yok",
	"ip_rotasyon":    "IP rotasyonu",
	"ulke_rotasyon":  "Ülke rotasyonu",
	"hiz_asimi":      "İstek hızı aşımı",
	"tuzak":          "Bot tuzağına düştü",
	"webdriver":      "WebDriver işareti",
	"headless_gpu":   "Yazılım grafik birimi",
	"tz_uyusmazlik":  "Saat dilimi uyuşmazlığı",
	"dil_uyusmazlik": "Dil uyuşmazlığı",
	"webrtc_farkli":  "WebRTC IP sızıntısı",
	"etkilesim_yok":  "Etkileşim yok",
	"sahte_crawler":  "Sahte arama motoru botu",
}

// sinifAdlari sınıf kodlarının panel karşılığı.
var sinifAdlari = map[string]string{
	"temiz":   "Gerçek",
	"supheli": "Şüpheli",
	"yuksek":  "Yüksek",
	"bot":     "Bot",
	"arama":   "Arama botu",
}

// turAdlari sağlayıcı türünün panel karşılığı.
var turAdlari = map[string]string{
	"hosting":    "veri merkezi",
	"vpn":        "VPN",
	"tor":        "Tor",
	"mobil":      "mobil",
	"konut":      "konut",
	"bilinmiyor": "bilinmiyor",
}

// handleTrafikGuvenligi paneli çizer.
func (s *Server) handleTrafikGuvenligi(w http.ResponseWriter, r *http.Request) {
	gun := 30
	if g, err := strconv.Atoi(r.URL.Query().Get("gun")); err == nil {
		switch g {
		case 7, 30, 90:
			gun = g
		}
	}

	ctx := r.Context()
	ozet, err := repo.RiskOzet(ctx, s.pool, gun)
	if err != nil {
		s.logger.Error("risk özeti", "error", err)
	}
	pol := s.guvenlikPolitikasi(ctx)

	v := templkit.TrafikGuvenligiView{
		Gun:          gun,
		Aktif:        pol.Aktif,
		VPNEngel:     pol.VPNEngel,
		HostingEngel: pol.HostingEngel,
		BotEngel:     pol.BotEngel,
		WebRTC:       pol.WebRTC,
		EsikYuksek:   s.ayarSayi(ctx, "guvenlik", "esik_yuksek", 60, 30, 100),
		EsikBot:      s.ayarSayi(ctx, "guvenlik", "esik_bot", 85, 50, 100),
		Toplam:       ozet.Toplam,
		Gercek:       ozet.Gercek,
		Supheli:      ozet.Supheli,
		Yuksek:       ozet.Yuksek,
		Bot:          ozet.Bot,
		Arama:        ozet.Arama,
		VPN:          ozet.VPN,
		Hosting:      ozet.Hosting,
		Tor:          ozet.Tor,
	}
	if ozet.Toplam > 0 {
		v.GercekYuzde = int((ozet.Gercek*100 + ozet.Toplam/2) / ozet.Toplam)
	}

	for _, a := range ozet.ASNDagilim {
		etiket := asnSaglayiciEtiket(a.Ad)
		if etiket == "" {
			etiket = "AS" + itoa(a.ASN)
		}
		satir := templkit.GuvenlikASNSatir{
			Etiket: etiket,
			Tur:    a.Tur,
			ASN:    a.ASN,
			Adet:   a.Adet,
		}
		if ozet.Toplam > 0 {
			satir.Pay = float64(a.Adet) * 100 / float64(ozet.Toplam)
		}
		v.ASN = append(v.ASN, satir)
	}

	// Sinyal paydası: sinyali olan ziyaretçi sayısı, toplam değil —
	// "20 sinyal / 100 ziyaretçi" yanıltıcı olurdu (bir ziyaretçide 5
	// sinyal olabilir). Bu yüzden payda en sık sinyalin sayısı.
	var sinyalMaks int64
	for _, sn := range ozet.Sinyaller {
		if sn.Adet > sinyalMaks {
			sinyalMaks = sn.Adet
		}
	}
	for _, sn := range ozet.Sinyaller {
		ad := sinyalAdlari[sn.Kod]
		if ad == "" {
			ad = sn.Kod
		}
		satir := templkit.GuvenlikSinyalSatir{Kod: sn.Kod, Ad: ad, Adet: sn.Adet}
		if sinyalMaks > 0 {
			satir.Pay = float64(sn.Adet) * 100 / float64(sinyalMaks)
		}
		v.Sinyaller = append(v.Sinyaller, satir)
	}

	for _, k := range ozet.Satirlar {
		// Sağlayıcı: AS adı bütçe içinde gelmediyse ASN numarası gösterilir.
		// Ölçüldü (14 Kasım 2026): Hetzner bot satırı "—" gösteriyordu çünkü
		// AS adı sorgusu 120 ms bütçesini aşmıştı; oysa ASN numarası (24940)
		// elde vardı ve teşhis için yeterli.
		saglayici := asnSaglayiciEtiket(k.ASNAd)
		if saglayici == "" && k.ASN > 0 {
			saglayici = "AS" + itoa(k.ASN)
		}
		satir := templkit.GuvenlikRiskSatir{
			Anahtar:     k.Anahtar,
			Sinif:       k.Sinif,
			SinifAd:     sinifGoster(k.Sinif),
			Puan:        k.Puan,
			Saglayici:   saglayici,
			Tur:         k.ASNTur,
			TurAd:       turAdlari[k.ASNTur],
			Ulke:        k.Ulke,
			IP:          maskeliIP(k.IP),
			WebRTCIP:    maskeliIP(k.WebRTCIP),
			IstekSayisi: k.IstekSayisi,
			SonGorulme:  k.SonGorulme.Local().Format("02.01 15:04"),
		}
		// staticcheck S1011: tek elemanlı loop yerine doğrudan append.
		satir.Sinyaller = append(satir.Sinyaller, sinyalListesi(k.Sinyaller)...)
		v.Satirlar = append(v.Satirlar, satir)
	}

	// NİYET — listede "ne yapmaya çalışıyor" sütunu.
	//
	// TEK SORGU, 120 ANAHTAR: ziyaretçi başına ayrı sorgu paneli
	// yavaşlatırdı (ziyaretçi konumu panelinde aynı kusur 14 Kasım'da
	// ölçülmüştü). Olay kaydı yoksa sütun "—" kalır: uydurma etiket
	// basmaktansa boş bırakmak dürüst.
	if len(v.Satirlar) > 0 && s.pool != nil {
		anahtarlar := make([]string, 0, len(v.Satirlar))
		for _, st := range v.Satirlar {
			anahtarlar = append(anahtarlar, st.Anahtar)
		}
		niyetler, err := repo.OlayNiyetleriToptan(ctx, s.pool, anahtarlar)
		if err != nil {
			s.logger.Debug("niyet sayımları okunamadı", "error", err)
		}
		for i := range v.Satirlar {
			sayim := niyetler[v.Satirlar[i].Anahtar]
			if len(sayim) == 0 {
				continue
			}
			toplam := 0
			for _, n := range sayim {
				toplam += n
			}
			n := NiyetCikar(sayim, toplam, 0)
			v.Satirlar[i].Niyet = n.Baslik
			v.Satirlar[i].NiyetSeviye = n.Seviye
		}
	}

	d := s.baseView("tr", "/admin", "Trafik Güvenliği")
	s.renderTemplPage(w, r, d, templkit.TrafikGuvenligiPanel(v))
}

func sinifGoster(s string) string {
	if ad := sinifAdlari[s]; ad != "" {
		return ad
	}
	return s
}

// sinyalListesi JSON dizi metnini kısa etiketlere çevirir.
//
// Sessiz bozulma koruması: ayrıştırılamayan kayıt boş liste döner, sayfa
// çizilmeye devam eder (tek bozuk satır tabloyu düşürmesin).
func sinyalListesi(ham string) []string {
	if ham == "" || ham == "[]" {
		return nil
	}
	type sinyal struct {
		K string `json:"k"`
		N string `json:"n"`
	}
	var liste []sinyal
	if err := json.Unmarshal([]byte(ham), &liste); err != nil {
		return nil
	}
	out := make([]string, 0, len(liste))
	for _, s := range liste {
		ad := sinyalAdlari[s.K]
		if ad == "" {
			ad = s.K
		}
		out = append(out, ad)
	}
	return out
}

// handleTrafikGuvenligiKaydet politikayı yazar.
func (s *Server) handleTrafikGuvenligiKaydet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		s.flashRedirect(w, r, "/admin/trafik-guvenligi", "error", "Form okunamadı.")
		return
	}
	yaz := func(anahtar string, acik bool) {
		v := "0"
		if acik {
			v = "1"
		}
		_ = repo.SetSetting(ctx, s.pool, "guvenlik", anahtar, []byte(strconv.Quote(v)))
	}
	yaz("aktif", r.FormValue("aktif") != "")
	yaz("hosting_engel", r.FormValue("hosting_engel") != "")
	yaz("vpn_engel", r.FormValue("vpn_engel") != "")
	yaz("bot_engel", r.FormValue("bot_engel") != "")
	yaz("webrtc", r.FormValue("webrtc") != "")

	// Eşikler: aralık dışı değer SESSİZCE yazılmaz; varsayılana döner.
	esikYuksek := ayarSayiForm(r.FormValue("esik_yuksek"), 60, 30, 100)
	esikBot := ayarSayiForm(r.FormValue("esik_bot"), 85, 50, 100)
	if esikBot <= esikYuksek {
		// Bot eşiği yüksek eşiğin altında olamaz: aksi hâlde "yüksek"
		// sınıfı hiç oluşmaz ve iki eşik aynı işi yapar.
		esikBot = esikYuksek + 25
		if esikBot > 100 {
			esikBot = 100
		}
	}
	_ = repo.SetSetting(ctx, s.pool, "guvenlik", "esik_yuksek",
		[]byte(strconv.Itoa(esikYuksek)))
	_ = repo.SetSetting(ctx, s.pool, "guvenlik", "esik_bot",
		[]byte(strconv.Itoa(esikBot)))

	s.invalidateContentCache()
	s.audit(r, "trafik_guvenligi_politika", "güncellendi")
	s.flashRedirect(w, r, "/admin/trafik-guvenligi", "info", "Politika kaydedildi.")
}

// ayarSayiForm form değerini aralık içinde int'e çevirir.
func ayarSayiForm(v string, varsayilan, alt, ust int) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < alt || n > ust {
		if err != nil {
			return varsayilan
		}
		if n < alt {
			return alt
		}
		return ust
	}
	return n
}
