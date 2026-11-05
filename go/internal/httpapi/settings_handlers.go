package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// settingsData Ayarlar sayfası verisidir.
type settingsData struct {
	Tab           string
	SiteBaslik    string
	Slogan        string
	Dil           string
	BakimModu     bool
	BakimMesaji   string
	SiteFont      string
	BaslikFont    string
	Theme         *ThemeConfig
	ModulSirasi   string
	IzinlerJSON   string
	GCode         string
	GVerification string
	AdSense       string
	RobotsHTML    string
	AIProvider    string
	AIModel       string
	AIBaseURL     string
	// AIKey KALDIRILDI: sağlayıcı anahtarı hiçbir şablonda gösterilmiyordu ama
	// ayar verisi her panel sayfası yüklemesinde anahtarı belleğe taşıyordu.
	// Anahtar artık yalnız ortam değişkeninden (OMNIROUTE_API_KEY) okunur.
	AIEnabled bool
	Types     []repo.ContentType
	IzinKeys  []string
	IzinMap   map[string][]string
	Logs      []repo.LogRow
	// Kunye BİK tebliği md. 8/1-d künye alanları. Deger yalnızca settings
	// "kunye" bölümünde SAKLANAN değerdir; kaydet boş bırakılırsa iletişim
	// ayarlarından gelen yedek devreye girer (bkz. bikKunyeDoldur).
	Kunye      []bikKunyeAlan
	KunyeEksik []string
	KunyeYedek map[string]string
}

// handleSettingsGet ayarları yükler ve sekmeyi gösterir.
func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildSettingsData(r)
	if IsHTMX(r) {
		s.renderAyarlarPanelV2(w, r, d)
		return
	}
	s.renderAyarlarPageV2(w, r, d)
}

func (s *Server) buildSettingsData(r *http.Request) *tmplData {
	ctx := r.Context()
	tab := r.URL.Query().Get("seke")
	if tab == "" {
		tab = "genel"
	}

	genel, _ := repo.GetSettings(ctx, s.pool, "genel")
	moduller, _ := repo.GetSettings(ctx, s.pool, "moduller")
	bag, _ := repo.GetSettings(ctx, s.pool, "baglantilar")
	aiSet, _ := repo.GetSettings(ctx, s.pool, "yapay_zeka")

	d := s.baseView("tr", "/admin", "Ayarlar")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	sc := &settingsData{Tab: tab}
	sc.SiteBaslik = jsonStr(genel["site_baslik"])
	sc.Slogan = jsonStr(genel["slogan"])
	sc.Dil = jsonStr(genel["dil"])
	if bm := obj(genel["bakim_modu"]); bm != nil {
		sc.BakimModu = bm["aktif"] == true
		sc.BakimMesaji, _ = bm["mesaj"].(string)
	}
	sc.Theme = s.themeConfig(ctx)
	sc.ModulSirasi = prettyJSON(moduller["anasayfa_sirasi"])
	sc.IzinlerJSON = prettyJSON(moduller["izinler"])
	sc.IzinMap = map[string][]string{}
	_ = json.Unmarshal(moduller["izinler"], &sc.IzinMap)
	if meta := obj(bag["meta"]); meta != nil {
		sc.GVerification, _ = meta["g_verification"].(string)
		// Google "arama kodu" = GA4 ölçüm kimliği. TEK kanonik kaynak
		// trend/ga4_olcum'dur (GA4 paneli de aynı anahtarı yazar); eski
		// baglantilar.meta.g_code yalnız geriye dönük yedek olarak okunur.
		// Neden: aynı değer iki yerde tutulunca biri güncellenip diğeri
		// unutuluyor ve iki panel farklı şey gösteriyordu.
		sc.GCode = s.ga4Measurement(ctx)
		if sc.GCode == "" {
			eskiKod, _ := meta["g_code"].(string)
			sc.GCode = validGA4Measurement(eskiKod)
		}
	}
	sc.AdSense = jsonStr(bag["adsense"])
	sc.RobotsHTML = jsonStr(bag["robots_html"])
	sc.AIProvider = jsonStr(aiSet["default_provider"])
	sc.AIModel = jsonStr(aiSet["model"])
	sc.AIBaseURL = jsonStr(aiSet["base_url"])
	sc.AIEnabled = s.ai.Enabled()
	if sc.AIEnabled {
		sc.AIProvider = strings.Join(s.ai.Providers(), ", ")
	}
	if types, err := repo.ListContentTypes(ctx, s.pool); err == nil {
		sc.Types = types
	}
	sc.IzinKeys = []string{"Okunma", "Paylaşım", "Yorumlar", "Sonsuz Scroll", "Editör", "Gözden Kaçmasın", "İlginizi Çekebilir", "Trend", "Son Eklenenler"}

	// Künye: formda SAKLANAN değer gösterilir, iletişim yedeği ayrı
	// gösterilir ki editör neyin nereden geldiğini bilsin. Eksik listesi
	// ise gerçek çıktıya göre (yedekler dahil) hesaplanır — panelde
	// "eksik" uyarısı ile canlıdaki künye aynı şeyi söylemeli.
	kunyeSet, _ := repo.GetSettings(ctx, s.pool, "kunye")
	for _, a := range bikKunyeAlanlari() {
		a.Value = strings.TrimSpace(bikSettingStr(kunyeSet, a.Key))
		sc.Kunye = append(sc.Kunye, a)
	}
	sc.KunyeEksik = bikKunyeEksikler(s.bikKunyeDoldur(ctx))
	c := s.contactConfig(ctx)
	sc.KunyeYedek = map[string]string{
		"telefon":      strings.TrimSpace(c.Telefon),
		"eposta":       strings.TrimSpace(c.Info),
		"yonetim_yeri": strings.TrimSpace(c.Adres),
	}

	if logs, err := repo.ListLogs(ctx, s.pool, 60); err == nil {
		sc.Logs = logs
	}
	d.Ayar = sc
	return d
}

