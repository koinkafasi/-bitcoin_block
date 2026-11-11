package httpapi

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/auth"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Abonelik katmanları: fiyat sayfası, iyzico checkout, dönüş ----

// abonelikData /abonelik ve /hesabim/abonelik görünümü.
type abonelikData struct {
	Tiers        []repo.Tier
	CurrentID    int64
	CurrentName  string
	Cycle        string // monthly | annual (sayfa varsayılanı)
	IyzicoOn     bool
	StripeOn     bool
	Error        string
	Info         string
	SelectedSlug string
	IsLoggedIn   bool
	MemberName   string
	MemberEmail  string
	CheckoutHTML template.HTML
}

// iyzicoConfig ayarlardan iyzico yapılandırmasını okur (section=iyzico).
func (s *Server) iyzicoConfig(r *http.Request) platform.IyzicoConfig {
	get := func(key string) string {
		raw, err := repo.GetSetting(r.Context(), s.pool, "iyzico", key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(strings.Trim(string(raw), `"`))
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	base := scheme + "://" + r.Host
	cb := get("callback_url")
	if cb == "" {
		cb = base + s.langBase(langOf(r, s)) + "/abonelik/geri-donus"
	}
	return platform.IyzicoConfig{
		APIKey:      get("api_key"),
		SecretKey:   get("secret_key"),
		Sandbox:     get("mode") != "production",
		CallbackURL: cb,
	}
}

// abonelikAktif iyzico ile ödeme alınıp alınamayacağını söyler.
func (s *Server) abonelikAktif(r *http.Request) bool {
	cfg := s.iyzicoConfig(r)
	if !cfg.Enabled() {
		return false
	}
	raw, err := repo.GetSetting(r.Context(), s.pool, "iyzico", "aktif")
	if err != nil {
		return false
	}
	return strings.TrimSpace(strings.Trim(string(raw), `"`)) == "1" ||
		strings.TrimSpace(strings.Trim(string(raw), `"`)) == "true"
}

// tierBySlugOrID katmanı slug veya sayısal kimlikle bulur.
func (s *Server) tierBySlugOrID(ctx context.Context, key string) (repo.Tier, error) {
	if id := parseID(key); id > 0 {
		return repo.GetTier(ctx, s.pool, id)
	}
	return repo.GetTierBySlug(ctx, s.pool, key)
}

// GET /abonelik — halka açık fiyat sayfası.
func (s *Server) handleAbonelikPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), "Abonelik")
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)

	ab := &abonelikData{Cycle: "monthly", IyzicoOn: s.abonelikAktif(r)}
	tiers, err := repo.ListTiers(ctx, s.pool, true)
	if err != nil {
		ab.Error = "Abonelik paketleri yüklenemedi."
	}
	ab.Tiers = tiers
	if stripe := s.stripeSettings(r); stripe["aktif"] == "1" && stripe["secret_key"] != "" {
		ab.StripeOn = true
	}
	if !d.MemberLoggedIn {
		if u, err := s.currentUser(r); err == nil && u.ID > 0 {
			d.MemberLoggedIn = true
			d.MemberName = u.Name
			d.MemberInitial = memberInitial(u.Name)
			d.MemberURL = "/hesabim"
			ab.IsLoggedIn = true
			ab.MemberName = u.Name
			ab.MemberEmail = u.Email
			if t, ok := repo.UserTier(ctx, s.pool, u.ID); ok {
				ab.CurrentID = t.ID
				ab.CurrentName = t.Name
			}
		}
	}
	if v := r.URL.Query().Get("plan"); v != "" {
		ab.SelectedSlug = v
	}
	d.Abonelik = ab
	s.renderPublic(w, r, "abonelik", d, cacheControlPublic)
}

// POST /abonelik/odeme — iyzico hosted checkout başlatır.
// sozlesmeSurumu, onaylanan ön bilgilendirme ve mesafeli satış sözleşmesi
// metinlerinin sürümüdür. Metinler güncellenirse burası da artırılmalı:
// kayıt, kullanıcının HANGİ metni onayladığını göstermeli.
const sozlesmeSurumu = "mesafeli-satis-sozlesmesi@v1"

