package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdHesapNavListesi hesap kenar menüsü (sabit sıra).
func cdHesapNavListesi(h *hesapData) []templkit.CDHesapNav {
	sayi := func(n int) int { return n }
	return []templkit.CDHesapNav{
		{Etiket: "Genel Bakış", Href: "/hesabim"},
		{Etiket: "Profilim", Href: "/hesabim/profil"},
		{Etiket: "Aboneliğim", Href: "/hesabim/abonelik"},
		{Etiket: "Kaydedilenler", Href: "/hesabim/kaydedilenler", Sayi: sayi(h.Stats.Bookmarks)},
		{Etiket: "Yorumlarım", Href: "/hesabim/yorumlar", Sayi: sayi(h.Stats.Comments)},
		{Etiket: "Takip ettiklerim", Href: "/hesabim/takip", Sayi: sayi(h.Stats.Follows)},
		{Etiket: "Bildirimler", Href: "/hesabim/bildirimler", Sayi: sayi(h.Stats.Unread), Yeni: h.Stats.Unread > 0},
		{Etiket: "Başarımlar", Href: "/hesabim/rozetler"},
	}
}

// cdHesapSayfaOlustur hesapData → templkit.CDHesapSayfa.
func (s *Server) cdHesapSayfaOlustur(r *http.Request, d *tmplData, h *hesapData, sekme, baslik, aciklama string) templkit.CDHesapSayfa {
	shell := s.cdShellCtx(r.Context(), r, "")
	shell.Ticker = s.cdTickerDoldur(r)
	shell.BaseURL = strings.TrimRight(s.cfg.BaseURL, "/")
	kanonik := strings.TrimRight(s.cfg.BaseURL, "/") + r.URL.Path
	// Sorgu parametresi kanonik adrese girmesin (?ok=…).
	if i := strings.IndexByte(kanonik, '?'); i >= 0 {
		kanonik = kanonik[:i]
	}

	planAd, planAciklama, premium := "Ücretsiz Üye", "Abone olarak özel içeriklerin tamamına erişin.", false
	if strings.EqualFold(h.Member.Membership, "premium") || h.TierName != "" {
		planAd = "Premium Üye"
		planAciklama = "Tüm abonelere özel içeriklere erişiminiz var."
		premium = true
	}

	v := templkit.CDHesapSayfa{
		Shell:     shell,
		Title:     baslik + " — " + s.siteName,
		Desc:      aciklama,
		Canonical: kanonik,
		JSONLD:    string(siteAdiJSONLD(s.siteName, kanonik)),
		Sekme:     sekme,
		Nav:       cdHesapNavListesi(h),

		Ad:               h.Member.Name,
		Eposta:           h.Member.Email,
		BasHarfler:       basHarf(h.Member.Name),
		AvatarURL:        h.AvatarURL,
		AvatarID:         strconv.FormatInt(h.AvatarID, 10),
		Rol:              h.Member.Role,
		EpostaDogrulandi: h.Member.EmailVerifiedAt != nil,

		PlanAd:       planAd,
		PlanAciklama: planAciklama,
		PlanBitis:    h.TierUntil,
		Premium:      premium,

		Slug:      h.Member.Slug,
		Unvan:     h.Member.Title,
		Biyografi: h.Member.Bio,

		Puan: h.Stats.Points,
		Seri: h.Stats.Streak,

		AbonelikVar:   h.TierName != "",
		AbonelikAd:    h.TierName,
		AbonelikRenk:  h.TierColor,
		AbonelikBitis: h.TierUntil,
		IptalTarih:    h.CancelAt,
		OdemeAcik:     h.StripeEnabled,
		OdemeEtiket:   h.PriceLabel,
	}
	if h.TierCycle == "annual" {
		v.AbonelikDongu = "Yıllık"
	} else if h.TierCycle == "monthly" {
		v.AbonelikDongu = "Aylık"
	}

	// Flash: hata varsa hata, bilgi varsa ok.
	if h.Error != "" {
		v.FlashTur, v.FlashMetin = "hata", h.Error
	} else if h.Info != "" {
		v.FlashTur, v.FlashMetin = "ok", h.Info
	}

	// Hub: sayı kutuları + kısayollar.
	v.Kutular = []templkit.CDHesapKutu{
		{Sayi: strconv.Itoa(h.Stats.Bookmarks), Etiket: "Kaydedilen", Href: "/hesabim/kaydedilenler"},
		{Sayi: strconv.Itoa(h.Stats.Comments), Etiket: "Yorum", Href: "/hesabim/yorumlar"},
		{Sayi: strconv.Itoa(h.Stats.Points), Etiket: "Okuma puanı", Href: "/hesabim/rozetler"},
		{Sayi: strconv.Itoa(h.Stats.Streak), Etiket: "Günlük seri", Href: "/hesabim/rozetler"},
	}
	v.Kisayollar = []templkit.CDHesapKisayol{
		{Baslik: "Profilim", Aciklama: "Ad, biyografi, sosyal medya ve fotoğraf", Href: "/hesabim/profil"},
		{Baslik: "Aboneliğim", Aciklama: "Plan, ödeme ve yenileme", Href: "/hesabim/abonelik"},
		{Baslik: "Bildirimler", Aciklama: bildirimOzet(h.Stats.Unread), Href: "/hesabim/bildirimler"},
		{Baslik: "Başarımlar", Aciklama: "Rozetler ve liderlik tablosu", Href: "/hesabim/rozetler"},
	}
	if h.Member.EmailVerifiedAt == nil {
		v.UyariMetin = "E-posta adresiniz doğrulanmadı."
	}

	// Profil: sosyal alanlar + avatarlar.
	for _, f := range hesapSosyal {
		v.Sosyal = append(v.Sosyal, templkit.CDSosyalAlan{
			Anahtar: f.Key, Etiket: f.Label, YerTutucu: f.Placeholder,
			Deger: h.Socials[f.Key],
		})
	}
	for _, a := range h.Avatars {
		v.Avatarlar = append(v.Avatarlar, templkit.CDAvatarSec{
			ID: strconv.FormatInt(a.ID, 10), URL: a.URL, Baslik: a.Title,
			Secili: a.ID == h.AvatarID,
		})
	}

	// Listeler.
	for _, c := range h.Comments {
		v.Yorumlar = append(v.Yorumlar, templkit.CDHesapYorum{
			Baslik: c.ContentTitle,
			Href:   "/" + c.ContentSlug + "-" + strconv.FormatInt(c.ContentID, 10),
			Tarih:  c.CreatedAt.Format("02.01.2006 15:04"),
			Metin:  c.Body,
			Durum:  yorumDurumMetni(c.Status),
			Begeni: c.Likes,
		})
	}
	for _, b := range h.Bookmarks {
		v.Kaydedilenler = append(v.Kaydedilenler, templkit.CDHesapKayit{
			Baslik: b.Title, Spot: b.Spot, Href: makaleURL(b),
			Gorsel: b.ImageURL, Zaman: b.UpdatedAt.Format("02.01.2006"),
		})
	}
	for _, f := range h.Follows {
		v.Takip = append(v.Takip, templkit.CDHesapTakip{
			Tur: takipTurMetni(f.Type), Etiket: takipEtiketi(f), Slug: f.Slug,
			Href: takipHref(f),
		})
	}
	for _, n := range h.Notifs {
		v.Bildirimler = append(v.Bildirimler, templkit.CDHesapBildirim{
			Tur: n.Kind, Baslik: n.Title, Metin: n.Body, Href: n.URL,
			Zaman: n.CreatedAt.Format("02.01.2006 15:04"), Okunmadi: !n.IsRead,
		})
	}
	var kazanilan map[string]bool
	if h.Points != nil {
		kazanilan = map[string]bool{}
		for _, k := range h.Points.Achievements {
			kazanilan[k] = true
		}
	}
	for _, a := range h.Defs {
		ac := ""
		if a.Min > 0 {
			ac = strconv.Itoa(a.Min) + "+ puan"
		} else if a.Streak > 0 {
			ac = strconv.Itoa(a.Streak) + "+ günlük seri"
		}
		v.Rozetler = append(v.Rozetler, templkit.CDHesapRozet{
			Ad: a.Ad, Aciklama: ac, Kazanildi: kazanilan[a.Key],
		})
	}
	for i, l := range h.Leader {
		v.Liderlik = append(v.Liderlik, templkit.CDHesapLider{
			Sira: i + 1, Ad: "#" + strconv.FormatInt(l.UserID, 10),
			Puan: l.Points, Seri: l.Streak, Ben: l.UserID == h.Member.ID,
		})
	}

	// Boş durum metinleri sekme bazlı.
	switch sekme {
	case "yorumlar":
		v.BosMetin = "Henüz yorum yapmadınız."
	case "takip":
		v.BosMetin = "Henüz bir kategori, etiket veya yazar takip etmiyorsunuz."
	case "kaydedilenler":
		v.BosMetin = "Henüz kaydettiğiniz haber yok. Haber sayfasındaki “Kaydet” düğmesini kullanın."
	case "bildirimler":
		v.BosMetin = "Bildirim yok."
	}
	return v
}

