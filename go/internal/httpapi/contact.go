package httpapi

// İletişim bilgileri ve iletişim formu (Faz 145).
//
// Sorun: kurumsal sayfalarda e-posta adresleri ve alan adı metnin içine
// gömülüydü. Alan adı değişince (ör. theblock.com.tr → yeni alan adı) tüm
// sayfaları tek tek düzeltmek gerekiyordu.
//
// Çözüm: tek bir "İletişim" ayarı (alan adı + adresler + telefon/adres) ve
// sayfa metinlerinde {eposta_info}, {domain} gibi yer tutucular. Yer tutucular
// hem sayfa gövdesinde hem sayfa metinlerinde render sırasında doldurulur.
// Böylece yönetimden alan adı değiştirildiğinde tüm sayfalar aynı anda
// güncellenir; adresler istenirse tek tek de özelleştirilebilir.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

// ContactConfig yönetimden düzenlenen iletişim bilgileri.
type ContactConfig struct {
	Domain    string `json:"domain"`
	Info      string `json:"info"`
	Haber     string `json:"haber"`
	Reklam    string `json:"reklam"`
	Destek    string `json:"destek"`
	Telif     string `json:"telif"`
	Duzeltme  string `json:"duzeltme"`
	KVKK      string `json:"kvkk"`
	Kariyer   string `json:"kariyer"`
	Telefon   string `json:"telefon"`
	Adres     string `json:"adres"`
	Calisma   string `json:"calisma"`
	FormAktif bool   `json:"form_aktif"`
}

// contactEmails düzenlenebilir adres alanlarını (etiket + değer işaretçisi)
// tek yerde toplar: kaydetme, varsayılan üretme ve şablon aynı listeyi kullanır.
func (c *ContactConfig) emailFields() []struct {
	Key   string
	Label string
	Hint  string
	Ptr   *string
} {
	return []struct {
		Key   string
		Label string
		Hint  string
		Ptr   *string
	}{
		{"info", "Genel iletişim", "Sayfalarda {eposta_info} olarak kullanılır.", &c.Info},
		{"haber", "Haber merkezi / ihbar", "{eposta_haber}", &c.Haber},
		{"reklam", "Reklam ve iş birliği", "{eposta_reklam}", &c.Reklam},
		{"destek", "Teknik destek", "{eposta_destek}", &c.Destek},
		{"telif", "Telif bildirimi", "{eposta_telif}", &c.Telif},
		{"duzeltme", "Düzeltme / yayın etiği", "{eposta_duzeltme}", &c.Duzeltme},
		{"kvkk", "KVKK başvuruları", "{eposta_kvkk}", &c.KVKK},
		{"kariyer", "İnsan kaynakları", "{eposta_kariyer}", &c.Kariyer},
	}
}

// contactDefaults alan adından varsayılan adresleri üretir.
func contactDefaults(domain string) ContactConfig {
	d := strings.TrimSpace(strings.ToLower(domain))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if d == "" {
		d = "theblock.com.tr"
	}
	return ContactConfig{
		Domain:    d,
		Info:      "info@" + d,
		Haber:     "haber@" + d,
		Reklam:    "reklam@" + d,
		Destek:    "destek@" + d,
		Telif:     "telif@" + d,
		Duzeltme:  "duzeltme@" + d,
		KVKK:      "info@" + d,
		Kariyer:   "kariyer@" + d,
		FormAktif: true,
	}
}

// normalizeContactConfig boş alanları alan adına göre tamamlar ve alan adını
// sadeleştirir (şema/yol yok, küçük harf).
func normalizeContactConfig(c ContactConfig) ContactConfig {
	def := contactDefaults(c.Domain)
	c.Domain = def.Domain
	for _, f := range c.emailFields() {
		*f.Ptr = strings.TrimSpace(*f.Ptr)
	}
	if c.Info == "" {
		c.Info = def.Info
	}
	if c.Haber == "" {
		c.Haber = def.Haber
	}
	if c.Reklam == "" {
		c.Reklam = def.Reklam
	}
	if c.Destek == "" {
		c.Destek = def.Destek
	}
	if c.Telif == "" {
		c.Telif = def.Telif
	}
	if c.Duzeltme == "" {
		c.Duzeltme = def.Duzeltme
	}
	if c.KVKK == "" {
		c.KVKK = c.Info
	}
	if c.Kariyer == "" {
		c.Kariyer = def.Kariyer
	}
	c.Telefon = strings.TrimSpace(c.Telefon)
	c.Adres = strings.TrimSpace(c.Adres)
	c.Calisma = strings.TrimSpace(c.Calisma)
	return c
}

