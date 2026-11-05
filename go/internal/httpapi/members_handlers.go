package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// handleMembersGet üyelik yönetim sayfası.
func (s *Server) handleMembersGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildMembersData(r)
	comp := templkit.UyelerPanel(s.uyelerViewFrom(r, d.Members, d.Flash, s.stripeSettings(r)))
	if IsHTMX(r) {
		s.renderTemplPanel(w, r, comp)
		return
	}
	s.renderTemplPage(w, r, d, comp)
}

func (s *Server) buildMembersData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Üyelikler (Paywall)")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ara := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, _ := repo.ListUsersAra(r.Context(), s.pool, ara)
	d.Members = rows
	return d
}

func (s *Server) renderMembersPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildMembersData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTemplPanel(w, r, templkit.UyelerPanel(s.uyelerViewFrom(r, d.Members, d.Flash, s.stripeSettings(r))))
}

func (s *Server) membersResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderMembersPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/members", kind, text)
}

// handleMembersSet üyeliği premium/yok yapar.
func (s *Server) handleMembersSet(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	plan := r.PathValue("plan")
	if plan != "premium" {
		plan = ""
	}
	if err := repo.UpdateMembership(r.Context(), s.pool, id, plan); err != nil {
		s.membersResult(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "membership_set", itoa(int(id))+" -> "+plan)
	s.membersResult(w, r, "info", "Üyelik güncellendi.")
}

// stripeSettings Stripe ayarlarını (section=stripe) okur.
func (s *Server) stripeSettings(r *http.Request) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"secret_key", "webhook_secret", "price_id", "aktif"} {
		if raw, err := repo.GetSetting(r.Context(), s.pool, "stripe", k); err == nil {
			out[k] = strings.Trim(strings.TrimSpace(string(raw)), `"`)
		}
	}
	return out
}

// handleMembersUye POST /admin/members/uye — üye kaydı anahtarını kaydeder.
func (s *Server) handleMembersUye(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	acik := "0"
	if r.FormValue("kayit_acik") == "1" {
		acik = "1"
	}
	if err := repo.SetSetting(r.Context(), s.pool, "uyelik", "kayit_acik", []byte(acik)); err != nil {
		s.membersResult(w, r, "error", "Ayar kaydedilemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	s.audit(r, "uye_kayit_ayari", "kayit_acik="+acik)
	s.membersResult(w, r, "info", "Üye kayıt ayarı güncellendi.")
}

// handleMembersStripe POST /admin/members/stripe — Stripe ayarlarını kaydeder.
// Sırlar boş bırakılırsa korunur; kapatmak tüm Stripe kayıtlarını siler.
func (s *Server) handleMembersStripe(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	set := func(key, val string) {
		_ = repo.SetSetting(r.Context(), s.pool, "stripe", key, []byte(strings.TrimSpace(val)))
	}
	aktif := strings.TrimSpace(r.FormValue("aktif"))
	if aktif == "" {
		// Kapatma = yapılandırmanın tamamı kaldırılır (boş ayar satırları
		// bırakmamak için silinir); idempotens olay kayıtları da temizlenir.
		_, _ = s.pool.Exec(r.Context(), `DELETE FROM settings WHERE section='stripe'`)
		_, _ = s.pool.Exec(r.Context(), `DELETE FROM stripe_events`)
		s.invalidateContentCache()
		s.audit(r, "stripe_ayar", "kapat")
		s.membersResult(w, r, "info", "Stripe kapatıldı; ayarlar temizlendi.")
		return
	}
	set("aktif", aktif)
	set("price_id", r.FormValue("price_id"))
	if v := strings.TrimSpace(r.FormValue("secret_key")); v != "" {
		set("secret_key", v)
	}
	if v := strings.TrimSpace(r.FormValue("webhook_secret")); v != "" {
		set("webhook_secret", v)
	}
	s.audit(r, "stripe_ayar", "kaydet")
	s.membersResult(w, r, "info", "Stripe ayarları kaydedildi.")
}

// handleMembersSil POST /admin/members/{id}/sil — kullanıcıyı siler (KVKK).
func (s *Server) handleMembersSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.membersResult(w, r, "error", "Geçersiz kullanıcı.")
		return
	}
	if err := repo.DeleteUser(r.Context(), s.pool, id); err != nil {
		s.membersResult(w, r, "error", "Silinemedi: kullanıcı içeriklere bağlı olabilir.")
		return
	}
	s.audit(r, "membership_sil", itoa(int(id)))
	s.membersResult(w, r, "info", "Kullanıcı silindi.")
}