// basHarf adın ilk harfini döndürür (avatar yedeği).
func basHarf(ad string) string {
	r := []rune(ad)
	if len(r) == 0 {
		return "?"
	}
	return string(r[:1])
}

func bildirimOzet(n int) string {
	if n > 0 {
		return strconv.Itoa(n) + " okunmamış bildirim"
	}
	return "Yeni bildirim yok"
}

func yorumDurumMetni(s string) string {
	switch s {
	case "onay_bekliyor":
		return "onay bekliyor"
	case "reddedildi":
		return "reddedildi"
	}
	return ""
}

func takipTurMetni(t string) string {
	switch t {
	case "kategori":
		return "Kategori"
	case "etiket":
		return "Etiket"
	case "yazar":
		return "Yazar"
	}
	return t
}

func takipEtiketi(f repo.Follow) string {
	switch f.Type {
	case "etiket":
		return "#" + f.Slug
	}
	return f.Slug
}

func takipHref(f repo.Follow) string {
	switch f.Type {
	case "kategori":
		return "/kategori/" + f.Slug
	case "etiket":
		return "/tag/" + f.Slug
	case "yazar":
		return "/yazar/" + f.Slug
	}
	return "/"
}

// cdHesapRender hesap sayfasını CoinDesk kabuğuyla basar.
func (s *Server) cdHesapRender(w http.ResponseWriter, r *http.Request, d *tmplData, h *hesapData, sekme, baslik, aciklama string) {
	v := s.cdHesapSayfaOlustur(r, d, h, sekme, baslik, aciklama)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := templkit.CDHesap(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd hesap render", "sekme", sekme, "error", err)
	}
}