func (s *Server) handleAbonelikOdeme(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireMember(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	if !s.abonelikAktif(r) {
		s.abonelikHata(w, r, "Ödeme altyapısı etkin değil.")
		return
	}
	key := strings.TrimSpace(r.FormValue("tier"))
	cycle := normalizeCycle(r.FormValue("cycle"))
	tier, err := s.tierBySlugOrID(r.Context(), key)
	if err != nil || !tier.IsActive {
		s.abonelikHata(w, r, "Seçilen paket bulunamadı.")
		return
	}
	// Ön bilgilendirme ve mesafeli satış sözleşmesi onayı — SUNUCU tarafında
	// zorunlu. Formdaki `required` yalnız tarayıcıda çalışır; istek doğrudan
	// gönderilirse onaysız sözleşme kurulurdu.
	//
	// Mesafeli Sözleşmeler Yönetmeliği uyarınca sağlayıcı, sözleşme
	// kurulmadan önce ön bilgilendirme yapmak zorundadır. Tüketici cayma
	// hakkı konusunda gereği gibi bilgilendirilmezse 14 günlük süre işlemez
	// ve cayma hakkı, sürenin bittiği tarihten bir yıl sonrasına kadar
	// kullanılabilir — yani eksik bilgilendirme riski aylarca açık kalır.
	if strings.TrimSpace(r.FormValue("sozlesme_onay")) != "1" {
		s.abonelikHata(w, r, "Ödemeye geçmek için Ön Bilgilendirme Formu'nu ve "+
			"Mesafeli Satış Sözleşmesi'ni onaylamanız gerekir.")
		return
	}

	tckn := strings.TrimSpace(r.FormValue("tckn"))
	if len(tckn) != 11 {
		s.abonelikHata(w, r, "TCKN 11 haneli olmalıdır (iyzico zorunlu tutar).")
		return
	}
	city := strings.TrimSpace(r.FormValue("city"))
	if city == "" {
		city = "Istanbul"
	}
	addr := strings.TrimSpace(r.FormValue("address"))
	if addr == "" {
		addr = "Belirtilmedi"
	}
	price := tier.MonthlyPrice
	if cycle == "annual" {
		price = tier.AnnualPrice
	}
	if price <= 0 {
		s.abonelikHata(w, r, "Bu paket için fiyat tanımlı değil.")
		return
	}
	// Kabul kaydı: hangi üyenin, hangi paket/döngü için, hangi sözleşme
	// sürümünü, ne zaman onayladığı. İspat yükü sağlayıcıdadır; sözleşme
	// metinleri değişebildiği için SÜRÜM de yazılır (sozlesmeSurumu).
	// TCKN bilerek KAYDEDİLMEZ: iyzico'ya iletilen bir zorunluluk, bizim
	// saklamamız gereken bir veri değil (veri minimizasyonu).
	s.audit(r, "abonelik_sozlesme_onay", fmt.Sprintf(
		"uye=%d paket=%s dongu=%s sozlesme=%s", u.ID, tier.Slug, cycle, sozlesmeSurumu))

	// İndirim kuponu (opsiyonel).
	discount := 0.0
	coupon := strings.ToUpper(strings.TrimSpace(r.FormValue("kupon")))
	if coupon != "" {
		promo, perr := repo.ValidPromo(r.Context(), s.pool, coupon, tier.Slug)
		if perr != nil {
			s.abonelikHata(w, r, "İndirim kodu geçersiz veya süresi dolmuş.")
			return
		}
		discount = price * float64(promo.Percent) / 100
		price -= discount
		if price < 0 {
			price = 0
		}
	}
	conv := platform.NewConversationID()
	pid, err := repo.CreatePayment(r.Context(), s.pool, u.ID, &tier.ID, "iyzico", conv,
		price, discount, "TRY", cycle, coupon)
	if err != nil {
		s.abonelikHata(w, r, "Ödeme kaydı oluşturulamadı: "+err.Error())
		return
	}
	_ = pid
	name, surname := splitName(u.Name)
	cfg := s.iyzicoConfig(r)
	res, err := cfg.Initialize(r.Context(), platform.IyzicoInitRequest{
		ConversationID: conv,
		Price:          price,
		Currency:       "TRY",
		BasketID:       tier.Slug + "-" + cycle,
		Buyer: platform.IyzicoBuyer{
			ID:               itoa(int(u.ID)),
			Name:             name,
			Surname:          surname,
			Email:            u.Email,
			IdentityNumber:   tckn,
			RegistrationAddr: addr,
			City:             city,
			Country:          "Turkey",
			IP:               clientIP(r),
		},
		Item: platform.IyzicoItem{
			ID:       tier.Slug,
			Name:     tier.Name + " (" + cycleTR(cycle) + ")",
			Category: "Abonelik",
			Price:    price,
			ItemType: "VIRTUAL",
		},
	})
	if err != nil {
		_ = repo.FailPayment(r.Context(), s.pool, conv, res.Raw)
		s.abonelikHata(w, r, "Ödeme başlatılamadı: "+err.Error())
		return
	}
	if res.Token != "" {
		_ = repo.SetPaymentToken(r.Context(), s.pool, conv, res.Token)
	}
	s.audit(r, "abonelik_odeme_baslat", tier.Slug+"/"+cycle)
	if res.PaymentPageURL != "" {
		http.Redirect(w, r, res.PaymentPageURL, http.StatusFound)
		return
	}
	// Bazı hesaplarda yalnız gömülü form döner.
	d := s.abonelikViewData(r, "Ödeme")
	d.Abonelik.Info = "Ödeme sayfası hazırlanıyor…"
	d.Abonelik.CheckoutHTML = template.HTML(res.CheckoutForm)
	s.renderPublic(w, r, "abonelik-odeme", d, "no-store")
}

// POST /abonelik/geri-donus — iyzico callback (token ile doğrulama).
func (s *Server) handleAbonelikGeriDonus(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		s.abonelikHata(w, r, "Ödeme yanıtı eksik (token yok).")
		return
	}
	// conversationId iyzico yanıtında gelmez; token'ı bekleyen kayıtlarla eşle.
	pay, err := s.paymentByToken(r.Context(), token)
	if err != nil {
		s.abonelikHata(w, r, "Ödeme kaydı bulunamadı.")
		return
	}
	cfg := s.iyzicoConfig(r)
	res, err := cfg.Retrieve(r.Context(), token, pay.ConversationID)
	if err != nil {
		_ = repo.FailPayment(r.Context(), s.pool, pay.ConversationID, res.Raw)
		s.abonelikHata(w, r, "Ödeme doğrulanamadı: "+err.Error())
		return
	}
	if !res.Success() {
		_ = repo.FailPayment(r.Context(), s.pool, pay.ConversationID, res.Raw)
		msg := res.ErrorMessage
		if msg == "" {
			msg = "Ödeme tamamlanmadı."
		}
		s.abonelikHata(w, r, msg)
		return
	}
	// Tekrar işleme koruması: kayıt zaten başarılıysa kademe ikinci kez uzatılmaz.
	if pay.Status == "success" {
		http.Redirect(w, r, "/hesabim/abonelik?ok=odeme", http.StatusFound)
		return
	}
	if pay.TierID == nil {
		s.abonelikHata(w, r, "Ödeme kaydında paket yok.")
		return
	}
	until, err := repo.ActivateSubscription(r.Context(), s.pool, pay.UserID, *pay.TierID, pay.BillingCycle)
	if err != nil {
		s.abonelikHata(w, r, "Abonelik etkinleştirilemedi: "+err.Error())
		return
	}
	_ = repo.CompletePayment(r.Context(), s.pool, pay.ConversationID, res.PaymentID, res.Raw)
	// Kupon kullanımını düş (kademe filtresi boş: kod zaten ödeme anında doğrulandı).
	if pay.CouponCode != "" {
		if promo, perr := repo.ValidPromo(r.Context(), s.pool, pay.CouponCode, ""); perr == nil {
			_ = repo.UsePromo(r.Context(), s.pool, promo.ID)
		}
	}
	if t, err := repo.GetTier(r.Context(), s.pool, *pay.TierID); err == nil {
		_ = repo.AddNotification(r.Context(), s.pool, pay.UserID, "uyelik",
			t.Name+" aboneliğiniz başladı",
			"Aboneliğiniz "+until.Format("02.01.2006")+" tarihine kadar geçerli.", "/hesabim/abonelik")
		if u, err := repo.GetUserByID(r.Context(), s.pool, pay.UserID); err == nil {
			note := ""
			if pay.Discount > 0 {
				note = " (indirim: " + formatTL(pay.Discount) + " ₺)"
			}
			s.sendMail(u.Email, "Aboneliğiniz başladı — "+t.Name,
				"<p>Sayın "+template.HTMLEscapeString(u.Name)+",</p><p><b>"+template.HTMLEscapeString(t.Name)+"</b> aboneliğiniz başladı. Bitiş: <b>"+
					until.Format("02.01.2006")+"</b>.</p><p>Tutar: <b>"+formatTL(pay.Amount)+" ₺</b>"+
					note+"</p><p>Hesabınız: /hesabim/abonelik</p>")
		}
	}
	s.audit(r, "abonelik_odeme_basarili", pay.ConversationID)
	http.Redirect(w, r, "/hesabim/abonelik?ok=odeme", http.StatusFound)
}

