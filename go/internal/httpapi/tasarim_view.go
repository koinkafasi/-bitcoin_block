package httpapi

import (
	"encoding/json"
	"net/http"

	"theblock/site/internal/webui/templkit"
)

// headerViewFrom HeaderConfig → templkit.HeaderView eşlemesi.
func headerViewFrom(cfg *HeaderConfig, fl *flashMsg) templkit.HeaderView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.HeaderView{FlashKind: kind, FlashText: text}
	if cfg == nil {
		return v
	}
	v.Model = cfg.Model
	v.Skin = cfg.Skin
	v.Width = cfg.Width
	v.Sticky = cfg.Sticky
	v.Topbar = cfg.Topbar
	v.TopbarText = cfg.TopbarText
	v.ShowTagline = cfg.ShowTagline
	v.ShowHome = cfg.ShowHome
	v.ShowSearch = cfg.ShowSearch
	v.ShowTheme = cfg.ShowTheme
	v.ShowMenu = cfg.ShowMenu
	v.HideMember = cfg.HideMember

	// header.js için yapılandırma JSON'u (orijinal anahtar adlarıyla).
	buttons := []map[string]string{}
	for _, b := range cfg.Buttons {
		buttons = append(buttons, map[string]string{
			"label": b.Label, "url": b.URL, "style": b.Style,
		})
	}
	raw, _ := json.Marshal(map[string]any{
		"model":        cfg.Model,
		"skin":         cfg.Skin,
		"width":        cfg.Width,
		"sticky":       cfg.Sticky,
		"topbar":       cfg.Topbar,
		"topbar_text":  cfg.TopbarText,
		"show_tagline": cfg.ShowTagline,
		"show_search":  cfg.ShowSearch,
		"show_theme":   cfg.ShowTheme,
		"show_menu":    cfg.ShowMenu,
		"show_home":    cfg.ShowHome,
		"hide_member":  cfg.HideMember,
		"buttons":      buttons,
	})
	v.ConfigJSON = string(raw)
	return v
}

// renderHeaderPanelV2 header panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderHeaderPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.HeaderPanel(headerViewFrom(d.Header, d.Flash)))
}

// renderHeaderPageV2 header panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderHeaderPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.HeaderPanel(headerViewFrom(d.Header, d.Flash)))
}

// footerViewFrom footerEditData → templkit.FooterView eşlemesi.
func footerViewFrom(f *footerEditData, fl *flashMsg) templkit.FooterView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.FooterView{FlashKind: kind, FlashText: text}
	if f == nil || f.Config == nil {
		return v
	}
	c := f.Config
	v.Layout = c.Layout
	v.Skin = c.Skin
	v.LogoPos = c.LogoPos
	v.Tagline = c.Tagline
	v.BottomText = c.BottomText
	v.ShowSocial = c.ShowSocial
	v.ShowBrand = c.ShowBrand
	v.ShowTagline = c.ShowTagline
	v.ShowBottom = c.ShowBottom
	for _, l := range c.Strip {
		v.Strip = append(v.Strip, templkit.FooterLinkV{Slug: l.Slug, Label: l.Label, URL: l.URL})
	}
	for _, col := range c.Cols {
		cv := templkit.FooterColV{Title: col.Title}
		for _, l := range col.Links {
			cv.Links = append(cv.Links, templkit.FooterLinkV{Slug: l.Slug, Label: l.Label, URL: l.URL})
		}
		v.Cols = append(v.Cols, cv)
	}
	for _, cat := range f.Categories {
		if cat.ParentID != nil {
			continue
		}
		v.Cats = append(v.Cats, templkit.MenuCatV{Slug: cat.Slug, Name: cat.Name})
	}
	return v
}

// renderFooterPanelV2 footer panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderFooterPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.FooterPanel(footerViewFrom(d.FooterEdit, d.Flash)))
}

// renderFooterPageV2 footer panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderFooterPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.FooterPanel(footerViewFrom(d.FooterEdit, d.Flash)))
}
