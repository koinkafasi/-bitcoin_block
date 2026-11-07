package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// ---- Üye hesap merkezi (/hesabim) ----
//
// Tüm alt sayfalar aynı kabuğu (sol menü + içerik) kullanır. Oturum yoksa
// /giris?redirect=hesabim adresine yönlendirilir.

// hesapData /hesabim sayfalarının görünüm verisi.
type hesapData struct {
	Tab       string
	Member    repo.Member
	Stats     repo.MemberStats
	AvatarURL string
	AvatarID  int64
	Avatars   []hesapAvatar
	Comments  []repo.MemberComment
	Bookmarks []repo.Content
	Follows   []repo.Follow
	Notifs    []repo.Notification
	Points    *repo.UserPoints
	Defs      []repo.Achievement
	Leader    []repo.UserPoints
	Error     string
	Info      string
	// Katman bilgisi (abonelik modeli).
	TierName  string
	TierColor string
	TierUntil string
	TierCycle string
	CancelAt  string
	// Ödeme (Stripe tek kademe; Faz 4'te katmanlara genişler).
	StripeEnabled bool
	PriceLabel    string
	Socials       map[string]string
}

// hesapAvatar avatar seçicideki görsel.
type hesapAvatar struct {
	ID    int64
	URL   string
	Title string
}

// hesapSosyal alanları (form sırasıyla).
var hesapSosyal = []struct{ Key, Label, Placeholder string }{
	{"x", "X (Twitter)", "https://x.com/kullanici"},
	{"instagram", "Instagram", "https://instagram.com/kullanici"},
	{"facebook", "Facebook", "https://facebook.com/kullanici"},
	{"youtube", "YouTube", "https://youtube.com/@kanal"},
	{"linkedin", "LinkedIn", "https://linkedin.com/in/kullanici"},
	{"tiktok", "TikTok", "https://tiktok.com/@kullanici"},
	{"telegram", "Telegram", "https://t.me/kullanici"},
	{"website", "Web sitesi", "https://ornek.com"},
}

// requireMember oturum yoksa giriş sayfasına yönlendirir.
func (s *Server) requireMember(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, err := s.currentUser(r)
	if err != nil || u.ID == 0 {
		http.Redirect(w, r, s.langBase(langOf(r, s))+"/giris?redirect=hesabim", http.StatusFound)
		return auth.User{}, false
	}
	return u, true
}

// hesapView üye merkezi sayfası için tmplData üretir.
func (s *Server) hesapView(r *http.Request, u auth.User, tab string) (*tmplData, *hesapData) {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Hesabım")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.MemberLoggedIn = true
	d.MemberName = u.Name
	d.MemberInitial = memberInitial(u.Name)
	d.MemberURL = "/hesabim"
	d.MemberStaff = auth.StaffRoles[u.Role]

	h := &hesapData{Tab: tab, Socials: map[string]string{}}
	m, err := repo.GetMember(ctx, s.pool, u.ID)
	if err != nil {
		h.Error = "Hesap bilgileri yüklenemedi."
	} else {
		h.Member = m
		h.Socials = m.Socials
		if h.Socials == nil {
			h.Socials = map[string]string{}
		}
		if m.AvatarMediaID != nil {
			h.AvatarURL = s.mediaURL(m.AvatarMediaID)
			h.AvatarID = *m.AvatarMediaID
		}
	}
	st, _ := repo.MemberStatsOf(ctx, s.pool, u.ID)
	h.Stats = st
	if t, ok := repo.UserTier(ctx, s.pool, u.ID); ok {
		h.TierName = t.Name
		h.TierColor = t.ColorHex
		h.TierCycle = m.MembershipBillingCycle
		h.TierUntil = ""
	}
	if m.MembershipExpiresAt != nil {
		h.TierUntil = m.MembershipExpiresAt.Format("02.01.2006")
	}
	if m.MembershipCancelAt != nil {
		h.CancelAt = m.MembershipCancelAt.Format("02.01.2006")
	}
	if stripe := s.stripeSettings(r); stripe["aktif"] == "1" && stripe["secret_key"] != "" && stripe["price_id"] != "" {
		h.StripeEnabled = true
		h.PriceLabel = stripe["price_label"]
	}
	d.Hesap = h
	return d, h
}

// hesapRender sayfayı üye kabuğuyla basar.
func (s *Server) hesapRender(w http.ResponseWriter, r *http.Request, d *tmplData, name string) {
	s.renderPublic(w, r, name, d, "no-store")
}

// GET /hesabim — hub.
func (s *Server) handleHesap(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "hub")
	if v := r.URL.Query().Get("ok"); v != "" {
		h.Info = hesapMesaj(v)
	}
	s.cdHesapRender(w, r, d, h, "hub", "Hesabım", "Hesabınızı, aboneliğinizi ve okuma listenizi yönetin.")
}

