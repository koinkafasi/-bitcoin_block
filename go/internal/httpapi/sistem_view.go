package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"theblock/site/internal/i18n"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// hasStrList yetki matrisinde (map[tür][]anahtar) üyelik kontrolü.
func hasStrList(m map[string][]string, key, val string) bool {
	for _, s := range m[key] {
		if s == val {
			return true
		}
	}
	return false
}

// ayarRenkAlani tema renk alanı tanımı (ad → etiket → varsayılan).
type ayarRenkTanım struct {
	Name, Label, Def string
	Number           bool
}

var ayarAcikRenkler = []ayarRenkTanım{
	{"theme_accent", "Tema / vurgu rengi", "#d10015", false},
	{"theme_accent_dark", "Koyu vurgu rengi", "#a30011", false},
	{"theme_header_bg", "Header arka planı", "#d10015", false},
	{"theme_menu_color", "Header menü yazısı", "#ffffff", false},
	{"theme_body_bg", "Sayfa arka planı", "#f3f3f3", false},
	{"theme_surface", "Kart / yüzey", "#ffffff", false},
	{"theme_surface_2", "İkincil yüzey", "#fafafa", false},
	{"theme_text", "Ana metin", "#18181b", false},
	{"theme_text_muted", "Soluk metin", "#71717a", false},
	{"theme_border", "Kenarlık", "#e4e4e7", false},
	{"theme_footer_bg", "Footer arka planı", "#18181b", false},
	{"theme_footer_link", "Footer link rengi", "#a1a1aa", false},
	{"theme_footer_heading", "Footer başlık rengi", "#a1a1aa", false},
	{"theme_radius", "Köşe yuvarlaklığı (px)", "", true},
}

var ayarKoyuRenkler = []ayarRenkTanım{
	{"theme_dark_body_bg", "Sayfa arka planı", "#000000", false},
	{"theme_dark_surface", "Kart / yüzey", "#18181b", false},
	{"theme_dark_surface_2", "İkincil yüzey", "#1f1f23", false},
	{"theme_dark_text", "Ana metin", "#f4f4f5", false},
	{"theme_dark_text_muted", "Soluk metin", "#a1a1aa", false},
	{"theme_dark_border", "Kenarlık", "#27272a", false},
	{"theme_dark_header_bg", "Header arka planı", "#0b0b0d", false},
	{"theme_dark_footer_bg", "Footer arka planı", "#18181b", false},
}

var ayarFontSecenekleri = []struct{ Value, Label string }{
	{"", "Varsayılan (Poppins)"}, {"system", "Sistem"}, {"poppins", "Poppins"},
	{"serif", "Serif"}, {"sans", "Sans"}, {"condensed", "Condensed"},
	{"display", "Display"}, {"mono", "Mono"},
}

// ayarlarViewFrom settingsData → templkit.AyarlarView eşlemesi.
func ayarlarViewFrom(sc *settingsData, locales []i18n.Locale, fl *flashMsg) templkit.AyarlarView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.AyarlarView{FlashKind: kind, FlashText: text}
	if sc == nil {
		return v
	}
	v.SiteBaslik = sc.SiteBaslik
	v.Slogan = sc.Slogan
	v.Dil = sc.Dil
	for _, l := range locales {
		v.Locales = append(v.Locales, templkit.IOpt{Value: l.Code, Label: l.NativeName, Selected: sc.Dil == l.Code})
	}
	v.BakimModu = sc.BakimModu
	v.BakimMesaji = sc.BakimMesaji

	th := sc.Theme
	themeVal := func(val string, def string) string {
		if val != "" {
			return val
		}
		return def
	}
	for _, t := range ayarAcikRenkler {
		av := ""
		if th != nil {
			av = themeField(th, t.Name)
		}
		v.Renkler = append(v.Renkler, templkit.AyarRenk{
			Name: t.Name, Label: t.Label, Value: themeVal(av, t.Def), Number: t.Number,
		})
	}
	for _, t := range ayarKoyuRenkler {
		av := ""
		if th != nil {
			av = themeField(th, t.Name)
		}
		v.KoyuRenkler = append(v.KoyuRenkler, templkit.AyarRenk{
			Name: t.Name, Label: t.Label, Value: themeVal(av, t.Def),
		})
	}
	// Tipografi: site fontu + başlık fontu
	fontOpts := func(cur string) []templkit.IOpt {
		var out []templkit.IOpt
		for _, f := range ayarFontSecenekleri {
			out = append(out, templkit.IOpt{Value: f.Value, Label: f.Label, Selected: cur == f.Value})
		}
		return out
	}
	curFont, curHead := "", ""
	if th != nil {
		curFont, curHead = th.Font, th.HeadingFont
	}
	v.Fonts = []templkit.AyarFont{
		{Name: "theme_font", Label: "Site fontu", Value: curFont, Options: fontOpts(curFont)},
		{Name: "theme_heading_font", Label: "Başlık fontu", Value: curHead, Options: fontOpts2(curHead)},
	}

	v.ModulSirasi = sc.ModulSirasi
	v.IzinKeys = sc.IzinKeys
	for _, t := range sc.Types {
		row := templkit.AyarIzinSatir{Key: t.Key, Label: t.Label}
		for _, k := range sc.IzinKeys {
			row.Cells = append(row.Cells, hasStrList(sc.IzinMap, t.Key, k))
		}
		v.IzinRows = append(v.IzinRows, row)
	}
	v.GCode = sc.GCode
	v.GVerification = sc.GVerification
	v.AdSense = sc.AdSense
	v.RobotsHTML = sc.RobotsHTML
	v.AIEnabled = sc.AIEnabled
	v.AIProvider = sc.AIProvider
	v.AIModel = sc.AIModel
	v.AIBaseURL = sc.AIBaseURL
	for _, p := range []struct{ Value, Label string }{
		{"omniroute", "OmniRoute"}, {"openai", "OpenAI"},
		{"anthropic", "Anthropic"}, {"gemini", "Gemini"},
	} {
		v.AIProviders = append(v.AIProviders, templkit.IOpt{
			Value: p.Value, Label: p.Label, Selected: sc.AIProvider == p.Value,
		})
	}
	for _, l := range sc.Logs {
		detail := ""
		if l.Detail != nil {
			detail = *l.Detail
		}
		v.Logs = append(v.Logs, templkit.AyarLog{
			Time: l.CreatedAt.Local().Format("02.01.2006 15:04"),
			User: l.UserName, Action: l.Action, Detail: detail,
		})
	}
	// Künye: saklanan değer + varsa iletişim yedeği. Yedek, alan boş
	// bırakıldığında canlıda ne basılacağını söyler.
	for _, a := range sc.Kunye {
		v.Kunye = append(v.Kunye, templkit.AyarKunye{
			Key: a.Key, Label: a.Label, Value: a.Value,
			Yedek: sc.KunyeYedek[a.Key],
		})
	}
	v.KunyeEksik = sc.KunyeEksik
	return v
}