// settingsResult form kaydı/log temizliği sonrası paneli tazeler.
func (s *Server) settingsResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	tab := strings.TrimSpace(r.FormValue("tab"))
	if tab == "" {
		tab = r.URL.Query().Get("seke")
	}
	if tab == "" {
		tab = "genel"
	}
	if !IsHTMX(r) {
		s.flashRedirect(w, r, "/admin/ayarlar?seke="+tab, kind, text)
		return
	}
	r2 := withHTMXCurrentQuery(r)
	q := r2.URL.Query()
	q.Set("seke", tab)
	r2.URL.RawQuery = q.Encode()
	d := s.buildSettingsData(r2)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderAyarlarPanelV2(w, r, d)
}

// handleSettingsSave formu kaydeder.
func (s *Server) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()

	set := func(section, key string, v any) {
		b, _ := json.Marshal(v)
		_ = repo.SetSetting(ctx, s.pool, section, key, b)
	}

	// genel
	// Başlık/slogan Marka Merkezi'nden yönetilir; alanlar yalnız formda
	// varsa yazılır (Marka sayfasına taşındı — yoksa silinmez).
	if r.Form.Has("site_baslik") {
		set("genel", "site_baslik", map[string]string{"tr": r.FormValue("site_baslik")})
	}
	if r.Form.Has("slogan") {
		set("genel", "slogan", map[string]string{"tr": r.FormValue("slogan")})
	}
	set("genel", "dil", r.FormValue("dil"))
	set("genel", "bakim_modu", map[string]any{"aktif": r.FormValue("bakim") != "", "mesaj": r.FormValue("bakim_mesaji")})
	// gorsel — tema renk/fontları (themeConfig üzerinden uygulanır).
	// NOT: site_font/baslik_font anahtarları artık yazılmıyor; ölü anahtarlardı
	// (tema fontları theme_font/theme_heading_font üzerinden çalışır).
	if r.FormValue("theme_reset") != "" {
		set("gorsel", "theme", ThemeConfig{})
	} else {
		theme := &ThemeConfig{
			Accent:        r.FormValue("theme_accent"),
			AccentDark:    r.FormValue("theme_accent_dark"),
			HeaderBg:      r.FormValue("theme_header_bg"),
			MenuColor:     r.FormValue("theme_menu_color"),
			BodyBg:        r.FormValue("theme_body_bg"),
			Surface:       r.FormValue("theme_surface"),
			Surface2:      r.FormValue("theme_surface_2"),
			Text:          r.FormValue("theme_text"),
			TextMuted:     r.FormValue("theme_text_muted"),
			Border:        r.FormValue("theme_border"),
			FooterBg:      r.FormValue("theme_footer_bg"),
			FooterLink:    r.FormValue("theme_footer_link"),
			FooterHeading: r.FormValue("theme_footer_heading"),
			DarkBodyBg:    r.FormValue("theme_dark_body_bg"),
			DarkSurface:   r.FormValue("theme_dark_surface"),
			DarkSurface2:  r.FormValue("theme_dark_surface_2"),
			DarkText:      r.FormValue("theme_dark_text"),
			DarkTextMuted: r.FormValue("theme_dark_text_muted"),
			DarkBorder:    r.FormValue("theme_dark_border"),
			DarkHeaderBg:  r.FormValue("theme_dark_header_bg"),
			DarkFooterBg:  r.FormValue("theme_dark_footer_bg"),
			Font:          r.FormValue("theme_font"),
			HeadingFont:   r.FormValue("theme_heading_font"),
		}
		if n := parseIntPtr(r.FormValue("theme_radius")); n != nil {
			theme.Radius = *n
			theme.RadiusSet = true
		}
		theme.normalize()
		set("gorsel", "theme", theme)
	}
	// moduller
	set("moduller", "anasayfa_sirasi", parseJSONList(r.FormValue("modul_sirasi")))
	set("moduller", "izinler", parseIzinler(r))
	// baglantilar
	//
	// g_code artık kanonik anahtara yazılır (trend/ga4_olcum): Sistem
	// sayfasındaki alan "Google Arama Kodu (G-TAG)" diyordu ama değer
	// hiçbir yerde basılmıyordu — kaydedilip unutulan ölü ayardı.
	// Doğrulama: biçimi tutmayan değer (örn. "abc") ayarı bozmaz.
	if r.Form.Has("g_code") {
		if kod := validGA4Measurement(r.FormValue("g_code")); kod != "" {
			_ = repo.SetTrendSetting(ctx, s.pool, "ga4_olcum", kod)
		}
	}
	set("baglantilar", "meta", map[string]string{
		"g_code": r.FormValue("g_code"), "g_verification": strings.TrimSpace(r.FormValue("g_verification")),
	})
	set("baglantilar", "adsense", r.FormValue("adsense"))
	set("baglantilar", "robots_html", r.FormValue("robots_html"))
	// yapay zeka (api anahtarı DB'ye yazılmaz; env — burada yalnız not)
	set("yapay_zeka", "default_provider", r.FormValue("ai_provider"))
	set("yapay_zeka", "model", r.FormValue("ai_model"))

	// kunye — BİK tebliği md. 8/1-d. Alan yalnızca formda VARSA yazılır:
	// başka bir sekme kaydedilirken bu bölümün sessizce silinmesini
	// engeller (Marka alanlarındaki aynı desen).
	s.saveKunyeSettings(r, set)

	s.audit(r, "ayar_kaydet", "Ayarlar güncellendi")
	s.settingsResult(w, r, "info", "Ayarlar kaydedildi.")
}