func hesapMesaj(key string) string {
	switch key {
	case "profil":
		return "Profiliniz kaydedildi."
	case "sifre":
		return "Şifreniz güncellendi."
	case "avatar":
		return "Profil fotoğrafınız güncellendi."
	case "onay":
		return "Doğrulama bağlantısı e-posta adresinize gönderildi."
	case "odeme":
		return "Ödemeniz alındı; aboneliğiniz etkinleştirildi."
	case "iptal":
		return "İptal talebiniz alındı. Dönem sonuna kadar erişiminiz devam eder."
	case "iptal-geri":
		return "İptal talebiniz geri alındı."
	default:
		return ""
	}
}

// GET /hesabim/profil
func (s *Server) handleHesapProfil(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "profil")
	if v := r.URL.Query().Get("ok"); v != "" {
		h.Info = hesapMesaj(v)
	}
	// Avatar seçici: kütüphanedeki son görseller.
	site := s.siteByDomain(r.Context(), r.Host)
	lib, _ := repo.ListMediaLibrary(r.Context(), s.pool, site.ID, repo.MediaFilter{Limit: 24})
	for _, m := range lib {
		if m.Kind != "image" {
			continue
		}
		h.Avatars = append(h.Avatars, hesapAvatar{ID: m.ID, URL: s.mediaThumbURL(&m.ID), Title: m.Title})
	}
	s.cdHesapRender(w, r, d, h, "profil", "Profilim", "Herkese açık bilgileriniz ve sosyal hesaplarınız.")
}

// POST /hesabim/profil
func (s *Server) handleHesapProfilPost(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || len([]rune(name)) > 120 {
		s.hesapHata(w, r, "profil", "Ad soyad gerekli (en fazla 120 karakter).")
		return
	}
	bio := strings.TrimSpace(r.FormValue("bio"))
	if len([]rune(bio)) > 600 {
		bio = string([]rune(bio)[:600])
	}
	slug := slugifyHesap(r.FormValue("slug"))
	socials := map[string]string{}
	for _, f := range hesapSosyal {
		v := strings.TrimSpace(r.FormValue("s_" + f.Key))
		if v == "" {
			continue
		}
		// Yalnız http(s) veya @kullanıcı kabul edilir (XSS/javascript: engeli).
		if strings.HasPrefix(v, "@") {
			handle := strings.TrimPrefix(v, "@")
			v = hesapSosyalURL(f.Key, handle)
		} else if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			v = "https://" + v
		}
		if _, err := url.ParseRequestURI(v); err != nil {
			continue
		}
		socials[f.Key] = v
	}
	if err := repo.UpdateMemberProfile(r.Context(), s.pool, u.ID, name,
		r.FormValue("title"), bio, slug, socials); err != nil {
		if strings.Contains(err.Error(), "ux_users_slug") {
			s.hesapHata(w, r, "profil", "Bu kullanıcı adı zaten alınmış.")
			return
		}
		s.hesapHata(w, r, "profil", "Profil kaydedilemedi.")
		return
	}
	s.audit(r, "uye_profil", itoa(int(u.ID)))
	s.invalidateContentCache()
	http.Redirect(w, r, "/hesabim/profil?ok=profil", http.StatusFound)
}

// hesapSosyalURL @kullanıcı biçimini platform adresine çevirir.
func hesapSosyalURL(key, handle string) string {
	switch key {
	case "x":
		return "https://x.com/" + handle
	case "instagram":
		return "https://instagram.com/" + handle
	case "facebook":
		return "https://facebook.com/" + handle
	case "youtube":
		return "https://youtube.com/@" + handle
	case "linkedin":
		return "https://linkedin.com/in/" + handle
	case "tiktok":
		return "https://tiktok.com/@" + handle
	case "telegram":
		return "https://t.me/" + handle
	default:
		return "https://" + handle
	}
}