// contactConfig kayıtlı iletişim ayarını (yoksa varsayılanı) döndürür.
func (s *Server) contactConfig(ctx context.Context) ContactConfig {
	var c ContactConfig
	if raw := s.cachedSetting(ctx, "iletisim", "ana"); len(raw) > 0 {
		if json.Unmarshal(raw, &c) == nil {
			return normalizeContactConfig(c)
		}
	}
	return contactDefaults(s.defaultDomain())
}

// defaultDomain site alan adını yapılandırmadan (yoksa sabitten) üretir.
func (s *Server) defaultDomain() string {
	base := s.cfg.BaseURL
	if base == "" {
		return "theblock.com.tr"
	}
	if u, err := url.Parse(base); err == nil && u.Host != "" {
		host := u.Hostname()
		if host != "" && host != "localhost" {
			return host
		}
	}
	return "theblock.com.tr"
}

// contactTokens metinlerde kullanılan yer tutucuları üretir.
func contactTokens(c ContactConfig) map[string]string {
	return map[string]string{
		"domain":          c.Domain,
		"eposta_info":     c.Info,
		"eposta_haber":    c.Haber,
		"eposta_reklam":   c.Reklam,
		"eposta_destek":   c.Destek,
		"eposta_telif":    c.Telif,
		"eposta_duzeltme": c.Duzeltme,
		"eposta_kvkk":     c.KVKK,
		"eposta_kariyer":  c.Kariyer,
		"eposta":          c.Info,
		"telefon":         c.Telefon,
		"adres":           c.Adres,
		"calisma":         c.Calisma,
	}
}

// applyContactTokens metindeki {domain}, {eposta_info}… yer tutucularını
// yönetimdeki iletişim bilgileriyle doldurur.
func (s *Server) applyContactTokens(ctx context.Context, text string) string {
	if text == "" || !strings.Contains(text, "{") {
		return text
	}
	return pageTextVars(text, contactTokens(s.contactConfig(ctx)))
}

// applyContactTokensWith saf sürüm: verilen ayarla doldurur (test/önizleme).
func applyContactTokensWith(text string, c ContactConfig) string {
	if text == "" || !strings.Contains(text, "{") {
		return text
	}
	return pageTextVars(text, contactTokens(c))
}

// shiftEmailsToDomain alan adı değiştiğinde eski alan adını taşıyan adresleri
// yeni alan adına taşır. Başka alan adlı özel adreslere dokunmaz.
func shiftEmailsToDomain(c ContactConfig, oldDomain string) ContactConfig {
	oldDomain = strings.TrimSpace(strings.ToLower(oldDomain))
	if oldDomain == "" || strings.EqualFold(oldDomain, c.Domain) {
		return c
	}
	from := "@" + oldDomain
	to := "@" + c.Domain
	for _, f := range c.emailFields() {
		cur := *f.Ptr
		if strings.HasSuffix(strings.ToLower(cur), from) {
			*f.Ptr = cur[:len(cur)-len(from)] + to
		}
	}
	return c
}

// contactView şablonun iletişim bloğunu basması için hazır görünüm.
type contactView struct {
	Domain    string
	Info      string
	Haber     string
	Reklam    string
	Destek    string
	Telif     string
	Duzeltme  string
	KVKK      string
	Kariyer   string
	Telefon   string
	Adres     string
	Calisma   string
	FormAktif bool
	Konular   []contactTopic
}

type contactTopic struct {
	Key  string
	Name string
}

// contactTopics formun konu seçenekleri.
func contactTopics() []contactTopic {
	return []contactTopic{
		{"haber", "Haber ihbarı"},
		{"duzeltme", "Düzeltme / yayın etiği"},
		{"reklam", "Reklam ve iş birliği"},
		{"kvkk", "KVKK / kişisel veriler"},
		{"teknik", "Teknik destek"},
		{"kariyer", "Kariyer / staj"},
		{"diger", "Diğer"},
	}
}