// paymentByToken iyzico callback token'ını ödeme kaydıyla eşler.
// Token checkout başlatılırken kaydedilir; başka bir bekleyen kayda
// eşleşme (ödeme çalma) mümkün değildir.
func (s *Server) paymentByToken(ctx context.Context, token string) (repo.Payment, error) {
	return repo.GetPaymentByToken(ctx, s.pool, token)
}

// abonelikViewData üyelik/ödeme sayfaları için ortak veri.
func (s *Server) abonelikViewData(r *http.Request, title string) *tmplData {
	ctx := r.Context()
	locale := langOf(r, s)
	d := s.baseView(locale, s.langBase(locale), title)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	ab := &abonelikData{Cycle: "monthly", IyzicoOn: s.abonelikAktif(r)}
	ab.Tiers, _ = repo.ListTiers(ctx, s.pool, true)
	if u, err := s.currentUser(r); err == nil && u.ID > 0 {
		d.MemberLoggedIn = true
		d.MemberName = u.Name
		d.MemberInitial = memberInitial(u.Name)
		d.MemberURL = "/hesabim"
		ab.IsLoggedIn = true
		ab.MemberName = u.Name
		ab.MemberEmail = u.Email
		if t, ok := repo.UserTier(ctx, s.pool, u.ID); ok {
			ab.CurrentID = t.ID
			ab.CurrentName = t.Name
		}
	}
	d.Abonelik = ab
	return d
}