// slugifyHesap kullanıcı adını güvenli URL parçasına çevirir.
func slugifyHesap(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	var b strings.Builder
	for _, r := range in {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.':
			b.WriteRune('-')
		case r == 'ç':
			b.WriteRune('c')
		case r == 'ğ':
			b.WriteRune('g')
		case r == 'ı':
			b.WriteRune('i')
		case r == 'ö':
			b.WriteRune('o')
		case r == 'ş':
			b.WriteRune('s')
		case r == 'ü':
			b.WriteRune('u')
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

// POST /hesabim/avatar — kütüphaneden avatar seçer (0 = kaldır).
func (s *Server) handleHesapAvatar(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	id := parseID(r.FormValue("media_id"))
	if r.FormValue("kaldir") == "1" {
		id = 0
	}
	if id > 0 {
		m, err := repo.GetMedia(r.Context(), s.pool, id)
		if err != nil || m.Kind != "image" {
			s.hesapHata(w, r, "profil", "Seçilen görsel bulunamadı.")
			return
		}
	}
	var mid *int64
	if id > 0 {
		mid = &id
	}
	if err := repo.SetMemberAvatar(r.Context(), s.pool, u.ID, mid); err != nil {
		s.hesapHata(w, r, "profil", "Profil fotoğrafı kaydedilemedi.")
		return
	}
	http.Redirect(w, r, "/hesabim/profil?ok=avatar", http.StatusFound)
}

// POST /hesabim/sifre — mevcut şifreyi doğrulayıp yenisini yazar.
func (s *Server) handleHesapSifre(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	cur := r.FormValue("current")
	pass := r.FormValue("password")
	pass2 := r.FormValue("password2")
	if len(pass) < 8 {
		s.hesapHata(w, r, "profil", "Yeni şifre en az 8 karakter olmalı.")
		return
	}
	if pass != pass2 {
		s.hesapHata(w, r, "profil", "Yeni şifreler eşleşmiyor.")
		return
	}
	row, err := repo.GetUserByID(r.Context(), s.pool, u.ID)
	if err != nil || !auth.VerifyPassword(row.PasswordHash, cur) {
		s.hesapHata(w, r, "profil", "Mevcut şifreniz hatalı.")
		return
	}
	hash, err := auth.HashPassword(pass)
	if err != nil {
		s.hesapHata(w, r, "profil", "Şifre değiştirilemedi.")
		return
	}
	if err := repo.UpdateUserPassword(r.Context(), s.pool, u.ID, hash); err != nil {
		s.hesapHata(w, r, "profil", "Şifre değiştirilemedi.")
		return
	}
	s.audit(r, "uye_sifre", itoa(int(u.ID)))
	http.Redirect(w, r, "/hesabim/profil?ok=sifre", http.StatusFound)
}

// POST /hesabim/e-posta-gonder — doğrulama e-postasını yeniden gönderir.
func (s *Server) handleHesapEpostaGonder(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	s.sendVerifyEmail(r, u.ID, u.Email)
	http.Redirect(w, r, "/hesabim?ok=onay", http.StatusFound)
}

// GET /hesabim/abonelik
func (s *Server) handleHesapAbonelik(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "abonelik")
	if v := r.URL.Query().Get("ok"); v != "" {
		h.Info = hesapMesaj(v)
	}
	s.cdHesapRender(w, r, d, h, "abonelik", "Aboneliğim", "Plan, ödeme ve yenileme bilgileri.")
}

// GET /hesabim/yorumlar
func (s *Server) handleHesapYorumlar(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "yorumlar")
	items, err := repo.MemberComments(r.Context(), s.pool, u.ID, 60)
	if err != nil {
		h.Error = "Yorumlar yüklenemedi."
	}
	h.Comments = items
	s.cdHesapRender(w, r, d, h, "yorumlar", "Yorumlarım", "Haberlerde yazdığınız yorumlar.")
}

// POST /hesabim/abonelik/iptal — dönem sonunda iptal (erişim süre sonuna kadar).
func (s *Server) handleHesapAbonelikIptal(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	if r.FormValue("geri_al") == "1" {
		if err := repo.UndoCancellation(r.Context(), s.pool, u.ID); err != nil {
			s.hesapHata(w, r, "abonelik", "İptal geri alınamadı.")
			return
		}
		http.Redirect(w, r, "/hesabim/abonelik?ok=iptal-geri", http.StatusFound)
		return
	}
	if err := repo.CancelSubscriptionAtPeriodEnd(r.Context(), s.pool, u.ID); err != nil {
		s.hesapHata(w, r, "abonelik", "İptal talebi kaydedilemedi (aktif abonelik yok).")
		return
	}
	s.audit(r, "abonelik_iptal", itoa(int(u.ID)))
	if usr, err := repo.GetUserByID(r.Context(), s.pool, u.ID); err == nil {
		s.sendMail(usr.Email, "Abonelik iptal talebiniz alındı",
			"<p>Sayın "+usr.Name+",</p><p>Aboneliğiniz dönem sonuna kadar aktif kalacak; "+
				"sonrasında yenilenmeyecek. Fikrinizi değiştirirseniz hesabınızdan iptali geri alabilirsiniz.</p>")
	}
	_ = repo.AddNotification(r.Context(), s.pool, u.ID, "uyelik-iptal",
		"Abonelik iptal talebiniz alındı",
		"Dönem sonuna kadar erişiminiz devam eder.", "/hesabim/abonelik")
	http.Redirect(w, r, "/hesabim/abonelik?ok=iptal", http.StatusFound)
}

