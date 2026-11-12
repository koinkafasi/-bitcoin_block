package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"strings"
)

// ThemeConfig site genelindeki renk, font ve köşe ayarları.
// Boş renk alanları public.css fallback değerini kullanır.
type ThemeConfig struct {
	Accent        string `json:"accent"`
	AccentDark    string `json:"accent_dark"`
	HeaderBg      string `json:"header_bg"`
	MenuColor     string `json:"menu_color"`
	BodyBg        string `json:"body_bg"`
	Surface       string `json:"surface"`
	Surface2      string `json:"surface_2"`
	Text          string `json:"text"`
	TextMuted     string `json:"text_muted"`
	Border        string `json:"border"`
	FooterBg      string `json:"footer_bg"`
	FooterLink    string `json:"footer_link"`
	FooterHeading string `json:"footer_heading"`
	DarkBodyBg    string `json:"dark_body_bg"`
	DarkSurface   string `json:"dark_surface"`
	DarkSurface2  string `json:"dark_surface_2"`
	DarkText      string `json:"dark_text"`
	DarkTextMuted string `json:"dark_text_muted"`
	DarkBorder    string `json:"dark_border"`
	DarkHeaderBg  string `json:"dark_header_bg"`
	DarkFooterBg  string `json:"dark_footer_bg"`
	Radius        int    `json:"radius"` // 0 = varsayılan
	RadiusSet     bool   `json:"radius_set"`
	Font          string `json:"font"`
	HeadingFont   string `json:"heading_font"`
}

var themeFontFamilies = map[string]string{
	"":          "",
	"system":    "system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif",
	"poppins":   "Poppins, system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif",
	"serif":     "Georgia, Times New Roman, Times, serif",
	"sans":      "Arial, Helvetica, sans-serif",
	"condensed": "Arial Narrow, Roboto Condensed, Impact, sans-serif",
	"display":   "Trebuchet MS, Segoe UI, Arial, sans-serif",
	"mono":      "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
}

func (s *Server) themeConfig(ctx context.Context) *ThemeConfig {
	if v := s.cachedSetting(ctx, "gorsel", "theme"); len(v) > 0 {
		var c ThemeConfig
		if json.Unmarshal(v, &c) == nil {
			c.normalize()
			return &c
		}
	}
	c := &ThemeConfig{}
	return c
}

func (c *ThemeConfig) normalize() {
	c.Accent = sanitizeHexColor(c.Accent)
	c.AccentDark = sanitizeHexColor(c.AccentDark)
	c.HeaderBg = sanitizeHexColor(c.HeaderBg)
	c.MenuColor = sanitizeHexColor(c.MenuColor)
	c.BodyBg = sanitizeHexColor(c.BodyBg)
	c.Surface = sanitizeHexColor(c.Surface)
	c.Surface2 = sanitizeHexColor(c.Surface2)
	c.Text = sanitizeHexColor(c.Text)
	c.TextMuted = sanitizeHexColor(c.TextMuted)
	c.Border = sanitizeHexColor(c.Border)
	c.FooterBg = sanitizeHexColor(c.FooterBg)
	c.FooterLink = sanitizeHexColor(c.FooterLink)
	c.FooterHeading = sanitizeHexColor(c.FooterHeading)
	c.DarkBodyBg = sanitizeHexColor(c.DarkBodyBg)
	c.DarkSurface = sanitizeHexColor(c.DarkSurface)
	c.DarkSurface2 = sanitizeHexColor(c.DarkSurface2)
	c.DarkText = sanitizeHexColor(c.DarkText)
	c.DarkTextMuted = sanitizeHexColor(c.DarkTextMuted)
	c.DarkBorder = sanitizeHexColor(c.DarkBorder)
	c.DarkHeaderBg = sanitizeHexColor(c.DarkHeaderBg)
	c.DarkFooterBg = sanitizeHexColor(c.DarkFooterBg)
	c.Radius = clampInt(c.Radius, 0, 40)
	if _, ok := themeFontFamilies[c.Font]; !ok {
		c.Font = ""
	}
	if _, ok := themeFontFamilies[c.HeadingFont]; !ok {
		c.HeadingFont = ""
	}
}

// themeTokensCSS, theme ayarını public_head'teki :root bloğuna çevirir.
func (s *Server) themeTokensCSS(ctx context.Context) template.CSS {
	c := s.themeConfig(ctx)
	var p []string
	add := func(name, value string) {
		if value != "" {
			p = append(p, name+":"+value)
		}
	}
	add("--hp-tema", c.Accent)
	add("--hp-tema-koyu", c.AccentDark)
	add("--accent", c.Accent)
	// Üstünde beyaz yazı taşıyan dolgular için türetilmiş ton (Faz 180).
	add("--accent-fill", accentFill(c.Accent))
	// Açık zemin üstünde metin tonu (Faz 180).
	add("--accent-text", accentTextFill(c.Accent))
	add("--accent-text-dark", accentTextKoyu(c.Accent))
	add("--hp-header-bg", c.HeaderBg)
	add("--hp-menu-renk", c.MenuColor)
	add("--hp-body-bg", c.BodyBg)
	add("--hp-surface", c.Surface)
	add("--hp-surface-2", c.Surface2)
	add("--hp-text", c.Text)
	add("--hp-text-muted", c.TextMuted)
	add("--hp-border", c.Border)
	add("--hp-footer-bg", c.FooterBg)
	add("--hp-footer-link", c.FooterLink)
	add("--hp-footer-baslik", c.FooterHeading)
	if c.RadiusSet || c.Radius > 0 {
		sm := c.Radius - 2
		if sm < 0 {
			sm = 0
		}
		add("--hp-radius", px(c.Radius))
		add("--hp-radius-sm", px(sm))
	}
	if f := themeFontFamilies[c.Font]; f != "" {
		add("--hp-font", f)
	}
	if f := themeFontFamilies[c.HeadingFont]; f != "" {
		add("--hp-heading-font", f)
	}
	var d []string
	dadd := func(name, value string) {
		if value != "" {
			d = append(d, name+":"+value)
		}
	}
	dadd("--hp-body-bg", c.DarkBodyBg)
	dadd("--hp-surface", c.DarkSurface)
	dadd("--hp-surface-2", c.DarkSurface2)
	dadd("--hp-text", c.DarkText)
	dadd("--hp-text-muted", c.DarkTextMuted)
	dadd("--hp-border", c.DarkBorder)
	dadd("--hp-header-bg", c.DarkHeaderBg)
	dadd("--hp-footer-bg", c.DarkFooterBg)
	if len(p) == 0 && len(d) == 0 {
		return ""
	}
	var out strings.Builder
	if len(p) > 0 {
		out.WriteString(":root{" + strings.Join(p, ";") + ";}")
	}
	if len(d) > 0 {
		out.WriteString(`html[data-theme="dark"]{` + strings.Join(d, ";") + ";}")
	}
	return template.CSS(out.String())
}