func (s *Server) abonelikHata(w http.ResponseWriter, r *http.Request, msg string) {
	d := s.abonelikViewData(r, "Abonelik")
	d.Abonelik.Error = msg
	s.renderPublic(w, r, "abonelik", d, "no-store")
}

func normalizeCycle(v string) string {
	if v == "annual" || v == "yillik" || v == "yearly" {
		return "annual"
	}
	return "monthly"
}

func cycleTR(cycle string) string {
	if cycle == "annual" {
		return "yıllık"
	}
	return "aylık"
}

func splitName(full string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(full))
	if len(parts) == 0 {
		return "Abone", "Uye"
	}
	if len(parts) == 1 {
		return parts[0], "Uye"
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// ---- Katman erişim kuralı ----

// contentRequiredWeight içerik için gereken en düşük katman ağırlığı.
// Özel içerik işaretli ama ağırlık verilmemişse 1 kabul edilir.
func contentRequiredWeight(isExclusive bool, minWeight int) int {
	if minWeight > 0 {
		return minWeight
	}
	if isExclusive {
		return 1
	}
	return 0
}

// tierAllows erişim ağırlığı yeterli mi?
func tierAllows(userWeight, required int) bool {
	return required <= 0 || userWeight >= required
}

// memberWeightAkif istek sahibinin katman ağırlığı (editoryal roller sınırsız).
func (s *Server) memberWeight(r *http.Request) int {
	if u, err := s.currentUser(r); err == nil && u.ID > 0 {
		if auth.StaffRoles[u.Role] {
			return 999
		}
		return repo.UserTierWeight(r.Context(), s.pool, u.ID)
	}
	// Çerezle tanınan eski premium üye.
	if em := s.memberCookieEmail(r); em != "" {
		if uid, ok, _ := repo.FindUserByEmailID(r.Context(), s.pool, em); ok {
			return repo.UserTierWeight(r.Context(), s.pool, uid)
		}
	}
	return 0
}

// ücretsiz erişimde kilit kararı (paywall anahtarı kapalıysa açık).
func (s *Server) paywallAktif(ctx context.Context) bool {
	b, err := repo.GetSetting(ctx, s.pool, "paywall", "aktif")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "true"
}

// formatTL tutarı "1.234,56" biçiminde yazar.
func formatTL(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String() + "," + parts[1]
}