// GET /hesabim/takip — takip edilen kategori/etiket/yazar listesi.
func (s *Server) handleHesapTakip(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	d, h := s.hesapView(r, u, "takip")
	list, err := repo.ListFollows(r.Context(), s.pool, u.ID)
	if err != nil {
		h.Error = "Takip listesi yüklenemedi."
	}
	h.Follows = list
	s.cdHesapRender(w, r, d, h, "takip", "Takip ettiklerim", "Takip ettiğiniz konular ve yazarlar.")
}

// POST /hesabim/sil — KVKK: hesabı kalıcı siler (şifre onayıyla).
func (s *Server) handleHesapSil(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	row, err := repo.GetUserByID(r.Context(), s.pool, u.ID)
	if err != nil || !auth.VerifyPassword(row.PasswordHash, r.FormValue("password")) {
		s.hesapHata(w, r, "profil", "Hesabı silmek için şifrenizi doğru girin.")
		return
	}
	if r.FormValue("onay") != "1" {
		s.hesapHata(w, r, "profil", "Silme işlemini onaylamanız gerekiyor.")
		return
	}
	if err := repo.DeleteMember(r.Context(), s.pool, u.ID); err != nil {
		s.hesapHata(w, r, "profil", "Hesap silinemedi.")
		return
	}
	s.audit(r, "uye_sil", itoa(int(u.ID)))
	s.clearMemberCookie(w)
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		_ = auth.DeleteSession(r.Context(), s.pool, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie()})
	http.Redirect(w, r, "/?hesap=silindi", http.StatusFound)
}

// hesapHata hata mesajını ilgili sekmede gösterir.
func (s *Server) hesapHata(w http.ResponseWriter, r *http.Request, tab, msg string) {
	u, err := s.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/giris", http.StatusFound)
		return
	}
	name, baslik, aciklama := "profil", "Profilim", "Herkese açık bilgileriniz ve sosyal hesaplarınız."
	if tab == "abonelik" {
		name, baslik, aciklama = "abonelik", "Aboneliğim", "Plan, ödeme ve yenileme bilgileri."
	}
	d, h := s.hesapView(r, u, tab)
	h.Error = msg
	s.cdHesapRender(w, r, d, h, name, baslik, aciklama)
}

// hesapKaydedilenRender /hesabim/kaydedilenler sayfası.
func (s *Server) hesapKaydedilenRender(w http.ResponseWriter, r *http.Request, u auth.User, d *tmplData, h *hesapData) {
	items, err := repo.ListBookmarks(r.Context(), s.pool, u.ID, 60)
	if err != nil {
		h.Error = "Kaydedilenler yüklenemedi."
	}
	h.Bookmarks = items
	s.cdHesapRender(w, r, d, h, "kaydedilenler", "Kaydedilenler", "Daha sonra okumak için kaydettiğiniz haberler.")
}

// hesapBildirimRender /hesabim/bildirimler sayfası.
func (s *Server) hesapBildirimRender(w http.ResponseWriter, r *http.Request, u auth.User, d *tmplData, h *hesapData) {
	list, err := repo.ListNotifications(r.Context(), s.pool, u.ID, 40)
	if err != nil {
		h.Error = "Bildirimler yüklenemedi."
	}
	_ = repo.MarkNotificationsRead(r.Context(), s.pool, u.ID)
	h.Notifs = list
	h.Stats.Unread = 0
	s.cdHesapRender(w, r, d, h, "bildirimler", "Bildirimler", "Hesabınızla ilgili son gelişmeler.")
}

// hesapRozetRender /hesabim/rozetler sayfası.
func (s *Server) hesapRozetRender(w http.ResponseWriter, r *http.Request, u auth.User, d *tmplData, h *hesapData) {
	pts, _, err := repo.TouchActivity(r.Context(), s.pool, u.ID, 0)
	if err != nil {
		h.Error = "Başarımlar yüklenemedi."
	}
	lead, _ := repo.Leaderboard(r.Context(), s.pool, 20)
	h.Points = pts
	h.Defs = repo.Achievements()
	h.Leader = lead
	s.cdHesapRender(w, r, d, h, "rozetler", "Başarımlar", "Rozetler, puan ve liderlik tablosu.")
}

// uyeBildirimSayisi header zili için okunmamış sayı (0 = gizle).
func (s *Server) uyeBildirimSayisi(ctx context.Context, userID int64) int {
	return repo.UnreadNotifications(ctx, s.pool, userID)
}