// iyzicoSettings iyzico ayarlarını (section=iyzico) okur.
func (s *Server) iyzicoSettings(r *http.Request) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"api_key", "secret_key", "mode", "aktif", "callback_url"} {
		if raw, err := repo.GetSetting(r.Context(), s.pool, "iyzico", k); err == nil {
			out[k] = strings.Trim(strings.TrimSpace(string(raw)), `"`)
		}
	}
	return out
}

// handleMembersIyzico POST /admin/members/iyzico — iyzico ödeme ayarları.
// Gizli anahtar boş bırakılırsa korunur; kapatınca sırlar silinir.
func (s *Server) handleMembersIyzico(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	set := func(key, val string) {
		_ = repo.SetSetting(r.Context(), s.pool, "iyzico", key, []byte(strings.TrimSpace(val)))
	}
	aktif := r.FormValue("aktif") == "1"
	mode := r.FormValue("mode")
	if mode != "production" {
		mode = "sandbox"
	}
	set("api_key", r.FormValue("api_key"))
	set("mode", mode)
	if !aktif && strings.TrimSpace(r.FormValue("api_key")) == "" {
		set("secret_key", "")
	} else if v := strings.TrimSpace(r.FormValue("secret_key")); v != "" {
		set("secret_key", v)
	}
	set("aktif", boolOnOff(aktif))
	s.invalidateContentCache()
	s.audit(r, "iyzico_ayar", "kaydet")
	s.membersResult(w, r, "info", "iyzico ayarları kaydedildi.")
}

func boolOnOff(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// handleMembersKademe POST /admin/members/kademe — katman ekler/günceller.
func (s *Server) handleMembersKademe(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := parseID(r.FormValue("id"))
	tier := repo.Tier{
		ID:          id,
		Slug:        slugifyHesap(r.FormValue("slug")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		ColorHex:    strings.TrimSpace(r.FormValue("color_hex")),
		IsActive:    r.FormValue("is_active") == "1",
	}
	if tier.Slug == "" || tier.Name == "" {
		s.membersResult(w, r, "error", "Katman adı ve kısa ad (slug) gerekli.")
		return
	}
	if tier.ColorHex == "" || !strings.HasPrefix(tier.ColorHex, "#") {
		tier.ColorHex = "#71717a"
	}
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("weight"))); err == nil && n >= 1 && n <= 20 {
		tier.Weight = n
	}
	tier.MonthlyPrice = parsePrice(r.FormValue("monthly_price"))
	tier.AnnualPrice = parsePrice(r.FormValue("annual_price"))
	for _, line := range strings.Split(r.FormValue("perks"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tier.Perks = append(tier.Perks, line)
		}
	}
	if _, err := repo.SaveTier(r.Context(), s.pool, tier); err != nil {
		s.membersResult(w, r, "error", "Katman kaydedilemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	s.audit(r, "abonelik_kademe", tier.Slug)
	s.membersResult(w, r, "info", "Katman kaydedildi.")
}

// handleMembersKademeSil POST /admin/members/kademe/{id}/sil
func (s *Server) handleMembersKademeSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.membersResult(w, r, "error", "Geçersiz katman.")
		return
	}
	if err := repo.DeleteTier(r.Context(), s.pool, id); err != nil {
		s.membersResult(w, r, "error", "Katman silinemedi: "+err.Error())
		return
	}
	s.invalidateContentCache()
	s.audit(r, "abonelik_kademe_sil", itoa(int(id)))
	s.membersResult(w, r, "info", "Katman silindi.")
}

// handleMembersAbonelikAta POST /admin/members/abonelik — elle kademe atama
// ve süre uzatma (edirnegercek admin davranışı).
func (s *Server) handleMembersAbonelikAta(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	uid := parseID(r.FormValue("user_id"))
	if uid <= 0 {
		s.membersResult(w, r, "error", "Kullanıcı seçilmedi.")
		return
	}
	tierID := parseID(r.FormValue("tier_id"))
	cycle := normalizeCycle(r.FormValue("cycle"))
	days, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("days")))
	if tierID > 0 {
		if _, err := repo.ActivateSubscription(r.Context(), s.pool, uid, tierID, cycle); err != nil {
			s.membersResult(w, r, "error", "Abonelik atanamadı: "+err.Error())
			return
		}
		if days > 0 {
			_, _ = repo.ExtendSubscription(r.Context(), s.pool, uid, days)
		}
		s.audit(r, "abonelik_ata", itoa(int(uid))+" tier="+itoa(int(tierID)))
		s.membersResult(w, r, "info", "Abonelik atandı.")
		return
	}
	if err := repo.SetUserTier(r.Context(), s.pool, uid, nil, "", nil); err != nil {
		s.membersResult(w, r, "error", "Abonelik kaldırılamadı: "+err.Error())
		return
	}
	s.audit(r, "abonelik_kaldir", itoa(int(uid)))
	s.membersResult(w, r, "info", "Abonelik kaldırıldı (ücretsiz).")
}