// fontOpts2 başlık fontu seçenekleri (ilk seçenek farklı etiket).
func fontOpts2(cur string) []templkit.IOpt {
	var out []templkit.IOpt
	for i, f := range ayarFontSecenekleri {
		label := f.Label
		if i == 0 {
			label = "Site fontuyla aynı"
		}
		out = append(out, templkit.IOpt{Value: f.Value, Label: label, Selected: cur == f.Value})
	}
	return out
}

// themeField tema alan adından değeri okur (ad eşlemesi).
func themeField(th *ThemeConfig, name string) string {
	switch name {
	case "theme_accent":
		return th.Accent
	case "theme_accent_dark":
		return th.AccentDark
	case "theme_header_bg":
		return th.HeaderBg
	case "theme_menu_color":
		return th.MenuColor
	case "theme_body_bg":
		return th.BodyBg
	case "theme_surface":
		return th.Surface
	case "theme_surface_2":
		return th.Surface2
	case "theme_text":
		return th.Text
	case "theme_text_muted":
		return th.TextMuted
	case "theme_border":
		return th.Border
	case "theme_footer_bg":
		return th.FooterBg
	case "theme_footer_link":
		return th.FooterLink
	case "theme_footer_heading":
		return th.FooterHeading
	case "theme_radius":
		if th.RadiusSet || th.Radius > 0 {
			return strconv.Itoa(th.Radius)
		}
		return ""
	case "theme_dark_body_bg":
		return th.DarkBodyBg
	case "theme_dark_surface":
		return th.DarkSurface
	case "theme_dark_surface_2":
		return th.DarkSurface2
	case "theme_dark_text":
		return th.DarkText
	case "theme_dark_text_muted":
		return th.DarkTextMuted
	case "theme_dark_border":
		return th.DarkBorder
	case "theme_dark_header_bg":
		return th.DarkHeaderBg
	case "theme_dark_footer_bg":
		return th.DarkFooterBg
	}
	return ""
}

// sitelerViewFrom siteler verisi → templkit.SitelerView eşlemesi.
func sitelerViewFrom(st *sitesData, locales []i18n.Locale, fl *flashMsg) templkit.SitelerView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.SitelerView{FlashKind: kind, FlashText: text}
	if st != nil {
		for _, it := range st.Items {
			brand := ""
			if it.Brand != nil {
				brand = *it.Brand
			}
			v.Items = append(v.Items, templkit.SiteSatir{
				ID: strconv.FormatInt(it.ID, 10), Name: it.Name, Brand: brand,
				Domain: it.Domain, Locale: it.DefaultLocale, Active: it.Status == "aktif",
				Content: strconv.Itoa(it.ContentCount), Default: it.ID == 1,
			})
		}
		v.RPM = fmt.Sprintf("%.2f", st.RPM)
		for _, s := range st.Stats {
			v.Stats = append(v.Stats, siteStatSatir(s))
		}
		v.Tot = siteStatSatir(st.Tot)
	}
	for _, l := range locales {
		v.Locales = append(v.Locales, templkit.IOpt{Value: l.Code, Label: l.NativeName})
	}
	return v
}

// siteStatSatir tek site gelir satırını çevirir.
func siteStatSatir(s repo.SiteStat) templkit.SiteStatSatir {
	return templkit.SiteStatSatir{
		Name:      s.Name,
		Domain:    s.Domain,
		Content:   strconv.FormatInt(s.Content, 10),
		Published: strconv.FormatInt(s.Published, 10),
		Hits:      strconv.FormatInt(s.Hits, 10),
		Revenue:   fmt.Sprintf("%.2f", s.Revenue),
		AICost:    fmt.Sprintf("%.2f", s.AICost),
		Profit:    fmt.Sprintf("%.2f", s.Profit),
		Positive:  s.Profit >= 0,
	}
}

// ---- Render yardımcıları ----

func (s *Server) renderAyarlarPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.AyarlarPanel(ayarlarViewFrom(d.Ayar, d.Locales, d.Flash)))
}

func (s *Server) renderAyarlarPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.AyarlarPanel(ayarlarViewFrom(d.Ayar, d.Locales, d.Flash)))
}

func (s *Server) renderSitelerPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.SitelerPanel(sitelerViewFrom(d.Siteler, d.Locales, d.Flash)))
}

func (s *Server) renderSitelerPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.SitelerPanel(sitelerViewFrom(d.Siteler, d.Locales, d.Flash)))
}