func contactViewOf(c ContactConfig) *contactView {
	return &contactView{
		Domain: c.Domain, Info: c.Info, Haber: c.Haber, Reklam: c.Reklam,
		Destek: c.Destek, Telif: c.Telif, Duzeltme: c.Duzeltme, KVKK: c.KVKK,
		Kariyer: c.Kariyer, Telefon: c.Telefon, Adres: c.Adres, Calisma: c.Calisma,
		FormAktif: c.FormAktif, Konular: contactTopics(),
	}
}

// ---- İletişim formu gönderimi ---------------------------------------------

// contactLimiter basit IP + dakika sınırı: formun spam ile şişirilmesini
// engeller (tek süreç içi, yeniden başlatmada sıfırlanır).
var contactLimiter = struct {
	mu   sync.Mutex
	last map[string]time.Time
}{last: map[string]time.Time{}}

func contactRateLimited(ip string) bool {
	now := time.Now()
	contactLimiter.mu.Lock()
	defer contactLimiter.mu.Unlock()
	if t, ok := contactLimiter.last[ip]; ok && now.Sub(t) < 45*time.Second {
		return true
	}
	contactLimiter.last[ip] = now
	// Ara sıra temizlik: haritanın sınırsız büyümesini önler.
	if len(contactLimiter.last) > 5000 {
		for k, v := range contactLimiter.last {
			if now.Sub(v) > 10*time.Minute {
				delete(contactLimiter.last, k)
			}
		}
	}
	return false
}

func validEmail(s string) bool {
	a, err := mail.ParseAddress(strings.TrimSpace(s))
	return err == nil && a.Address != "" && strings.Contains(a.Address, "@")
}

// handleContactSubmit /iletisim formunu alır ve mesajı kaydeder.
func (s *Server) handleContactSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	cfg := s.contactConfig(ctx)
	if !cfg.FormAktif {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "error", "İletişim formu şu anda kapalı.")
		return
	}
	// Bot tuzağı: gizli alan doldurulduysa sessizce başarılı görün.
	if strings.TrimSpace(r.FormValue("web")) != "" {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim", "ok", "Mesajınız alındı. Teşekkürler.")
		return
	}
	ip := clientIP(r)
	if contactRateLimited(ip) {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "error",
			"Çok sık gönderim yapıldı. Lütfen bir dakika sonra tekrar deneyin.")
		return
	}
	name := strings.TrimSpace(r.FormValue("ad"))
	email := strings.TrimSpace(r.FormValue("eposta"))
	phone := strings.TrimSpace(r.FormValue("telefon"))
	subject := strings.TrimSpace(r.FormValue("konu"))
	body := strings.TrimSpace(r.FormValue("mesaj"))
	if r.FormValue("onay") == "" {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "error",
			"Devam etmek için aydınlatma metnini onaylamanız gerekir.")
		return
	}
	if name == "" || body == "" || !validEmail(email) {
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "error",
			"Ad, geçerli bir e-posta ve mesaj zorunludur.")
		return
	}
	if len([]rune(name)) > 120 {
		name = string([]rune(name)[:120])
	}
	if len([]rune(body)) > 5000 {
		body = string([]rune(body)[:5000])
	}
	if subject == "" {
		subject = "diger"
	}
	known := false
	for _, t := range contactTopics() {
		if t.Key == subject {
			known = true
			break
		}
	}
	if !known {
		subject = "diger"
	}
	if _, err := repo.AddContactMessage(ctx, s.pool, repo.ContactMessage{
		SiteID: s.curSiteID(r), Name: name, Email: email, Phone: phone,
		Subject: subject, Body: body, IP: ip, UserAgent: r.UserAgent(),
	}); err != nil {
		s.logger.Warn("iletişim mesajı kaydedilemedi", "err", err)
		s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "error",
			"Mesaj kaydedilemedi. Lütfen daha sonra tekrar deneyin.")
		return
	}
	s.audit(r, "iletisim_mesaj", subject+": "+name)
	s.flashRedirect(w, r, s.langBase(langOf(r, s))+"/iletisim#iletisim-form", "ok",
		"Mesajınız bize ulaştı. En kısa sürede yanıt vereceğiz.")
}