// handleMembersOdemeIade POST /admin/members/odeme/{id}/iade
func (s *Server) handleMembersOdemeIade(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.membersResult(w, r, "error", "Geçersiz ödeme.")
		return
	}
	if err := repo.RefundPayment(r.Context(), s.pool, id, "admin iadesi"); err != nil {
		s.membersResult(w, r, "error", "İade işaretlenemedi: yalnız başarılı ödemeler iade edilebilir.")
		return
	}
	s.audit(r, "odeme_iade", itoa(int(id)))
	s.membersResult(w, r, "info", "Ödeme iade edildi olarak işaretlendi.")
}

// handleMembersKupon POST /admin/members/kupon — indirim kodu ekler/günceller.
func (s *Server) handleMembersKupon(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := repo.Promo{
		ID:          parseID(r.FormValue("id")),
		Code:        strings.ToUpper(strings.TrimSpace(r.FormValue("code"))),
		Description: strings.TrimSpace(r.FormValue("description")),
		TierSlug:    slugifyHesap(r.FormValue("tier_slug")),
		IsActive:    r.FormValue("is_active") == "1",
	}
	if p.Code == "" {
		s.membersResult(w, r, "error", "Kod gerekli.")
		return
	}
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("percent"))); err == nil && n > 0 && n <= 100 {
		p.Percent = n
	} else {
		s.membersResult(w, r, "error", "İndirim oranı 1-100 arasında olmalı.")
		return
	}
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("max_uses"))); err == nil && n >= 0 {
		p.MaxUses = n
	}
	if v := strings.TrimSpace(r.FormValue("expires_at")); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			p.ExpiresAt = &end
		}
	}
	if _, err := repo.SavePromo(r.Context(), s.pool, p); err != nil {
		s.membersResult(w, r, "error", "Kupon kaydedilemedi: "+err.Error())
		return
	}
	s.audit(r, "kupon_kaydet", p.Code)
	s.membersResult(w, r, "info", "İndirim kodu kaydedildi.")
}

// handleMembersKuponSil POST /admin/members/kupon/{id}/sil
func (s *Server) handleMembersKuponSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.membersResult(w, r, "error", "Geçersiz kupon.")
		return
	}
	if err := repo.DeletePromo(r.Context(), s.pool, id); err != nil {
		s.membersResult(w, r, "error", "Kupon silinemedi.")
		return
	}
	s.audit(r, "kupon_sil", itoa(int(id)))
	s.membersResult(w, r, "info", "Kupon silindi.")
}

// parsePrice "150,50" / "150.50" → float64.
func parsePrice(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	v, _ := strconv.ParseFloat(s, 64)
	if v < 0 {
		return 0
	}
	return v
}

// googleSettings Google giriş ayarlarını (section=google) okur.
func (s *Server) googleSettings(r *http.Request) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"client_id", "client_secret"} {
		if raw, err := repo.GetSetting(r.Context(), s.pool, "google", k); err == nil {
			out[k] = strings.Trim(strings.TrimSpace(string(raw)), `"`)
		}
	}
	return out
}

// handleMembersGoogle POST /admin/members/google — Google giriş ayarları.
// Client secret boş bırakılırsa korunur.
func (s *Server) handleMembersGoogle(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	set := func(key, val string) {
		_ = repo.SetSetting(r.Context(), s.pool, "google", key, []byte(strings.TrimSpace(val)))
	}
	set("client_id", r.FormValue("client_id"))
	// Secret boş bırakılırsa korunur; giriş KAPATILIRSA (client_id boş) silinir.
	if strings.TrimSpace(r.FormValue("client_id")) == "" {
		set("client_secret", "")
	} else if v := strings.TrimSpace(r.FormValue("client_secret")); v != "" {
		set("client_secret", v)
	}
	s.audit(r, "google_ayar", "kaydet")
	s.membersResult(w, r, "info", "Google giriş ayarları kaydedildi.")
}
