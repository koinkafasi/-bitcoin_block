package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// FootLink tek footer bağlantısı. slug tanımlıysa kategoriye, yoksa URL'e gider.
type FootLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Slug  string `json:"slug,omitempty"`
}

// FootCol footer sütunu: başlık + sıralı linkler.
type FootCol struct {
	Title string     `json:"title"`
	Links []FootLink `json:"links"`
}

type FooterConfig struct {
	LogoPos     string     `json:"logo"`
	Layout      string     `json:"model"` // klasik | compact | sade
	Skin        string     `json:"skin"`  // auto | dark | light | accent
	Tagline     string     `json:"tagline"`
	BottomText  string     `json:"bottom_text"`
	ShowSocial  bool       `json:"show_social"`
	ShowBrand   bool       `json:"show_brand"`
	ShowTagline bool       `json:"show_tagline"`
	ShowBottom  bool       `json:"show_bottom"`
	Configured  bool       `json:"configured"`
	Strip       []FootLink `json:"strip"`
	Cols        []FootCol  `json:"cols"`
}

// footerEditData /admin/footer sayfası verisi.
type footerEditData struct {
	Categories []repo.Category
	Config     *FooterConfig
}

// urlAbs göreli adresi dil köküyle tamamlar; http(s) ise aynen bırakır.
func urlAbs(href, base string) string {
	href = trimSpace(href)
	if href == "" {
		return "#"
	}
	// Dilim taşmasını önlemek için her kontrolün uzunluğu ayrı korunur.
	if len(href) >= 8 && href[:8] == "https://" {
		return href
	}
	if len(href) >= 7 && href[:7] == "http://" {
		return href
	}
	if len(href) >= 2 && href[:2] == "//" {
		return href
	}
	if href[0] != '/' {
		href = "/" + href
	}
	return base + href
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

func normalizePageURL(u string) string {
	if u == "" {
		return u
	}
	// "/sayfa/x" veya "/en/sayfa/x" → "/x" veya "/en/x"
	if strings.HasPrefix(u, "/sayfa/") {
		return u[len("/sayfa"):]
	}
	if len(u) > len("/sayfa/") && u[0] == '/' {
		if i := strings.Index(u, "/sayfa/"); i == 3 {
			return u[:3] + u[i+len("/sayfa"):]
		}
	}
	return u
}

func (s *Server) footerConfig(ctx context.Context) *FooterConfig {
	if v := s.cachedSetting(ctx, "footer", "ana"); len(v) > 0 {
		var c FooterConfig
		if json.Unmarshal(v, &c) == nil {
			c.normalize()
			ensureFooterRSS(&c)
			for i := range c.Strip {
				c.Strip[i].URL = normalizePageURL(c.Strip[i].URL)
			}
			for i := range c.Cols {
				for j := range c.Cols[i].Links {
					c.Cols[i].Links[j].URL = normalizePageURL(c.Cols[i].Links[j].URL)
				}
			}
			// Alt metin/slogan {domain} gibi yer tutucular taşıyabilir.
			c.Tagline = s.applyContactTokens(ctx, c.Tagline)
			c.BottomText = s.applyContactTokens(ctx, c.BottomText)
			return &c
		}
	}
	fc := &FooterConfig{
		LogoPos:    "right",
		Layout:     "klasik",
		Skin:       "auto",
		ShowSocial: true, ShowBrand: true, ShowTagline: true, ShowBottom: true,
		Configured: true,
		Strip: []FootLink{
			{Label: "Bize Ulaşın", URL: "/bize-ulasin"},
			{Label: "Künyemiz", URL: "/kunye"},
			{Label: "Arşivler", URL: "/arsiv"},
			{Label: "Gizlilik Politikası", URL: "/gizlilik-politikasi"},
		},
		Cols: []FootCol{
			{Title: "Sayfalar", Links: []FootLink{
				{Label: "Akış", URL: "/akis"},
				{Label: "Arşiv", URL: "/arsiv"},
				{Label: "Piyasa ve Borsa", URL: "/borsa"},
			}},
			{Title: "Kurumsal", Links: []FootLink{
				{Label: "Çerez Politikası", URL: "/cerez-politikasi"},
				{Label: "Kullanım Koşulları", URL: "/kullanim-kosullari"},
				{Label: "Gizlilik Politikası", URL: "/gizlilik-politikasi"},
				{Label: "KVKK Aydınlatma", URL: "/kvkk"},
				{Label: "İlgili Kişi Başvuru Formu", URL: "/kvkk-basvuru"},
				{Label: "Telif Hakları", URL: "/telif-haklari"},
				{Label: "Editoryal İlkeler", URL: "/editoryal-ilkeler"},
				{Label: "Bilgi Toplum Hizmetleri", URL: "/bilgi-toplum-hizmetleri"},
				{Label: "Künyemiz", URL: "/kunye"},
				{Label: "Bize Ulaşın", URL: "/iletisim"},
				{Label: "Kariyer", URL: "/insan-kaynaklari"},
				{Label: "Sıkça Sorulan Sorular", URL: "/sikca-sorulan-sorular"},
				{Label: "Bülten", URL: "/bulten"},
				{Label: "RSS Akışları", URL: "/rss-listesi"},
			}},
		},
	}
	cats := s.cachedCategories(ctx, siteIDFromCtx(ctx))
	var kl, extra []FootLink
	for _, c := range cats {
		if c.Slug == "" || c.Slug == "diger" || c.Status != "yayinda" {
			continue
		}
		if len(kl) < 8 {
			kl = append(kl, FootLink{Slug: c.Slug, Label: c.Name})
		} else {
			extra = append(extra, FootLink{Slug: c.Slug, Label: c.Name})
		}
	}
	var pre []FootCol
	if len(kl) > 0 {
		pre = append(pre, FootCol{Title: "Kategoriler", Links: kl})
	}
	if len(extra) > 0 {
		pre = append(pre, FootCol{Title: "Hızlı Linkler", Links: extra})
	}
	fc.Cols = append(pre, fc.Cols...)
	fc.normalize()
	ensureFooterRSS(fc)
	return fc
}

// ensureFooterRSS, RSS Akışları bağlantısını Kurumsal sütununa ekler.
// Eski kayıtlarda sütun yoksa oluşturur; mevcut bağlantı varsa çoğaltmaz.
func ensureFooterRSS(c *FooterConfig) {
	for i := range c.Cols {
		if !strings.EqualFold(strings.TrimSpace(c.Cols[i].Title), "Kurumsal") {
			continue
		}
		for _, l := range c.Cols[i].Links {
			if strings.TrimSpace(l.URL) == "/rss-listesi" || strings.EqualFold(strings.TrimSpace(l.Slug), "rss") {
				return
			}
		}
		c.Cols[i].Links = append(c.Cols[i].Links, FootLink{Label: "RSS Akışları", URL: "/rss-listesi"})
		return
	}
	c.Cols = append(c.Cols, FootCol{
		Title: "Kurumsal",
		Links: []FootLink{{Label: "RSS Akışları", URL: "/rss-listesi"}},
	})
}

func (c *FooterConfig) normalize() {
	if c.LogoPos != "left" && c.LogoPos != "center" && c.LogoPos != "right" {
		c.LogoPos = "right"
	}
	switch c.Layout {
	case "klasik", "compact", "sade":
	default:
		c.Layout = "klasik"
	}
	switch c.Skin {
	case "auto", "dark", "light", "accent":
	default:
		c.Skin = "auto"
	}
	if !c.Configured {
		// Eski kayıtlarda yeni görünürlük alanları yok: eski footer
		// görünümünü korumak için hepsini aç.
		c.ShowSocial, c.ShowBrand, c.ShowTagline, c.ShowBottom = true, true, true, true
		c.Configured = true
	}
}

// handleFooterGet /admin/footer — profesyonel footer yöneticisi.
func (s *Server) handleFooterGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildFooterData(r)
	if IsHTMX(r) {
		s.renderFooterPanelV2(w, r, d)
		return
	}
	s.renderFooterPageV2(w, r, d)
}

func (s *Server) buildFooterData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Footer Yönetimi")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ctx := r.Context()
	cats, _ := repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	d.FooterEdit = &footerEditData{Categories: cats, Config: s.footerConfig(ctx)}
	return d
}

func (s *Server) renderFooterPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildFooterData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderFooterPanelV2(w, r, d)
}

// handleFooterSave POST /admin/footer/kaydet — JSON {config}.
func (s *Server) handleFooterSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cfg FooterConfig
	if IsHTMX(r) {
		_ = r.ParseForm()
		raw := r.FormValue("config")
		if raw == "" || json.Unmarshal([]byte(raw), &cfg) != nil {
			s.renderFooterPanel(w, r, "error", "Footer verisi geçersiz.")
			return
		}
	} else {
		defer r.Body.Close()
		var body struct {
			Config FooterConfig `json:"config"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&body); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "JSON geçersiz: " + err.Error()})
			return
		}
		cfg = body.Config
	}
	cfg.normalize()
	raw, err := json.Marshal(cfg)
	if err != nil {
		if IsHTMX(r) {
			s.renderFooterPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "footer", "ana", raw); err != nil {
		if IsHTMX(r) {
			s.renderFooterPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderFooterPanel(w, r, "info", "Footer kaydedildi.")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
