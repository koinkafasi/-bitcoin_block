package httpapi

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/webui/templkit"
)

// kategorilerViewFrom catsData → templkit.KategorilerView eşlemesi.
func kategorilerViewFrom(c *catsData) templkit.KategorilerView {
	v := templkit.KategorilerView{}
	for _, p := range c.Parents {
		v.NewParents = append(v.NewParents, templkit.IOpt{
			Value: strconv.FormatInt(p.ID, 10), Label: p.Name,
		})
	}
	for _, it := range c.Items {
		row := templkit.KategoriRow{
			ID:           strconv.FormatInt(it.ID, 10),
			Name:         it.Name,
			Slug:         it.Slug,
			Sort:         strconv.Itoa(it.SortOrder),
			Child:        strVal(it.ParentSlug) != "",
			UstAd:        strVal(it.ParentName),
			OzelVar:      it.CustomDesign,
			OzelTasarim:  it.LayoutPreset == "" && it.CustomDesign,
			DesignName:   it.DesignName,
			ContentCount: strconv.Itoa(it.ContentCount),
			ChildCount:   strconv.Itoa(it.ChildCount),
			InHeader:     it.InHeader,
			InMenu:       it.InMenu,
			InFooter:     it.InFooter,
			MinSkor:      strconv.Itoa(it.MinScore),
			AnaSkor:      strconv.Itoa(it.HomepageMinScore),
			Ikon:         it.Icon,
			GorselServis: strVal(it.ImageService),
			AdEN:         strVal(it.NameEN),
			AdES:         strVal(it.NameES),
			Etiket:       strVal(it.Label),
		}
		// Kategori varsayilan abonelik katmani secenekleri.
		for _, k := range []struct {
			val int
			lbl string
		}{
			{0, "Katman kilidi yok"},
			{1, "Filiz (1) ve üzeri"},
			{2, "Bronz (2) ve üzeri"},
			{3, "Gümüş (3) ve üzeri"},
			{4, "Altın (4) ve üzeri"},
			{5, "Elmas (5)"},
		} {
			row.Katmanlar = append(row.Katmanlar, templkit.IOpt{
				Value:    strconv.Itoa(k.val),
				Label:    k.lbl,
				Selected: it.MinTierWeight == k.val,
			})
		}
		for _, p := range c.Parents {
			if p.ID == it.ID {
				continue
			}
			row.Parents = append(row.Parents, templkit.IOpt{
				Value:    strconv.FormatInt(p.ID, 10),
				Label:    p.Name,
				Selected: strVal(it.ParentSlug) != "" && p.Slug == strVal(it.ParentSlug),
			})
		}
		for _, g := range []struct{ val, lbl string }{
			{"", "Genel ayar"}, {"klasik", "Klasik"}, {"liste", "Sade liste"},
			{"izgara", "Izgara"}, {"mozaik", "Mozaik"},
		} {
			row.Gorunumler = append(row.Gorunumler, templkit.IOpt{
				Value: g.val, Label: g.lbl, Selected: it.ListView == g.val,
			})
		}
		for _, pr := range c.Presets {
			row.Sablonlar = append(row.Sablonlar, templkit.IOpt{
				Value: pr.Key, Label: pr.Name,
				Selected: pr.Key == it.LayoutPreset ||
					(pr.Key == "genel" && it.LayoutPreset == "" && !it.CustomDesign),
			})
		}
		v.Rows = append(v.Rows, row)
	}
	return v
}

// renderKategorilerPanel kategoriler panelini HTMX fragment'ı olarak yazar.
func (s *Server) renderKategorilerPanel(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := kategorilerViewFrom(d.Cats)
	if d.Flash != nil {
		v.FlashKind = d.Flash.Kind
		v.FlashText = d.Flash.Text
	}
	var sb strings.Builder
	if err := templkit.KategorilerPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("kategoriler render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	_, _ = w.Write([]byte(sb.String()))
}

// renderKategorilerPage kategorileri tam sayfa (admin kabuğu + panel) yazar.
func (s *Server) renderKategorilerPage(w http.ResponseWriter, r *http.Request, d *tmplData) {
	v := kategorilerViewFrom(d.Cats)
	if d.Flash != nil {
		v.FlashKind = d.Flash.Kind
		v.FlashText = d.Flash.Text
	}
	var sb strings.Builder
	if err := templkit.KategorilerPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("kategoriler render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	d.SlotHTML = template.HTML(sb.String())
	s.render(w, r, "admin_bilesenler", d)
}