// saveKunyeSettings künye alanlarını kaydeder.
//
// Boş bırakılan alan settings'ten SİLİNMEZ, boş dize olarak yazılır: böylece
// iletişim ayarlarından gelen yedek yeniden devreye girer ve editör "buraya
// özel değer girmek istemiyorum" kararını verebilir. Kayıt yalnızca ilgili
// form alanı gönderildiğinde yapılır.
func (s *Server) saveKunyeSettings(r *http.Request, set func(section, key string, v any)) {
	for _, a := range bikKunyeAlanlari() {
		ad := "kunye_" + a.Key
		if !r.Form.Has(ad) {
			continue
		}
		set("kunye", a.Key, strings.TrimSpace(r.FormValue(ad)))
	}
}

// handleSettingsLogsClear işlem kayıtlarını temizler.
func (s *Server) handleSettingsLogsClear(w http.ResponseWriter, r *http.Request) {
	if err := repo.ClearLogs(r.Context(), s.pool); err != nil {
		s.settingsResult(w, r, "error", "Temizlenemedi.")
		return
	}
	s.settingsResult(w, r, "info", "Kayıtlar temizlendi.")
}

// audit panel işlemini kaydeder.
func (s *Server) audit(r *http.Request, action, detail string) {
	u := userFromContext(r.Context())
	var uid *int64
	if u.ID != 0 {
		uid = &u.ID
	}
	_ = repo.AddAuditLog(r.Context(), s.pool, uid, action, detail)
}

// ---- yardımcılar ----

func jsonStr(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	return ""
}

func obj(b []byte) map[string]any {
	if len(b) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) == nil {
		return m
	}
	return nil
}

func prettyJSON(b []byte) string {
	if len(b) == 0 {
		return "[]"
	}
	var v any
	if json.Unmarshal(b, &v) != nil {
		return string(b)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(b)
	}
	return string(out)
}

func parseJSONList(s string) []string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var l []string
		if json.Unmarshal([]byte(s), &l) == nil {
			return l
		}
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' })
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(strings.Trim(p, "\"")); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseIzinler sekme formundan tür->anahtar(lar) matrisini üretir.
func parseIzinler(r *http.Request) map[string][]string {
	out := map[string][]string{}
	for _, tip := range r.Form["izin_tip"] {
		for _, k := range r.Form["izin_"+tip] {
			out[tip] = append(out[tip], k)
		}
	}
	return out
}

var _ = context.Background
