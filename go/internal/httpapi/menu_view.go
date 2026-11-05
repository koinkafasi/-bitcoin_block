package httpapi

import (
	"net/http"
	"strconv"

	"theblock/site/internal/webui/templkit"
)

// menuViewFrom menuEditData → templkit.MenuView eşlemesi.
func menuViewFrom(m *menuEditData, fl *flashMsg) templkit.MenuView {
	kind, text := "", ""
	if fl != nil {
		kind, text = fl.Kind, fl.Text
	}
	v := templkit.MenuView{FlashKind: kind, FlashText: text}
	if m == nil {
		return v
	}
	for _, r := range m.Rows {
		v.Rows = append(v.Rows, templkit.MenuRowV{
			Slug: r.Slug, URL: r.URL, Label: r.Label, Name: r.Name, Sub: r.Sub,
		})
	}
	for _, c := range m.Categories {
		if c.ParentID != nil {
			continue // yalnız üst kategoriler eklenebilir
		}
		v.Cats = append(v.Cats, templkit.MenuCatV{Slug: c.Slug, Name: c.Name})
	}
	for _, o := range m.MegaOptions {
		v.MegaOptions = append(v.MegaOptions, templkit.MenuDesignOpt{
			Key: o.Key, Name: o.Name, Desc: o.Desc, Selected: o.Key == m.MegaDesign,
		})
	}
	for _, o := range m.CatNavOptions {
		v.CatNav = append(v.CatNav, templkit.MenuDesignOpt{
			Key: o.Key, Name: o.Name, Desc: o.Desc, Selected: o.Key == m.CatNavDesign,
		})
	}
	if bg := m.Bg; bg != nil && bg.Config != nil {
		overlay := 0
		if bg.Config.Overlay != nil {
			overlay = *bg.Config.Overlay
		}
		v.Bg = templkit.MenuBgV{
			HasCustom: bg.HasCustom,
			URL:       bg.URL,
			Disabled:  bg.Config.Disabled,
			Size:      bg.Config.Size,
			Position:  bg.Config.Position,
			Repeat:    bg.Config.Repeat,
			Attach:    bg.Config.Attach,
			Overlay:   strconv.Itoa(overlay),
		}
	}
	return v
}

// renderMenuPanelV2 menü panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderMenuPanelV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPanel(w, r, templkit.MenuPanel(menuViewFrom(d.MenuEdit, d.Flash)))
}

// renderMenuPageV2 menü panelini tam sayfa (admin kabuğu) yazar.
func (s *Server) renderMenuPageV2(w http.ResponseWriter, r *http.Request, d *tmplData) {
	s.renderTemplPage(w, r, d, templkit.MenuPanel(menuViewFrom(d.MenuEdit, d.Flash)))
}
